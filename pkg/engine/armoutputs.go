// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-engine/pkg/api"
	"github.com/Azure/aks-engine/pkg/api/common"
)

func GetKubernetesOutputs(cs *api.ContainerService) map[string]interface{} {
	outputs := map[string]interface{}{
		"resourceGroup": map[string]interface{}{
			"type":  "string",
			"value": "[variables('resourceGroup')]",
		},
		"vnetResourceGroup": map[string]interface{}{
			"type":  "string",
			"value": "[variables('virtualNetworkResourceGroupName')]",
		},
		"subnetName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('subnetName')]",
		},
		"securityGroupName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('nsgName')]",
		},
		"virtualNetworkName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('virtualNetworkName')]",
		},
		"routeTableName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('routeTableName')]",
		},
		"primaryAvailabilitySetName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('primaryAvailabilitySetName')]",
		},
		"primaryScaleSetName": map[string]interface{}{
			"type":  "string",
			"value": "[variables('primaryScaleSetName')]",
		},
	}

	isHostedMaster := cs.Properties.IsHostedMasterProfile()

	if !isHostedMaster {
		for k, v := range getMasterOutputs(cs) {
			outputs[k] = v
		}
	}

	for _, profile := range cs.Properties.AgentPoolProfiles {
		if profile.IsAvailabilitySets() && profile.IsStorageAccount() {
			agentName := profile.Name
			outputs[fmt.Sprintf("%sStorageAccountOffset", agentName)] = map[string]interface{}{
				"type":  "int",
				"value": fmt.Sprintf("[variables('%sStorageAccountOffset')]", agentName),
			}
			outputs[fmt.Sprintf("%sStorageAccountCount", agentName)] = map[string]interface{}{
				"type":  "int",
				"value": fmt.Sprintf("[variables('%sStorageAccountsCount')]", agentName),
			}
			outputs[fmt.Sprintf("%sSubnetName", agentName)] = map[string]interface{}{
				"type":  "string",
				"value": fmt.Sprintf("[variables('%sSubnetName')]", agentName),
			}
		}
	}

	if cs.Properties.OrchestratorProfile.KubernetesConfig.IsAddonEnabled(common.AppGwIngressAddonName) {
		outputs["applicationGatewayName"] = map[string]interface{}{
			"type":  "string",
			"value": "[variables('appGwName')]",
		}
		outputs["appGwIdentityResourceId"] = map[string]interface{}{
			"type":  "string",
			"value": "[variables('appGwICIdentityId')]",
		}
		outputs["appGwIdentityClientId"] = map[string]interface{}{
			"type":  "string",
			"value": "[reference(variables('appGwICIdentityId'), variables('apiVersionManagedIdentity')).clientId]",
		}
	}

	if cs.Properties.OrchestratorProfile.KubernetesConfig.SystemAssignedIDEnabled() {
		outputValue := "variables('vmasRoleAssignmentNames')"

		if cs.Properties.MasterProfile.IsAvailabilitySet() {

			// These master role assignments for agent pools have to directly here and can't be put into a variable first.
			// The reason is that they need the `reference` function which can't be used in a variable.
			var masterRoleAssignmentForAgentPools []string

			for _, agentPool := range cs.Properties.AgentPoolProfiles {
				resourceGroup := fmt.Sprintf("variables('%sSubnetResourceGroup')", agentPool.Name)

				for masterIdx := 0; masterIdx < cs.Properties.MasterProfile.Count; masterIdx++ {
					masterVMReference := fmt.Sprintf("reference(resourceId(resourceGroup().name, 'Microsoft.Compute/virtualMachines', concat(variables('masterVMNamePrefix'), %d)), '2017-03-30', 'Full').identity.principalId", masterIdx)

					masterRoleAssignmentForAgentPools = append(masterRoleAssignmentForAgentPools,
						fmt.Sprintf("resourceId(%s, 'Microsoft.Network/virtualNetworks/providers/roleAssignments', variables('%sVnet'), 'Microsoft.Authorization', guid(uniqueString(%s)))",
							resourceGroup, agentPool.Name, masterVMReference))
				}
			}

			outputValue += ", createArray(" + strings.Join(masterRoleAssignmentForAgentPools, ", ") + ")"
		}

		var vmssSysRoleAssignmentVariables []string

		for _, profile := range cs.Properties.AgentPoolProfiles {
			if profile.IsVirtualMachineScaleSets() {
				vmssSysRoleAssignmentVariables = append(vmssSysRoleAssignmentVariables,
					fmt.Sprintf("variables('%sVMSSSysRoleAssignmentName')", profile.Name))
			}
		}

		if len(vmssSysRoleAssignmentVariables) > 0 {
			outputValue += ", createArray(" + strings.Join(vmssSysRoleAssignmentVariables, ", ") + ")"
		}

		// TODO: Rename this output and all contributing variables (ARM & Go) to "ids".

		outputs["roleAssignmentNames"] = map[string]string {
			"type":  "array",
			"value": fmt.Sprintf("[concat(%s)]", outputValue),
		}
	}

	return outputs
}

func getMasterOutputs(cs *api.ContainerService) map[string]interface{} {
	outputs := map[string]interface{}{}
	masterFQDN := ""

	if !cs.Properties.OrchestratorProfile.IsPrivateCluster() {
		masterFQDN = "[reference(concat('Microsoft.Network/publicIPAddresses/', variables('masterPublicIPAddressName'))).dnsSettings.fqdn]"
	}

	outputs["masterFQDN"] = map[string]interface{}{
		"type":  "string",
		"value": masterFQDN,
	}

	if cs.Properties.HasVMASAgentPool() {
		outputs["agentStorageAccountSuffix"] = map[string]interface{}{
			"type":  "string",
			"value": "[variables('storageAccountBaseName')]",
		}
		outputs["agentStorageAccountPrefixes"] = map[string]interface{}{
			"type":  "array",
			"value": "[variables('storageAccountPrefixes')]",
		}
	}

	return outputs
}

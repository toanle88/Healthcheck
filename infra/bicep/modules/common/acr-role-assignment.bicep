param acrName string
param principalId string

resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = {
  name: acrName
}

// AcrPull role definition ID: 7f951dda-40cb-4ad4-9f49-b7897002d680
resource acrPullAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(acr.id, principalId, 'AcrPull')
  scope: acr
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '7f951dda-40cb-4ad4-9f49-b7897002d680')
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}

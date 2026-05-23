param location string
param environment string

resource githubActionsIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'id-github-actions-${environment}'
  location: location
}

// Built-in Contributor Role Definition ID
var contributorRoleId = 'b24988ac-6180-42a0-ab88-20f7382dd24c'

resource allowGithubContributor 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(githubActionsIdentity.id, resourceGroup().id, contributorRoleId)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', contributorRoleId)
    principalId: githubActionsIdentity.properties.principalId
    principalType: 'ServicePrincipal'
  }
}

resource appsIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: 'id-healthcheck-apps-${environment}'
  location: location
}

output client_id string = githubActionsIdentity.properties.clientId
output tenant_id string = githubActionsIdentity.properties.tenantId
output app_identity_id string = appsIdentity.id
output app_identity_principal_id string = appsIdentity.properties.principalId
output app_identity_name string = appsIdentity.name
output app_identity_client_id string = appsIdentity.properties.clientId

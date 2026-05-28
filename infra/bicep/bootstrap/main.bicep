targetScope = 'subscription'

param location string = 'eastasia'
param githubOrg string = 'toanle88'
param githubRepo string = 'Healthcheck'

// 1. Create the Bootstrap Resource Group
resource rgBootstrap 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: 'rg-healthcheck-bootstrap'
  location: location
  tags: {
    environment: 'bootstrap'
    project: 'healthcheck'
  }
}

// Subscription level role assignments require subscription target scope.
// Built-in Contributor Role ID: b24988ac-6180-42a0-ab88-20f7382dd24c
var contributorRoleId = 'b24988ac-6180-42a0-ab88-20f7382dd24c'
// Built-in User Access Administrator Role ID: 18d7d88d-d35e-4fb5-a5c3-7773c20a72d9
var uaaRoleId = '18d7d88d-d35e-4fb5-a5c3-7773c20a72d9'
// Built-in Storage Blob Data Owner Role ID: b7e6dc26-a60b-4bfe-b0ef-37d7b8470fd4
var sdoRoleId = 'b7e6dc26-a60b-4bfe-b0ef-37d7b8470fd4'

// Deploys the bootstrap resources in the scope of rgBootstrap
module bootstrapResources './modules/bootstrap-resources.bicep' = {
  name: 'bootstrap-resources-deployment'
  scope: rgBootstrap
  params: {
    location: location
    githubOrg: githubOrg
    githubRepo: githubRepo
  }
}

// BCP120 Workaround: Statically construct the resource ID at start of deployment.
var githubActionsIdentityId = subscriptionResourceId('rg-healthcheck-bootstrap', 'Microsoft.ManagedIdentity/userAssignedIdentities', 'id-github-actions-bootstrap')

// Assign Contributor to githubActions user identity at subscription level
resource allowGithubContributor 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(githubActionsIdentityId, subscription().id, contributorRoleId)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', contributorRoleId)
    principalId: bootstrapResources.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Assign User Access Administrator to githubActions user identity at subscription level
resource allowGithubUaa 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(githubActionsIdentityId, subscription().id, uaaRoleId)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', uaaRoleId)
    principalId: bootstrapResources.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// Assign Storage Blob Data Owner to githubActions user identity at subscription level
resource allowGithubSdo 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(githubActionsIdentityId, subscription().id, sdoRoleId)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', sdoRoleId)
    principalId: bootstrapResources.outputs.identityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

output AZURE_ACR_NAME string = bootstrapResources.outputs.acrName
output AZURE_STORAGE_ACCOUNT string = bootstrapResources.outputs.storageAccountName
output AZURE_STORAGE_CONTAINER string = bootstrapResources.outputs.containerName
output AZURE_CLIENT_ID string = bootstrapResources.outputs.identityClientId
output AZURE_TENANT_ID string = subscription().tenantId
output AZURE_SUBSCRIPTION_ID string = subscription().subscriptionId

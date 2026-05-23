param location string
param environment string
param deployerPrincipalId string = ''

var kvSuffix = uniqueString(resourceGroup().id, environment)
var kvName = 'kv-hc-${environment}-${substring(kvSuffix, 0, 8)}'

resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' = {
  name: kvName
  location: location
  properties: {
    enabledForDiskEncryption: true
    tenantId: subscription().tenantId
    sku: {
      name: 'standard'
      family: 'A'
    }
    softDeleteRetentionInDays: 7
    enablePurgeProtection: false
    enableRbacAuthorization: true
    publicNetworkAccess: 'Enabled'
    networkAcls: {
      bypass: 'AzureServices'
      defaultAction: 'Allow'
    }
  }
}

// Built-in Key Vault Secrets Officer Role ID: b86a8fe4-44ce-4948-aee5-eccb2c155cd7
var secretsOfficerRoleId = 'b86a8fe4-44ce-4948-aee5-eccb2c155cd7'

resource deployerSecretsOfficer 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (!empty(deployerPrincipalId)) {
  name: guid(keyVault.id, deployerPrincipalId, secretsOfficerRoleId)
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', secretsOfficerRoleId)
    principalId: deployerPrincipalId
    principalType: 'ServicePrincipal'
  }
}

output id string = keyVault.id
output vault_uri string = keyVault.properties.vaultUri
output name string = keyVault.name

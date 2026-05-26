targetScope = 'subscription'

param location string = 'eastasia'
param environment string = 'pro'
param githubOrgOrUser string = 'toanle88'
param githubRepoName string = 'Healthcheck'
param acrName string
param entraClientId string
param ciamTenantId string = 'cea4bf39-5592-4b9c-bed9-0729bbf40cd4'
@secure()
param alertWebhookUrl string = ''
param alertEmail string = 'toanle88@outlook.com'
param deployerPrincipalId string = ''

param apiImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
param workerImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
param webImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'
param migrateImage string = 'mcr.microsoft.com/azuredocs/containerapps-helloworld:latest'

// Resource Group
resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: 'rg-healthcheck-${environment}'
  location: location
  tags: {
    environment: environment
    project: 'healthcheck'
    githubOrg: githubOrgOrUser
    githubRepo: githubRepoName
  }
}

// 1. Identity Module (Common)
module identity '../../modules/common/identity.bicep' = {
  name: 'identity-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
  }
}

// 2. Network Module (Production Override - private endpoint subnet & NSG lock)
module network '../../modules/pro/network.bicep' = {
  name: 'network-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
  }
}

// 3. Postgres Module (Production Override - AD-only passwordless authentication & geo backups)
module postgres '../../modules/pro/postgres.bicep' = {
  name: 'postgres-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
    vnetId: network.outputs.vnet_id
    subnetId: network.outputs.db_subnet_id
    aadAdminObjectId: identity.outputs.app_identity_principal_id
    aadAdminName: identity.outputs.app_identity_name
  }
}

// 4. Key Vault Module (Production Override - public access disabled, private endpoint active)
module keyvault '../../modules/pro/keyvault.bicep' = {
  name: 'keyvault-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
    deployerPrincipalId: deployerPrincipalId
    subnetId: network.outputs.endpoints_subnet_id
  }
}

// 5. Application Insights Module (Common)
module appinsights '../../modules/common/appinsights.bicep' = {
  name: 'appinsights-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
  }
}

// 6. Container App Module (Common)
module containerapp '../../modules/common/containerapp.bicep' = {
  name: 'containerapp-deployment-${environment}'
  scope: rg
  params: {
    location: location
    environment: environment
    subnetId: network.outputs.apps_subnet_id
    acrName: acrName
    keyVaultName: keyvault.outputs.name
    apiImage: apiImage
    workerImage: workerImage
    webImage: webImage
    migrateImage: migrateImage
    appVersion: apiImage
    dbHost: postgres.outputs.host
    dbName: 'healthcheck'
    dbUser: identity.outputs.app_identity_name
    entraClientId: entraClientId
    tenantId: ciamTenantId
    appInsightsConnectionString: appinsights.outputs.connectionString
    appIdentityId: identity.outputs.app_identity_id
    appIdentityPrincipalId: identity.outputs.app_identity_principal_id
    appIdentityClientId: identity.outputs.app_identity_client_id
  }
  dependsOn: [
    keyvaultWebhookSecret
  ]
}

// 7. Alerts Module (Common)
module monitor '../../modules/common/monitor.bicep' = {
  name: 'monitor-deployment-${environment}'
  scope: rg
  params: {
    environment: environment
    alertEmail: alertEmail
    apiContainerAppId: containerapp.outputs.api_app_id
  }
}

// 8. Policy Definitions (Subscription Scoped)
resource requireEnvTagDef 'Microsoft.Authorization/policyDefinitions@2021-06-01' = {
  name: 'require-env-tag-${environment}'
  properties: {
    policyType: 'Custom'
    mode: 'Indexed'
    displayName: '[${toUpper(environment)}] Require \'environment\' tag on all resources'
    description: 'Denies creation or update of any resource that is missing the \'environment\' tag.'
    policyRule: {
      if: {
        field: 'tags[\'environment\']'
        exists: 'false'
      }
      then: {
        effect: 'deny'
      }
    }
  }
}

resource requireProjTagDef 'Microsoft.Authorization/policyDefinitions@2021-06-01' = {
  name: 'require-proj-tag-${environment}'
  properties: {
    policyType: 'Custom'
    mode: 'Indexed'
    displayName: '[${toUpper(environment)}] Require \'project\' tag on all resources'
    description: 'Denies creation or update of any resource that is missing the \'project\' tag.'
    policyRule: {
      if: {
        field: 'tags[\'project\']'
        exists: 'false'
      }
      then: {
        effect: 'deny'
      }
    }
  }
}

// 9. Policy Assignments Module (Common - scoped to RG)
module policy '../../modules/common/policy.bicep' = {
  name: 'policy-deployment-${environment}'
  scope: rg
  params: {
    environment: environment
    requireEnvTagDefId: requireEnvTagDef.id
    requireProjTagDefId: requireProjTagDef.id
  }
}

// 10. ACR Role Assignment (Common - Scoped to Bootstrap RG)
module acrRoleAssignment '../../modules/common/acr-role-assignment.bicep' = {
  name: 'acr-role-assignment-${environment}'
  scope: resourceGroup('rg-healthcheck-bootstrap')
  params: {
    acrName: acrName
    principalId: identity.outputs.app_identity_principal_id
  }
}

// 11. Key Vault secret alert-webhook-url upload (Common)
module keyvaultWebhookSecret '../../modules/common/keyvault-secret.bicep' = {
  name: 'keyvault-webhook-secret-deployment-${environment}'
  scope: rg
  params: {
    keyVaultName: keyvault.outputs.name
    secretName: 'alert-webhook-url'
    secretValue: empty(alertWebhookUrl) ? 'dummy' : alertWebhookUrl
  }
}

// Outputs
output ACR_LOGIN_SERVER string = '${acrName}.azurecr.io'
output API_URL string = containerapp.outputs.api_url
output WEB_URL string = containerapp.outputs.web_url

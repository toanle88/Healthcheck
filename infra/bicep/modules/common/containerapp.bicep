param location string
param environment string
param subnetId string
param acrName string
param keyVaultName string
param apiImage string
param workerImage string
param webImage string
param migrateImage string
param appVersion string
param dbHost string
param dbName string
param dbUser string
param entraClientId string
param tenantId string
param appInsightsConnectionString string
param appIdentityId string
param appIdentityPrincipalId string
param appIdentityClientId string

var acrLoginServer = '${acrName}.azurecr.io'

// 1. Log Analytics Workspace
resource logWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: 'log-healthcheck-${environment}'
  location: location
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

// 2. Container Apps Environment (VNet integrated)
resource containerAppEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: 'cae-healthcheck-${environment}'
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logWorkspace.properties.customerId
        sharedKey: logWorkspace.listKeys().primarySharedKey
      }
    }
    vnetConfiguration: {
      infrastructureSubnetId: subnetId
      internal: false
    }
  }
}

// 3. Role Assignments

// Key Vault Secrets User role definition ID: 46334581-17ef-4b9d-b4d5-a6e16e140d29
resource keyVault 'Microsoft.KeyVault/vaults@2023-07-01' existing = {
  name: keyVaultName
}

resource kvSecretsAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(keyVault.id, appIdentityPrincipalId, 'KeyVaultSecretsUser')
  scope: keyVault
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '46334581-17ef-4b9d-b4d5-a6e16e140d29')
    principalId: appIdentityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

// 4. API Container App
resource apiApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: 'ca-healthcheck-api-${environment}'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${appIdentityId}': {}
    }
  }
  dependsOn: [
    kvSecretsAssignment
  ]
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      activeRevisionsMode: 'Multiple'
      ingress: {
        external: true
        targetPort: 8080
        transport: 'auto'
        corsPolicy: {
          allowedOrigins: [
            'https://ca-healthcheck-web-${environment}.${containerAppEnv.properties.defaultDomain}'
          ]
          allowedMethods: [
            'GET'
            'POST'
            'DELETE'
            'OPTIONS'
          ]
          allowedHeaders: [
            '*'
          ]
        }
      }
      registries: [
        {
          server: acrLoginServer
          identity: appIdentityId
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'api'
          image: apiImage
          resources: {
            cpu: any('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'PORT'
              value: '8080'
            }
            {
              name: 'ENV'
              value: environment
            }
            {
              name: 'DB_HOST'
              value: dbHost
            }
            {
              name: 'DB_NAME'
              value: dbName
            }
            {
              name: 'DB_USER'
              value: dbUser
            }
            {
              name: 'AZURE_CLIENT_ID'
              value: appIdentityClientId
            }
            {
              name: 'ENTRA_TENANT_ID'
              value: tenantId
            }
            {
              name: 'ENTRA_CLIENT_ID'
              value: entraClientId
            }
            {
              name: 'APPLICATIONINSIGHTS_CONNECTION_STRING'
              value: appInsightsConnectionString
            }
          ]
        }
      ]
      scale: {
        minReplicas: 0
        maxReplicas: 3
      }
    }
  }
}

// 5. Worker Job (Scheduled)
resource workerJob 'Microsoft.App/jobs@2023-05-01' = {
  name: 'caj-healthcheck-worker-${environment}'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${appIdentityId}': {}
    }
  }
  dependsOn: [
    kvSecretsAssignment
  ]
  properties: {
    environmentId: containerAppEnv.id
    configuration: {
      triggerType: 'Schedule'
      replicaTimeout: 60
      replicaRetryLimit: 1
      registries: [
        {
          server: acrLoginServer
          identity: appIdentityId
        }
      ]
      scheduleTriggerConfig: {
        cronExpression: '*/1 * * * *'
      }
      secrets: [
        {
          name: 'alert-webhook'
          keyVaultUrl: '${keyVault.properties.vaultUri}secrets/alert-webhook-url'
          identity: appIdentityId
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'worker'
          image: workerImage
          resources: {
            cpu: any('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'WORKER_MODE'
              value: 'job'
            }
            {
              name: 'ENV'
              value: environment
            }
            {
              name: 'DB_HOST'
              value: dbHost
            }
            {
              name: 'DB_NAME'
              value: dbName
            }
            {
              name: 'DB_USER'
              value: dbUser
            }
            {
              name: 'AZURE_CLIENT_ID'
              value: appIdentityClientId
            }
            {
              name: 'APPLICATIONINSIGHTS_CONNECTION_STRING'
              value: appInsightsConnectionString
            }
            {
              name: 'ALERT_WEBHOOK_URL'
              secretRef: 'alert-webhook'
            }
          ]
        }
      ]
    }
  }
}

// 6. Web Container App (Frontend)
resource webApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: 'ca-healthcheck-web-${environment}'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${appIdentityId}': {}
    }
  }
  dependsOn: [
    kvSecretsAssignment
  ]
  properties: {
    managedEnvironmentId: containerAppEnv.id
    configuration: {
      activeRevisionsMode: 'Multiple'
      ingress: {
        external: true
        targetPort: 80
        transport: 'auto'
      }
      registries: [
        {
          server: acrLoginServer
          identity: appIdentityId
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'web'
          image: webImage
          resources: {
            cpu: any('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'VITE_API_URL'
              value: 'https://${apiApp.properties.configuration.ingress.fqdn}'
            }
            {
              name: 'VITE_APP_VERSION'
              value: appVersion
            }
            {
              name: 'VITE_ENTRA_CLIENT_ID'
              value: entraClientId
            }
            {
              name: 'VITE_ENTRA_TENANT_ID'
              value: tenantId
            }
            {
              name: 'ENV'
              value: environment
            }
          ]
        }
      ]
      scale: {
        minReplicas: 0
        maxReplicas: 3
      }
    }
  }
}

// 7. Migrate Job (Manual)
resource migrateJob 'Microsoft.App/jobs@2023-05-01' = {
  name: 'caj-healthcheck-migrate-${environment}'
  location: location
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${appIdentityId}': {}
    }
  }
  dependsOn: [
    kvSecretsAssignment
  ]
  properties: {
    environmentId: containerAppEnv.id
    configuration: {
      triggerType: 'Manual'
      replicaTimeout: 180
      replicaRetryLimit: 0
      registries: [
        {
          server: acrLoginServer
          identity: appIdentityId
        }
      ]
      manualTriggerConfig: {
        parallelism: 1
        replicaCompletionCount: 1
      }
    }
    template: {
      containers: [
        {
          name: 'migrate'
          image: migrateImage
          resources: {
            cpu: any('0.25')
            memory: '0.5Gi'
          }
          env: [
            {
              name: 'ENV'
              value: environment
            }
            {
              name: 'DB_HOST'
              value: dbHost
            }
            {
              name: 'DB_NAME'
              value: dbName
            }
            {
              name: 'DB_USER'
              value: dbUser
            }
            {
              name: 'AZURE_CLIENT_ID'
              value: appIdentityClientId
            }
            {
              name: 'APPLICATIONINSIGHTS_CONNECTION_STRING'
              value: appInsightsConnectionString
            }
          ]
        }
      ]
    }
  }
}

output default_domain string = containerAppEnv.properties.defaultDomain
output api_url string = apiApp.properties.configuration.ingress.fqdn
output web_url string = webApp.properties.configuration.ingress.fqdn
output container_app_environment_id string = containerAppEnv.id
output api_app_id string = apiApp.id
output worker_job_id string = workerJob.id
output migrate_job_name string = migrateJob.name

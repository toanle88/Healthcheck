param location string
param environment string
param vnetId string
param subnetId string
@secure()
param adminPassword string
param aadAdminObjectId string
param aadAdminName string

// 1. Private DNS Zone
resource privateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' = {
  name: 'healthcheck.postgres.database.azure.com'
  location: 'global'
}

// 2. VNet Link
resource vnetLink 'Microsoft.Network/privateDnsZones/virtualNetworkLinks@2020-06-01' = {
  parent: privateDnsZone
  name: 'pdzlink-healthcheck'
  location: 'global'
  properties: {
    registrationEnabled: false
    virtualNetwork: {
      id: vnetId
    }
  }
}

// 3. PostgreSQL Flexible Server (Dev Configuration)
resource postgresServer 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {
  name: 'psql-healthcheck-${environment}'
  location: location
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'
  }
  properties: {
    version: '16'
    network: {
      delegatedSubnetResourceId: subnetId
      privateDnsZoneArmResourceId: privateDnsZone.id
    }
    administratorLogin: 'psqladmin'
    #disable-next-line use-secure-value-for-secure-inputs
    administratorLoginPassword: adminPassword
    authConfig: {
      activeDirectoryAuth: 'Enabled'
      passwordAuth: 'Enabled'
    }
    storage: {
      storageSizeGB: 32
    }
    backup: {
      geoRedundantBackup: 'Disabled'
    }
  }
  dependsOn: [
    vnetLink
  ]
}

// 4. PostgreSQL Database
resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2023-06-01-preview' = {
  parent: postgresServer
  name: 'healthcheck'
  properties: {
    charset: 'UTF8'
    collation: 'en_US.utf8'
  }
}

// 5. Active Directory Administrator
resource aadAdmin 'Microsoft.DBforPostgreSQL/flexibleServers/administrators@2023-06-01-preview' = {
  parent: postgresServer
  name: aadAdminObjectId
  properties: {
    principalType: 'ServicePrincipal'
    principalName: aadAdminName
    tenantId: subscription().tenantId
  }
}

output host string = postgresServer.properties.fullyQualifiedDomainName

param location string
param environment string

// 1. NSG for Database
resource dbNsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'nsg-db-${environment}'
  location: location
  properties: {
    securityRules: [
      {
        name: 'AllowAppsToDB'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '5432'
          sourceAddressPrefix: '10.0.1.0/24'
          destinationAddressPrefix: '10.0.2.0/24'
        }
      }
      {
        name: 'DenyAllInbound'
        properties: {
          priority: 1000
          direction: 'Inbound'
          access: 'Deny'
          protocol: '*'
          sourcePortRange: '*'
          destinationPortRange: '*'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
    ]
  }
}

// 2. NSG for Container Apps (Allows HTTP and HTTPS in Dev)
resource appsNsg 'Microsoft.Network/networkSecurityGroups@2023-09-01' = {
  name: 'nsg-apps-${environment}'
  location: location
  properties: {
    securityRules: [
      {
        name: 'AllowHTTPInbound'
        properties: {
          priority: 100
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '80'
          sourceAddressPrefix: 'Internet'
          destinationAddressPrefix: '*'
        }
      }
      {
        name: 'AllowHTTPSInbound'
        properties: {
          priority: 110
          direction: 'Inbound'
          access: 'Allow'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '443'
          sourceAddressPrefix: 'Internet'
          destinationAddressPrefix: '*'
        }
      }
      {
        name: 'DenySSHInbound'
        properties: {
          priority: 120
          direction: 'Inbound'
          access: 'Deny'
          protocol: 'Tcp'
          sourcePortRange: '*'
          destinationPortRange: '22'
          sourceAddressPrefix: '*'
          destinationAddressPrefix: '*'
        }
      }
    ]
  }
}

// 3. Virtual Network
resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-healthcheck-${environment}'
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
    subnets: [
      {
        name: 'snet-apps'
        properties: {
          addressPrefix: '10.0.1.0/24'
          networkSecurityGroup: {
            id: appsNsg.id
          }
          delegations: [
            {
              name: 'aca-delegation'
              properties: {
                serviceName: 'Microsoft.App/environments'
              }
            }
          ]
        }
      }
      {
        name: 'snet-db'
        properties: {
          addressPrefix: '10.0.2.0/24'
          networkSecurityGroup: {
            id: dbNsg.id
          }
          delegations: [
            {
              name: 'fs-delegation'
              properties: {
                serviceName: 'Microsoft.DBforPostgreSQL/flexibleServers'
              }
            }
          ]
        }
      }
    ]
  }
}

output vnet_id string = vnet.id
output apps_subnet_id string = resourceId('Microsoft.Network/virtualNetworks/subnets', vnet.name, 'snet-apps')
output db_subnet_id string = resourceId('Microsoft.Network/virtualNetworks/subnets', vnet.name, 'snet-db')

resource "azurerm_virtual_network" "main" {
  name                = "vnet-healthcheck-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  address_space       = ["10.0.0.0/16"]
}

# Subnet for Azure Container Apps (requires delegation)
resource "azurerm_subnet" "container_apps" {
  name                 = "snet-apps"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]

  delegation {
    name = "aca-delegation"
    service_delegation {
      name    = "Microsoft.App/environments"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action", "Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action"]
    }
  }
}

# Subnet for Database (Private Endpoint)
resource "azurerm_subnet" "database" {
  name                 = "snet-db"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.2.0/24"]
  
  # Delegate to Postgres Flexible Server
  delegation {
    name = "fs-delegation"
    service_delegation {
      name = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

output "vnet_id" {
  value = azurerm_virtual_network.main.id
}

output "apps_subnet_id" {
  value = azurerm_subnet.container_apps.id
}

output "db_subnet_id" {
  value = azurerm_subnet.database.id
}

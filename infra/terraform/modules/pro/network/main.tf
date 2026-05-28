resource "azurerm_virtual_network" "main" {
  name                = "vnet-healthcheck-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  address_space       = ["10.0.0.0/16"]
  tags                = var.tags
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
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

# Subnet for Private Endpoints (Needed for Key Vault and other endpoints in pro)
resource "azurerm_subnet" "endpoints" {
  name                 = "snet-endpoints"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.3.0/24"]
}

# 3. NETWORK SECURITY GROUP (The Internal Firewall)
resource "azurerm_network_security_group" "db" {
  name                = "nsg-db-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  tags                = var.tags

  # RULE: Only allow the Apps Subnet to talk to the DB on port 5432
  security_rule {
    name                       = "AllowAppsToDB"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "5432"
    source_address_prefix      = "10.0.1.0/24" # The Apps Subnet
    destination_address_prefix = "10.0.2.0/24" # The DB Subnet
  }

  # RULE: Deny everything else (High priority number means it runs last)
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

# 4. NETWORK SECURITY GROUP FOR APPS
resource "azurerm_network_security_group" "apps" {
  name                = "nsg-apps-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  tags                = var.tags

  # Deny HTTP (Port 80) in Production (complying with Checkov CKV_AZURE_160)
  security_rule {
    name                       = "DenyHTTPInbound"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }

  # Allow HTTPS (Port 443)
  security_rule {
    name                       = "AllowHTTPSInbound"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }

  # EXPLICIT DENY SSH (Checkov CKV_AZURE_10)
  security_rule {
    name                       = "DenySSHInbound"
    priority                   = 120
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "apps" {
  subnet_id                 = azurerm_subnet.container_apps.id
  network_security_group_id = azurerm_network_security_group.apps.id
}

# 5. LINK NSG TO DB SUBNET
resource "azurerm_subnet_network_security_group_association" "db" {
  subnet_id                 = azurerm_subnet.database.id
  network_security_group_id = azurerm_network_security_group.db.id
}

# 6. NSG FOR PRIVATE ENDPOINTS (CKV2_AZURE_31)
resource "azurerm_network_security_group" "endpoints" {
  name                = "nsg-endpoints-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  tags                = var.tags

  # Allow inbound from container apps subnet (for Key Vault access, etc.)
  security_rule {
    name                       = "AllowAppsToEndpoints"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "10.0.1.0/24"
    destination_address_prefix = "10.0.3.0/24"
  }

  # Deny all other inbound
  security_rule {
    name                       = "DenyAllInbound"
    priority                   = 1000
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "endpoints" {
  subnet_id                 = azurerm_subnet.endpoints.id
  network_security_group_id = azurerm_network_security_group.endpoints.id
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

output "endpoints_subnet_id" {
  value = azurerm_subnet.endpoints.id
}

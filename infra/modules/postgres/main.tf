# 1. PRIVATE DNS ZONE
# Because the database has no public IP, we need a private domain name 
# (e.g., my-db.postgres.database.azure.com) that only works inside our VNet.
resource "azurerm_private_dns_zone" "postgres" {
  name                = "healthcheck.postgres.database.azure.com"
  resource_group_name = var.resource_group_name
}

# 2. VNET LINK
# This "attaches" the Private DNS zone to our VNet so our API can resolve the DB address.
resource "azurerm_private_dns_zone_virtual_network_link" "main" {
  name                  = "pdzlink-healthcheck"
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = var.vnet_id
  resource_group_name   = var.resource_group_name
}

# 3. POSTGRES FLEXIBLE SERVER
# checkov:skip=CKV2_AZURE_57:Using VNet Integration (Delegated Subnet) instead of Private Endpoint
resource "azurerm_postgresql_flexible_server" "main" {
  name                = "psql-healthcheck-${var.environment}"
  resource_group_name = var.resource_group_name
  location            = var.location
  version             = "16"
  delegated_subnet_id = var.subnet_id
  private_dns_zone_id = azurerm_private_dns_zone.postgres.id

  # SECURITY: This ensures the database is NOT reachable from the internet.
  public_network_access_enabled = false

  administrator_login    = "psqladmin"
  administrator_password = var.admin_password

  # SKU: B1ms is the "Burstable" tier, perfect for dev/learning at ~$0.017/hour.
  sku_name   = "B_Standard_B1ms"
  storage_mb = 32768

  lifecycle {
    ignore_changes = [zone]
  }

  depends_on = [azurerm_private_dns_zone_virtual_network_link.main]
}

# 4. THE ACTUAL DATABASE
resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = "healthcheck"
  server_id = azurerm_postgresql_flexible_server.main.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# 5. AZURE AD AUTHENTICATION
# This allows our Managed Identity to log in without a password.
resource "azurerm_postgresql_flexible_server_active_directory_administrator" "main" {
  server_name         = azurerm_postgresql_flexible_server.main.name
  resource_group_name = var.resource_group_name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  object_id           = var.aad_admin_object_id
  principal_name      = var.aad_admin_name
  principal_type      = "ServicePrincipal"
}

data "azurerm_client_config" "current" {}

output "host" {
  value = azurerm_postgresql_flexible_server.main.fqdn
}

resource "azurerm_container_registry" "main" {
  name                = "crhealthcheck${var.environment}" # Must be alphanumeric
  resource_group_name = var.resource_group_name
  location            = var.location
  sku                 = "Basic"
  admin_enabled       = false # Modern standard: use Managed Identity instead of admin keys
}

output "login_server" {
  value = azurerm_container_registry.main.login_server
}

output "id" {
  value = azurerm_container_registry.main.id
}

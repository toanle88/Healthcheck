# Enforce required tags on all resources within the resource group.
#
# The policy definition uses the built-in "Require a tag on resources" pattern
# applied twice — once for the "environment" tag and once for the "project" tag.
# Each definition is scoped to the resource group via a policy assignment so that
# it does not accidentally affect other subscriptions or resource groups.
#
# Effect: "deny" — a resource CREATE/UPDATE without the required tag is rejected
# at the ARM API layer, before any resource is provisioned.

# ── Policy definition: enforce "environment" tag ────────────────────────────

resource "azurerm_policy_definition" "require_environment_tag" {
  name         = "require-environment-tag-${var.environment}"
  policy_type  = "Custom"
  mode         = "Indexed"
  display_name = "[${upper(var.environment)}] Require 'environment' tag on all resources"
  description  = "Denies creation or update of any resource that is missing the 'environment' tag."

  policy_rule = jsonencode({
    "if" = {
      field  = "tags['environment']"
      exists = "false"
    }
    "then" = {
      effect = "deny"
    }
  })
}

# ── Policy definition: enforce "project" tag ─────────────────────────────────

resource "azurerm_policy_definition" "require_project_tag" {
  name         = "require-project-tag-${var.environment}"
  policy_type  = "Custom"
  mode         = "Indexed"
  display_name = "[${upper(var.environment)}] Require 'project' tag on all resources"
  description  = "Denies creation or update of any resource that is missing the 'project' tag."

  policy_rule = jsonencode({
    "if" = {
      field  = "tags['project']"
      exists = "false"
    }
    "then" = {
      effect = "deny"
    }
  })
}

# ── Assignments: scoped to the dev resource group ────────────────────────────
# Scoping to the resource group (not subscription) prevents accidental impact
# on other resources during development.

resource "azurerm_resource_group_policy_assignment" "require_environment_tag" {
  name                 = "assign-require-env-tag-${var.environment}"
  resource_group_id    = var.resource_group_id
  policy_definition_id = azurerm_policy_definition.require_environment_tag.id
  display_name         = "[${upper(var.environment)}] Require 'environment' tag"
  description          = "Assigned to ${var.resource_group_name} — denies resources missing the 'environment' tag."

  # Provide the default tag value so Azure Policy can optionally use it in
  # modify-effect extensions in the future without changing the assignment.
  parameters = jsonencode({
    tagName = { value = "environment" }
  })
}

resource "azurerm_resource_group_policy_assignment" "require_project_tag" {
  name                 = "assign-require-proj-tag-${var.environment}"
  resource_group_id    = var.resource_group_id
  policy_definition_id = azurerm_policy_definition.require_project_tag.id
  display_name         = "[${upper(var.environment)}] Require 'project' tag"
  description          = "Assigned to ${var.resource_group_name} — denies resources missing the 'project' tag."

  parameters = jsonencode({
    tagName = { value = "project" }
  })
}

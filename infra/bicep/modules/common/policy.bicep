param environment string
param requireEnvTagDefId string
param requireProjTagDefId string

// 1. Policy Assignment for Environment Tag
resource requireEnvTagAssign 'Microsoft.Authorization/policyAssignments@2022-06-01' = {
  name: 'assign-req-env-${environment}'
  properties: {
    policyDefinitionId: requireEnvTagDefId
    displayName: '[${toUpper(environment)}] Require \'environment\' tag'
    description: 'Denies resources missing the \'environment\' tag.'
  }
}

// 2. Policy Assignment for Project Tag
resource requireProjTagAssign 'Microsoft.Authorization/policyAssignments@2022-06-01' = {
  name: 'assign-req-proj-${environment}'
  properties: {
    policyDefinitionId: requireProjTagDefId
    displayName: '[${toUpper(environment)}] Require \'project\' tag'
    description: 'Denies resources missing the \'project\' tag.'
  }
}

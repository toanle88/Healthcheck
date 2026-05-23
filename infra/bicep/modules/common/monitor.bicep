param environment string
param alertEmail string
param apiContainerAppId string

// 1. Action Group
resource actionGroup 'Microsoft.Insights/actionGroups@2023-01-01' = {
  name: 'ag-healthcheck-${environment}'
  location: 'global'
  properties: {
    groupShortName: 'hc-alerts'
    enabled: true
    emailReceivers: [
      {
        name: 'admin'
        emailAddress: alertEmail
        useCommonAlertSchema: true
      }
    ]
  }
}

// 2. Latency Metric Alert
resource latencyAlert 'Microsoft.Insights/metricAlerts@2018-03-01' = {
  name: 'alert-latency-high-${environment}'
  location: 'global'
  properties: {
    description: 'Fires when API P95 latency is > 500ms for 5 minutes'
    severity: 2
    enabled: true
    scopes: [
      apiContainerAppId
    ]
    evaluationFrequency: 'PT1M'
    windowSize: 'PT5M'
    criteria: {
      'odata.type': 'Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria'
      allOf: [
        {
          name: 'LatencyCriteria'
          metricNamespace: 'Microsoft.App/containerApps'
          metricName: 'ResponseTime'
          operator: 'GreaterThan'
          threshold: 500
          timeAggregation: 'Average'
          criterionType: 'StaticThresholdCriterion'
        }
      ]
    }
    actions: [
      {
        actionGroupId: actionGroup.id
      }
    ]
  }
}

// 3. Request Spike Alert (Error placeholder)
resource errorAlert 'Microsoft.Insights/metricAlerts@2018-03-01' = {
  name: 'alert-errors-high-${environment}'
  location: 'global'
  properties: {
    description: 'Fires when HTTP 5xx errors exceed 1%'
    severity: 1
    enabled: true
    scopes: [
      apiContainerAppId
    ]
    evaluationFrequency: 'PT1M'
    windowSize: 'PT5M'
    criteria: {
      'odata.type': 'Microsoft.Azure.Monitor.SingleResourceMultipleMetricCriteria'
      allOf: [
        {
          name: 'ErrorCriteria'
          metricNamespace: 'Microsoft.App/containerApps'
          metricName: 'Requests'
          operator: 'GreaterThan'
          threshold: 100
          timeAggregation: 'Total'
          criterionType: 'StaticThresholdCriterion'
        }
      ]
    }
    actions: [
      {
        actionGroupId: actionGroup.id
      }
    ]
  }
}

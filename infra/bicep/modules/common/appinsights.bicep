param location string
param environment string

// Application Insights
resource appInsights 'Microsoft.Insights/components@2020-02-02' = {
  name: 'appi-healthcheck-${environment}'
  location: location
  kind: 'web'
  properties: {
    Application_Type: 'web'
    SamplingPercentage: 100
  }
}

output connectionString string = appInsights.properties.ConnectionString

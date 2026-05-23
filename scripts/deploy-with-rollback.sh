#!/bin/bash
# Enable exit on error
set -e

# Arguments
ENVIRONMENT=$1
RESOURCE_GROUP=$2
ACR_NAME=$3
IMAGE_TAG=$4

if [ -z "$ENVIRONMENT" ] || [ -z "$RESOURCE_GROUP" ] || [ -z "$ACR_NAME" ] || [ -z "$IMAGE_TAG" ]; then
  echo "Usage: $0 <environment> <resource-group> <acr-name> <image-tag>"
  exit 1
fi

# Resolve application names based on environment
API_APP_NAME="ca-healthcheck-api-${ENVIRONMENT}"
WEB_APP_NAME="ca-healthcheck-web-${ENVIRONMENT}"
WORKER_JOB_NAME="caj-healthcheck-worker-${ENVIRONMENT}"
MIGRATE_JOB_NAME="caj-healthcheck-migrate-${ENVIRONMENT}"

echo "========================================="
echo "🚀 Starting Deployment & Rollback Handler"
echo "Environment:    $ENVIRONMENT"
echo "Resource Group: $RESOURCE_GROUP"
echo "ACR Name:       $ACR_NAME"
echo "Image Tag:      $IMAGE_TAG"
echo "========================================="

# Fetch current images for rollback backup
echo "🔍 Fetching current app images for backup..."
PREV_API_IMAGE=$(az containerapp show -n $API_APP_NAME -g $RESOURCE_GROUP --query "properties.template.containers[0].image" -o tsv || echo "")
PREV_WEB_IMAGE=$(az containerapp show -n $WEB_APP_NAME -g $RESOURCE_GROUP --query "properties.template.containers[0].image" -o tsv || echo "")
PREV_WORKER_IMAGE=$(az containerapp job show -n $WORKER_JOB_NAME -g $RESOURCE_GROUP --query "properties.template.containers[0].image" -o tsv || echo "")
PREV_MIGRATE_IMAGE=$(az containerapp job show -n $MIGRATE_JOB_NAME -g $RESOURCE_GROUP --query "properties.template.containers[0].image" -o tsv || echo "")

echo "📦 Backup Images:"
echo "  - API:     $PREV_API_IMAGE"
echo "  - Web:     $PREV_WEB_IMAGE"
echo "  - Worker:  $PREV_WORKER_IMAGE"
echo "  - Migrate: $PREV_MIGRATE_IMAGE"

# Track if rollback is needed
ROLLBACK_NEEDED=false

# Rollback function
rollback() {
  echo "🚨 Deployment failed or health check failed! Initiating rollback..."
  
  if [ -n "$PREV_API_IMAGE" ]; then
    echo "Reverting API App to $PREV_API_IMAGE..."
    az containerapp update -n $API_APP_NAME -g $RESOURCE_GROUP --image $PREV_API_IMAGE || echo "⚠️ Failed to revert API App"
  fi
  
  if [ -n "$PREV_WEB_IMAGE" ]; then
    echo "Reverting Web App to $PREV_WEB_IMAGE..."
    az containerapp update -n $WEB_APP_NAME -g $RESOURCE_GROUP --image $PREV_WEB_IMAGE || echo "⚠️ Failed to revert Web App"
  fi
  
  if [ -n "$PREV_WORKER_IMAGE" ]; then
    echo "Reverting Worker Job to $PREV_WORKER_IMAGE..."
    az containerapp job update -n $WORKER_JOB_NAME -g $RESOURCE_GROUP --image $PREV_WORKER_IMAGE || echo "⚠️ Failed to revert Worker Job"
  fi
  
  if [ -n "$PREV_MIGRATE_IMAGE" ]; then
    echo "Reverting Migrate Job to $PREV_MIGRATE_IMAGE..."
    az containerapp job update -n $MIGRATE_JOB_NAME -g $RESOURCE_GROUP --image $PREV_MIGRATE_IMAGE || echo "⚠️ Failed to revert Migrate Job"
  fi
  
  echo "🛑 Rollback execution finished."
}

# Define exit trap to trigger rollback if deployment fails before we finish
trap 'if [ "$ROLLBACK_NEEDED" = "true" ]; then rollback; fi' EXIT

# 1. Database migrations (safe to run first)
echo "⚡ Updating database migration job..."
az containerapp job update \
  --name $MIGRATE_JOB_NAME \
  --resource-group $RESOURCE_GROUP \
  --image ${ACR_NAME}.azurecr.io/migrate:${IMAGE_TAG}

echo "⚡ Starting migration execution..."
EXECUTION_NAME=$(az containerapp job start \
  --name $MIGRATE_JOB_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "name" -o tsv)
echo "Migration job started: $EXECUTION_NAME"

echo "⏳ Waiting for migration job to complete..."
while true; do
  STATUS=$(az containerapp job execution show \
    --name $MIGRATE_JOB_NAME \
    --resource-group $RESOURCE_GROUP \
    --job-execution-name $EXECUTION_NAME \
    --query "properties.status" -o tsv)
  echo "Current status: $STATUS"
  if [ "$STATUS" = "Succeeded" ]; then
    echo "✅ Migrations completed successfully"
    break
  elif [ "$STATUS" = "Failed" ] || [ "$STATUS" = "Canceled" ]; then
    echo "❌ Migrations failed or were canceled"
    exit 1
  fi
  sleep 10
done

# From this point on, if anything fails, we need to roll back the updated containers
ROLLBACK_NEEDED=true

# 2. Update API App
echo "⚡ Updating API Container App..."
az containerapp update \
  --name $API_APP_NAME \
  --resource-group $RESOURCE_GROUP \
  --image ${ACR_NAME}.azurecr.io/api:${IMAGE_TAG}

# 3. Update Web App
echo "⚡ Updating Web Container App..."
az containerapp update \
  --name $WEB_APP_NAME \
  --resource-group $RESOURCE_GROUP \
  --image ${ACR_NAME}.azurecr.io/web:${IMAGE_TAG} \
  --set-env-vars VITE_APP_VERSION=${IMAGE_TAG}

# 4. Update Worker Job
echo "⚡ Updating Worker Job..."
az containerapp job update \
  --name $WORKER_JOB_NAME \
  --resource-group $RESOURCE_GROUP \
  --image ${ACR_NAME}.azurecr.io/worker:${IMAGE_TAG}

# Fetch the live endpoint URL
echo "🔍 Fetching API URL..."
API_FQDN=$(az containerapp show \
  --name $API_APP_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.configuration.ingress.fqdn" \
  --output tsv)
API_URL="https://${API_FQDN}"
echo "API URL: $API_URL"

# Output API URL for the pipeline (checks environment to set correct format)
if [ -n "$GITHUB_OUTPUT" ]; then
  echo "api_url=${API_URL}" >> $GITHUB_OUTPUT
elif [ -n "$BUILD_BUILDID" ]; then
  echo "##vso[task.setvariable variable=api_url;isOutput=true]${API_URL}"
fi

# 5. Smoke Test / Health Check
echo "🌡️ Checking application health..."
HEALTH_SUCCESS=false

for attempt in $(seq 1 9); do
  HTTP_STATUS=$(curl --silent --output /dev/null \
    --write-out "%{http_code}" \
    --max-time 10 \
    "${API_URL}/health" || echo "000")

  echo "Attempt ${attempt}/9 → HTTP ${HTTP_STATUS}"

  if [ "${HTTP_STATUS}" -eq 200 ]; then
    echo "✅ Smoke test passed (HTTP 200)"
    HEALTH_SUCCESS=true
    break
  fi

  sleep 10
done

if [ "$HEALTH_SUCCESS" = "false" ]; then
  echo "❌ Smoke test failed — /health did not return 200 after 90 seconds"
  exit 1
fi

# Deployment succeeded, disable rollback trap
ROLLBACK_NEEDED=false
echo "🎉 Deployment succeeded!"
exit 0

#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/terraform"

echo "Starting Ministack AWS emulator..."
docker compose up -d

echo "Waiting for Ministack to be ready..."
attempt=0
max_attempts=30
until curl -s http://localhost:4566/_ministack/health > /dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ $attempt -ge $max_attempts ]; then
    echo "Error: Ministack failed to start within expected time"
    exit 1
  fi
  sleep 1
done

echo "Ministack is ready!"

echo "Initializing OpenTofu..."
cd "$TERRAFORM_DIR"
tofu init

echo "Applying OpenTofu configuration..."
tofu apply -auto-approve

echo ""
echo "======================================"
echo "Local AWS environment is ready!"
echo "======================================"
echo ""
echo "Test with AWS CLI:"
echo "  aws --endpoint-url=http://localhost:4566 --region us-east-1 s3 ls"
echo "  aws --endpoint-url=http://localhost:4566 --region us-east-1 secretsmanager list-secrets"
echo ""
echo "OpenTofu outputs:"
tofu output

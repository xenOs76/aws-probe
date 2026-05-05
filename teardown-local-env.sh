#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/terraform"

echo "Destroying OpenTofu resources..."
cd "$TERRAFORM_DIR"
tofu destroy -auto-approve

echo "Stopping Ministack..."
cd "$SCRIPT_DIR"
docker compose down

echo "Local AWS environment torn down."

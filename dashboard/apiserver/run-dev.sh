#!/bin/bash

# Use colored output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}===== OCM Dashboard Development Mode =====${NC}"

# Set development environment variables
export DASHBOARD_DEBUG=true
export DASHBOARD_BYPASS_AUTH=true
export DASHBOARD_USE_MOCK=true

echo -e "${YELLOW}Environment variables set:${NC}"
echo "DASHBOARD_DEBUG=true       - Enable debug logging"
echo "DASHBOARD_BYPASS_AUTH=true - Skip authentication checks"
echo "DASHBOARD_USE_MOCK=true    - Use mock data instead of real clusters"
echo -e ""

# Get the script's directory and change to it
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo -e "${GREEN}Starting backend service...${NC}"
echo "API will be available at http://localhost:8080"

# Run from the backend directory
go run main.go $@
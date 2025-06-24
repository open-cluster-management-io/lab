#!/bin/bash

echo "===== Kubernetes Client Debug Tool ====="
echo "Checking KUBECONFIG..."

if [ -z "$KUBECONFIG" ]; then
  echo "❌ KUBECONFIG environment variable is not set"
  echo "Please set it using: export KUBECONFIG=/path/to/your/kubeconfig"
else
  echo "✅ KUBECONFIG = $KUBECONFIG"

  if [ -f "$KUBECONFIG" ]; then
    echo "✅ KUBECONFIG file exists"

    # Check file permissions
    if [ -r "$KUBECONFIG" ]; then
      echo "✅ KUBECONFIG file is readable"
    else
      echo "❌ KUBECONFIG file is not readable, please check permissions"
      echo "Run: chmod 600 $KUBECONFIG"
    fi

    # Validate file content
    if grep -q "server:" "$KUBECONFIG"; then
      echo "✅ KUBECONFIG file contains server configuration"
    else
      echo "❌ KUBECONFIG file may not contain valid cluster configuration"
    fi
  else
    echo "❌ KUBECONFIG file does not exist: $KUBECONFIG"
  fi
fi

echo ""
echo "Testing Kubernetes connection..."

# Test if we can get basic information
if command -v kubectl &> /dev/null; then
  echo "Attempting to connect to cluster..."
  kubectl version --short 2>/dev/null

  if [ $? -eq 0 ]; then
    echo "✅ Successfully connected to Kubernetes cluster"
    echo "✅ Validating OCM resources..."

    # Check if OCM resources exist
    if kubectl api-resources | grep -q "managedclusters"; then
      echo "✅ ManagedClusters resource type exists"
      echo "Listing ManagedClusters in current cluster:"
      kubectl get managedclusters
    else
      echo "❌ ManagedClusters resource type does not exist, OCM needs to be installed"
    fi
  else
    echo "❌ Cannot connect to Kubernetes cluster"
  fi
else
  echo "❌ kubectl is not installed, cannot test connection"
fi

echo ""
echo "===== Environment Variables ====="
echo "Run the following commands to enable debug mode:"
echo "export DASHBOARD_BYPASS_AUTH=true"
echo "export DASHBOARD_DEBUG=true"
echo ""
echo "Run the following command to start the application:"
echo "cd .. && make dev-backend"
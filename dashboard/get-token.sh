#!/bin/bash

set -e

echo "🔧 OCM Dashboard Token Generator"
echo "================================="
echo

# Service account name and namespace
SA_NAME="dashboard-user"
NAMESPACE="default"

echo "📋 Creating service account: $SA_NAME"
if kubectl get serviceaccount $SA_NAME -n $NAMESPACE &>/dev/null; then
    echo "✅ Service account '$SA_NAME' already exists"
else
    kubectl create serviceaccount $SA_NAME -n $NAMESPACE
    echo "✅ Service account '$SA_NAME' created"
fi

echo
echo "🔑 Creating cluster role binding..."
if kubectl get clusterrolebinding $SA_NAME &>/dev/null; then
    echo "✅ Cluster role binding '$SA_NAME' already exists"
else
    kubectl create clusterrolebinding $SA_NAME \
        --clusterrole=cluster-admin \
        --serviceaccount=$NAMESPACE:$SA_NAME
    echo "✅ Cluster role binding '$SA_NAME' created"
fi

echo
echo "🎫 Generating token (valid for 24 hours)..."
TOKEN=$(kubectl create token $SA_NAME --duration=24h --namespace=$NAMESPACE)

echo
echo "🎉 SUCCESS! Your OCM Dashboard token:"
echo "======================================"
echo
echo $TOKEN
echo
echo "💡 Instructions:"
echo "1. Copy the token above"
echo "2. Go to OCM Dashboard login page"
echo "3. Paste the token in the 'Bearer Token' field"
echo "4. Click 'Sign In'"
echo
echo "⚠️  Note: This token is valid for 24 hours only."
echo "    You'll need to generate a new one after expiration."
echo
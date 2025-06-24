#!/bin/bash

# Deploy Rundeck Infrastructure for AIOps Platform
# This script deploys the Rundeck orchestrator as the primary remediation system

set -e

echo "🚀 Deploying Rundeck Infrastructure for AIOps Platform"
echo "====================================================="

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot access Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

echo "✅ Kubernetes cluster accessible"

# Deploy Rundeck namespace
echo "📦 Creating Rundeck namespace..."
kubectl apply -f kubernetes/rundeck/namespace.yaml

# Deploy PostgreSQL database
echo "🗄️  Deploying PostgreSQL database for Rundeck..."
kubectl apply -f kubernetes/rundeck/postgresql.yaml

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgresql -n rundeck --timeout=300s

# Deploy RBAC
echo "🔐 Setting up RBAC for Rundeck..."
kubectl apply -f kubernetes/rundeck/rundeck-rbac.yaml

# Deploy Rundeck jobs configuration
echo "⚙️  Configuring Rundeck jobs..."
kubectl apply -f kubernetes/rundeck/rundeck-jobs.yaml

# Deploy Rundeck StatefulSet
echo "🏗️  Deploying Rundeck orchestrator..."
kubectl apply -f kubernetes/rundeck/rundeck-statefulset.yaml

# Deploy Rundeck services
echo "🌐 Setting up Rundeck services..."
kubectl apply -f kubernetes/rundeck/rundeck-service.yaml

# Wait for Rundeck to be ready
echo "⏳ Waiting for Rundeck to be ready..."
kubectl wait --for=condition=ready pod -l app=rundeck -n rundeck --timeout=600s

echo ""
echo "✅ Rundeck Infrastructure Deployment Complete!"
echo "=============================================="
echo ""
echo "📋 Access Information:"
echo "  • Rundeck UI: http://localhost:30440"
echo "  • Admin Username: admin"
echo "  • Admin Password: admin123"
echo ""
echo "🔍 Useful Commands:"
echo "  • Check Rundeck status: kubectl get pods -n rundeck"
echo "  • View Rundeck logs: kubectl logs -f deployment/rundeck -n rundeck"
echo "  • Access Rundeck shell: kubectl exec -it deployment/rundeck -n rundeck -- bash"
echo ""
echo "🔗 Integration:"
echo "  • ML Pipeline API: http://anomaly-detector.aiops:8080"
echo "  • Rundeck API: http://rundeck.rundeck:4440"
echo ""
echo "📚 Next Steps:"
echo "  1. Access Rundeck UI and verify job definitions"
echo "  2. Test ML Pipeline → Rundeck integration"
echo "  3. Configure additional node sources as needed"
echo "  4. Set up monitoring alerts to trigger jobs"


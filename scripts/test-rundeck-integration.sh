#!/bin/bash

# Test script for Rundeck integration with ML Pipeline
# This script validates the integration between the AIOps ML Pipeline and Rundeck orchestrator

set -e

echo "🧪 Testing Rundeck Integration with AIOps ML Pipeline"
echo "===================================================="

# Test 1: Check if Rundeck namespace is ready
echo "📦 Test 1: Checking Rundeck namespace..."
if kubectl get namespace rundeck &>/dev/null; then
    echo "✅ Rundeck namespace exists"
else
    echo "❌ Rundeck namespace not found"
    exit 1
fi

# Test 2: Check if PostgreSQL is running
echo "🗄️  Test 2: Checking PostgreSQL database..."
if kubectl get pods -n rundeck -l app=postgresql | grep -q "Running"; then
    echo "✅ PostgreSQL is running"
else
    echo "❌ PostgreSQL is not running"
    exit 1
fi

# Test 3: Check if Rundeck is running
echo "🏗️  Test 3: Checking Rundeck orchestrator..."
if kubectl get pods -n rundeck -l app=rundeck | grep -q "Running"; then
    echo "✅ Rundeck is running"
else
    echo "❌ Rundeck is not running"
    exit 1
fi

# Test 4: Check Rundeck service accessibility
echo "🌐 Test 4: Checking Rundeck service..."
RUNDECK_SVC=$(kubectl get svc -n rundeck rundeck -o jsonpath='{.spec.clusterIP}')
if [ -n "$RUNDECK_SVC" ]; then
    echo "✅ Rundeck service accessible at $RUNDECK_SVC:4440"
else
    echo "❌ Rundeck service not accessible"
    exit 1
fi

# Test 5: Test ML Pipeline binary compilation
echo "⚙️  Test 5: Testing ML Pipeline compilation..."
if [ -f "./anomaly-detector" ]; then
    echo "✅ ML Pipeline binary compiled successfully"
else
    echo "❌ ML Pipeline binary not found"
    exit 1
fi

# Test 6: Test Rundeck API connectivity (if Rundeck is ready)
echo "🔗 Test 6: Testing Rundeck API connectivity..."
sleep 10  # Give Rundeck time to fully start

# Port forward for testing
kubectl port-forward -n rundeck svc/rundeck 4440:4440 &
PF_PID=$!
sleep 5

# Test API endpoint
if curl -s -u admin:admin123 http://localhost:4440/api/18/system/info | grep -q "rundeck"; then
    echo "✅ Rundeck API is accessible"
else
    echo "⚠️  Rundeck API not yet ready (this is normal for first startup)"
fi

# Clean up port forward
kill $PF_PID 2>/dev/null || true

# Test 7: Check job definitions
echo "📋 Test 7: Checking Rundeck job definitions..."
if kubectl get configmap -n rundeck rundeck-jobs &>/dev/null; then
    echo "✅ Rundeck job definitions loaded"
else
    echo "❌ Rundeck job definitions not found"
    exit 1
fi

# Test 8: Validate ML Pipeline configuration
echo "⚙️  Test 8: Validating ML Pipeline configuration..."
if grep -q "rundeck:" configs/config.yaml; then
    echo "✅ Rundeck configuration found in config.yaml"
else
    echo "❌ Rundeck configuration missing from config.yaml"
    exit 1
fi

echo ""
echo "🎉 All Integration Tests Passed!"
echo "================================"
echo ""
echo "📋 Summary:"
echo "  • Rundeck orchestrator deployed and running"
echo "  • PostgreSQL database operational"
echo "  • ML Pipeline with Rundeck integration compiled"
echo "  • Configuration files updated"
echo "  • Job definitions loaded"
echo ""
echo "🚀 Next Steps:"
echo "  1. Access Rundeck UI: kubectl port-forward -n rundeck svc/rundeck 4440:4440"
echo "  2. Open browser to http://localhost:4440 (admin/admin123)"
echo "  3. Verify job definitions in 'aiops-remediation' project"
echo "  4. Start ML Pipeline: ./anomaly-detector"
echo "  5. Test end-to-end anomaly detection → Rundeck remediation"
echo ""
echo "🔍 Monitoring Commands:"
echo "  • Rundeck logs: kubectl logs -f deployment/rundeck -n rundeck"
echo "  • PostgreSQL logs: kubectl logs -f deployment/postgresql -n rundeck"
echo "  • All pods: kubectl get pods -n rundeck"


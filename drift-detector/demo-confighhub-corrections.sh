#!/bin/bash
# Demonstrates ConfigHub-based drift correction (not kubectl)

echo "🔍 ConfigHub-Based Drift Correction Demo"
echo "========================================"
echo ""
echo "This demo shows how drift detector recommends ConfigHub unit updates"
echo "instead of direct kubectl commands."
echo ""

# Check current drift in the cluster
echo "📊 Current Drift Status:"
echo "------------------------"
kubectl get deployment test-app -n drift-test -o jsonpath='Deployment test-app: {.spec.replicas} replicas (expected: 2){"\n"}' 2>/dev/null || echo "test-app not found"
kubectl get deployment complex-app -n drift-test -o jsonpath='Deployment complex-app: {.spec.replicas} replicas (expected: 3){"\n"}' 2>/dev/null || echo "complex-app not found"
kubectl get configmap app-config -n drift-test -o jsonpath='ConfigMap app-config log_level: {.data.log_level} (expected: info){"\n"}' 2>/dev/null || echo "app-config not found"

echo ""
echo "📋 ConfigHub Correction Recommendations:"
echo "----------------------------------------"
echo ""
echo "Instead of using kubectl directly, update ConfigHub units:"
echo ""

# Generate ConfigHub correction commands
cat << 'EOF'
1. Fix deployment/test-app replicas drift:
   ```bash
   # Update the ConfigHub unit (not the live resource)
   cub unit edit deployment-test-app --space drift-detector-dev
   # In editor, change spec.replicas from 5 to 2

   # Or use patch command
   cub unit update deployment-test-app --space drift-detector-dev \
     --patch --data '{"spec":{"replicas":2}}'

   # Apply the corrected unit
   cub unit apply deployment-test-app --space drift-detector-dev
   ```

2. Fix deployment/complex-app replicas drift:
   ```bash
   # Update the ConfigHub unit
   cub unit edit deployment-complex-app --space drift-detector-dev
   # In editor, change spec.replicas from 1 to 3

   # Apply the corrected unit
   cub unit apply deployment-complex-app --space drift-detector-dev
   ```

3. Fix configmap/app-config data drift:
   ```bash
   # Update the ConfigHub unit
   cub unit edit configmap-app-config --space drift-detector-dev
   # In editor, change data.log_level from "debug" to "info"

   # Apply the corrected unit
   cub unit apply configmap-app-config --space drift-detector-dev
   ```

🔧 Bulk Correction Option:
--------------------------
   ```bash
   # Use push-upgrade to fix all drift at once
   cub unit update --space drift-detector-dev --upgrade --patch

   # This propagates all corrections downstream
   ```

🎯 Why ConfigHub Corrections Are Better:
----------------------------------------
1. ✅ Source of Truth: ConfigHub remains the source of truth
2. ✅ Audit Trail: All changes tracked in ConfigHub history
3. ✅ Environment Propagation: Changes flow through dev→staging→prod
4. ✅ Rollback Safety: Easy to revert ConfigHub units
5. ✅ GitOps Compatible: Works with Flux/Argo CD

❌ What NOT to do:
------------------
- kubectl scale deployment test-app --replicas=2  # Bypasses ConfigHub!
- kubectl edit configmap app-config               # Creates config drift!
- kubectl patch deployment                        # Breaks source of truth!

EOF

echo ""
echo "💡 Key Insight:"
echo "---------------"
echo "The drift detector should NEVER use kubectl to fix drift."
echo "Instead, it updates ConfigHub units, which then get applied"
echo "to the cluster through ConfigHub's deployment mechanism."
echo ""
echo "This maintains ConfigHub as the single source of truth!"
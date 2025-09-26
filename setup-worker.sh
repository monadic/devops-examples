#!/bin/bash
# ConfigHub Worker Setup for DevOps Examples
# This script sets up a ConfigHub worker to bridge between ConfigHub and Kubernetes

set -e

echo "🚀 ConfigHub Worker Setup for DevOps Examples"
echo "============================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisites() {
    echo -e "\n${YELLOW}Checking prerequisites...${NC}"

    # Check if cub is installed
    if ! command -v cub &> /dev/null; then
        echo -e "${RED}❌ 'cub' CLI not found. Please install ConfigHub CLI first.${NC}"
        echo "Visit: https://confighub.com/docs/getting-started"
        exit 1
    fi

    # Check if authenticated
    if ! cub auth get-token &> /dev/null; then
        echo -e "${RED}❌ Not authenticated to ConfigHub. Running 'cub auth login'...${NC}"
        cub auth login
    fi

    # Check if kubectl is configured
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}❌ kubectl not configured or cluster not accessible.${NC}"
        echo "Please ensure you have a Kubernetes cluster running (e.g., kind, minikube)"
        exit 1
    fi

    echo -e "${GREEN}✅ All prerequisites met${NC}"
}

# Create worker
create_worker() {
    WORKER_NAME="${1:-devops-worker-$(date +%s)}"
    SPACE="${2:-default}"

    echo -e "\n${YELLOW}Creating ConfigHub worker...${NC}"
    echo "Worker name: $WORKER_NAME"
    echo "Space: $SPACE"

    # Check if worker already exists
    if cub worker list --space "$SPACE" | grep -q "$WORKER_NAME"; then
        echo -e "${YELLOW}⚠️  Worker '$WORKER_NAME' already exists${NC}"
    else
        # Create the worker
        cub worker create "$WORKER_NAME" --space "$SPACE" || {
            echo -e "${RED}❌ Failed to create worker${NC}"
            exit 1
        }
        echo -e "${GREEN}✅ Worker created successfully${NC}"
    fi
}

# Create target for Kind cluster
create_target() {
    CONTEXT="${1:-kind-kind}"
    NAMESPACE="${2:-default}"
    TARGET_NAME="${3:-kind-target-$NAMESPACE}"
    SPACE="${4:-default}"
    WORKER_NAME="${5:-devops-worker}"

    echo -e "\n${YELLOW}Creating Kubernetes target...${NC}"
    echo "Target: $TARGET_NAME"
    echo "Context: $CONTEXT"
    echo "Namespace: $NAMESPACE"

    # Check if target already exists
    if cub target list --space "$SPACE" | grep -q "$TARGET_NAME"; then
        echo -e "${YELLOW}⚠️  Target '$TARGET_NAME' already exists${NC}"
    else
        # Create the target with the worker
        PARAMS='{"KubeContext":"'$CONTEXT'","KubeNamespace":"'$NAMESPACE'","WaitTimeout":"2m0s"}'
        cub target create "$TARGET_NAME" "$PARAMS" "$WORKER_NAME" \
            --space "$SPACE" \
            --provider Kubernetes \
            --toolchain "Kubernetes/YAML" || {
            echo -e "${RED}❌ Failed to create target${NC}"
            echo "Note: Worker needs to be running for target creation to succeed"
        }
        echo -e "${GREEN}✅ Target configured${NC}"
    fi
}

# Update units with target
assign_target_to_units() {
    SPACE="${1:-drift-test-demo}"
    TARGET_NAME="${2:-kind-target-drift-test}"

    echo -e "\n${YELLOW}Assigning target to units in space '$SPACE'...${NC}"

    # Get target ID first
    TARGET_ID=$(cub target get "$TARGET_NAME" --space "$SPACE" --json 2>/dev/null | jq -r '.TargetID' || echo "")

    if [ -z "$TARGET_ID" ]; then
        echo -e "${YELLOW}⚠️  Target not found in space. Skipping unit assignment.${NC}"
        return
    fi

    # List units and update their target
    cub unit list --space "$SPACE" --quiet | while read -r unit_name _; do
        if [ -n "$unit_name" ]; then
            echo "  Updating unit: $unit_name"
            cub unit update "$unit_name" --space "$SPACE" \
                --patch --data '{"TargetID":"'$TARGET_ID'"}' &>/dev/null || {
                echo -e "${YELLOW}    ⚠️  Could not update $unit_name${NC}"
            }
        fi
    done

    echo -e "${GREEN}✅ Units updated with target${NC}"
}

# Main setup
main() {
    echo -e "\n${GREEN}Starting ConfigHub Worker Setup${NC}"

    # Parse arguments
    WORKER_NAME="${1:-devops-worker}"
    KUBE_CONTEXT="${2:-$(kubectl config current-context)}"

    # Check prerequisites
    check_prerequisites

    # Get current Kubernetes context
    echo -e "\n${YELLOW}Kubernetes Configuration:${NC}"
    echo "Current context: $KUBE_CONTEXT"
    kubectl get nodes

    # Create worker
    create_worker "$WORKER_NAME" "default"

    # Create targets for common namespaces
    echo -e "\n${YELLOW}Setting up targets for DevOps examples...${NC}"

    # For drift-detector
    create_target "$KUBE_CONTEXT" "drift-test" "kind-drift-test" "drift-test-demo" "$WORKER_NAME"
    assign_target_to_units "drift-test-demo" "kind-drift-test"

    # For cost-optimizer (uses cluster-wide view)
    create_target "$KUBE_CONTEXT" "default" "kind-default" "default" "$WORKER_NAME"

    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ ConfigHub Worker Setup Complete!${NC}"
    echo -e "${GREEN}========================================${NC}"

    echo -e "\n${YELLOW}Next steps:${NC}"
    echo "1. Run the worker (in a separate terminal):"
    echo -e "   ${GREEN}cub worker run $WORKER_NAME${NC}"
    echo ""
    echo "2. Apply units to Kubernetes:"
    echo -e "   ${GREEN}cub unit apply --all --space drift-test-demo${NC}"
    echo ""
    echo "3. Run the DevOps examples:"
    echo -e "   ${GREEN}cd drift-detector && ./drift-detector${NC}"
    echo -e "   ${GREEN}cd cost-optimizer && ./cost-optimizer${NC}"

    echo -e "\n${YELLOW}Note:${NC} The worker needs to stay running for ConfigHub ↔ Kubernetes sync"
}

# Run main function
main "$@"
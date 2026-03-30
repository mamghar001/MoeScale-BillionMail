#!/bin/bash
# Noez GRE Tunnel Email Routing Setup for BillionMail
# This script configures the host and container to route email through the Noez GRE tunnel
# Usage: sudo bash setup_noez_email_routing.sh

set -e

echo "=========================================="
echo "Noez GRE Tunnel Email Routing Setup"
echo "=========================================="
echo ""

# Configuration
NOEZ_IP="5.230.168.0"
CONTAINER_NAME="billionmail-postfix-billionmail-1"
CONTAINER_NET_NS_IP="172.66.2.100"
DOCKER_BRIDGE_GW="172.66.2.1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use sudo)"
    exit 1
fi

# Check if container is running
print_status "Checking if Postfix container is running..."
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null) || {
    print_error "Container $CONTAINER_NAME not found or not running"
    exit 1
}
print_status "Container PID: $CONTAINER_PID"

# Step 1: Add Noez IP to container's loopback interface
print_status "Step 1: Adding $NOEZ_IP to container's loopback interface..."
nsenter -t $CONTAINER_PID -n ip addr add $NOEZ_IP/32 dev lo 2>/dev/null || {
    print_warning "IP $NOEZ_IP may already exist in container"
}

# Step 2: Add routing rule inside container
print_status "Step 2: Adding routing rules inside container..."
nsenter -t $CONTAINER_PID -n ip rule add from $NOEZ_IP table 100 2>/dev/null || true
nsenter -t $CONTAINER_PID -n ip route add default via $DOCKER_BRIDGE_GW table 100 2>/dev/null || {
    print_warning "Route may already exist"
}

# Step 3: Setup host iptables for packet marking
print_status "Step 3: Setting up host iptables packet marking..."
if ! iptables -t mangle -C PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100 2>/dev/null; then
    iptables -t mangle -A PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100
    print_status "Added mangle rule for packet marking"
else
    print_status "Mangle rule already exists"
fi

# Step 4: Add policy routing rule on host
print_status "Step 4: Setting up host policy routing..."
ip rule del fwmark 100 table 20 2>/dev/null || true
ip rule add fwmark 100 table 20 prio 50

# Step 5: Add SNAT rule
print_status "Step 5: Setting up SNAT..."
if ! iptables -t nat -C POSTROUTING -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP 2>/dev/null; then
    iptables -t nat -I POSTROUTING 1 -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP
    print_status "Added SNAT rule"
else
    print_status "SNAT rule already exists"
fi

# Step 6: Allow forwarding
print_status "Step 6: Setting up forwarding rules..."
if ! iptables -C FORWARD -s $NOEZ_IP -j ACCEPT 2>/dev/null; then
    iptables -I FORWARD -s $NOEZ_IP -j ACCEPT
fi
if ! iptables -C FORWARD -d $NOEZ_IP -j ACCEPT 2>/dev/null; then
    iptables -I FORWARD -d $NOEZ_IP -j ACCEPT
fi

# Step 7: Ensure ip rule for Noez IP exists
print_status "Step 7: Verifying Noez IP routing rule..."
ip rule del from $NOEZ_IP table 20 2>/dev/null || true
ip rule add from $NOEZ_IP table 20 prio 200

# Flush route cache
ip route flush cache

echo ""
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""

# Test connectivity
print_status "Testing connectivity from container..."
if nsenter -t $CONTAINER_PID -n ping -c 2 8.8.8.8 > /dev/null 2>&1; then
    print_status "✅ Container can reach internet via GRE tunnel!"
else
    print_error "❌ Container cannot reach internet"
    exit 1
fi

echo ""
print_status "Current routing rules:"
ip rule | grep -E "(fwmark|from $NOEZ_IP|172.66.2)"

echo ""
print_status "Current SNAT rule:"
iptables -t nat -L POSTROUTING -n -v | grep $NOEZ_IP | head -1

echo ""
echo "=========================================="
print_status "Noez GRE tunnel routing is now active!"
echo "Emails from moescaleb2b.site will be sent from $NOEZ_IP"
echo "=========================================="

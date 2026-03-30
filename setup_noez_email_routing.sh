#!/bin/bash
# Noez GRE Tunnel Email Routing Setup for BillionMail
# This script configures the host and container to route email through the Noez GRE tunnel
#
# IMPORTANT: You MUST customize the CONFIGURATION section below for your setup!
#
# Usage: sudo bash setup_noez_email_routing.sh

set -e

echo "=========================================="
echo "Noez GRE Tunnel Email Routing Setup"
echo "=========================================="
echo ""

# =============================================================================
# CONFIGURATION - MODIFY THESE VALUES FOR YOUR SETUP
# =============================================================================

# Your Noez IP (the IP you want to send emails from)
NOEZ_IP="5.230.168.0"

# Your host's main IP (the IP Noez created the GRE tunnel to)
HOST_IP="85.121.241.162"

# Noez GRE tunnel remote endpoint (provided by Noez)
NOEZ_GRE_REMOTE="5.230.205.35"

# GRE tunnel internal IPs (usually provided by Noez)
GRE_LOCAL="192.168.31.2"
GRE_REMOTE="192.168.31.1"
GRE_SUBNET="192.168.31.0/30"

# Docker container settings
CONTAINER_NAME="billionmail-postfix-billionmail-1"
CONTAINER_NET_NS_IP="172.66.2.100"  # The container's IP in the Docker network
DOCKER_BRIDGE_GW="172.66.2.1"       # The Docker bridge gateway

# Domain to use for Noez IP sending
DOMAIN="moescaleb2b.site"

# =============================================================================
# END CONFIGURATION
# =============================================================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_section() {
    echo ""
    echo -e "${BLUE}==========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}==========================================${NC}"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use sudo)"
    exit 1
fi

print_section "STEP 0: Validating Configuration"

# Validate configuration
if [ "$NOEZ_IP" == "YOUR_NOEZ_IP_HERE" ]; then
    print_error "You MUST edit this script and set your Noez IP!"
    echo "Open the script and modify the CONFIGURATION section."
    exit 1
fi

if [ "$HOST_IP" == "YOUR_HOST_IP_HERE" ]; then
    print_error "You MUST edit this script and set your host IP!"
    exit 1
fi

# Check if GRE tunnel exists
if ! ip link show gre1 &>/dev/null; then
    print_warning "GRE tunnel 'gre1' not found!"
    echo ""
    echo "You need to set up the GRE tunnel first. Run these commands:"
    echo ""
    echo "  ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255"
    echo "  ip addr add $GRE_LOCAL/30 dev gre1"
    echo "  ip addr add $NOEZ_IP/32 dev gre1"
    echo "  ip link set gre1 up"
    echo "  ip rule add from $NOEZ_IP table 20 prio 200"
    echo "  ip route add default via $GRE_REMOTE dev gre1 table 20"
    echo ""
    read -p "Do you want me to set up the GRE tunnel now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Setting up GRE tunnel..."
        ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255 || true
        ip addr add $GRE_LOCAL/30 dev gre1 || true
        ip addr add $NOEZ_IP/32 dev gre1 || true
        ip link set gre1 up
        ip rule add from $NOEZ_IP table 20 prio 200 || true
        ip route add default via $GRE_REMOTE dev gre1 table 20 || true
        print_status "GRE tunnel setup complete!"
    else
        print_error "Please set up the GRE tunnel manually first, then run this script again."
        exit 1
    fi
else
    print_status "GRE tunnel 'gre1' found"
fi

# Check if container is running
print_status "Checking if Postfix container is running..."
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null) || {
    print_error "Container $CONTAINER_NAME not found or not running"
    echo ""
    echo "Make sure BillionMail is running:"
    echo "  cd /opt/billionmail && docker compose up -d"
    exit 1
}
print_status "Container PID: $CONTAINER_PID"

print_section "STEP 1: Adding Noez IP to Container"

# Add Noez IP to container's loopback interface
print_status "Adding $NOEZ_IP to container's loopback interface..."
if nsenter -t $CONTAINER_PID -n ip addr add $NOEZ_IP/32 dev lo 2>/dev/null; then
    print_status "Successfully added $NOEZ_IP to container"
else
    print_warning "IP $NOEZ_IP may already exist in container (this is OK)"
fi

# Verify
if nsenter -t $CONTAINER_PID -n ip addr show lo | grep -q "$NOEZ_IP"; then
    print_status "✓ Verified: $NOEZ_IP is in container"
else
    print_error "Failed to add $NOEZ_IP to container"
    exit 1
fi

print_section "STEP 2: Container Routing Setup"

# Add routing rule inside container
print_status "Adding routing rules inside container..."
nsenter -t $CONTAINER_PID -n ip rule add from $NOEZ_IP table 100 2>/dev/null || true
nsenter -t $CONTAINER_PID -n ip route add default via $DOCKER_BRIDGE_GW table 100 2>/dev/null || {
    print_warning "Route may already exist (this is OK)"
}

# Verify
if nsenter -t $CONTAINER_PID -n ip route show table 100 | grep -q "default"; then
    print_status "✓ Verified: Container routing rule added"
else
    print_warning "Container routing rule may not be set correctly"
fi

print_section "STEP 3: Host Packet Marking (Mangle)"

# Setup host iptables for packet marking
print_status "Setting up host iptables packet marking..."
if ! iptables -t mangle -C PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100 2>/dev/null; then
    iptables -t mangle -A PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100
    print_status "✓ Added mangle rule for packet marking"
else
    print_status "✓ Mangle rule already exists"
fi

print_section "STEP 4: Host Policy Routing"

# Add policy routing rule on host
print_status "Setting up host policy routing..."
ip rule del fwmark 100 table 20 2>/dev/null || true
ip rule add fwmark 100 table 20 prio 50
print_status "✓ Added policy routing rule (fwmark 100 -> table 20)"

# Ensure ip rule for Noez IP exists
print_status "Verifying Noez IP routing rule..."
ip rule del from $NOEZ_IP table 20 2>/dev/null || true
ip rule add from $NOEZ_IP table 20 prio 200
print_status "✓ Added Noez IP routing rule"

print_section "STEP 5: Source NAT (SNAT) Setup"

# Add SNAT rule
print_status "Setting up SNAT..."
if ! iptables -t nat -C POSTROUTING -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP 2>/dev/null; then
    iptables -t nat -I POSTROUTING 1 -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP
    print_status "✓ Added SNAT rule"
else
    print_status "✓ SNAT rule already exists"
fi

print_section "STEP 6: Firewall Rules"

# Allow forwarding
print_status "Setting up forwarding rules..."
ADDED_RULES=0

if ! iptables -C FORWARD -s $NOEZ_IP -j ACCEPT 2>/dev/null; then
    iptables -I FORWARD -s $NOEZ_IP -j ACCEPT
    print_status "✓ Added FORWARD rule for outgoing traffic"
    ((ADDED_RULES++))
else
    print_status "✓ FORWARD rule for outgoing already exists"
fi

if ! iptables -C FORWARD -d $NOEZ_IP -j ACCEPT 2>/dev/null; then
    iptables -I FORWARD -d $NOEZ_IP -j ACCEPT
    print_status "✓ Added FORWARD rule for incoming traffic"
    ((ADDED_RULES++))
else
    print_status "✓ FORWARD rule for incoming already exists"
fi

# Flush route cache
ip route flush cache
print_status "✓ Route cache flushed"

print_section "STEP 7: Testing Connectivity"

# Test connectivity
echo "Testing connectivity from container to internet..."
if nsenter -t $CONTAINER_PID -n ping -c 3 8.8.8.8 > /dev/null 2>&1; then
    print_status "✅ SUCCESS: Container can reach internet via GRE tunnel!"
else
    print_error "❌ FAILED: Container cannot reach internet"
    echo ""
    echo "Troubleshooting tips:"
    echo "1. Check if GRE tunnel is up: ip link show gre1"
    echo "2. Check routing: ip route show table 20"
    echo "3. Check iptables: iptables -t nat -L POSTROUTING -n -v"
    exit 1
fi

print_section "Configuration Summary"

echo "Current routing rules:"
echo "----------------------"
ip rule | grep -E "(fwmark|from $NOEZ_IP|172.66)" || true

echo ""
echo "Current SNAT rule:"
echo "------------------"
iptables -t nat -L POSTROUTING -n -v | grep "$NOEZ_IP" | head -1 || true

echo ""
echo "Container network:"
echo "------------------"
echo "Container IP: $CONTAINER_NET_NS_IP"
nsenter -t $CONTAINER_PID -n ip addr show lo | grep "$NOEZ_IP" || true

print_section "✅ Setup Complete!"

echo ""
echo "Your BillionMail server is now configured to send emails from:"
echo "  IP: $NOEZ_IP"
echo "  Domain: $DOMAIN"
echo "  Via: GRE tunnel to $NOEZ_GRE_REMOTE"
echo ""
echo "To test, run:"
echo "  docker exec -i $CONTAINER_NAME sh -c 'echo \"Test\" | sendmail -v test@example.com'"
echo ""
echo "To make this persistent across reboots, add this script to /etc/rc.local"
echo "or run it manually after each container restart."
echo ""
echo "=========================================="

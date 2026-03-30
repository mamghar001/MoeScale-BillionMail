#!/bin/bash
# Add Additional Noez IP for BillionMail Email Sending
# This script adds a new Noez IP to your existing GRE tunnel setup
#
# Usage: sudo bash add_noez_ip.sh <NOEZ_IP> <DOMAIN>
# Example: sudo bash add_noez_ip.sh 5.230.168.1 anotherdomain.com

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_section() { echo -e "\n${BLUE}==========================================${NC}"; echo -e "${BLUE}$1${NC}"; echo -e "${BLUE}==========================================${NC}"; }

# Check root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use sudo)"
    exit 1
fi

# Get arguments
NEW_NOEZ_IP="${1:-}"
NEW_DOMAIN="${2:-}"

# Validate arguments
if [ -z "$NEW_NOEZ_IP" ] || [ -z "$NEW_DOMAIN" ]; then
    echo "Usage: sudo bash add_noez_ip.sh <NOEZ_IP> <DOMAIN>"
    echo "Example: sudo bash add_noez_ip.sh 5.230.168.1 anotherdomain.com"
    exit 1
fi

# Configuration (modify these if different from your setup)
CONTAINER_NAME="billionmail-postfix-billionmail-1"
CONTAINER_NET_NS_IP="172.66.2.100"

print_section "Adding New Noez IP: $NEW_NOEZ_IP for Domain: $NEW_DOMAIN"

# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null) || {
    print_error "Container $CONTAINER_NAME not found"
    exit 1
}

print_status "Container PID: $CONTAINER_PID"

# Step 1: Add Noez IP to container
print_section "Step 1: Adding $NEW_NOEZ_IP to Container"
nsenter -t $CONTAINER_PID -n ip addr add $NEW_NOEZ_IP/32 dev lo 2>/dev/null || {
    print_warning "IP $NEW_NOEZ_IP may already exist"
}

# Add container routing
nsenter -t $CONTAINER_PID -n ip rule add from $NEW_NOEZ_IP table 100 2>/dev/null || true

print_status "✓ Added $NEW_NOEZ_IP to container"

# Step 2: Add iptables rules
print_section "Step 2: Adding iptables Rules"

# Check if this is the same subnet as existing IPs
# For additional IPs in the same subnet, we just need SNAT
# For different subnets, we need additional routing

# Add SNAT rule for this specific IP
if ! iptables -t nat -C POSTROUTING -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NEW_NOEZ_IP 2>/dev/null; then
    iptables -t nat -I POSTROUTING 1 -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NEW_NOEZ_IP
    print_status "✓ Added SNAT rule for $NEW_NOEZ_IP"
else
    print_status "✓ SNAT rule already exists"
fi

# Add forwarding rules
iptables -C FORWARD -s $NEW_NOEZ_IP -j ACCEPT 2>/dev/null || iptables -I FORWARD -s $NEW_NOEZ_IP -j ACCEPT
iptables -C FORWARD -d $NEW_NOEZ_IP -j ACCEPT 2>/dev/null || iptables -I FORWARD -d $NEW_NOEZ_IP -j ACCEPT

# Add routing rule
ip rule del from $NEW_NOEZ_IP table 20 2>/dev/null || true
ip rule add from $NEW_NOEZ_IP table 20 prio 200

ip route flush cache

print_status "✓ Added routing and firewall rules"

# Step 3: Add Postfix transport
print_section "Step 3: Adding Postfix Transport"

TRANSPORT_NAME="smtp_bind_ip_$(echo $NEW_NOEZ_IP | tr '.' '_')"

# Check if transport already exists in master.cf
if grep -q "^$TRANSPORT_NAME" /opt/billionmail/conf/postfix/master.cf; then
    print_status "✓ Transport $TRANSPORT_NAME already exists in master.cf"
else
    cat >> /opt/billionmail/conf/postfix/master.cf << EOF

# Noez IP for $NEW_DOMAIN
$TRANSPORT_NAME unix  -       -       n       -       -       smtp
EOF
    print_status "✓ Added transport $TRANSPORT_NAME to master.cf"
fi

# Step 4: Add domain mapping in database
print_section "Step 4: Adding Domain Mapping"

cd /opt/billionmail
source .env 2>/dev/null || true

# Check if domain already exists
docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -t -c \
    "SELECT COUNT(*) FROM bm_domain_smtp_transport WHERE domain = '@$NEW_DOMAIN';" 2>/dev/null | grep -q "1" && {
    print_warning "Domain @$NEW_DOMAIN already exists, updating..."
    docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
        "UPDATE bm_domain_smtp_transport SET smtp_name = '$TRANSPORT_NAME' WHERE domain = '@$NEW_DOMAIN';" 2>/dev/null
} || {
    docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
        "INSERT INTO bm_domain_smtp_transport (atype, domain, smtp_name) VALUES ('dedicated_ip', '@$NEW_DOMAIN', '$TRANSPORT_NAME');" 2>/dev/null
    print_status "✓ Added domain mapping: @$NEW_DOMAIN -> $TRANSPORT_NAME"
}

# Step 5: Reload Postfix
print_section "Step 5: Reloading Postfix"
docker exec $CONTAINER_NAME postfix reload 2>/dev/null || true
print_status "✓ Postfix reloaded"

# Step 6: Test
print_section "Step 6: Testing"
echo "Testing connectivity..."
if nsenter -t $CONTAINER_PID -n ping -c 2 8.8.8.8 > /dev/null 2>&1; then
    print_status "✅ Container connectivity: OK"
else
    print_warning "⚠️  Container connectivity test failed"
fi

print_section "✅ Setup Complete!"
echo ""
echo "New configuration:"
echo "  IP: $NEW_NOEZ_IP"
echo "  Domain: $NEW_DOMAIN"
echo "  Transport: $TRANSPORT_NAME"
echo ""
echo "Test command:"
echo "  docker exec -i $CONTAINER_NAME sh -c 'cat << \"EOF\" | sendmail -v test@mail-tester.com"
echo "From: admin@$NEW_DOMAIN"
echo "To: test@mail-tester.com"
echo "Subject: Test"
echo ""
echo "Test"
echo "EOF'"
echo ""

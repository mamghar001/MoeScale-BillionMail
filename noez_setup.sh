#!/bin/bash
# Noez GRE Tunnel Complete Setup for BillionMail
# One script to handle everything from fresh install to working email sending
#
# Usage:
#   sudo bash noez_setup.sh              # Run complete setup
#   sudo bash noez_setup.sh test         # Test email sending
#   sudo bash noez_setup.sh add IP DOMAIN # Add new IP/domain
#   sudo bash noez_setup.sh status       # Check status
#
# Version: 3.0 - Now with API support!

set -e

# =============================================================================
# CONFIGURATION - EDIT THESE VALUES FOR YOUR SETUP
# =============================================================================

# REQUIRED: Your Noez IP (the IP you want to send emails from)
NOEZ_IP="5.230.168.0"

# REQUIRED: Your VPS public IP (the IP Noez created the GRE tunnel to)
HOST_IP="85.121.241.162"

# REQUIRED: Noez GRE tunnel remote endpoint (provided by Noez)
NOEZ_GRE_REMOTE="5.230.205.35"

# REQUIRED: Domain to use for sending
DOMAIN="moescaleb2b.site"

# GRE tunnel internal IPs (usually provided by Noez)
GRE_LOCAL="192.168.31.2"     # Your side
GRE_REMOTE="192.168.31.1"    # Noez side
GRE_SUBNET="192.168.31.0/30" # GRE subnet

# BillionMail API Settings (optional - for API-based domain creation)
# Leave empty to use database directly (faster, works without API)
BM_API_URL=""  # e.g., "https://mail.yourdomain.com"
BM_API_TOKEN=""  # Your API token

# Docker settings (auto-detected if possible)
CONTAINER_NAME="billionmail-postfix-billionmail-1"
CONTAINER_NET_NS_IP="172.66.2.100"  # Will be auto-detected
DOCKER_BRIDGE_GW="172.66.2.1"       # Will be auto-detected

# =============================================================================
# END CONFIGURATION
# =============================================================================

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[✓]${NC} $1"; }
print_error() { echo -e "${RED}[✗]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
print_info() { echo -e "${CYAN}[i]${NC} $1"; }
print_section() { 
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Check root
if [ "$EUID" -ne 0 ]; then 
    print_error "Please run as root (use sudo)"
    exit 1
fi

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load .env for DB credentials
source .env 2>/dev/null || true

# Auto-detect Docker network settings
detect_docker_network() {
    print_info "Auto-detecting Docker network settings..."
    
    # Find the multi-ip network (usually contains "b2bscale" or "net-" in name)
    CONTAINER_NET_NS_IP=$(docker inspect -f '{{json .NetworkSettings.Networks}}' $CONTAINER_NAME 2>/dev/null | \
        python3 -c "import sys,json; d=json.load(sys.stdin); nets=[v['IPAddress'] for k,v in d.items() if 'b2bscale' in k or 'net-' in k]; print(nets[0] if nets else list(d.values())[0]['IPAddress'])")
    
    if [ -z "$CONTAINER_NET_NS_IP" ]; then
        print_warning "Could not auto-detect container IP"
        # Fallback to hardcoded if available
        if [ -n "$CONTAINER_NET_NS_IP" ] && [ "$CONTAINER_NET_NS_IP" != "172.66.2.100" ]; then
            print_info "Using configured IP: $CONTAINER_NET_NS_IP"
        fi
    else
        # Extract gateway from IP (e.g., 172.66.2.100 -> 172.66.2.1)
        DOCKER_BRIDGE_GW=$(echo $CONTAINER_NET_NS_IP | sed 's/\.[0-9]*$/.1/')
        print_status "Detected container IP: $CONTAINER_NET_NS_IP, Gateway: $DOCKER_BRIDGE_GW"
    fi
}

# Validate configuration
validate_config() {
    print_section "Configuration"
    
    local errors=0
    local interactive=0
    
    # Check if running interactively
    if [ -t 0 ]; then
        interactive=1
    fi
    
    # NOEZ_IP
    if [ "$NOEZ_IP" == "YOUR_NOEZ_IP_HERE" ] || [ -z "$NOEZ_IP" ]; then
        if [ $interactive -eq 1 ]; then
            print_warning "NOEZ_IP not configured"
            read -p "Enter your Noez IP (e.g., 5.230.168.0): " NOEZ_IP
        else
            print_error "NOEZ_IP not configured! Run interactively or edit the script."
            ((errors++))
        fi
    fi
    
    # HOST_IP
    if [ "$HOST_IP" == "YOUR_VPS_IP_HERE" ] || [ -z "$HOST_IP" ]; then
        if [ $interactive -eq 1 ]; then
            print_warning "HOST_IP not configured"
            # Try to auto-detect
            DETECTED_IP=$(ip route get 8.8.8.8 2>/dev/null | head -1 | grep -oP 'src \K[\d.]+')
            if [ -n "$DETECTED_IP" ]; then
                read -p "Enter your VPS IP [detected: $DETECTED_IP]: " HOST_IP
                HOST_IP=${HOST_IP:-$DETECTED_IP}
            else
                read -p "Enter your VPS IP: " HOST_IP
            fi
        else
            print_error "HOST_IP not configured! Run interactively or edit the script."
            ((errors++))
        fi
    fi
    
    # NOEZ_GRE_REMOTE
    if [ -z "$NOEZ_GRE_REMOTE" ] || [ "$NOEZ_GRE_REMOTE" == "NOEZ_ENDPOINT" ]; then
        if [ $interactive -eq 1 ]; then
            print_warning "NOEZ_GRE_REMOTE not configured"
            read -p "Enter Noez GRE remote endpoint (from Noez panel): " NOEZ_GRE_REMOTE
        else
            print_error "NOEZ_GRE_REMOTE not configured! Run interactively or edit the script."
            ((errors++))
        fi
    fi
    
    # DOMAIN
    if [ -z "$DOMAIN" ] || [ "$DOMAIN" == "yourdomain.com" ]; then
        if [ $interactive -eq 1 ]; then
            print_warning "DOMAIN not configured"
            read -p "Enter domain to send from: " DOMAIN
        else
            print_error "DOMAIN not configured! Run interactively or edit the script."
            ((errors++))
        fi
    fi
    
    if [ $errors -gt 0 ]; then
        echo ""
        print_error "Configuration incomplete!"
        print_info "Either:"
        print_info "  1. Run this script interactively (with a TTY)"
        print_info "  2. Edit $0 and set the values in the CONFIGURATION section"
        exit 1
    fi
    
    # Show final configuration
    echo ""
    print_status "Configuration:"
    echo "  Noez IP: $NOEZ_IP"
    echo "  Host IP: $HOST_IP"
    echo "  GRE Endpoint: $NOEZ_GRE_REMOTE"
    echo "  Domain: $DOMAIN"
    echo ""
    
    if [ $interactive -eq 1 ]; then
        read -p "Is this correct? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_error "Configuration cancelled"
            exit 1
        fi
    fi
    
    print_status "Configuration confirmed!"
}

# Add domain to BillionMail (via API or database)
add_domain_to_billionmail() {
    local DOMAIN_TO_ADD="$1"
    
    print_section "Adding Domain to BillionMail: $DOMAIN_TO_ADD"
    
    # Check if domain already exists
    local domain_exists=$(docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -t -c \
        "SELECT COUNT(*) FROM domain WHERE domain='$DOMAIN_TO_ADD';" 2>/dev/null | xargs)
    
    if [ "$domain_exists" == "1" ]; then
        print_status "Domain $DOMAIN_TO_ADD already exists in BillionMail"
        return 0
    fi
    
    # Method 1: API (if configured)
    if [ -n "$BM_API_URL" ] && [ -n "$BM_API_TOKEN" ]; then
        print_info "Using BillionMail API to add domain..."
        
        # Create domain via API
        local response=$(curl -s -w "\n%{http_code}" -X POST "${BM_API_URL}/api/v1/domain" \
            -H "Authorization: Bearer ${BM_API_TOKEN}" \
            -H "Content-Type: application/json" \
            -d "{\"domain\":\"${DOMAIN_TO_ADD}\",\"mailboxes\":50,\"mailbox_quota\":5368709120,\"quota\":10737418240}" 2>/dev/null)
        
        local http_code=$(echo "$response" | tail -n1)
        local body=$(echo "$response" | sed '$d')
        
        if [ "$http_code" == "200" ] || [ "$http_code" == "201" ]; then
            print_status "✓ Domain created via API"
            return 0
        else
            print_warning "API call failed (HTTP $http_code), falling back to database"
            print_info "Response: $body"
        fi
    fi
    
    # Method 2: Direct database insert
    print_info "Adding domain directly to database..."
    
    local current_time=$(date +%s)
    
    docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
        "INSERT INTO domain (domain, a_record, mailboxes, mailbox_quota, quota, rate_limit, create_time, active, urls, hasbrandinfo, current_usage) 
         VALUES ('$DOMAIN_TO_ADD', '', 50, 5368709120, 10737418240, 12, $current_time, 1, '{}', 0, 0) 
         ON CONFLICT (domain) DO NOTHING;" 2>/dev/null
    
    # Verify
    domain_exists=$(docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -t -c \
        "SELECT COUNT(*) FROM domain WHERE domain='$DOMAIN_TO_ADD';" 2>/dev/null | xargs)
    
    if [ "$domain_exists" == "1" ]; then
        print_status "✓ Domain added to BillionMail"
    else
        print_error "Failed to add domain"
        return 1
    fi
}

# Setup GRE tunnel
setup_gre_tunnel() {
    print_section "Setting Up GRE Tunnel"
    
    # Check if GRE tunnel already exists
    if ip link show gre1 &>/dev/null; then
        print_status "GRE tunnel 'gre1' already exists"
        
        # Check if it has our IP
        if ip addr show gre1 | grep -q "$NOEZ_IP"; then
            print_status "GRE tunnel already configured with $NOEZ_IP"
            return 0
        else
            print_warning "GRE tunnel exists but may have different configuration"
            read -p "Recreate GRE tunnel? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                ip link del gre1 2>/dev/null || true
            else
                return 0
            fi
        fi
    fi
    
    print_info "Creating GRE tunnel..."
    print_info "  Local: $HOST_IP"
    print_info "  Remote: $NOEZ_GRE_REMOTE"
    print_info "  Noez IP: $NOEZ_IP"
    
    # Create GRE tunnel
    ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255 || {
        print_error "Failed to create GRE tunnel!"
        print_info "You may need to load the gre module: modprobe ip_gre"
        return 1
    }
    
    # Add addresses
    ip addr add $GRE_LOCAL/30 dev gre1
    ip addr add $NOEZ_IP/32 dev gre1
    
    # Bring up
    ip link set gre1 up
    
    # Add routing policy
    ip rule add from $NOEZ_IP table 20 prio 200 2>/dev/null || true
    ip route add default via $GRE_REMOTE dev gre1 table 20 2>/dev/null || true
    
    print_status "GRE tunnel created successfully!"
    
    # Test
    print_info "Testing GRE tunnel..."
    if ping -c 3 -I $NOEZ_IP 8.8.8.8 &>/dev/null; then
        print_status "✅ GRE tunnel working! Can reach internet via Noez IP"
    else
        print_warning "⚠️  GRE tunnel may not be fully operational yet"
        print_info "This is normal - continue with setup"
    fi
}

# Setup container networking
setup_container() {
    print_section "Setting Up Container Networking"
    
    # Get container PID
    CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null) || {
        print_error "Container $CONTAINER_NAME not found!"
        print_info "Start BillionMail first: docker compose up -d"
        exit 1
    }
    
    print_status "Container PID: $CONTAINER_PID"
    
    # Add Noez IP to container's loopback
    print_info "Adding $NOEZ_IP to container..."
    if nsenter -t $CONTAINER_PID -n ip addr add $NOEZ_IP/32 dev lo 2>/dev/null; then
        print_status "Added $NOEZ_IP to container"
    else
        print_warning "$NOEZ_IP may already exist in container"
    fi
    
    # Add container routing
    nsenter -t $CONTAINER_PID -n ip rule add from $NOEZ_IP table 100 2>/dev/null || true
    nsenter -t $CONTAINER_PID -n ip route add default via $DOCKER_BRIDGE_GW table 100 2>/dev/null || true
    
    print_status "Container networking configured!"
}

# Setup host iptables and routing
setup_host_routing() {
    print_section "Setting Up Host Routing"
    
    # Mark packets from container
    print_info "Adding iptables packet marking..."
    if ! iptables -t mangle -C PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100 2>/dev/null; then
        iptables -t mangle -A PREROUTING -s $CONTAINER_NET_NS_IP -j MARK --set-mark 100
        print_status "Added packet marking rule"
    else
        print_status "Packet marking rule already exists"
    fi
    
    # Add policy routing
    print_info "Adding policy routing..."
    ip rule del fwmark 100 table 20 2>/dev/null || true
    ip rule add fwmark 100 table 20 prio 50
    print_status "Added policy routing (mark 100 -> table 20)"
    
    # Add SNAT
    print_info "Adding SNAT rule..."
    if ! iptables -t nat -C POSTROUTING -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP 2>/dev/null; then
        iptables -t nat -I POSTROUTING 1 -s $CONTAINER_NET_NS_IP -j SNAT --to-source $NOEZ_IP
        print_status "Added SNAT rule"
    else
        print_status "SNAT rule already exists"
    fi
    
    # Add forwarding rules
    print_info "Adding forwarding rules..."
    iptables -C FORWARD -s $NOEZ_IP -j ACCEPT 2>/dev/null || iptables -I FORWARD -s $NOEZ_IP -j ACCEPT
    iptables -C FORWARD -d $NOEZ_IP -j ACCEPT 2>/dev/null || iptables -I FORWARD -d $NOEZ_IP -j ACCEPT
    print_status "Added forwarding rules"
    
    # Flush route cache
    ip route flush cache
    print_status "Route cache flushed"
}

# Setup Postfix
setup_postfix() {
    print_section "Configuring Postfix"
    
    TRANSPORT_NAME="smtp_bind_ip_$(echo $NOEZ_IP | tr '.' '_')"
    
    # Check if transport already exists
    if grep -q "^$TRANSPORT_NAME" conf/postfix/master.cf; then
        print_status "Transport $TRANSPORT_NAME already exists"
    else
        print_info "Adding transport to master.cf..."
        cat >> conf/postfix/master.cf << EOF

# Noez IP for $DOMAIN
$TRANSPORT_NAME unix  -       -       n       -       -       smtp
EOF
        print_status "Added transport $TRANSPORT_NAME"
    fi
    
    # Add domain mapping in database
    print_info "Adding domain transport mapping..."
    
    # Check if mapping already exists
    local existing=$(docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -t -c \
        "SELECT COUNT(*) FROM bm_domain_smtp_transport WHERE domain = '@$DOMAIN';" 2>/dev/null | xargs)
    
    if [ "$existing" == "1" ]; then
        print_info "Domain transport mapping already exists, updating..."
        docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
            "UPDATE bm_domain_smtp_transport SET smtp_name = '$TRANSPORT_NAME' WHERE domain = '@$DOMAIN';" 2>/dev/null
    else
        docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
            "INSERT INTO bm_domain_smtp_transport (atype, domain, smtp_name) VALUES ('dedicated_ip', '@$DOMAIN', '$TRANSPORT_NAME');" 2>/dev/null
        print_status "Added domain transport mapping: @$DOMAIN -> $TRANSPORT_NAME"
    fi
    
    # Reload Postfix
    print_info "Reloading Postfix..."
    docker exec $CONTAINER_NAME postfix reload 2>/dev/null || {
        print_warning "Postfix reload failed, may need restart"
    }
    print_status "Postfix configured!"
}

# Test connectivity
test_connectivity() {
    print_section "Testing Connectivity"
    
    CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null)
    
    print_info "Testing container internet connectivity..."
    if nsenter -t $CONTAINER_PID -n ping -c 3 8.8.8.8 &>/dev/null; then
        print_status "✅ Container can reach internet!"
    else
        print_error "❌ Container cannot reach internet"
        print_info "Check: ip rule, ip route show table 20, iptables -t nat -L"
        return 1
    fi
    
    print_info "Testing DNS resolution..."
    if nsenter -t $CONTAINER_PID -n nslookup google.com &>/dev/null; then
        print_status "✅ DNS working!"
    else
        print_warning "⚠️  DNS may have issues"
    fi
    
    return 0
}

# Send test email
test_email() {
    print_section "Sending Test Email"
    
    echo ""
    print_info "Getting test email address from mail-tester.com..."
    print_info "Please go to: https://www.mail-tester.com/"
    read -p "Paste the test email address (or press Enter for default): " TEST_EMAIL
    
    if [ -z "$TEST_EMAIL" ]; then
        TEST_EMAIL="test-$(date +%s)@srv1.mail-tester.com"
        print_info "Using: $TEST_EMAIL"
    fi
    
    echo ""
    print_info "Sending test email from $DOMAIN via $NOEZ_IP..."
    
    docker exec -i $CONTAINER_NAME sh -c "cat << 'EOFMAIL' | sendmail -v $TEST_EMAIL
From: admin@$DOMAIN
To: $TEST_EMAIL
Subject: Test from Noez IP $NOEZ_IP
MIME-Version: 1.0
Content-Type: text/plain

This is a test email from BillionMail.

Details:
- Domain: $DOMAIN
- Source IP: $NOEZ_IP
- Sent via: GRE Tunnel
- Timestamp: $(date)

If you received this, the setup is working!
EOFMAIL" 2>&1 | tail -20
    
    echo ""
    print_status "Test email sent!"
    print_info "Check your score at: https://www.mail-tester.com/"
    print_info "Or check logs: tail -f logs/postfix/mail.log"
}

# Add new IP/domain
add_new_ip() {
    local NEW_IP="$1"
    local NEW_DOMAIN="$2"
    
    if [ -z "$NEW_IP" ] || [ -z "$NEW_DOMAIN" ]; then
        print_error "Usage: $0 add <NOEZ_IP> <DOMAIN>"
        exit 1
    fi
    
    print_section "Adding New IP: $NEW_IP for Domain: $NEW_DOMAIN"
    
    # Save original values
    ORIG_NOEZ_IP="$NOEZ_IP"
    ORIG_DOMAIN="$DOMAIN"
    
    # Set new values
    NOEZ_IP="$NEW_IP"
    DOMAIN="$NEW_DOMAIN"
    
    # Detect network first
    detect_docker_network
    
    # Validate we have container network settings
    if [ -z "$CONTAINER_NET_NS_IP" ]; then
        print_error "Could not detect container network settings"
        print_info "Please set CONTAINER_NET_NS_IP manually in the script"
        exit 1
    fi
    
    # Add domain to BillionMail first
    add_domain_to_billionmail "$NEW_DOMAIN"
    
    # Run setup steps
    setup_container
    setup_host_routing
    setup_postfix
    test_connectivity
    
    # Restore original values
    NOEZ_IP="$ORIG_NOEZ_IP"
    DOMAIN="$ORIG_DOMAIN"
    
    print_section "✅ New IP Added Successfully!"
    print_info "IP: $NEW_IP"
    print_info "Domain: $NEW_DOMAIN"
    print_info ""
    print_info "Note: If using API, domain should appear in BillionMail UI immediately."
    print_info "If using database method, refresh the BillionMail UI to see the domain."
}

# Show status
show_status() {
    print_section "Noez GRE Tunnel Status"
    
    echo ""
    echo -e "${CYAN}Configuration:${NC}"
    echo "  Noez IP: $NOEZ_IP"
    echo "  Domain: $DOMAIN"
    echo "  Host IP: $HOST_IP"
    echo "  GRE Remote: $NOEZ_GRE_REMOTE"
    
    echo ""
    echo -e "${CYAN}GRE Tunnel:${NC}"
    if ip link show gre1 &>/dev/null; then
        echo -e "  ${GREEN}✓${NC} GRE tunnel exists"
        ip addr show gre1 | grep "inet " | head -1 | sed 's/^/  /'
    else
        echo -e "  ${RED}✗${NC} GRE tunnel not found"
    fi
    
    echo ""
    echo -e "${CYAN}Container:${NC}"
    if docker ps | grep -q "$CONTAINER_NAME"; then
        echo -e "  ${GREEN}✓${NC} Container running"
        CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null)
        if nsenter -t $CONTAINER_PID -n ip addr show lo 2>/dev/null | grep -q "$NOEZ_IP"; then
            echo -e "  ${GREEN}✓${NC} Noez IP ($NOEZ_IP) in container"
        else
            echo -e "  ${RED}✗${NC} Noez IP not in container"
        fi
    else
        echo -e "  ${RED}✗${NC} Container not running"
    fi
    
    echo ""
    echo -e "${CYAN}Routing Rules:${NC}"
    ip rule | grep -E "(fwmark|from $NOEZ_IP)" | sed 's/^/  /' || echo "  (none found)"
    
    echo ""
    echo -e "${CYAN}iptables SNAT:${NC}"
    iptables -t nat -L POSTROUTING -n -v 2>/dev/null | grep "$NOEZ_IP" | head -1 | sed 's/^/  /' || echo "  (not found)"
    
    echo ""
    echo -e "${CYAN}Domain in BillionMail:${NC}"
    local domain_count=$(docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -t -c \
        "SELECT COUNT(*) FROM domain WHERE domain='$DOMAIN';" 2>/dev/null | xargs)
    if [ "$domain_count" == "1" ]; then
        echo -e "  ${GREEN}✓${NC} Domain '$DOMAIN' exists in BillionMail"
    else
        echo -e "  ${RED}✗${NC} Domain '$DOMAIN' NOT found in BillionMail"
    fi
    
    echo ""
    echo -e "${CYAN}Transport Mapping:${NC}"
    docker exec -i $CONTAINER_NAME psql -U billionmail -d billionmail -c \
        "SELECT domain, smtp_name FROM bm_domain_smtp_transport WHERE domain='@$DOMAIN';" 2>/dev/null | sed 's/^/  /' || echo "  (database error)"
}

# Main function
main() {
    local COMMAND="${1:-setup}"
    
    case "$COMMAND" in
        setup|"")
            print_section "Noez GRE Tunnel Setup for BillionMail"
            validate_config
            detect_docker_network
            
            # Add domain to BillionMail first
            add_domain_to_billionmail "$DOMAIN"
            
            setup_gre_tunnel
            setup_container
            setup_host_routing
            setup_postfix
            test_connectivity
            
            print_section "✅ Setup Complete!"
            print_info "Your domain $DOMAIN is now configured to send from $NOEZ_IP"
            print_info "Run: $0 test  - to send a test email"
            print_info "Run: $0 status - to check status"
            print_info ""
            print_info "Note: Domain should appear in BillionMail UI. If not visible immediately,"
            print_info "      refresh the page or check the Domain Management section."
            ;;
        test)
            validate_config
            detect_docker_network
            test_connectivity && test_email
            ;;
        add)
            add_new_ip "$2" "$3"
            ;;
        status)
            show_status
            ;;
        remove)
            print_error "Remove functionality not yet implemented"
            print_info "To remove an IP, manually:"
            print_info "  1. Remove from container: nsenter -t PID ip addr del IP/32 dev lo"
            print_info "  2. Remove iptables rules"
            print_info "  3. Remove from database"
            exit 1
            ;;
        help|--help|-h)
            echo "Usage: $0 [COMMAND]"
            echo ""
            echo "Commands:"
            echo "  setup       Run complete setup (default)"
            echo "  test        Test email sending"
            echo "  add IP DOM  Add new IP and domain"
            echo "  status      Show current status"
            echo "  help        Show this help"
            echo ""
            echo "Examples:"
            echo "  sudo $0                    # Run setup"
            echo "  sudo $0 test               # Send test email"
            echo "  sudo $0 add 5.230.168.1 newdomain.com"
            echo ""
            echo "API Configuration (optional):"
            echo "  Edit BM_API_URL and BM_API_TOKEN in the script to use API"
            echo "  instead of direct database access for domain creation."
            ;;
        *)
            print_error "Unknown command: $COMMAND"
            echo "Run '$0 help' for usage"
            exit 1
            ;;
    esac
}

# Run main
main "$@"

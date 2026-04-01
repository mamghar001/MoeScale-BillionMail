#!/bin/bash
# One-Command Install for BillionMail MoeScale V5.0.0 (with Noez GRE Tunnel Support)
# Usage: curl -sSL https://raw.githubusercontent.com/mamghar001/MoeScale-BillionMail/moescale-fixed/one-command-install.sh | sudo bash
# Or: sudo bash one-command-install.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BILLIONMAIL_DIR="/opt/billionmail"
REPO_URL="https://github.com/mamghar001/MoeScale-BillionMail.git"
BRANCH="moescale-fixed"  # Updated to use the branch with Noez support

# Logging
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if root
check_root() {
    if [ "$(id -u)" != "0" ]; then
        error "This script must be run as root"
        exit 1
    fi
    success "Running as root"
}

# Check OS
check_os() {
    log "Checking operating system..."
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VERSION=$VERSION_ID
    else
        error "Cannot detect OS"
        exit 1
    fi
    
    case $OS in
        "Ubuntu")
            if [ "$VERSION" != "20.04" ] && [ "$VERSION" != "22.04" ] && [ "$VERSION" != "24.04" ]; then
                warning "Ubuntu $VERSION not tested. Recommended: 20.04 or 22.04"
            fi
            ;;
        "Debian GNU/Linux")
            warning "Debian detected. Ubuntu recommended."
            ;;
        *)
            error "OS $OS not supported. Use Ubuntu 20.04 or 22.04"
            exit 1
            ;;
    esac
    success "OS check passed: $OS $VERSION"
}

# Check and install dependencies
install_dependencies() {
    log "Installing dependencies..."
    
    apt-get update -qq
    
    # Essential packages
    PACKAGES="curl wget git jq net-tools dnsutils"
    
    # Docker check
    if ! command -v docker &> /dev/null; then
        log "Installing Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl enable docker
        systemctl start docker
        success "Docker installed"
    else
        success "Docker already installed"
    fi
    
    # Docker Compose check
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log "Installing Docker Compose..."
        apt-get install -y docker-compose-plugin
        success "Docker Compose installed"
    else
        success "Docker Compose already installed"
    fi
    
    apt-get install -y $PACKAGES -qq
    success "Dependencies installed"
}

# Get user input
gather_info() {
    log "Gathering configuration information..."
    echo ""
    echo "==================================="
    echo "BillionMail Configuration"
    echo "==================================="
    echo ""
    
    # Main domain
    read -p "Enter your main domain (e.g., b2bscale.xyz): " MAIN_DOMAIN
    if [ -z "$MAIN_DOMAIN" ]; then
        error "Domain is required"
        exit 1
    fi
    
    # Admin credentials
    read -p "Enter admin username [admin]: " ADMIN_USER
    ADMIN_USER=${ADMIN_USER:-admin}
    
    read -s -p "Enter admin password: " ADMIN_PASS
    echo ""
    if [ -z "$ADMIN_PASS" ]; then
        ADMIN_PASS=$(openssl rand -base64 12)
        warning "Generated random password: $ADMIN_PASS"
    fi
    
    # Safe path
    read -p "Enter admin panel path [admin888]: " SAFE_PATH
    SAFE_PATH=${SAFE_PATH:-admin888}
    
    # IPs
    log "Detecting server IP..."
    PRIMARY_IP=$(curl -s4 ifconfig.me || curl -s4 icanhazip.com || hostname -I | awk '{print $1}')
    read -p "Enter primary server IP [$PRIMARY_IP]: " VPS_IP
    VPS_IP=${VPS_IP:-$PRIMARY_IP}
    
    read -p "Enter additional IPs (comma-separated, or press Enter for none): " EXTRA_IPS
    
    # SMTP Relay option
    echo ""
    echo "==================================="
    echo "For reliable Gmail/Hotmail delivery, SMTP relay is recommended."
    echo "Options: SendGrid (free 100/day), Amazon SES, Mailgun"
    echo ""
    read -p "Set up SMTP relay now? (yes/no) [no]: " SETUP_RELAY
    SETUP_RELAY=${SETUP_RELAY:-no}
    
    if [[ "$SETUP_RELAY" == "yes" || "$SETUP_RELAY" == "y" ]]; then
        echo "Select relay provider:"
        echo "1) SendGrid (free 100/day)"
        echo "2) Amazon SES"
        echo "3) Mailgun"
        echo "4) Other"
        read -p "Choice [1]: " RELAY_CHOICE
        RELAY_CHOICE=${RELAY_CHOICE:-1}
        
        case $RELAY_CHOICE in
            1) RELAY_HOST="smtp.sendgrid.net"; RELAY_PORT="587" ;;
            2) RELAY_HOST="email-smtp.us-east-1.amazonaws.com"; RELAY_PORT="587" ;;
            3) RELAY_HOST="smtp.mailgun.org"; RELAY_PORT="587" ;;
            4) 
                read -p "Enter SMTP host: " RELAY_HOST
                read -p "Enter SMTP port [587]: " RELAY_PORT
                RELAY_PORT=${RELAY_PORT:-587}
                ;;
        esac
        
        read -p "Enter SMTP username: " RELAY_USER
        read -s -p "Enter SMTP password: " RELAY_PASS
        echo ""
    fi
    
    # Noez GRE Tunnel option
    echo ""
    echo "==================================="
    echo "Noez GRE Tunnel Setup (Optional)"
    echo "For sending from dedicated Noez IP addresses via GRE tunnel."
    echo "Requires: Noez account with GRE tunnel service"
    echo ""
    read -p "Set up Noez GRE Tunnel now? (yes/no) [no]: " SETUP_NOEZ
    SETUP_NOEZ=${SETUP_NOEZ:-no}
    
    if [[ "$SETUP_NOEZ" == "yes" || "$SETUP_NOEZ" == "y" ]]; then
        echo ""
        echo "Enter Noez configuration:"
        read -p "Noez IP address (e.g., 5.230.168.0): " NOEZ_IP
        read -p "Noez GRE endpoint (from Noez panel, e.g., 5.230.205.35): " NOEZ_GRE_REMOTE
        read -p "Cloudflare API token (optional, for auto-DNS): " CF_API_TOKEN
        CF_API_TOKEN=${CF_API_TOKEN:-}
    fi
    
    # Generate secure passwords (alphanumeric only, no base64 padding)
    # Note: Base64 passwords with == padding break DockerEnv which splits on =
    DB_PASS=$(LC_ALL=C </dev/urandom tr -dc A-Za-z0-9 2>/dev/null | head -c 32)
    REDIS_PASS=$(LC_ALL=C </dev/urandom tr -dc A-Za-z0-9 2>/dev/null | head -c 32)
    
    success "Configuration gathered"
}

# Setup server IPs
setup_ips() {
    log "Setting up IP addresses..."
    
    # Detect network interface
    INTERFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    success "Network interface: $INTERFACE"
    
    # Add primary IP if not already set
    if ! ip addr show dev $INTERFACE | grep -q "$VPS_IP"; then
        warning "Primary IP $VPS_IP not found on interface"
        warning "Make sure your VPS has this IP assigned"
    fi
    
    # Add extra IPs
    if [ -n "$EXTRA_IPS" ]; then
        IFS=',' read -ra IPS <<< "$EXTRA_IPS"
        for ip in "${IPS[@]}"; do
            ip=$(echo $ip | xargs) # trim whitespace
            if ! ip addr show dev $INTERFACE | grep -q "$ip"; then
                log "Adding IP $ip to $INTERFACE"
                ip addr add $ip/32 dev $INTERFACE 2>/dev/null || warning "Failed to add $ip (may need manual setup)"
            else
                success "IP $ip already configured"
            fi
        done
    fi
    
    # Make IPs persistent
    NETPLAN_FILE="/etc/netplan/50-cloud-init.yaml"
    if [ -f "$NETPLAN_FILE" ]; then
        warning "To make IPs persistent, add them to $NETPLAN_FILE"
        warning "Example:"
        echo "            addresses:"
        echo "            - $VPS_IP/24"
        for ip in $EXTRA_IPS; do
            echo "            - $ip/32"
        done
    fi
}

# Clone repository
clone_repo() {
    log "Cloning BillionMail repository..."
    
    if [ -d "$BILLIONMAIL_DIR" ]; then
        warning "Directory $BILLIONMAIL_DIR exists"
        read -p "Remove and reinstall? (yes/no) [no]: " REINSTALL
        if [[ "$REINSTALL" == "yes" || "$REINSTALL" == "y" ]]; then
            rm -rf $BILLIONMAIL_DIR
        else
            error "Installation cancelled"
            exit 1
        fi
    fi
    
    git clone -b "$BRANCH" "$REPO_URL" "$BILLIONMAIL_DIR"
    cd "$BILLIONMAIL_DIR"
    success "Repository cloned to $BILLIONMAIL_DIR"
}

# Create .env file
create_env() {
    log "Creating environment configuration..."
    
    cat > .env << EOF
# Database
DBUSER=billionmail
DBNAME=billionmail
DBPASS=$DB_PASS

# Redis
REDISPASS=$REDIS_PASS

# Timezone
TZ=UTC

# Hostname
BILLIONMAIL_HOSTNAME=$MAIN_DOMAIN

# Admin credentials
ADMIN_USERNAME=$ADMIN_USER
ADMIN_PASSWORD=$ADMIN_PASS
SafePath=$SAFE_PATH

# Network
IPV4_NETWORK=172.66.1

# Ports
HTTP_PORT=80
HTTPS_PORT=443
SMTP_PORT=25
SMTPS_PORT=465
SUBMISSION_PORT=587
IMAP_PORT=143
IMAPS_PORT=993
POP_PORT=110
POPS_PORT=995
SQL_PORT=25432
REDIS_PORT=26379

# Settings
RETENTION_DAYS=7
FAIL2BAN_INIT=y
IP_WHITELIST_ENABLE=false
EOF

    success ".env file created"
}

# Fix SQL configs with correct password (before install)
fix_sql_configs() {
    log "Configuring SQL connection files..."
    
    # Update all Postfix SQL config files with correct password
    for f in conf/postfix/sql/pgsql_*.cf; do
        if [ -f "$f" ]; then
            sed -i "s/^password = .*/password = $DB_PASS/" "$f"
        fi
    done
    
    # Update Dovecot SQL config with correct password
    if [ -f "conf/dovecot/conf.d/dovecot-sql.conf.ext" ]; then
        sed -i "s/^connect = .*/connect = host=pgsql dbname=$DBNAME user=$DBUSER password=$DB_PASS/" conf/dovecot/conf.d/dovecot-sql.conf.ext
    fi
    
    # Update Dovecot SSL config to use correct domain
    if [ -f "conf/dovecot/conf.d/10-ssl.conf" ]; then
        # Replace any existing mail.* domain with mail.$MAIN_DOMAIN
        sed -i "s/mail\.[^.]*\.[^.]*{/mail.$MAIN_DOMAIN {/g" conf/dovecot/conf.d/10-ssl.conf
        sed -i "s|/mail\.[^/]*/|/mail.$MAIN_DOMAIN/|g" conf/dovecot/conf.d/10-ssl.conf
    fi
    
    # Update Rspamd Redis config with correct password
    if [ -f "conf/rspamd/local.d/redis.conf" ]; then
        sed -i "s/^servers.*password=.*/servers = \"redis:6379\";\n  password = \"$REDIS_PASS\";/" conf/rspamd/local.d/redis.conf
    fi
    
    success "SQL configs updated"
}

# Clean up old domain configs from previous installations
cleanup_old_configs() {
    log "Cleaning up old configuration files..."
    
    # Reset DKIM signing config to clean state (domains will be re-added via API)
    if [ -f "conf/rspamd/local.d/dkim_signing.conf" ]; then
        # Keep the header, remove all domain blocks
        head -1 conf/rspamd/local.d/dkim_signing.conf > /tmp/dkim_signing.conf.tmp
        cat >> /tmp/dkim_signing.conf.tmp << 'EOF'

use_esld = false;

domain {
#BT_DOMAIN_DKIM_BEGIN

}
#BT_DOMAIN_DKIM_END
EOF
        mv /tmp/dkim_signing.conf.tmp conf/rspamd/local.d/dkim_signing.conf
    fi
    
    # Clean up old vmail_ssl.map entries
    if [ -f "conf/postfix/conf/vmail_ssl.map" ]; then
        echo -n > conf/postfix/conf/vmail_ssl.map
    fi
    
    success "Old configs cleaned up"
}

# Run install script
run_install() {
    log "Running BillionMail install script..."
    
    # Clean up old configs from previous installations
    cleanup_old_configs
    
    # Fix SQL configs before install
    fix_sql_configs
    
    # Fix Postfix myhostname to match domain (for PTR record alignment)
    log "Configuring Postfix..."
    sed -i "s/^myhostname = .*/myhostname = $MAIN_DOMAIN/" conf/postfix/main.cf 2>/dev/null || true
    
    chmod +x install.sh
    
    # Remove existing config files to avoid overwrite prompts
    rm -f billionmail.conf .env 2>/dev/null || true
    
    # Re-create .env file (install.sh uses it)
    cat > .env << EOF
DBUSER=billionmail
DBNAME=billionmail
DBPASS=$DB_PASS
REDISPASS=$REDIS_PASS
TZ=UTC
BILLIONMAIL_HOSTNAME=$MAIN_DOMAIN
ADMIN_USERNAME=$ADMIN_USER
ADMIN_PASSWORD=$ADMIN_PASS
SafePath=$SAFE_PATH
IPV4_NETWORK=172.66.1
HTTP_PORT=80
HTTPS_PORT=443
SMTP_PORT=25
SMTPS_PORT=465
SUBMISSION_PORT=587
IMAP_PORT=143
IMAPS_PORT=993
POP_PORT=110
POPS_PORT=995
SQL_PORT=25432
REDIS_PORT=26379
RETENTION_DAYS=7
FAIL2BAN_INIT=y
IP_WHITELIST_ENABLE=false
EOF
    
    # Run install in background and show progress
    ./install.sh 2>&1 | tee install.log &
    INSTALL_PID=$!
    
    # Show spinner
    spin='-\|/'
    i=0
    while kill -0 $INSTALL_PID 2>/dev/null; do
        i=$(( (i+1) %4 ))
        printf "\r${spin:$i:1} Installing... (this may take 5-10 minutes)"
        sleep .5
    done
    printf "\r"
    
    wait $INSTALL_PID
    if [ $? -eq 0 ]; then
        success "BillionMail installed successfully"
    else
        error "Installation failed. Check install.log"
        exit 1
    fi
}

# Setup SMTP relay if requested
setup_smtp_relay() {
    if [[ "$SETUP_RELAY" == "yes" || "$SETUP_RELAY" == "y" ]]; then
        log "Setting up SMTP relay..."
        
        # Create sasl_passwd
        echo "$RELAY_USER:$RELAY_PASS" > conf/postfix/conf/sasl_passwd
        docker exec billionmail-postfix-billionmail-1 postmap /etc/postfix/conf/sasl_passwd 2>/dev/null || true
        
        # Add to database
        docker exec -e PGPASSWORD=$DB_PASS billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail << EOF
INSERT INTO bm_relay_config (remark, rtype, relay_host, relay_port, auth_user, auth_password, active)
VALUES ('SMTP Relay', 'custom', '$RELAY_HOST', $RELAY_PORT, '$RELAY_USER', '$RELAY_PASS', 1)
ON CONFLICT DO NOTHING;
EOF
        
        # Map domains to relay
        docker exec -e PGPASSWORD=$DB_PASS billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail << EOF
INSERT INTO bm_relay_domain_mapping (sender_domain, relay_id)
SELECT domain, 1 FROM domain
ON CONFLICT DO NOTHING;
EOF
        
        # Restart postfix
        docker compose restart postfix-billionmail
        
        success "SMTP relay configured: $RELAY_HOST"
    fi
}

# Setup Noez GRE Tunnel if requested
setup_noez() {
    if [[ "$SETUP_NOEZ" == "yes" || "$SETUP_NOEZ" == "y" ]]; then
        log "Setting up Noez GRE Tunnel..."
        
        if [ ! -f "$BILLIONMAIL_DIR/noez_setup.sh" ]; then
            warning "Noez setup script not found. Skipping Noez setup."
            return
        fi
        
        cd "$BILLIONMAIL_DIR"
        
        # Create noez_setup.env
        cat > noez_setup.env << EOF
# Noez GRE Tunnel Configuration
NOEZ_IP="$NOEZ_IP"
HOST_IP="$VPS_IP"
NOEZ_GRE_REMOTE="$NOEZ_GRE_REMOTE"
DOMAIN="$MAIN_DOMAIN"
ALL_NOEZ_IPS="$NOEZ_IP"
CF_API_TOKEN="$CF_API_TOKEN"
EOF
        
        # Run Noez setup
        log "Running Noez setup..."
        bash noez_setup.sh || warning "Noez setup had some issues. Check logs above."
        
        success "Noez GRE Tunnel configured for IP: $NOEZ_IP"
    fi
}

# Verify installation
verify_install() {
    log "Verifying installation..."
    
    cd "$BILLIONMAIL_DIR"
    
    # Check containers
    if docker compose ps | grep -q "running"; then
        success "Containers are running"
    else
        error "Containers not running properly"
        docker compose ps
        exit 1
    fi
    
    # Wait for core to be ready
    log "Waiting for core service to initialize..."
    sleep 5
    
    # Check if core is responding (JWT fix verification)
    if curl -s http://localhost:${HTTP_PORT:-80}/api/health > /dev/null 2>&1 || curl -s http://localhost:${HTTP_PORT:-80}/ > /dev/null 2>&1; then
        success "Core web service is responding"
    else
        warning "Core service may still be initializing (JWT requires DBPASS/REDISPASS)"
        warning "If installation hangs, check: docker logs billionmail-core-billionmail-1"
    fi
    
    # Check Postfix config
    MYHOSTNAME=$(docker exec billionmail-postfix-billionmail-1 postconf -h myhostname 2>/dev/null)
    if [ "$MYHOSTNAME" == "$MAIN_DOMAIN" ]; then
        success "Postfix hostname correct: $MYHOSTNAME"
    else
        warning "Postfix hostname is $MYHOSTNAME (expected $MAIN_DOMAIN)"
    fi
    
    # Test database connection from Postfix container
    if docker exec -e PGPASSWORD=$DB_PASS billionmail-postfix-billionmail-1 psql -h pgsql -U billionmail -d billionmail -c "SELECT 1;" > /dev/null 2>&1; then
        success "Database connection from Postfix OK"
    else
        warning "Database connection from Postfix failed - may need SQL config fix"
    fi
    
    # Test database connection
    if docker exec -e PGPASSWORD=$DB_PASS billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c "SELECT 1;" > /dev/null 2>&1; then
        success "Database connection OK"
    else
        error "Database connection failed"
    fi
    
    success "Installation verification complete"
}

# Display final info
show_info() {
    echo ""
    echo "==================================="
    echo "✅ BILLIONMAIL INSTALLATION COMPLETE"
    echo "==================================="
    echo ""
    echo "🔗 Access URLs:"
    echo "  HTTPS: https://$VPS_IP/$SAFE_PATH"
    echo "  Domain: https://$MAIN_DOMAIN/$SAFE_PATH (after DNS setup)"
    echo ""
    echo "🔐 Admin Credentials:"
    echo "  Username: $ADMIN_USER"
    echo "  Password: $ADMIN_PASS"
    echo ""
    echo "📁 Installation Directory: $BILLIONMAIL_DIR"
    echo ""
    echo "⚙️  Useful Commands:"
    echo "  cd $BILLIONMAIL_DIR"
    echo "  sudo docker compose ps        # Check status"
    echo "  sudo docker compose logs -f   # View logs"
    echo "  sudo bash bm.sh               # Management menu"
    echo ""
    echo "📧 Testing Email:"
    echo "  echo 'Test' | sendmail -v yourgmail@gmail.com"
    echo ""
    
    if [[ "$SETUP_RELAY" == "yes" || "$SETUP_RELAY" == "y" ]]; then
        echo "✉️  SMTP Relay: $RELAY_HOST"
        echo "   Gmail/Hotmail delivery: ENABLED"
    else
        echo "⚠️  SMTP Relay: NOT CONFIGURED"
        echo "   For Gmail/Hotmail delivery, run:"
        echo "   sudo bash $BILLIONMAIL_DIR/setup_smtp_relay.sh"
    fi
    
    if [[ "$SETUP_NOEZ" == "yes" || "$SETUP_NOEZ" == "y" ]]; then
        echo ""
        echo "🌐 Noez GRE Tunnel: CONFIGURED"
        echo "   IP: $NOEZ_IP"
        echo "   Sending domain: $MAIN_DOMAIN"
        echo "   Test: Send email from admin@$MAIN_DOMAIN"
    fi
    
    echo ""
    echo "📝 NEXT STEPS:"
    echo "1. Set up DNS records (A, MX, SPF, DKIM, DMARC)"
    echo "2. Add domains in BillionMail web UI"
    echo "3. Generate DKIM keys"
    echo "4. Test email delivery"
    
    if [[ "$SETUP_NOEZ" == "yes" || "$SETUP_NOEZ" == "y" ]]; then
        echo ""
        echo "📚 Noez Documentation:"
        echo "   cat $BILLIONMAIL_DIR/NOEZ_SETUP.md"
        echo "   cat $BILLIONMAIL_DIR/SKILLS.md (for AI agents)"
    fi
    
    echo ""
    echo "==================================="
}

# Main execution
main() {
    echo "==================================="
    echo "BillionMail MoeScale V5.0.0"
    echo "One-Command Installer (with Noez Support)"
    echo "==================================="
    echo ""
    
    check_root
    check_os
    install_dependencies
    gather_info
    setup_ips
    clone_repo
    create_env
    run_install
    setup_smtp_relay
    setup_noez
    verify_install
    show_info
}

# Run main function
main

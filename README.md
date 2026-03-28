# BillionMail - MoeScale Edition

> **🚀 ONE-COMMAND INSTALL (Recommended):**
> ```bash
> sudo curl -sSL https://raw.githubusercontent.com/mamghar001/MoeScale-BillionMail/moescale-fixed/one-command-install.sh | sudo bash
> ```
> This installs the latest working version with all fixes applied!

---

## 📦 Quick Start Options

### Option 1: One-Command Install (Easiest)
```bash
sudo curl -sSL https://raw.githubusercontent.com/mamghar001/MoeScale-BillionMail/moescale-fixed/one-command-install.sh | sudo bash
```

This script will:
- Check your system requirements
- Install Docker & dependencies
- Ask for your domain and configuration
- Set up IP addresses
- Clone and install BillionMail V5.0.0
- Configure everything automatically

### Option 2: Manual Install
```bash
# Clone the latest stable version (MoeScale V5.0.0)
git clone -b MoeScale-V5.0.0 https://github.com/mamghar001/MoeScale-BillionMail.git

# Or checkout the tag after cloning
git checkout MoeScale-V5.0.0
```

**Why use V5.0.0?** This version includes critical fixes for:
- Multi-IP configuration
- Database connection issues
- SPF record problems
- External email delivery

---

## ✅ Setup Checklist

Before installing, complete these steps:

### 1. Server Requirements
- [ ] Ubuntu 20.04 or 22.04 VPS
- [ ] 4GB+ RAM, 20GB+ disk space
- [ ] Root access
- [ ] Ports open: 25, 465, 587, 80, 443, 110, 143, 993, 995

### 2. Domain & DNS Setup
- [ ] Register your domain (e.g., example.com)
- [ ] Point domain to your VPS IP:
  - A record: `@` → Your VPS IP
  - A record: `mail` → Your VPS IP
  - MX record: `@` → `mail.yourdomain.com` (priority 10)
  - SPF: `v=spf1 +a +mx +ip4:YOUR_IP ~all`
  - DMARC: `v=DMARC1;p=quarantine;rua=mailto:dmarc@yourdomain.com`
  - (After install) DKIM: Add keys from BillionMail web UI

### 3. IP Configuration
- [ ] Primary IP assigned to VPS
- [ ] Additional IPs added to network interface (if using multi-IP)

### 4. Credentials (for later)
- [ ] Prepare admin password for BillionMail
- [ ] (Optional) Get Cloudflare API token for DNS sync
- [ ] (Optional) Get SMTP relay credentials (SendGrid/AWS SES) for Gmail delivery

### 5. Install
- [ ] Run one-command install
- [ ] Complete web UI setup
- [ ] Generate DKIM keys
- [ ] Add DKIM DNS records
- [ ] Test email delivery

---

## ✨ What's New in V5.0.0

### Critical Fixes Applied:
1. **Network Configuration** - Fixed Docker subnet mismatch (172.66.2.0/24)
2. **Database Passwords** - Fixed all SQL config files
3. **Postfix Configuration** - Fixed myhostname and multi-IP routing
4. **SPF Records** - Disabled broken relayhost_maps causing failures
5. **External Delivery** - Emails now deliver to mail-tester.com and external providers

### 🔒 Security Improvement:
**All credentials removed from code!** No hardcoded API tokens, passwords, or sensitive data. All configuration is now done via:
- Environment variables (`.env` file)
- User input during installation
- Generated at runtime

**This makes the repository safe to share publicly.**

---

## 📋 Prerequisites

See the [✅ Setup Checklist](#-setup-checklist) above for complete requirements.

**Quick Summary:**
- Ubuntu 20.04/22.04 VPS with 4GB+ RAM
- Domain with DNS records (A, MX, SPF, DMARC)
- Ports 25, 465, 587, 80, 443, 110, 143, 993, 995 open
- Primary IP assigned to VPS (additional IPs optional for multi-domain)

---

## 🚀 Installation

### One-Command Install:
```bash
sudo curl -sSL https://raw.githubusercontent.com/mamghar001/MoeScale-BillionMail/moescale-fixed/one-command-install.sh | sudo bash
```

The script will guide you through:
- Domain setup
- IP configuration
- Admin credentials
- Optional SMTP relay (for reliable Gmail/Hotmail delivery)

### After Installation:

1. **Access BillionMail:**
   ```
   https://YOUR_IP/admin-path
   ```

2. **Add your domain in the web UI:**
   - Go to Domain Management
   - Add your domain
   - Generate DKIM keys
   - Add DKIM TXT records to DNS

3. **Test email delivery:**
   ```bash
   echo "Test email" | sendmail -v your-email@gmail.com
   ```

---

## 🔐 Configuration & Credentials

### Environment Variables

All sensitive configuration is done through environment variables. Create a `.env` file:

```bash
cd /opt/billionmail
cat > .env << 'EOF'
# Database
DBUSER=billionmail
DBNAME=billionmail
DBPASS=your_secure_password_here

# Redis
REDISPASS=your_redis_password_here

# Main Domain
BILLIONMAIL_HOSTNAME=your-domain.com

# Admin Credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your_admin_password
SafePath=admin888

# Network
IPV4_NETWORK=172.66.1
EOF
```

### API Tokens (for helper scripts)

If using helper scripts like `dns_auto_sync.py` or `setup_forwards.sh`, set credentials via environment variables:

```bash
# Export before running scripts
export BILLIONMAIL_TOKEN="your_billionmail_api_token"
export NAMECHEAP_API_USER="your_username"
export NAMECHEAP_API_KEY="your_api_key"
export CLOUDFLARE_API_TOKEN="your_cloudflare_token"
export VPS_IP="your_server_ip"

# Then run the script
python3 dns_auto_sync.py
```

**⚠️ Never commit credentials to git!** The `.env` file is in `.gitignore` by default.

---

## 🔧 Management Commands

After installation, use the `bm` command:

```bash
cd /opt/billionmail

# View status
sudo bash bm.sh status

# Restart all services
sudo bash bm.sh restart

# Restart specific service
sudo bash bm.sh restart-service postfix

# View logs
sudo bash bm.sh log-container postfix
sudo bash bm.sh log-file postfix

# Multi-IP setup (after adding domains)
sudo bash bm.sh multi_ip
```

---

## ⚠️ Important Notes

### IP Reputation & Gmail/Hotmail

**Your IPs may be blocked initially** by Gmail and Hotmail due to:
- Fresh IP with no sending history
- Possible blocklist presence

**Solutions:**
1. **Use SMTP Relay** (Recommended) - Set up during install or run:
   ```bash
   sudo bash /opt/billionmail/setup_smtp_relay.sh
   ```
   This routes emails through SendGrid/Amazon SES for immediate delivery.

2. **IP Warmup** - Send gradually (10-50 emails/day) for 2-4 weeks

3. **Check Blocklists:**
   - https://www.spamhaus.org/query/ip/YOUR_IP
   - https://mxtoolbox.com/blacklists.aspx

### DNS Configuration

**SPF Record Example (include ALL your IPs):**
```
v=spf1 +a +mx +ip4:YOUR_PRIMARY_IP +ip4:YOUR_SECONDARY_IP_1 +ip4:YOUR_SECONDARY_IP_2 ~all
```

**Example with multiple IPs:**
```
v=spf1 +a +mx +ip4:203.0.113.10 +ip4:203.0.113.11 +ip4:203.0.113.12 ~all
```

---

## 🐛 Troubleshooting

### Issue: "address resolver failure"
**Fix:** Database passwords were wrong - fixed in V5.0.0

### Issue: "SPF fail - not authorized"
**Fix:** Update SPF record to include your server's IP

### Issue: "The IP you're using to send mail is not authorized"
**Fix:** This is Gmail blocking your IP. Use SMTP relay or wait for IP warmup.

### Check logs:
```bash
sudo tail -f /opt/billionmail/logs/postfix/mail.log
```

---

## 📂 Repository Structure

```
.
├── conf/               # Configuration files (Postfix, Dovecot, etc.)
├── core/               # BillionMail core application
├── one-command-install.sh  # ⭐ One-command installer
├── bm.sh               # Management script
├── install.sh          # Traditional installer
├── docker-compose.yml  # Docker configuration
└── README.md           # This file
```

---

## 🏷️ Releases

| Version | Status | Notes |
|---------|--------|-------|
| MoeScale-V5.0.0 | ✅ Latest | Working version with all fixes |
| MoeScale-V4.9.0 | ⚠️ Old | Has bugs (database, network, SPF) |

**Always use V5.0.0 or newer for fresh installs.**

### What's in V5.0.0?
- ✅ One-command install script
- ✅ Fixed multi-IP configuration
- ✅ Fixed database connection issues
- ✅ Fixed SPF/Email delivery problems
- ✅ No hardcoded credentials (security)
- ✅ Complete setup checklist
- ✅ Cloudflare DNS sync script

---

## 📞 Support

- **Issues:** https://github.com/mamghar001/MoeScale-BillionMail/issues
- **Releases:** https://github.com/mamghar001/MoeScale-BillionMail/releases

---

## 📝 License

MIT License - See original BillionMail repository for details.

---

**⭐ Star this repository if it helps you!**

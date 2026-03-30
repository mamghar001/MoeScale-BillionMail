# Noez GRE Tunnel Setup for BillionMail

> **ONE SCRIPT - ONE COMMAND - DOES EVERYTHING**
>
> ✅ GRE Tunnel Setup
> ✅ BillionMail Domain Configuration  
> ✅ Container Networking
> ✅ Postfix Configuration
> ✅ Cloudflare DNS (automatic SPF, A, DMARC records)
> ✅ Auto-start on Boot

## 🚀 ONE COMMAND SETUP

```bash
sudo bash /opt/billionmail/noez_setup.sh
```

That's it! This single command will:
- Create auto-start service (boot persistence)
- Create GRE tunnel
- Add domain to BillionMail
- Setup Cloudflare DNS (SPF, A, DMARC records)
- Configure container networking
- Setup iptables rules
- Configure Postfix
- Test connectivity

## 📋 Prerequisites

- VPS with public IP and root access
- Noez account with GRE tunnel service
- BillionMail installed (`/opt/billionmail`)
- Cloudflare account (for automatic DNS)

## 🔧 Configuration

Edit `/opt/billionmail/noez_setup.sh` and set:

```bash
# REQUIRED: Your Noez IP
NOEZ_IP="5.230.168.0"

# REQUIRED: Your VPS public IP
HOST_IP="YOUR_VPS_IP_HERE"

# REQUIRED: Noez GRE endpoint (from Noez panel)
NOEZ_GRE_REMOTE="5.230.205.35"

# REQUIRED: Domain to send from
DOMAIN="yourdomain.com"

# ALL Noez IPs for auto-setup on boot
ALL_NOEZ_IPS="5.230.168.0 5.230.168.1 5.230.168.10"

# Cloudflare API (for automatic DNS)
CF_API_TOKEN="your-cloudflare-token"  # From https://dash.cloudflare.com/profile/api-tokens
CF_ZONE_ID=""  # Optional - script auto-detects
```

**Get Cloudflare API Token:**
1. Go to: https://dash.cloudflare.com/profile/api-tokens
2. Create token with: `Zone:Read`, `DNS:Edit` permissions
3. Copy the token

## 📊 Commands

| Command | Description |
|---------|-------------|
| `sudo bash noez_setup.sh` | **COMPLETE SETUP** - does everything |
| `sudo bash noez_setup.sh test` | Send test email |
| `sudo bash noez_setup.sh add IP DOMAIN` | Add new IP/domain |
| `sudo bash noez_setup.sh status` | Check status |

## 🔄 Adding More IPs/Domains

```bash
# Add another IP and domain with automatic DNS
sudo bash noez_setup.sh add 5.230.168.X newdomain.com

# The script will:
# - Add domain to BillionMail
# - Create A record: mail.DOMAIN -> HOST_IP
# - Create SPF record: v=spf1 ip4:NOEZ_IP ~all
# - Create DMARC record
# - Setup container networking
# - Configure Postfix
# - Add to auto-start service
```

Then update `ALL_NOEZ_IPS`:
```bash
ALL_NOEZ_IPS="5.230.168.0 5.230.168.1 5.230.168.10 5.230.168.X"
```

## ✅ What the Script Does

### 1. Auto-Start Service
- Creates `/opt/billionmail/setup_noez_ips.sh`
- Creates `/etc/systemd/system/noez-ips.service`
- Enables service with `systemctl enable`
- Starts service immediately

### 2. GRE Tunnel Setup
- Creates tunnel if not exists
- Adds Noez IP to tunnel
- Sets up routing policies

### 3. BillionMail Integration
- Adds domain to `domain` table
- Adds transport mapping
- Adds to `bm_multi_ip_domain` (shows Dedicated IP in UI)

### 4. Cloudflare DNS (Auto)
- **A record:** mail.DOMAIN → HOST_IP
- **SPF record:** v=spf1 ip4:NOEZ_IP ~all
- **DMARC record:** _dmarc.DOMAIN

### 5. Container Networking
- Adds Noez IP to container loopback
- Configures container routing

### 6. Host Routing
- iptables packet marking
- Policy routing
- SNAT for source IP rewriting

### 7. Postfix Configuration
- Creates transport in master.cf
- Maps domain to transport
- Restarts Postfix

## 🆘 Troubleshooting

### "SPF fail - not authorized"

The SPF record is missing your Noez IP. The script now **automatically fixes this** when you add a domain with `CF_API_TOKEN` set!

If still failing, check Cloudflare DNS:
```bash
dig +short TXT yourdomain.com | grep spf
```

Should include: `ip4:YOUR_NOEZ_IP`

### "Container not found"
```bash
cd /opt/billionmail && docker compose up -d
```

### "Domain not showing in UI"
- Refresh browser: Ctrl+Shift+R
- Run: `sudo bash noez_setup.sh status`

### Check logs:
```bash
tail -f /opt/billionmail/logs/postfix/mail.log
```

## 🔍 Verification

After running:
```bash
sudo bash noez_setup.sh status
```

Should show:
- ✅ GRE tunnel exists
- ✅ Container running
- ✅ Noez IP in container
- ✅ Domain in BillionMail
- ✅ Auto-start service enabled

## 💾 Auto-Start on Boot

The script **automatically enables** auto-start. To verify:

```bash
systemctl is-enabled noez-ips.service
# Output: enabled
```

If you need to manually re-add IPs after container restart:
```bash
sudo bash /opt/billionmail/setup_noez_ips.sh
```

## 📝 Files

| File | Purpose |
|------|---------|
| `noez_setup.sh` | **MAIN SCRIPT** - Run this |
| `NOEZ_SETUP.md` | This documentation |
| `setup_noez_ips.sh` | Auto-created helper for systemd |

---

**Version:** 5.0  
**Status:** ✅ PRODUCTION READY - Cloudflare DNS auto-integration working

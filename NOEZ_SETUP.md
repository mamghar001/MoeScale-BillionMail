# Noez GRE Tunnel Setup for BillionMail

> **ONE SCRIPT - ONE COMMAND - DOES EVERYTHING**

## Quick Start

```bash
sudo bash /opt/billionmail/noez_setup.sh
```

That's it! This single command will:
- Create auto-start service (boot persistence)
- Create GRE tunnel
- Add domain to BillionMail
- Configure container networking
- Setup iptables rules
- Configure Postfix
- Test connectivity

## Files

| File | Purpose |
|------|---------|
| `noez_setup.sh` | **MAIN SCRIPT** - Run this |
| `NOEZ_SETUP.md` | This documentation |

The `setup_noez_ips.sh` is auto-created by the main script for systemd service.

## Configuration

Edit `noez_setup.sh` and set:

```bash
NOEZ_IP="5.230.168.0"           # Your Noez IP
HOST_IP="YOUR_VPS_IP"           # Your VPS public IP
NOEZ_GRE_REMOTE="5.230.205.35"  # From Noez panel
DOMAIN="yourdomain.com"         # Domain to send from
ALL_NOEZ_IPS="5.230.168.0 5.230.168.1"  # All IPs for auto-setup
```

Or run interactively and the script will prompt you.

## Commands

```bash
# Complete setup
sudo bash noez_setup.sh

# Test email
sudo bash noez_setup.sh test

# Add another IP/domain
sudo bash noez_setup.sh add 5.230.168.1 newdomain.com

# Check status
sudo bash noez_setup.sh status
```

## After Container Restart

IPs are automatically added on boot via systemd service.

If you need to manually re-add:
```bash
sudo bash /opt/billionmail/setup_noez_ips.sh
```

## Requirements

- VPS with public IP
- Noez GRE tunnel service
- BillionMail installed at `/opt/billionmail`
- Root access

## Troubleshooting

```bash
# Check status
sudo bash noez_setup.sh status

# View logs
tail -f /opt/billionmail/logs/postfix/mail.log

# Check auto-start service
systemctl status noez-ips.service
```

## 🔧 Fixing SPF Issues

If you get "SPF fail - not authorized" errors:

### Option 1: Automatic (with Cloudflare API)

1. Edit the script and add your API token:
```bash
nano noez_setup.sh
# Change:
CF_API_TOKEN="your-cloudflare-api-token-here"
```

2. Get token from: https://dash.cloudflare.com/profile/api-tokens
   - Create token with: Zone:Read, DNS:Edit permissions

3. Re-run for the domain:
```bash
sudo bash noez_setup.sh add 5.230.168.X domain.com
```

### Option 2: Manual (Cloudflare Dashboard)

1. Go to https://dash.cloudflare.com
2. Select your domain
3. Go to DNS
4. Find TXT record with SPF
5. Update to include your Noez IP:
```
v=spf1 +a +mx +ip4:YOUR_VPS_IP +ip4:NOEZ_IP ~all
```

Example for 5.230.168.10:
```
v=spf1 +a +mx +ip4:66.55.64.133 +ip4:5.230.168.8 +ip4:5.230.168.10 ~all
```

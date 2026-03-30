# Noez GRE Tunnel - Complete Setup for BillionMail

> **ONE SCRIPT DOES IT ALL** - Complete solution for sending emails from Noez IP via GRE tunnel

## 🚀 ONE COMMAND SETUP

```bash
sudo bash /opt/billionmail/noez_setup.sh
```

That's it! This single command will:
1. ✅ Prompt for configuration (or use pre-configured values)
2. ✅ Create and enable auto-start systemd service
3. ✅ Create GRE tunnel (if needed)
4. ✅ Add domain to BillionMail
5. ✅ Configure container networking
6. ✅ Setup iptables rules
7. ✅ Configure Postfix
8. ✅ Test connectivity
9. ✅ Auto-configure IPs on every boot

---

## 📋 Prerequisites

- VPS with public IP and root access
- Noez account with GRE tunnel service
- BillionMail installed (`/opt/billionmail`)

---

## 🔧 Configuration

Edit `/opt/billionmail/noez_setup.sh` and set these values:

```bash
# REQUIRED: Your Noez IP (the IP you want to send emails from)
NOEZ_IP="5.230.168.0"

# REQUIRED: Your VPS public IP
HOST_IP="YOUR_VPS_IP_HERE"

# REQUIRED: Noez GRE tunnel remote endpoint (from Noez)
NOEZ_GRE_REMOTE="5.230.205.35"

# REQUIRED: Domain to send from
DOMAIN="yourdomain.com"

# ALL Noez IPs for auto-setup on boot (space-separated)
ALL_NOEZ_IPS="5.230.168.0 5.230.168.1 5.230.168.2"
```

---

## 📊 Commands Reference

| Command | Description |
|---------|-------------|
| `sudo bash noez_setup.sh` | **COMPLETE SETUP** - does everything including auto-start |
| `sudo bash noez_setup.sh test` | Send test email |
| `sudo bash noez_setup.sh add IP DOMAIN` | Add new IP/domain |
| `sudo bash noez_setup.sh status` | Check status of everything |
| `sudo bash noez_setup.sh help` | Show help |

---

## 🔄 Adding More IPs/Domains

```bash
# Add another IP and domain
sudo bash noez_setup.sh add 5.230.168.1 anotherdomain.com

# The script will:
# - Add domain to BillionMail
# - Add IP to container
# - Configure routing
# - Update Postfix
# - Add to auto-start service
```

Then update `ALL_NOEZ_IPS` in the script:
```bash
ALL_NOEZ_IPS="5.230.168.0 5.230.168.1"
```

And run once more to update the auto-start service:
```bash
sudo bash noez_setup.sh status
```

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         YOUR VPS (HOST_IP)                          │
│                                                                     │
│  ┌──────────────────┐         ┌─────────────────────────────────┐   │
│  │   Docker Bridge  │         │         GRE Tunnel (gre1)       │   │
│  │  172.66.2.0/24   │◄────────│  Internal: 192.168.31.2/30      │   │
│  │                  │  route  │  Noez IP: 5.230.168.0/32        │   │
│  │  Gateway: .1     │         │  Peer: 5.230.205.35             │   │
│  └────────┬─────────┘         └─────────────────────────────────┘   │
│           │                                                         │
│  ┌────────▼──────────────────┐                                      │
│  │   Postfix Container       │  • Sends emails from NOEZ_IP        │
│  │   172.66.2.100            │  • Uses SNAT for IP rewriting       │
│  │   NOEZ_IP/32 (loopback)   │  • Routes via GRE tunnel            │
│  └───────────────────────────┘                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

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
- Adds domain to `domain` table (shows in UI)
- Adds transport mapping
- Adds to `bm_multi_ip_domain` (shows Dedicated IP in UI)

### 4. Container Networking
- Adds Noez IP to container loopback
- Configures container routing

### 5. Host Routing
- iptables packet marking (mangle)
- Policy routing (fwmark)
- SNAT for source IP rewriting
- Forwarding rules

### 6. Postfix Configuration
- Creates transport in master.cf
- Maps domain to transport
- Restarts Postfix

---

## 🆘 Troubleshooting

### "Container not found"
```bash
cd /opt/billionmail && docker compose up -d
```

### "Domain not showing in UI"
- Refresh browser: Ctrl+Shift+R
- Run: `sudo bash noez_setup.sh status`

### "Emails not sending"
```bash
# Check logs
tail -f /opt/billionmail/logs/postfix/mail.log

# Check status
sudo bash noez_setup.sh status
```

---

## 🔍 Verification

After running the script:

```bash
# Check everything
sudo bash noez_setup.sh status
```

Should show:
- ✅ GRE tunnel exists
- ✅ Container running
- ✅ Noez IP in container
- ✅ Domain in BillionMail
- ✅ Auto-start service enabled

---

## 💾 Auto-Start on Boot

The script **automatically enables** auto-start. To verify:

```bash
# Check if enabled
systemctl is-enabled noez-ips.service

# Check status
systemctl status noez-ips.service
```

If you ever need to manually run the IP setup:
```bash
# After container restart
sudo bash /opt/billionmail/setup_noez_ips.sh
```

---

## 📞 Support

**Debug mode:**
```bash
sudo bash -x /opt/billionmail/noez_setup.sh
```

**Check logs:**
```bash
tail -f /opt/billionmail/logs/postfix/mail.log
```

---

**Version:** 4.0  
**Status:** ✅ ONE SCRIPT DOES IT ALL

# Noez GRE Tunnel - Complete Setup for BillionMail

> **One script, one guide** - Complete solution for sending emails from Noez IP via GRE tunnel

## 📋 Quick Start

```bash
# 1. Configure your values
nano /opt/billionmail/noez_setup.sh

# 2. Run setup
sudo bash /opt/billionmail/noez_setup.sh

# 3. Test
sudo bash /opt/billionmail/noez_setup.sh test
```

---

## 📁 Files

| File | Purpose |
|------|---------|
| `noez_setup.sh` | **Main script** - handles everything |
| `NOEZ_COMPLETE_SETUP.md` | **This guide** |

---

## 🔧 Configuration

Edit `noez_setup.sh` and set these values:

```bash
# REQUIRED: Your Noez IP (the IP you want to send emails from)
NOEZ_IP="5.230.168.0"

# REQUIRED: Your VPS public IP
HOST_IP="YOUR_VPS_IP_HERE"

# REQUIRED: Noez GRE tunnel remote endpoint (from Noez)
NOEZ_GRE_REMOTE="5.230.205.35"

# REQUIRED: Domain to send from
DOMAIN="yourdomain.com"

# GRE internal IPs (usually provided by Noez)
GRE_LOCAL="192.168.31.2"     # Your side
GRE_REMOTE="192.168.31.1"    # Noez side
```

---

## 🚀 Usage

### Fresh Install / New VPS

```bash
cd /opt/billionmail
sudo bash noez_setup.sh
```

This will:
1. ✅ Create GRE tunnel (if not exists)
2. ✅ Configure host routing
3. ✅ Setup container networking
4. ✅ Configure iptables (SNAT + marking)
5. ✅ Setup Postfix transport
6. ✅ Add domain mapping
7. ✅ Test connectivity

### Test Email Sending

```bash
sudo bash noez_setup.sh test
```

### Add More IPs/Domains

```bash
# Add another IP and domain
sudo bash noez_setup.sh add 5.230.168.1 anotherdomain.com

# Add another
sudo bash noez_setup.sh add 5.230.168.2 moredomains.com
```

### Check Status

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

**How it works:**
1. Container sends email from 172.66.2.100
2. Host iptables marks packets with mark 100
3. Policy routing sends marked packets through GRE tunnel (table 20)
4. SNAT changes source IP from 172.66.2.100 to 5.230.168.0
5. Email arrives from 5.230.168.0 ✓

---

## 📋 Step-by-Step Guide

### Step 1: Order Noez GRE Tunnel

1. Log into [Noez](https://noez.de)
2. Order **GRE tunnel** service
3. Provide your VPS IP
4. Note these values:
   - **Noez IP**: Your send IP (e.g., `5.230.168.0`)
   - **GRE endpoint**: Noez's tunnel IP (e.g., `5.230.205.35`)
   - **GRE subnet**: Internal network (e.g., `192.168.31.0/30`)

### Step 2: Install BillionMail

Follow BillionMail installation guide. Ensure:
- Docker is installed
- BillionMail is running
- Postfix container is up

### Step 3: Configure Script

```bash
nano /opt/billionmail/noez_setup.sh
```

Set these values in the CONFIGURATION section:
```bash
NOEZ_IP="5.230.168.0"
HOST_IP="YOUR_VPS_IP"
NOEZ_GRE_REMOTE="5.230.205.35"
DOMAIN="yourdomain.com"
```

### Step 4: Run Setup

```bash
sudo bash /opt/billionmail/noez_setup.sh
```

If GRE tunnel doesn't exist, the script will offer to create it.

### Step 5: Verify

```bash
# Check status
sudo bash noez_setup.sh status

# Send test email
sudo bash noez_setup.sh test
```

---

## 🔍 Troubleshooting

### "Container not found"

Start BillionMail first:
```bash
cd /opt/billionmail && docker compose up -d
```

### "GRE tunnel not found"

The script will offer to create it. Or manually:
```bash
ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255
ip addr add $GRE_LOCAL/30 dev gre1
ip addr add $NOEZ_IP/32 dev gre1
ip link set gre1 up
ip rule add from $NOEZ_IP table 20 prio 200
ip route add default via $GRE_REMOTE dev gre1 table 20
```

### "Cannot reach internet"

Check routing:
```bash
ip rule                          # Should show fwmark and from rules
ip route show table 20           # Should show default via GRE
tail -f logs/postfix/mail.log    # Check Postfix logs
```

### Check iptables:
```bash
iptables -t mangle -L PREROUTING -n -v   # Should show MARK rule
iptables -t nat -L POSTROUTING -n -v     # Should show SNAT rule
```

### Emails not from Noez IP

Verify domain mapping:
```bash
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT * FROM bm_domain_smtp_transport;"
```

Check Postfix transport:
```bash
grep "smtp_bind_ip" /opt/billionmail/conf/postfix/master.cf
```

---

## 📊 Commands Reference

| Command | Description |
|---------|-------------|
| `sudo bash noez_setup.sh` | Complete setup |
| `sudo bash noez_setup.sh test` | Send test email |
| `sudo bash noez_setup.sh add IP DOMAIN` | Add IP/domain |
| `sudo bash noez_setup.sh status` | Show status |
| `sudo bash noez_setup.sh help` | Show help |

---

## 🔄 Adding More IPs/Domains

### Same GRE Tunnel, Additional IPs

If Noez gave you multiple IPs in the same subnet:

```bash
sudo bash noez_setup.sh add 5.230.168.1 domain1.com
sudo bash noez_setup.sh add 5.230.168.2 domain2.com
sudo bash noez_setup.sh add 5.230.168.3 domain3.com
```

### One IP, Multiple Domains

```bash
# Same IP, different domains
sudo bash noez_setup.sh add 5.230.168.0 domain1.com
sudo bash noez_setup.sh add 5.230.168.0 domain2.com
sudo bash noez_setup.sh add 5.230.168.0 domain3.com
```

### New GRE Tunnel (Different Location)

For a completely new GRE tunnel:

1. Set up new GRE tunnel manually (different interface name like `gre2`)
2. Update script with new values
3. Run setup

---

## 💾 Persistence

### Option 1: Systemd Service (Recommended)

```bash
sudo tee /etc/systemd/system/noez-gre.service > /dev/null << 'EOF'
[Unit]
Description=Noez GRE Tunnel Setup
After=docker.service

[Service]
Type=oneshot
ExecStart=/opt/billionmail/noez_setup.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable noez-gre.service
```

### Option 2: Cron @reboot

```bash
sudo crontab -e
```

Add:
```
@reboot sleep 30 && /opt/billionmail/noez_setup.sh >> /var/log/noez-setup.log 2>&1
```

---

## 🆘 Common Issues

| Issue | Solution |
|-------|----------|
| GRE module not found | `modprobe ip_gre` |
| IP already exists | Normal - script continues |
| Domain already exists | Script updates the mapping |
| Container can't ping | Check iptables FORWARD chain |
| Wrong source IP | Check SNAT rule in POSTROUTING |

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

**Verify setup:**
```bash
sudo bash /opt/billionmail/noez_setup.sh status
```

---

## ✅ Verification Checklist

- [ ] Noez GRE tunnel ordered and active
- [ ] Script configured with correct IPs
- [ ] GRE tunnel created (or script created it)
- [ ] Container has Noez IP on loopback
- [ ] iptables SNAT rule exists
- [ ] Policy routing configured (fwmark 100)
- [ ] Postfix transport added
- [ ] Domain mapping in database
- [ ] Test email sent successfully
- [ ] Source IP verified (mail-tester.com)

---

**Version:** 2.0  
**Working:** ✅ Yes - Successfully tested sending from 5.230.168.0, 5.230.168.1, 5.230.168.4 via GRE tunnel

---

## 🚀 Interactive Mode

The script now **prompts you for values** if they're not set!

### If Values Are Not Set:

```bash
$ sudo bash noez_setup.sh

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Noez GRE Tunnel Setup for BillionMail
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Configuration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[!] NOEZ_IP not configured
Enter your Noez IP (e.g., 5.230.168.0): 5.230.168.0

[!] HOST_IP not configured
Enter your VPS IP [detected: 85.121.241.162]: 

[!] NOEZ_GRE_REMOTE not configured
Enter Noez GRE remote endpoint (from Noez panel): 5.230.205.35

[!] DOMAIN not configured
Enter domain to send from: yourdomain.com

Configuration:
  Noez IP: 5.230.168.0
  Host IP: 85.121.241.162
  GRE Endpoint: 5.230.205.35
  Domain: yourdomain.com

Is this correct? (y/n) y

[✓] Configuration confirmed!
```

### Two Ways to Configure:

**Option 1: Interactive (Easiest)**
Just run the script and answer the prompts:
```bash
sudo bash noez_setup.sh
```

**Option 2: Edit Script (For Automation)**
Edit the CONFIGURATION section at the top:
```bash
nano noez_setup.sh
# Set these values:
NOEZ_IP="5.230.168.0"
HOST_IP="85.121.241.162"
NOEZ_GRE_REMOTE="5.230.205.35"
DOMAIN="yourdomain.com"
```

### Auto-Detection

The script will try to **auto-detect your VPS IP**:
```
Enter your VPS IP [detected: 85.121.241.162]:
```

Just press Enter to use the detected IP, or type a different one.


---

## 🔌 API Support (Optional)

The script can use the BillionMail API instead of direct database access:

### Why Use API?
- More "official" method
- Triggers any API hooks/webhooks
- Better for multi-server setups

### Configure API:

Edit the script and set:
```bash
BM_API_URL="https://mail.yourdomain.com"
BM_API_TOKEN="your-api-token"
```

Get your API token from BillionMail:
1. Go to **System Settings** → **API Tokens**
2. Generate new token
3. Copy the token

### API vs Database:

| Method | Speed | Requires Token | Shows in UI Immediately |
|--------|-------|----------------|------------------------|
| **API** | Slower | Yes | Yes |
| **Database** | Faster | No | Yes (may need refresh) |

**Default:** Database method (works out of the box)

---

## 🎯 What the Script Does

For each domain/IP, the script:

1. **Creates domain in BillionMail** (`domain` table)
   - So it shows in the UI
   - Sets proper quotas and limits

2. **Adds transport mapping** (`bm_domain_smtp_transport` table)
   - Links domain to Noez IP

3. **Configures Postfix** (`master.cf`)
   - Creates transport for the IP

4. **Sets up networking**
   - Adds IP to container
   - Configures iptables SNAT
   - Sets up policy routing

5. **Tests connectivity**
   - Verifies internet access
   - Tests DNS resolution

---

## ✅ Verification

After running the script, verify:

```bash
# Check domain is in BillionMail
sudo bash noez_setup.sh status

# Or manually:
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT domain, active FROM domain WHERE domain='yourdomain.com';"
```

Should show:
```
    domain     | active 
---------------+--------
yourdomain.com|      1
```

**If domain not visible in UI:**
- Refresh the BillionMail page
- Check browser cache (Ctrl+Shift+R)
- Verify domain is in database using command above


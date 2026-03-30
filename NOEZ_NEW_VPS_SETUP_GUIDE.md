# Noez GRE Tunnel Setup Guide for New VPS

This guide explains how to set up the Noez GRE tunnel email routing on a **new VPS** with **new Noez IPs**.

## Prerequisites

1. A VPS with a public IP address
2. Noez account with GRE tunnel service
3. BillionMail installed and running
4. Root access to the VPS

## Step 1: Order Noez GRE Tunnel

1. Log into your Noez account
2. Order a GRE tunnel service
3. Note down the following information:
   - Your Noez IP (e.g., `5.230.168.0`)
   - Noez GRE remote endpoint (e.g., `5.230.205.35`)
   - GRE tunnel internal subnet (e.g., `192.168.31.0/30`)

## Step 2: Initial GRE Tunnel Setup (Host)

On your **host VPS**, run these commands to create the GRE tunnel:

```bash
# Set variables (replace with your actual values)
HOST_IP="YOUR_VPS_IP"           # Your VPS public IP
NOEZ_IP="YOUR_NOEZ_IP"          # The Noez IP you want to use
NOEZ_GRE_REMOTE="NOEZ_ENDPOINT" # The Noez GRE endpoint IP
GRE_LOCAL="192.168.31.2"        # Your side of GRE tunnel
GRE_REMOTE="192.168.31.1"       # Noez side of GRE tunnel

# Create GRE tunnel
ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255

# Add internal GRE IP
ip addr add $GRE_LOCAL/30 dev gre1

# Add your Noez IP to the tunnel
ip addr add $NOEZ_IP/32 dev gre1

# Bring tunnel up
ip link set gre1 up

# Add routing policy for Noez IP
ip rule add from $NOEZ_IP table 20 prio 200
ip route add default via $GRE_REMOTE dev gre1 table 20
```

### Test the GRE Tunnel

```bash
# Test ping from host via Noez IP
ping -c 3 -I $NOEZ_IP 8.8.8.8
```

If this works, your GRE tunnel is set up correctly!

## Step 3: Configure BillionMail Docker Network

### Option A: Using the Setup Script (Recommended)

1. **Edit the configuration section** of `setup_noez_email_routing.sh`:

```bash
nano /opt/billionmail/setup_noez_email_routing.sh
```

2. **Modify these values** at the top of the script:

```bash
# Your Noez IP (the IP you want to send emails from)
NOEZ_IP="5.230.168.0"  # <-- CHANGE THIS

# Your host's main IP
HOST_IP="85.121.241.162"  # <-- CHANGE THIS

# Noez GRE tunnel remote endpoint
NOEZ_GRE_REMOTE="5.230.205.35"  # <-- CHANGE THIS

# GRE tunnel internal IPs
GRE_LOCAL="192.168.31.2"
GRE_REMOTE="192.168.31.1"

# Domain to use for Noez IP sending
DOMAIN="yourdomain.com"  # <-- CHANGE THIS
```

3. **Find your container's Docker network IP**:

```bash
docker inspect billionmail-postfix-billionmail-1 | grep -A 20 "Networks"
```

Look for the IP in the `billionmail-net-*` network, e.g., `172.66.2.100`

4. **Update the container network settings** in the script:

```bash
CONTAINER_NET_NS_IP="172.66.2.100"  # <-- CHANGE THIS to match your setup
DOCKER_BRIDGE_GW="172.66.2.1"       # Usually .1 of the subnet
```

5. **Run the script**:

```bash
cd /opt/billionmail
sudo bash setup_noez_email_routing.sh
```

### Option B: Manual Setup

If you prefer manual configuration, follow these steps:

#### 3.1 Add Noez IP to Container

```bash
# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Add Noez IP to container's loopback
nsenter -t $CONTAINER_PID -n ip addr add YOUR_NOEZ_IP/32 dev lo

# Add container routing rule
nsenter -t $CONTAINER_PID -n ip rule add from YOUR_NOEZ_IP table 100
nsenter -t $CONTAINER_PID -n ip route add default via 172.66.2.1 table 100
```

#### 3.2 Configure Host Policy Routing

```bash
# Mark packets from container
iptables -t mangle -A PREROUTING -s 172.66.2.100 -j MARK --set-mark 100

# Route marked packets through GRE tunnel
ip rule add fwmark 100 table 20 prio 50

# SNAT to change source IP
iptables -t nat -I POSTROUTING 1 -s 172.66.2.100 -j SNAT --to-source YOUR_NOEZ_IP

# Allow forwarding
iptables -I FORWARD -s YOUR_NOEZ_IP -j ACCEPT
iptables -I FORWARD -d YOUR_NOEZ_IP -j ACCEPT

# Add Noez IP routing rule
ip rule add from YOUR_NOEZ_IP table 20 prio 200

# Flush route cache
ip route flush cache
```

## Step 4: Configure Postfix

### 4.1 Add Transport to master.cf

Edit `/opt/billionmail/conf/postfix/master.cf` and add:

```
smtp_bind_ip_YOUR_NOEZ_IP unix  -       -       n       -       -       smtp
```

Replace dots with underscores in the transport name, e.g., `smtp_bind_ip_5_230_168_0`

### 4.2 Add Domain Mapping in Database

```bash
cd /opt/billionmail
source .env

docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"INSERT INTO bm_domain_smtp_transport (atype, domain, smtp_name) 
VALUES ('dedicated_ip', '@yourdomain.com', 'smtp_bind_ip_YOUR_NOEZ_IP');"
```

Replace `yourdomain.com` and `smtp_bind_ip_YOUR_NOEZ_IP` with your actual values.

### 4.3 Reload Postfix

```bash
docker exec billionmail-postfix-billionmail-1 postfix reload
```

## Step 5: Test the Setup

### Test Connectivity

```bash
# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Test ping from container
nsenter -t $CONTAINER_PID -n ping -c 3 8.8.8.8
```

### Test Email Sending

```bash
# Send test email
docker exec -i billionmail-postfix-billionmail-1 sh -c 'cat << "EOF" | sendmail -v test@mail-tester.com
From: admin@yourdomain.com
To: test@mail-tester.com
Subject: Test from Noez IP

Test message
EOF'

# Check logs
tail -f /opt/billionmail/logs/postfix/mail.log
```

### Verify Source IP

Send an email to `test@mail-tester.com` or check the received headers to confirm the email is coming from your Noez IP.

## Step 6: Make It Persistent

### Option A: Systemd Service (Recommended)

Create a systemd service file:

```bash
sudo tee /etc/systemd/system/noez-gre-setup.service > /dev/null << 'EOF'
[Unit]
Description=Noez GRE Tunnel Email Routing Setup
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/opt/billionmail/setup_noez_email_routing.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable noez-gre-setup.service
sudo systemctl start noez-gre-setup.service
```

### Option B: Cron Job

Add to root's crontab:

```bash
sudo crontab -e
```

Add:
```
@reboot sleep 30 && /opt/billionmail/setup_noez_email_routing.sh >> /var/log/noez-setup.log 2>&1
```

### Option C: Docker Compose Post-Start Hook

Add to your `docker-compose.yml` under the postfix service:

```yaml
postfix-billionmail:
  # ... existing config ...
  command: >
    sh -c "/opt/billionmail/setup_noez_email_routing.sh && /usr/bin/supervisord -c /etc/supervisor/supervisord.conf"
```

## Troubleshooting

### Issue: Container cannot reach internet

**Check:**
```bash
# 1. Is GRE tunnel up?
ip link show gre1

# 2. Are routes correct?
ip route show table 20
ip rule

# 3. Are iptables rules set?
iptables -t mangle -L PREROUTING -n -v | grep 172.66
iptables -t nat -L POSTROUTING -n -v | grep SNAT

# 4. Is the container IP correct?
docker inspect billionmail-postfix-billionmail-1 | grep -A 5 "billionmail-net"
```

### Issue: Emails not sending from Noez IP

**Check:**
```bash
# 1. Is the domain mapped correctly?
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT * FROM bm_domain_smtp_transport;"

# 2. Is the transport defined in master.cf?
grep "smtp_bind_ip" /opt/billionmail/conf/postfix/master.cf

# 3. Check Postfix logs
tail -50 /opt/billionmail/logs/postfix/mail.log | grep -E "(status=|error|warning)"
```

### Issue: GRE tunnel not working

**Check:**
```bash
# Can you ping the Noez gateway?
ping 192.168.31.1

# Is the tunnel interface up?
ip addr show gre1

# Check kernel modules
lsmod | grep gre
```

## Common Mistakes

1. **Wrong Noez IP**: Make sure you're using the IP assigned to you by Noez
2. **Wrong GRE endpoint**: Double-check the remote endpoint IP from Noez
3. **Wrong container IP**: The container IP must match the Docker network subnet
4. **iptables not saved**: iptables rules are lost on reboot - use persistent methods
5. **GRE tunnel not persistent**: The GRE tunnel must be recreated after reboot

## Variables Summary

| Variable | Description | Example |
|----------|-------------|---------|
| `HOST_IP` | Your VPS public IP | `85.121.241.162` |
| `NOEZ_IP` | Your Noez IP for sending | `5.230.168.0` |
| `NOEZ_GRE_REMOTE` | Noez GRE endpoint | `5.230.205.35` |
| `GRE_LOCAL` | Your GRE tunnel IP | `192.168.31.2` |
| `GRE_REMOTE` | Noez GRE tunnel IP | `192.168.31.1` |
| `CONTAINER_NET_NS_IP` | Container's Docker IP | `172.66.2.100` |
| `DOCKER_BRIDGE_GW` | Docker bridge gateway | `172.66.2.1` |

## Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review the logs: `/opt/billionmail/logs/postfix/mail.log`
3. Verify your Noez GRE tunnel is working independently
4. Check the original documentation: `NOEZ_GRE_TUNNEL_SETUP.md`

# Adding More Noez IPs and Domains

This guide explains how to add additional Noez IPs and domains to your BillionMail setup.

## Quick Decision Guide

| Scenario | Recommended Method | Difficulty |
|----------|-------------------|------------|
| Same GRE tunnel, additional IP in same subnet | Use script (Option 1) | Easy |
| New GRE tunnel (different location) | Use script + manual GRE setup | Medium |
| Just adding a new domain to existing IP | BillionMail UI or script | Very Easy |
| Multiple new IPs at once | Use script | Easy |

---

## Option 1: Using Scripts (Recommended)

### For Additional IPs in Same GRE Tunnel

If Noez assigned you **additional IPs in the same subnet** (e.g., you have 5.230.168.0 and want to add 5.230.168.1):

```bash
# Add a new IP and domain
sudo bash /opt/billionmail/add_noez_ip.sh 5.230.168.1 newdomain.com
```

This script will:
1. ✅ Add the IP to the container
2. ✅ Add iptables SNAT rules
3. ✅ Add Postfix transport
4. ✅ Add domain mapping in database
5. ✅ Reload Postfix

### For a Completely New GRE Tunnel

If you ordered a **new GRE tunnel** in a different location:

1. **Set up the new GRE tunnel** (similar to initial setup):
```bash
# Example: New GRE tunnel to different Noez endpoint
ip tunnel add gre2 mode gre local YOUR_VPS_IP remote NEW_NOEZ_ENDPOINT ttl 255
ip addr add NEW_GRE_LOCAL/30 dev gre2
ip addr add NEW_NOEZ_IP/32 dev gre2
ip link set gre2 up
ip rule add from NEW_NOEZ_IP table 30 prio 300
ip route add default via NEW_GRE_REMOTE dev gre2 table 30
```

2. **Use the main setup script** with new values:
```bash
# Edit the script with new IP values
nano /opt/billionmail/setup_noez_email_routing.sh
# Then run
sudo bash /opt/billionmail/setup_noez_email_routing.sh
```

---

## Option 2: Using BillionMail Web Interface

### What Works in the UI

✅ **Adding a new domain to an EXISTING IP**:
1. Go to BillionMail Admin Panel
2. Navigate to "Domain Management"
3. Add your new domain
4. Select the existing Noez IP from the dropdown

⚠️ **Limitations of the UI**:
- ❌ Cannot add new Noez IPs (only select from existing)
- ❌ Cannot configure GRE tunnels
- ❌ May create incorrect transport names (we've seen this issue)

### When to Use the UI

Use the BillionMail UI **only** when:
- You're adding a new domain
- You want to use an **already configured** Noez IP
- The IP is already working for another domain

### How to Use UI for New Domain

1. **Log into BillionMail Admin** (usually https://your-vip/admin)
2. **Go to "Domain Management"**
3. **Click "Add Domain"**
4. **Enter your domain name**
5. **Select "Dedicated IP"**
6. **Choose your Noez IP from the dropdown**
7. **Save**

8. **Verify it worked**:
```bash
# Check database
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT * FROM bm_domain_smtp_transport WHERE domain='@yourdomain.com';"
```

---

## Option 3: Manual Method (Full Control)

If you prefer to do it manually or the script doesn't work:

### Step 1: Add IP to Container

```bash
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Add new Noez IP to container loopback
nsenter -t $CONTAINER_PID -n ip addr add NEW_NOEZ_IP/32 dev lo

# Add container routing
nsenter -t $CONTAINER_PID -n ip rule add from NEW_NOEZ_IP table 100
```

### Step 2: Add Host iptables Rules

```bash
# SNAT rule
iptables -t nat -I POSTROUTING 1 -s 172.66.2.100 -j SNAT --to-source NEW_NOEZ_IP

# Forwarding rules
iptables -I FORWARD -s NEW_NOEZ_IP -j ACCEPT
iptables -I FORWARD -d NEW_NOEZ_IP -j ACCEPT

# Routing rule
ip rule add from NEW_NOEZ_IP table 20 prio 200
ip route flush cache
```

### Step 3: Add Postfix Transport

Edit `/opt/billionmail/conf/postfix/master.cf`:

```
smtp_bind_ip_NEW_NOEZ_IP unix  -       -       n       -       -       smtp
```

(Replace dots with underscores in the transport name)

### Step 4: Add Domain to Database

```bash
cd /opt/billionmail
source .env

# Replace NEW_NOEZ_IP_FORMAT with IP using underscores
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"INSERT INTO bm_domain_smtp_transport (atype, domain, smtp_name) 
VALUES ('dedicated_ip', '@newdomain.com', 'smtp_bind_ip_NEW_NOEZ_IP_FORMAT');"
```

### Step 5: Reload Postfix

```bash
docker exec billionmail-postfix-billionmail-1 postfix reload
```

---

## Verification

After adding a new IP/domain, always verify:

### 1. Check Container Has the IP
```bash
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)
nsenter -t $CONTAINER_PID -n ip addr show lo | grep YOUR_NOEZ_IP
```

### 2. Test Connectivity
```bash
nsenter -t $CONTAINER_PID -n ping -c 3 8.8.8.8
```

### 3. Check Database Mapping
```bash
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT * FROM bm_domain_smtp_transport;"
```

### 4. Test Email Sending
```bash
# Get a test address from mail-tester.com
docker exec -i billionmail-postfix-billionmail-1 sh -c 'cat << "EOF" | sendmail -v test-xxx@srv1.mail-tester.com
From: admin@yourdomain.com
To: test-xxx@srv1.mail-tester.com
Subject: Test New IP

Test message
EOF'
```

### 5. Check Source IP
Go to mail-tester.com and verify the email came from your new Noez IP.

---

## Common Scenarios

### Scenario 1: Same Subnet, Multiple IPs

If Noez gave you: `5.230.168.0`, `5.230.168.1`, `5.230.168.2`

**Solution**: Use the `add_noez_ip.sh` script for each:

```bash
sudo bash add_noez_ip.sh 5.230.168.0 domain1.com
sudo bash add_noez_ip.sh 5.230.168.1 domain2.com
sudo bash add_noez_ip.sh 5.230.168.2 domain3.com
```

### Scenario 2: Multiple GRE Tunnels (Different Locations)

If you have GRE tunnels to different Noez locations:

**Solution**: 
1. Set up each GRE tunnel with different interface names (`gre1`, `gre2`, etc.)
2. Create separate routing tables (table 20, table 30, etc.)
3. Run the setup script for each configuration

```bash
# Tunnel 1 (Germany)
ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GERMANY ttl 255
ip addr add 192.168.31.2/30 dev gre1
ip addr add 5.230.168.0/32 dev gre1
ip link set gre1 up
ip rule add from 5.230.168.0 table 20 prio 200
ip route add default via 192.168.31.1 dev gre1 table 20

# Tunnel 2 (France)
ip tunnel add gre2 mode gre local $HOST_IP remote $NOEZ_FRANCE ttl 255
ip addr add 192.168.32.2/30 dev gre2
ip addr add 5.231.100.0/32 dev gre2
ip link set gre2 up
ip rule add from 5.231.100.0 table 30 prio 300
ip route add default via 192.168.32.1 dev gre2 table 30
```

### Scenario 3: One IP, Multiple Domains

If you want to send from multiple domains using the same IP:

**Solution**: Just add the domains in BillionMail UI:

1. Go to Domain Management
2. Add each domain
3. Select the same Noez IP for all

Or use the script:
```bash
sudo bash add_noez_ip.sh 5.230.168.0 domain1.com
sudo bash add_noez_ip.sh 5.230.168.0 domain2.com
sudo bash add_noez_ip.sh 5.230.168.0 domain3.com
```

---

## Troubleshooting

### Issue: "IP already exists" error

**Cause**: The IP was already added to the container

**Solution**: This is fine, the script will continue

### Issue: Domain not sending from correct IP

**Check**:
```bash
# Verify transport mapping
docker exec -i billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c \
"SELECT domain, smtp_name FROM bm_domain_smtp_transport WHERE domain='@yourdomain.com';"

# Verify transport exists in master.cf
grep "smtp_bind_ip" /opt/billionmail/conf/postfix/master.cf
```

### Issue: Emails bouncing

**Check Postfix logs**:
```bash
tail -50 /opt/billionmail/logs/postfix/mail.log | grep -E "(status=bounced|error|warning)"
```

---

## Summary

| Task | Method | Command/Location |
|------|--------|------------------|
| Add new domain to existing IP | BillionMail UI | Domain Management → Add Domain |
| Add new domain to existing IP | Script | `bash add_noez_ip.sh IP domain` |
| Add new IP (same subnet) | Script | `bash add_noez_ip.sh IP domain` |
| Add new IP (new GRE tunnel) | Manual | Follow "New GRE Tunnel" section |
| Multiple domains, one IP | UI or Script | Add multiple times with same IP |

---

## Recommendation

**For most users**: Use the `add_noez_ip.sh` script - it's the fastest and most reliable method.

**When to ask me for help**:
- Setting up a completely new VPS from scratch
- Complex multi-tunnel configurations
- Troubleshooting when things don't work
- Creating custom automation scripts

**When to use BillionMail UI**:
- Just adding a domain to an already-working IP
- You prefer GUI over command line
- Quick testing

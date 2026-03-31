# BillionMail Noez GRE Tunnel - AI Agent Skill

> **Quick Reference for AI Agents Working on This Project**
> 
> This document contains all knowledge gained from debugging and setting up the Noez GRE tunnel integration with BillionMail. Read this first before making changes.

---

## 📚 Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Quick Start for Fresh Install](#quick-start-for-fresh-install)
3. [Common Issues & Solutions](#common-issues--solutions)
4. [Debugging Guide](#debugging-guide)
5. [File Reference](#file-reference)
6. [Testing Checklist](#testing-checklist)

---

## Architecture Overview

### System Design

```
┌─────────────────────────────────────────────────────────────────┐
│                          VPS HOST                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    GRE Tunnel (gre1)                     │    │
│  │  Noez IPs: 5.230.168.0, .1, .2, .10, .11, .12         │    │
│  │  Route: GRE → Noez Network → Internet                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            │                                     │
│                   Docker Bridge (br-*)                           │
│                            │                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │         BillionMail Postfix Container                    │    │
│  │  ┌─────────────────────────────────────────────────┐   │    │
│  │  │ Loopback (lo):                                  │   │    │
│  │  │   5.230.168.0/32  ← smtp_bind_ip_5_230_168_0   │   │    │
│  │  │   5.230.168.1/32  ← smtp_bind_ip_5_230_168_1   │   │    │
│  │  │   ...                                           │   │    │
│  │  └─────────────────────────────────────────────────┘   │    │
│  │  Container eth0: 172.66.2.100 (Docker internal)        │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Purpose | Critical Config |
|-----------|---------|-----------------|
| **GRE Tunnel** | Connects to Noez network | IPs bound to gre1 interface |
| **Postfix Transports** | Routes emails per IP | `smtp_bind_address` option |
| **Container Loopback** | Binds Noez IPs inside container | IP aliases on lo |
| **Host Routing** | Forwards replies to container | Routes via bridge, not local |
| **iptables** | NAT/SNAT for source IP | Source-based SNAT rules |

---

## Quick Start for Fresh Install

### 1. Prerequisites Check

```bash
# Verify BillionMail is running
docker ps | grep billionmail-postfix

# Verify Noez GRE tunnel endpoint is reachable
ping 5.230.205.35  # Replace with actual Noez endpoint

# Check GRE module is available
modprobe ip_gre
```

### 2. Configuration Setup

```bash
cd /opt/billionmail
cp noez_setup.env.example noez_setup.env
nano noez_setup.env
```

**Required fields:**
```bash
NOEZ_IP="5.230.168.0"           # The Noez IP for sending
HOST_IP="YOUR_VPS_PUBLIC_IP"    # VPS public IP
NOEZ_GRE_REMOTE="5.230.205.35"  # From Noez panel
DOMAIN="yourdomain.com"         # Sending domain
ALL_NOEZ_IPS="5.230.168.0 ..."  # All your Noez IPs
```

### 3. Run Setup

```bash
sudo bash noez_setup.sh
```

**Expected output:**
- GRE tunnel created
- Container networking configured
- Domain added to BillionMail
- DNS records created (if CF_API_TOKEN set)
- Systemd service enabled

---

## Common Issues & Solutions

### Issue 1: Email Sent from Wrong IP

**Symptom:** Mail-tester shows emails coming from different IP than configured.

**Root Cause:** Postfix transport missing `smtp_bind_address` option.

**Check:**
```bash
grep -A1 "smtp_bind_ip_5_230_168" conf/postfix/master.cf
```

**Expected:**
```
smtp_bind_ip_5_230_168_10 unix - - n - - smtp
  -o smtp_bind_address=5.230.168.10
```

**Fix:**
```bash
# Add the bind address line if missing
sed -i '/^smtp_bind_ip_5_230_168_10 unix/a\  -o smtp_bind_address=5.230.168.10' conf/postfix/master.cf
docker restart billionmail-postfix-billionmail-1
```

---

### Issue 2: No Internet Connectivity from Container

**Symptom:** Container cannot ping 8.8.8.8 from Noez IP.

**Root Cause:** Host has local route for Noez IP, intercepts replies.

**Check:**
```bash
ip route show table local | grep 5.230.168
ip route | grep 5.230.168
```

**Problem:**
```
local 5.230.168.10 dev gre1 scope host  # ← WRONG! Host intercepts
```

**Solution:**
```bash
# Delete local route
ip route del table local 5.230.168.10 dev gre1

# Add route via bridge to container
ip route add 5.230.168.10 dev br-825cda742c28
```

**Verify:**
```bash
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)
nsenter -t $CONTAINER_PID -n ping -c 2 -I 5.230.168.10 8.8.8.8
```

---

### Issue 3: Docker Bridge Not Detected

**Symptom:** Script fails to find Docker bridge interface.

**Check:**
```bash
# List Docker networks
ip route | grep "172.66"
docker network ls

# Find bridge for specific network
ip route | grep "172.66.2.0/24"
```

**Manual Fix:**
```bash
export DOCKER_BRIDGE="br-825cda742c28"
export DOCKER_BRIDGE_GW="172.66.2.1"
```

---

### Issue 4: Postfix Transport Not Loading

**Symptom:** New transport added to master.cf but not used.

**Root Cause:** Postfix needs full restart (not reload) for new transports.

**Fix:**
```bash
docker restart billionmail-postfix-billionmail-1
sleep 3

# Re-add IPs after restart
sudo bash /opt/billionmail/setup_noez_ips.sh
```

---

### Issue 5: Container Loses IPs After Reboot

**Symptom:** After VPS reboot, emails fail to send.

**Root Cause:** Container IPs not persisted (they're in container namespace).

**Solution:** Systemd service should auto-run. Check:
```bash
systemctl status noez-ips.service

# If not running:
sudo systemctl start noez-ips.service
# or manually:
sudo bash /opt/billionmail/setup_noez_ips.sh
```

---

## Debugging Guide

### Step 1: Verify GRE Tunnel

```bash
# Check tunnel exists
ip link show gre1

# Check IPs on tunnel
ip addr show gre1 | grep "inet "

# Test from host
ping -c 2 -I 5.230.168.10 8.8.8.8
```

### Step 2: Verify Container IPs

```bash
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Check IPs in container
nsenter -t $CONTAINER_PID -n ip addr show lo | grep "inet "

# Test ping from container
nsenter -t $CONTAINER_PID -n ping -c 2 -I 5.230.168.10 8.8.8.8
```

### Step 3: Verify Host Routing

```bash
# Should show route via bridge, not local
ip route | grep 5.230.168.10

# Should NOT show in local table
ip route show table local | grep 5.230.168.10
```

### Step 4: Verify Postfix Transport

```bash
# Check transport exists
grep "smtp_bind_ip_5_230_168" conf/postfix/master.cf

# Check transport has bind address
grep -A1 "smtp_bind_ip_5_230_168_10" conf/postfix/master.cf

# Check database mapping
docker exec -i billionmail-postfix-billionmail-1 psql -U billionmail -d billionmail -c \
  "SELECT * FROM bm_domain_smtp_transport WHERE domain='@yourdomain.com';"
```

### Step 5: Check Mail Logs

```bash
tail -50 /opt/billionmail/logs/postfix/mail.log | grep -E "(status=|relay=|error)"
```

### Step 6: Test SMTP Connection

```bash
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Test SMTP to mail-tester
nsenter -t $CONTAINER_PID -n bash -c 'echo "QUIT" | nc -s 5.230.168.10 reception.mail-tester.com 25'
```

---

## File Reference

### User-Created Files

| File | Created By | Purpose |
|------|-----------|---------|
| `noez_setup.env` | You | Configuration (IPs, tokens, domains) |

### Main Script

| File | Run By | Purpose |
|------|--------|---------|
| `noez_setup.sh` | You (sudo) | Main setup script - handles everything |

### Auto-Generated Files

| File | Created By | Purpose |
|------|-----------|---------|
| `setup_noez_ips.sh` | `noez_setup.sh` | Boot helper to re-add IPs |
| `/etc/systemd/system/noez-ips.service` | `noez_setup.sh` | Systemd service for auto-start |

### Postfix Config

| File | Modified By | Purpose |
|------|-------------|---------|
| `conf/postfix/master.cf` | `noez_setup.sh` | Postfix transports with IP binding |

---

## Testing Checklist

### After Fresh Install

- [ ] GRE tunnel created: `ip link show gre1`
- [ ] Noez IP on gre1: `ip addr show gre1`
- [ ] Container running: `docker ps | grep postfix`
- [ ] IP in container: `nsenter -t $PID -n ip addr show lo`
- [ ] Host routes to bridge: `ip route | grep 5.230.168`
- [ ] Domain in BillionMail: Check UI
- [ ] Postfix transport: `grep smtp_bind_ip conf/postfix/master.cf`
- [ ] Transport has bind address: `grep -A1 smtp_bind_ip conf/postfix/master.cf`
- [ ] DNS records created: `dig TXT yourdomain.com | grep spf`
- [ ] Test email sends: Send to mail-tester.com
- [ ] Correct IP shown: Verify in mail-tester results

### After Reboot

- [ ] Systemd service active: `systemctl status noez-ips.service`
- [ ] IPs in container: `nsenter -t $PID -n ip addr show lo`
- [ ] Host routes correct: `ip route | grep 5.230.168`
- [ ] Test email works

---

## Commands Quick Reference

### Essential Commands

```bash
# Full setup
sudo bash noez_setup.sh

# Add new IP/domain
sudo bash noez_setup.sh add 5.230.168.X domain.com

# Check status
sudo bash noez_setup.sh status

# Fix after reboot
sudo bash setup_noez_ips.sh

# Restart Postfix
docker restart billionmail-postfix-billionmail-1

# View logs
tail -f logs/postfix/mail.log
```

### Debug Commands

```bash
# Container shell
docker exec -it billionmail-postfix-billionmail-1 bash

# Container namespace commands
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)
nsenter -t $CONTAINER_PID -n ip addr
nsenter -t $CONTAINER_PID -n ip route
nsenter -t $CONTAINER_PID -n ping 8.8.8.8

# Check iptables
iptables -t nat -L POSTROUTING -n -v
iptables -L FORWARD -n -v | head -10

# Check routes
ip rule
ip route show table 20
ip route show table local
```

---

## Important Notes for AI Agents

### Critical Fix: Host Routing

**ALWAYS** ensure host routes Noez IPs to container, not locally:

```bash
# WRONG - causes reply packets to be dropped
local 5.230.168.10 dev gre1 scope host

# CORRECT - forwards to container
5.230.168.10 dev br-825cda742c28 scope link
```

### Critical Fix: Postfix Transport

**ALWAYS** include `smtp_bind_address`:

```bash
# WRONG - uses default IP
smtp_bind_ip_5_230_168_10 unix - - n - - smtp

# CORRECT - binds to specific IP
smtp_bind_ip_5_230_168_10 unix - - n - - smtp
  -o smtp_bind_address=5.230.168.10
```

### Container Namespace

Container IPs are added to **loopback (lo)** interface inside container namespace, not to host. Use `nsenter` to access:

```bash
nsenter -t $CONTAINER_PID -n <command>
```

### Multiple Domains per IP

Perfectly valid to have multiple domains using same IP:

```bash
# Both domains use same transport
@domain1.com → smtp_bind_ip_5_230_168_10
@domain2.com → smtp_bind_ip_5_230_168_10
```

Both send from `5.230.168.10`, each has own DNS records.

---

## Version History

- **v6.1** (Current): Production-ready, robust auto-detection, simplified iptables
- **v5.0**: Initial working version with Cloudflare integration

---

## Support Resources

- **Main Script:** `noez_setup.sh`
- **Config Template:** `noez_setup.env.example`
- **Documentation:** `NOEZ_SETUP.md`
- **This Skill File:** `SKILLS.md`

---

*Last Updated: 2026-03-31*  
*Author: AI Agent Documentation*  
*Purpose: Preserve knowledge for future AI agents*

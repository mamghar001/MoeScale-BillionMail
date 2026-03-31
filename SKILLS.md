# BillionMail Noez GRE Tunnel - Complete AI Agent Knowledge Base

> **⚠️ CRITICAL: READ THIS ENTIRE FILE BEFORE MODIFYING ANYTHING**
> 
> This document contains ALL knowledge gained from 20+ hours of debugging the Noez GRE tunnel integration. Missing any section will likely cause you to repeat our mistakes.

---

## 📋 TABLE OF CONTENTS

1. [Project Overview](#1-project-overview)
2. [Complete Architecture](#2-complete-architecture)
3. [All Project Files](#3-all-project-files)
4. [Configuration Deep Dive](#4-configuration-deep-dive)
5. [Complete Setup Process](#5-complete-setup-process)
6. [ALL Bugs & Solutions (Chronological)](#6-all-bugs--solutions-chronological)
7. [Troubleshooting Decision Trees](#7-troubleshooting-decision-trees)
8. [Emergency Recovery Procedures](#8-emergency-recovery-procedures)
9. [Testing & Verification](#9-testing--verification)
10. [Integration Points](#10-integration-points)
11. [Command Reference](#11-command-reference)
12. [Common Pitfalls](#12-common-pitfalls)

---

## 1. PROJECT OVERVIEW

### What This Project Does

Integrates **Noez GRE tunnel service** with **BillionMail** to allow sending emails from dedicated Noez IPs through a GRE tunnel.

### Why GRE Tunnel?

- Noez provides clean IP addresses for email sending
- GRE tunnel routes traffic through Noez network
- Container binds to Noez IPs for source IP control
- Postfix uses specific IPs per domain

### Components Involved

1. **Host VPS** - Runs Docker, GRE tunnel, iptables rules
2. **GRE Tunnel** - Connects to Noez (gre1 interface)
3. **Docker Container** - BillionMail Postfix
4. **Noez Network** - Provides clean IP reputation
5. **Cloudflare** - DNS management (optional)

---

## 2. COMPLETE ARCHITECTURE

### Network Flow (Detailed)

```
OUTBOUND EMAIL FLOW:
====================

1. User sends email via BillionMail UI
   ↓
2. BillionMail core → Postfix container
   (via internal Docker network 172.66.1.x)
   ↓
3. Postfix determines transport based on sender domain
   - Looks up domain in bm_domain_smtp_transport table
   - Finds: @domain.com → smtp_bind_ip_5_230_168_10
   ↓
4. Postfix uses transport with smtp_bind_address
   - Binds to 5.230.168.10 on container lo interface
   ↓
5. Packet leaves container via eth0 (172.66.2.100)
   - Container routing: from 5.230.168.10 → table 100
   - Table 100 default via 172.66.2.1 (Docker bridge)
   ↓
6. Host receives packet on Docker bridge (br-825cda742c28)
   - Host sees source IP: 5.230.168.10
   - iptables SNAT not needed (source is already correct)
   ↓
7. Host routing: from 5.230.168.10 → table 20
   - Table 20: default via 192.168.31.1 dev gre1
   ↓
8. Packet enters GRE tunnel
   - Encapsulated in GRE header
   - Sent to Noez endpoint (5.230.205.35)
   ↓
9. Noez network routes to destination
   - Destination sees source: 5.230.168.10
   ↓
10. Email delivered ✓


REPLY/INBOUND FLOW:
===================

1. Remote server replies to 5.230.168.10
   ↓
2. Noez network receives reply
   ↓
3. GRE tunnel delivers to host
   - Host sees packet for 5.230.168.10
   ↓
4. CRITICAL: Host routing decision
   
   WRONG (causes failure):
   - ip route show table local → local 5.230.168.10 dev gre1
   - Host thinks IP is LOCAL, processes packet itself
   - Packet never reaches container!
   
   CORRECT (what we need):
   - ip route show → 5.230.168.10 dev br-825cda742c28
   - Host forwards packet to Docker bridge
   ↓
5. Docker bridge delivers to container
   - Container receives reply on eth0
   - Routing delivers to application
   ↓
6. Postfix receives reply ✓
```

### Component Relationships

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              HOST VPS                                    │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     GRE TUNNEL (gre1)                            │   │
│  │  ┌─────────────────────────────────────────────────────────┐   │   │
│  │  │ IPs bound to interface:                                  │   │   │
│  │  │   • 192.168.31.2/30 (tunnel local)                      │   │   │
│  │  │   • 5.230.168.0/32 (Noez IP 1)                          │   │   │
│  │  │   • 5.230.168.1/32 (Noez IP 2)                          │   │   │
│  │  │   • 5.230.168.2/32 (Noez IP 3)                          │   │   │
│  │  │   • 5.230.168.10/32 (Noez IP 4)                         │   │   │
│  │  │   • 5.230.168.11/32 (Noez IP 5)                         │   │   │
│  │  │   • 5.230.168.12/32 (Noez IP 6)                         │   │   │
│  │  └─────────────────────────────────────────────────────────┘   │   │
│  │  Tunnel endpoint: 5.230.205.35                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                              │                                           │
│                              │ GRE tunnel interface                      │
│                              │                                           │
│  ┌───────────────────────────┼───────────────────────────────────────┐ │
│  │                           │    HOST NETWORKING                     │ │
│  │  ┌──────────────────────┴─────────────────────────────────────┐  │ │
│  │  │ Routing Tables:                                              │  │ │
│  │  │   • Main table: default via VPS gateway                     │  │ │
│  │  │   • Table 20: default via 192.168.31.1 (for Noez IPs)       │  │ │
│  │  │   • ip rule: from 5.230.168.x lookup 20                     │  │ │
│  │  └─────────────────────────────────────────────────────────────┘  │ │
│  │                                                                     │ │
│  │  ┌─────────────────────────────────────────────────────────────┐   │ │
│  │  │ iptables Rules:                                              │   │ │
│  │  │   • FORWARD: ACCEPT for 5.230.168.x (both directions)       │   │ │
│  │  │   • NAT: Not needed for source (already correct)            │   │ │
│  │  └─────────────────────────────────────────────────────────────┘   │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                              │                                           │
│                              │ Docker bridge                             │
│                              │ (br-825cda742c28)                        │
│                              │                                           │
│  ┌───────────────────────────┴───────────────────────────────────────┐ │
│  │                         DOCKER CONTAINER                           │ │
│  │              (billionmail-postfix-billionmail-1)                   │ │
│  │                                                                    │ │
│  │  ┌──────────────────────────────────────────────────────────────┐ │ │
│  │  │ Container Interfaces:                                         │ │ │
│  │  │   • eth0: 172.66.2.100/24 (Docker network)                   │ │ │
│  │  │   • eth1: 172.66.1.100/24 (BillionMail internal)             │ │ │
│  │  │   • lo: 127.0.0.1 + Noez IPs as aliases                      │ │ │
│  │  │          - 5.230.168.0/32                                     │ │ │
│  │  │          - 5.230.168.1/32                                     │ │ │
│  │  │          - 5.230.168.2/32                                     │ │ │
│  │  │          - 5.230.168.10/32                                    │ │ │
│  │  │          - 5.230.168.11/32                                    │ │ │
│  │  │          - 5.230.168.12/32                                    │ │ │
│  │  └──────────────────────────────────────────────────────────────┘ │ │
│  │                                                                     │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ Container Routing:                                            │  │ │
│  │  │   • ip rule: from 5.230.168.x lookup 100                     │  │ │
│  │  │   • table 100: default via 172.66.2.1                        │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │                                                                     │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ Postfix Configuration:                                        │  │ │
│  │  │   master.cf:                                                 │  │ │
│  │  │     smtp_bind_ip_5_230_168_0 unix - - n - - smtp            │  │ │
│  │  │       -o smtp_bind_address=5.230.168.0                       │  │ │
│  │  │     smtp_bind_ip_5_230_168_1 unix - - n - - smtp            │  │ │
│  │  │       -o smtp_bind_address=5.230.168.1                       │  │ │
│  │  │     ...                                                      │  │ │
│  │  │                                                              │  │ │
│  │  │   Database:                                                  │  │ │
│  │  │     bm_domain_smtp_transport:                                │  │ │
│  │  │       @domain1.com → smtp_bind_ip_5_230_168_0               │  │ │
│  │  │       @domain2.com → smtp_bind_ip_5_230_168_1               │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. ALL PROJECT FILES

### Core Noez Files (User Focused)

| File | Purpose | Who Edits | Who Runs |
|------|---------|-----------|----------|
| `noez_setup.sh` | **MAIN SCRIPT** - Does everything | Never | Admin (sudo) |
| `noez_setup.env` | **PRIVATE CONFIG** - IPs, tokens | Admin | Sourced by main script |
| `noez_setup.env.example` | Template for noez_setup.env | AI dev | Reference only |
| `NOEZ_SETUP.md` | User documentation | AI dev | Reference only |
| `setup_noez_ips.sh` | Boot helper (auto-generated) | Never | systemd (auto) |
| `SKILLS.md` | This file - AI knowledge base | AI dev | AI agents must read |

### BillionMail Core Files (Integration Points)

| File | Purpose | Modified By |
|------|---------|-------------|
| `conf/postfix/master.cf` | Postfix transports | noez_setup.sh |
| `conf/postfix/main.cf` | Main Postfix config | BillionMail install |
| `docker-compose.yml` | Container orchestration | BillionMail install |
| `.env` | DB credentials | BillionMail install |
| `data/postgresql/` | Database files | Runtime |
| `logs/postfix/mail.log` | Email logs | Runtime |

### Auto-Generated/Runtime Files

| File | Purpose | Created By |
|------|---------|------------|
| `/etc/systemd/system/noez-ips.service` | Boot service | noez_setup.sh |
| `postfix-data/pid/unix.smtp_bind_ip_*` | Transport sockets | Postfix runtime |
| `core-data/session/billion-mail/` | Telegram sessions | BillionMail runtime |

---

## 4. CONFIGURATION DEEP DIVE

### noez_setup.env - All Options Explained

```bash
# =============================================================================
# REQUIRED FIELDS (Must be filled)
# =============================================================================

# NOEZ_IP
# The specific Noez IP you want to use for this domain
# Format: x.x.x.x
# Get from: Noez control panel
# Example: NOEZ_IP="5.230.168.0"
NOEZ_IP=""

# HOST_IP  
# Your VPS public IP address (the one Noez creates GRE tunnel to)
# Format: x.x.x.x
# Get from: curl ifconfig.me or VPS dashboard
# Example: HOST_IP="85.121.241.162"
HOST_IP=""

# NOEZ_GRE_REMOTE
# Noez GRE tunnel endpoint IP
# Format: x.x.x.x
# Get from: Noez control panel (GRE tunnel section)
# Example: NOEZ_GRE_REMOTE="5.230.205.35"
NOEZ_GRE_REMOTE=""

# DOMAIN
# The domain you'll send emails from
# Format: domain.com
# Must: Have DNS managed by you (Cloudflare recommended)
# Example: DOMAIN="mydomain.com"
DOMAIN=""

# ALL_NOEZ_IPS
# ALL Noez IPs you own (space-separated)
# Used by: systemd service for auto-setup on boot
# Format: "ip1 ip2 ip3 ..."
# Example: ALL_NOEZ_IPS="5.230.168.0 5.230.168.1 5.230.168.2"
ALL_NOEZ_IPS=""

# =============================================================================
# OPTIONAL FIELDS (But highly recommended)
# =============================================================================

# CF_API_TOKEN
# Cloudflare API token for automatic DNS
# Permissions needed: Zone:Read, DNS:Edit
# Get from: https://dash.cloudflare.com/profile/api-tokens
# If empty: Manual DNS setup required
CF_API_TOKEN=""

# CF_ZONE_ID
# Cloudflare Zone ID (auto-detected if empty)
# Get from: Cloudflare domain overview page (bottom right)
CF_ZONE_ID=""

# =============================================================================
# ADVANCED FIELDS (Usually auto-detected)
# =============================================================================

# CONTAINER_NAME
# Docker container name for Postfix
# Default: billionmail-postfix-billionmail-1
# Only change if: Custom Docker setup
CONTAINER_NAME="billionmail-postfix-billionmail-1"

# DOCKER_SUBNET
# Docker network subnet
# Default: 172.66.2.0/24
# Auto-detected from: ip route | grep 172.66
DOCKER_SUBNET="172.66.2.0/24"

# GRE_LOCAL
# Local IP for GRE tunnel (your side)
# Default: 192.168.31.2
# Only change if: Noez provides different
GRE_LOCAL="192.168.31.2"

# GRE_REMOTE
# Remote IP for GRE tunnel (Noez side)
# Default: 192.168.31.1
# Only change if: Noez provides different
GRE_REMOTE="192.168.31.1"
```

### Configuration Validation Rules

```bash
# All REQUIRED fields must be non-empty
# IP addresses must be valid IPv4
# DOMAIN must not include protocol (no https://)
# ALL_NOEZ_IPS must include NOEZ_IP
```

---

## 5. COMPLETE SETUP PROCESS

### Phase 1: Validation (noez_setup.sh)

```bash
1. Check running as root
2. Load .env file
3. Validate NOEZ_IP, HOST_IP, NOEZ_GRE_REMOTE, DOMAIN
4. If interactive: prompt for missing values
5. If non-interactive: fail with error
6. Show configuration summary
7. Get user confirmation
```

### Phase 2: Docker Bridge Detection

```bash
1. Get container network info from Docker
2. Try to find bridge interface:
   a. Look for network with "b2bscale" or "net-" in name
   b. Extract IP from container networks
   c. Find bridge from routing table
3. Fallback: Try common bridge names (br-*, br-billionmail, docker0)
4. Set DOCKER_BRIDGE, DOCKER_BRIDGE_GW, CONTAINER_NET_NS_IP
5. Fail if cannot detect
```

### Phase 3: GRE Tunnel Setup

```bash
1. Check if gre1 exists
2. If exists and has our IP: skip
3. If exists but different: ask to recreate
4. Create tunnel:
   ip tunnel add gre1 mode gre local $HOST_IP remote $NOEZ_GRE_REMOTE ttl 255
5. Add addresses:
   ip addr add $GRE_LOCAL/30 dev gre1
   ip addr add $NOEZ_IP/32 dev gre1
6. Bring up:
   ip link set gre1 up
7. Add routing policy:
   ip rule add from $NOEZ_IP table 20 prio 200
   ip route add default via $GRE_REMOTE dev gre1 table 20
8. Test: ping -I $NOEZ_IP 8.8.8.8
```

### Phase 4: Cloudflare DNS Setup

```bash
1. Skip if CF_API_TOKEN not set
2. Find Zone ID (if not provided)
3. Delete old SPF records (cleanup duplicates)
4. Delete old DMARC records (cleanup duplicates)
5. Create/Update A record: mail.DOMAIN → HOST_IP
6. Create SPF: v=spf1 ip4:NOEZ_IP ~all
7. Create DMARC: v=DMARC1; p=quarantine; rua=mailto:dmarc@DOMAIN
8. Report results
```

### Phase 5: BillionMail Domain Setup

```bash
1. Check if domain exists in 'domain' table
2. If not: INSERT INTO domain (...)
3. Add/update bm_multi_ip_domain entry
4. Show domain in BillionMail UI
```

### Phase 6: Container Networking

```bash
1. Get container PID
2. Add IP to container loopback:
   nsenter -t $PID -n ip addr add $NOEZ_IP/32 dev lo
3. Add container routing:
   nsenter -t $PID -n ip rule add from $NOEZ_IP table 100
   nsenter -t $PID -n ip route add default via $DOCKER_BRIDGE_GW table 100
```

### Phase 7: Host Routing

```bash
1. Add forwarding rules:
   iptables -I FORWARD -s $NOEZ_IP -j ACCEPT
   iptables -I FORWARD -d $NOEZ_IP -j ACCEPT

2. CRITICAL: Fix local route issue
   # Delete local route (prevents host from intercepting replies)
   ip route del table local $NOEZ_IP dev gre1
   
   # Add route via bridge to container
   ip route add $NOEZ_IP dev $DOCKER_BRIDGE

3. Flush route cache
```

### Phase 8: Postfix Configuration

```bash
1. Check if transport exists in master.cf
2. If exists without smtp_bind_address:
   - Add: -o smtp_bind_address=$NOEZ_IP
3. If not exists:
   - Append transport with smtp_bind_address
4. Update database:
   INSERT/UPDATE bm_domain_smtp_transport
5. Restart Postfix container (not reload - need restart for new transports)
6. Wait 3 seconds
```

### Phase 9: Systemd Service Setup

```bash
1. Create /opt/billionmail/setup_noez_ips.sh
2. Create /etc/systemd/system/noez-ips.service
3. systemctl daemon-reload
4. systemctl enable noez-ips.service
5. systemctl start noez-ips.service
```

### Phase 10: Testing

```bash
1. Test container internet connectivity:
   nsenter -t $PID -n ping -c 3 8.8.8.8

2. Test DNS resolution:
   nsenter -t $PID -n nslookup google.com

3. Report results
```

---

## 6. ALL BUGS & SOLUTIONS (CHRONOLOGICAL)

### Bug #1: Wrong Source IP (CRITICAL)

**Date Discovered:** 2026-03-30
**Severity:** CRITICAL - Emails sent from wrong IP

**Symptom:**
```
Mail-tester shows:
"Your IP address 5.230.168.12 is associated with the domain 11.0-255.168.230.5.in-addr.arpa.
Nevertheless your message appears to be sent from mail.b2bscale.xyz"
```

**Root Cause:**
Postfix transport created WITHOUT `smtp_bind_address` option. Postfix uses default IP.

**How to Detect:**
```bash
grep "smtp_bind_ip" conf/postfix/master.cf
```

**Problem Output:**
```
smtp_bind_ip_5_230_168_10 unix - - n - - smtp
# No -o smtp_bind_address line!
```

**Solution:**
```bash
# Add the bind address to transport
sed -i '/^smtp_bind_ip_5_230_168_10 unix/a\  -o smtp_bind_address=5.230.168.10' conf/postfix/master.cf

# Verify
grep -A1 "smtp_bind_ip_5_230_168_10" conf/postfix/master.cf
# Should show:
# smtp_bind_ip_5_230_168_10 unix - - n - - smtp
#   -o smtp_bind_address=5.230.168.10

# Restart Postfix
docker restart billionmail-postfix-billionmail-1
```

**Code Fix Applied:**
Modified `setup_postfix()` to always include `-o smtp_bind_address=$NOEZ_IP`

---

### Bug #2: No Reply Packets (CRITICAL)

**Date Discovered:** 2026-03-30
**Severity:** CRITICAL - No bidirectional connectivity

**Symptom:**
```
Container can send ping but gets no replies:
--- 8.8.8.8 ping statistics ---
2 packets transmitted, 0 received, 100% packet loss
```

**Root Cause:**
When adding IP to GRE tunnel, kernel creates LOCAL route:
```
local 5.230.168.10 dev gre1 scope host
```

Host thinks IP belongs to itself, intercepts reply packets, drops them (no process listening).

**How to Detect:**
```bash
ip route show table local | grep 5.230.168.10
# Shows: local 5.230.168.10 dev gre1 scope host ← BAD!

# Should be:
ip route | grep 5.230.168.10
# Shows: 5.230.168.10 dev br-825cda742c28 scope link ← GOOD!
```

**Solution:**
```bash
# Delete local route
ip route del table local 5.230.168.10 dev gre1

# Add route to container via bridge
ip route add 5.230.168.10 dev br-825cda742c28

# Verify
ip route get 5.230.168.10
# Should show: 5.230.168.10 dev br-825cda742c28 src 172.66.2.1
```

**Code Fix Applied:**
Added to `setup_host_routing()`:
```bash
ip route del table local $NOEZ_IP dev gre1 2>/dev/null || true
ip route add $NOEZ_IP dev $DOCKER_BRIDGE 2>/dev/null || true
```

---

### Bug #3: Docker Bridge Detection Failure

**Date Discovered:** 2026-03-30
**Severity:** HIGH - Script fails on fresh install

**Symptom:**
```
Could not detect Docker bridge!
Please set DOCKER_BRIDGE manually in noez_setup.env
```

**Root Cause:**
Container has TWO networks:
- 172.66.2.100 (b2bscale network - for Noez)
- 172.66.1.100 (billionmail network - internal)

Script only looked for specific names, didn't handle multiple networks.

**How to Detect:**
```bash
# Check container networks
docker inspect billionmail-postfix-billionmail-1 --format='{{range $k,$v := .NetworkSettings.Networks}}{{$k}}:{{$v.IPAddress}} {{end}}'
# Output: billionmail_billionmail-net-b2bscale:172.66.2.100 billionmail_billionmail-network:172.66.1.100
```

**Solution:**
Rewrote `detect_docker_bridge()` to:
1. Look for network with "b2bscale" or "net-" in name
2. Extract IP from that network
3. Find bridge from routing table
4. Fallback to trying common bridge names

---

### Bug #4: IP Not Added to GRE Tunnel on 'add' Command

**Date Discovered:** 2026-03-30
**Severity:** MEDIUM - Adding new IP/domain fails

**Symptom:**
```
Running: noez_setup.sh add 5.230.168.12 newdomain.com
Email sends from wrong IP or fails
```

**Root Cause:**
`add_new_ip()` function skipped adding IP to GRE tunnel - only added to container.

**Solution:**
Modified `add_new_ip()` to call GRE tunnel setup first.

---

### Bug #5: Systemd Service Missing Host Routing

**Date Discovered:** 2026-03-30
**Severity:** MEDIUM - Reboot breaks connectivity

**Symptom:**
After VPS reboot, emails fail until manually running setup script.

**Root Cause:**
`setup_systemd_service()` created script that only added IPs to container, didn't fix host routes.

**Solution:**
Rewrote generated `setup_noez_ips.sh` to:
1. Detect bridge automatically
2. Delete local routes
3. Add routes via bridge
4. Re-add container IPs

---

### Bug #6: iptables Mangle Rules Conflicting

**Date Discovered:** 2026-03-30
**Severity:** MEDIUM - Packet marking conflicts

**Symptom:**
Complex iptables rules not working, packets bypassing SNAT.

**Root Cause:**
Tried to use packet marking (mark 0x64) + SNAT by mark + Docker MASQUERADE rules.
Too complex, rules in wrong order.

**Solution:**
Simplified to source-based SNAT only:
```bash
# Deleted complex mangle rules
# Deleted mark-based SNAT
# Kept simple source-based forwarding
iptables -I FORWARD -s $NOEZ_IP -j ACCEPT
iptables -I FORWARD -d $NOEZ_IP -j ACCEPT
```

---

## 7. TROUBLESHOOTING DECISION TREES

### Email Not Sending

```
Email not sending?
│
├─ Check: docker ps | grep postfix
│  └─ Container not running?
│     └─ cd /opt/billionmail && docker compose up -d
│
├─ Check: tail -f logs/postfix/mail.log
│  ├─ "Connection timed out"?
│  │  └─ Network issue → Check connectivity tree
│  ├─ "SPF fail"?
│  │  └─ Wrong source IP → Check Bug #1 fix
│  └─ "relay=none"?
│     └─ DNS or routing issue
│
└─ Check: ping -I 5.230.168.X 8.8.8.8
   └─ Host can't ping?
      └─ Check GRE tunnel tree
```

### Wrong Source IP (Bug #1)

```
Email sent from wrong IP?
│
├─ Check: grep -A1 "smtp_bind_ip_5_230_168_X" conf/postfix/master.cf
│  ├─ Shows "-o smtp_bind_address=X.X.X.X"?
│  │  └─ Transport OK, check database
│  │     └─ SELECT * FROM bm_domain_smtp_transport WHERE domain='@yourdomain'
│  └─ Missing smtp_bind_address?
│     └─ Fix: sed -i '/^transport_name/a\  -o smtp_bind_address=IP' conf/postfix/master.cf
│         docker restart billionmail-postfix-billionmail-1
```

### No Reply Packets (Bug #2)

```
Container can't get replies?
│
├─ Check: ip route show table local | grep 5.230.168
│  ├─ Shows "local 5.230.168.X dev gre1"?
│  │  └─ BAD! Host intercepting packets
│  │     └─ Fix: ip route del table local 5.230.168.X dev gre1
│  │              ip route add 5.230.168.X dev br-XXX
│  └─ No local routes?
│     └─ Check: ip route | grep 5.230.168
│        ├─ Shows "dev br-"?
│        │  └─ Routing OK, check iptables
│        └─ No route?
│           └─ Fix: ip route add 5.230.168.X dev br-XXX
```

### GRE Tunnel Issues

```
GRE tunnel problems?
│
├─ Check: ip link show gre1
│  └─ "Device does not exist"?
│     └─ Create: ip tunnel add gre1 mode gre local HOST_IP remote NOEZ_GRE_REMOTE ttl 255
│
├─ Check: ip addr show gre1 | grep "inet "
│  └─ Missing Noez IP?
│     └─ Add: ip addr add NOEZ_IP/32 dev gre1
│
├─ Check: ping -I NOEZ_IP 8.8.8.8
│  └─ "Network unreachable"?
│     └─ Add route: ip route add default via GRE_REMOTE dev gre1 table 20
│         Add rule: ip rule add from NOEZ_IP table 20
│
└─ Check: ip rule | grep "from NOEZ_IP"
   └─ No rule?
      └─ Add: ip rule add from NOEZ_IP table 20
```

---

## 8. EMERGENCY RECOVERY PROCEDURES

### Complete Reset (Nuclear Option)

**Use when:** Everything broken, start fresh

```bash
#!/bin/bash
# EMERGENCY RESET - Run as root

cd /opt/billionmail

# 1. Stop services
systemctl stop noez-ips.service 2>/dev/null
docker stop billionmail-postfix-billionmail-1

# 2. Remove GRE tunnel
ip link del gre1 2>/dev/null || true

# 3. Clean up routes
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    ip route del $IP 2>/dev/null || true
    ip route del table local $IP 2>/dev/null || true
    ip rule del from $IP 2>/dev/null || true
done
ip route flush cache

# 4. Clean up iptables (carefully!)
iptables -L FORWARD -n --line-numbers | grep "5.230.168" | awk '{print $1}' | sort -rn | while read num; do
    iptables -D FORWARD $num 2>/dev/null
done

# 5. Remove Postfix transports from master.cf
# Backup first
cp conf/postfix/master.cf conf/postfix/master.cf.backup.$(date +%Y%m%d)
# Remove Noez transport lines
sed -i '/^# Noez IP/d' conf/postfix/master.cf
sed -i '/^smtp_bind_ip_5_230_/d' conf/postfix/master.cf
sed -i '/^-o smtp_bind_address=5\.230\./d' conf/postfix/master.cf

# 6. Restart container
docker start billionmail-postfix-billionmail-1

# 7. Re-run setup
sudo bash noez_setup.sh
```

### Quick Fix After Reboot

```bash
# If emails fail after VPS reboot:
sudo bash /opt/billionmail/setup_noez_ips.sh

# Or restart the systemd service:
sudo systemctl restart noez-ips.service
```

### Fix Single Domain

```bash
# If one domain not working:
sudo bash noez_setup.sh add 5.230.168.X domain.com
```

---

## 9. TESTING & VERIFICATION

### Complete Test Suite

```bash
#!/bin/bash
# Run all tests

ERRORS=0

echo "=== TEST 1: GRE Tunnel ==="
if ip link show gre1 &>/dev/null; then
    echo "✓ GRE tunnel exists"
else
    echo "✗ GRE tunnel missing"
    ERRORS=$((ERRORS+1))
fi

echo "=== TEST 2: GRE Tunnel IPs ==="
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    if ip addr show gre1 | grep -q "$IP"; then
        echo "✓ $IP on GRE tunnel"
    else
        echo "✗ $IP missing from GRE tunnel"
        ERRORS=$((ERRORS+1))
    fi
done

echo "=== TEST 3: Container Running ==="
if docker ps | grep -q "billionmail-postfix"; then
    echo "✓ Container running"
else
    echo "✗ Container not running"
    ERRORS=$((ERRORS+1))
fi

echo "=== TEST 4: Container IPs ==="
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1 2>/dev/null)
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    if nsenter -t $CONTAINER_PID -n ip addr show lo 2>/dev/null | grep -q "$IP"; then
        echo "✓ $IP in container"
    else
        echo "✗ $IP missing from container"
        ERRORS=$((ERRORS+1))
    fi
done

echo "=== TEST 5: Host Routes ==="
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    if ip route | grep -q "$IP.*dev br-"; then
        echo "✓ $IP routed via bridge"
    else
        echo "✗ $IP not routed correctly"
        ERRORS=$((ERRORS+1))
    fi
done

echo "=== TEST 6: No Local Routes ==="
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    if ip route show table local 2>/dev/null | grep -q "local $IP"; then
        echo "✗ $IP has local route (BAD!)"
        ERRORS=$((ERRORS+1))
    else
        echo "✓ $IP no local route"
    fi
done

echo "=== TEST 7: Connectivity ==="
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1 2>/dev/null)
if nsenter -t $CONTAINER_PID -n ping -c 2 -W 5 -I 5.230.168.10 8.8.8.8 &>/dev/null; then
    echo "✓ Container can reach internet from 5.230.168.10"
else
    echo "✗ Container connectivity failed"
    ERRORS=$((ERRORS+1))
fi

echo "=== TEST 8: Postfix Transports ==="
for IP in 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11 5.230.168.12; do
    TRANSPORT="smtp_bind_ip_$(echo $IP | tr '.' '_')"
    if grep -q "^$TRANSPORT" conf/postfix/master.cf; then
        if grep -A1 "^$TRANSPORT" conf/postfix/master.cf | grep -q "smtp_bind_address=$IP"; then
            echo "✓ $IP transport has bind address"
        else
            echo "✗ $IP transport missing bind address"
            ERRORS=$((ERRORS+1))
        fi
    else
        echo "✗ $IP transport missing"
        ERRORS=$((ERRORS+1))
    fi
done

echo "=== TEST 9: Systemd Service ==="
if systemctl is-enabled noez-ips.service &>/dev/null; then
    echo "✓ Systemd service enabled"
else
    echo "✗ Systemd service not enabled"
    ERRORS=$((ERRORS+1))
fi

echo ""
echo "========================================="
if [ $ERRORS -eq 0 ]; then
    echo "✅ ALL TESTS PASSED!"
else
    echo "❌ $ERRORS TESTS FAILED"
    exit 1
fi
```

---

## 10. INTEGRATION POINTS

### BillionMail Database

**Table: `domain`**
- Stores domains for BillionMail UI
- Script adds entry during setup

**Table: `bm_domain_smtp_transport`**
- Maps domains to Postfix transports
- Columns: atype, domain, smtp_name
- Example: ('dedicated_ip', '@domain.com', 'smtp_bind_ip_5_230_168_0')

**Table: `bm_multi_ip_domain`**
- Shows Dedicated IP in UI
- Columns: domain, ip, in_use

### Postfix Integration

**master.cf transports:**
```
# Format:
smtp_bind_ip_X_X_X_X unix - - n - - smtp
  -o smtp_bind_address=X.X.X.X
```

**sender_dependent_default_transport_maps:**
- Configured in main.cf to use PostgreSQL
- Query: `SELECT CONCAT(smtp_name, ':') FROM bm_domain_smtp_transport WHERE domain = '%s'`

### Cloudflare API

**Required Permissions:**
- Zone:Read
- DNS:Edit

**Records Created:**
- A: mail.DOMAIN → HOST_IP
- TXT: DOMAIN → v=spf1 ip4:NOEZ_IP ~all
- TXT: _dmarc.DOMAIN → v=DMARC1; p=quarantine; rua=mailto:dmarc@DOMAIN

### Systemd Integration

**Service: `noez-ips.service`**
- Type: oneshot
- Runs: `/opt/billionmail/setup_noez_ips.sh`
- After: docker.service
- Enabled: multi-user.target

---

## 11. COMMAND REFERENCE

### Essential Commands

```bash
# Full setup
sudo bash noez_setup.sh

# Add new IP/domain
sudo bash noez_setup.sh add 5.230.168.X domain.com

# Check status
sudo bash noez_setup.sh status

# Fix after reboot
sudo bash /opt/billionmail/setup_noez_ips.sh

# Restart Postfix
docker restart billionmail-postfix-billionmail-1

# View mail logs
tail -f /opt/billionmail/logs/postfix/mail.log

# Check systemd service
systemctl status noez-ips.service
```

### Container Namespace Commands

```bash
# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Container shell
nsenter -t $CONTAINER_PID -n /bin/bash

# Container IP addresses
nsenter -t $CONTAINER_PID -n ip addr

# Container routing
nsenter -t $CONTAINER_PID -n ip route
nsenter -t $CONTAINER_PID -n ip rule

# Test from container
nsenter -t $CONTAINER_PID -n ping -I 5.230.168.10 8.8.8.8
nsenter -t $CONTAINER_PID -n nslookup google.com
```

### Network Debugging

```bash
# GRE tunnel
ip tunnel show
gre1: gre/ip remote 5.230.205.35 local 85.121.241.162 ttl 255

ip addr show gre1
ip route show table 20
ip rule | grep "from 5.230"

# Docker
ip route | grep 172.66
docker network ls
docker inspect billionmail-postfix-billionmail-1 --format='{{json .NetworkSettings.Networks}}'

# iptables
iptables -L FORWARD -n -v | head -20
iptables -t nat -L POSTROUTING -n -v
iptables -t mangle -L PREROUTING -n -v

# Routes
ip route show table local | grep 5.230
ip route | grep 5.230
```

### Database Queries

```bash
# List domains
docker exec -i billionmail-postfix-billionmail-1 psql -U billionmail -d billionmail -c "SELECT domain FROM domain;"

# Check transport mapping
docker exec -i billionmail-postfix-billionmail-1 psql -U billionmail -d billionmail -c "SELECT * FROM bm_domain_smtp_transport;"

# Check multi-IP domains
docker exec -i billionmail-postfix-billionmail-1 psql -U billionmail -d billionmail -c "SELECT * FROM bm_multi_ip_domain;"
```

---

## 12. COMMON PITFALLS

### Pitfall #1: Running Script Without sudo

**Problem:** Script fails with permission errors  
**Solution:** Always use `sudo bash noez_setup.sh`

### Pitfall #2: Wrong Container Name

**Problem:** Script can't find container  
**Check:** `docker ps | grep postfix`  
**Fix:** Update CONTAINER_NAME in noez_setup.env

### Pitfall #3: Missing ALL_NOEZ_IPS

**Problem:** After reboot, only some IPs work  
**Cause:** systemd service doesn't know about all IPs  
**Fix:** Update ALL_NOEZ_IPS in noez_setup.env with ALL IPs

### Pitfall #4: Not Including NOEZ_IP in ALL_NOEZ_IPS

**Problem:** Current IP not re-added on boot  
**Fix:** Ensure NOEZ_IP is listed in ALL_NOEZ_IPS

### Pitfall #5: Editing Auto-Generated Files

**Problem:** Changes lost when script runs again  
**Files not to edit:**
- `setup_noez_ips.sh` (auto-generated)
- `/etc/systemd/system/noez-ips.service` (auto-generated)

**Instead:** Edit `noez_setup.sh` to change generation logic

### Pitfall #6: Multiple Domains Same IP - SPF Record

**Problem:** Adding second domain to same IP overwrites SPF  
**Current Behavior:** Script deletes old SPF, creates new one  
**Fix Needed:** Should append IP to existing SPF, not replace  
**Workaround:** Manually edit SPF in Cloudflare to include all IPs

### Pitfall #7: Cloudflare API Token Permissions

**Problem:** DNS records not created  
**Check:** Token has Zone:Read + DNS:Edit permissions  
**Not Zone:Edit alone!**

### Pitfall #8: Postfix Reload vs Restart

**Problem:** New transports not recognized  
**Cause:** `postfix reload` doesn't pick up master.cf changes  
**Solution:** Must use `docker restart` or full postfix stop/start

---

## APPENDIX A: File Modification History

| Date | File | Change | Author |
|------|------|--------|--------|
| 2026-03-28 | noez_setup.sh | Initial creation | AI Agent |
| 2026-03-30 | noez_setup.sh | Add smtp_bind_address fix | AI Agent |
| 2026-03-30 | noez_setup.sh | Add host routing fix | AI Agent |
| 2026-03-30 | setup_noez_ips.sh | Add auto-detection | AI Agent |
| 2026-03-31 | SKILLS.md | Create knowledge base | AI Agent |

---

## APPENDIX B: External Dependencies

- Noez GRE tunnel service
- Cloudflare account (optional but recommended)
- BillionMail v5.x
- Docker 20.x+
- iproute2 (ip command)
- iptables
- nsenter (util-linux)
- PostgreSQL client (docker exec)

---

**Document Version:** 2.0 (Comprehensive)  
**Last Updated:** 2026-03-31  
**Author:** AI Agent Knowledge Preservation  
**Purpose:** Enable any AI agent to understand and fix this system

---

**⚠️ FINAL NOTE FOR AI AGENTS:**

This system works when:
1. GRE tunnel carries Noez IPs
2. Container binds to those IPs
3. Postfix uses correct source IP
4. Host forwards replies to container (NOT local!)

The #1 mistake is forgetting #4 - always check `ip route show table local` and ensure Noez IPs are NOT there!

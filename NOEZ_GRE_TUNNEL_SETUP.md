# Noez GRE Tunnel Email Routing Setup

## Problem Description

The BillionMail MoeScale V5.0.0 email server needed to send emails from a Noez GRE tunnel IP (5.230.168.0) instead of the host's primary IP (85.121.241.162). 

### The Challenge
- The GRE tunnel was set up on the host with IP 5.230.168.0 on interface `gre1`
- Postfix runs inside a Docker container with internal IP 172.66.2.100
- When Postfix tried to bind to 5.230.168.0 using `smtp_bind_address`, it failed because the IP didn't exist in the container's network namespace
- Direct IP binding inside the container was not possible since the GRE tunnel exists only on the host

### Initial Failed Attempts
1. **Direct container IP binding** - Failed because 5.230.168.0 wasn't in the container's namespace
2. **Adding IP to container** - Added 5.230.168.0 to container's lo interface, but routing failed
3. **SNAT without policy routing** - Container traffic was not being routed through the GRE tunnel

## Solution

The working solution uses a combination of:
1. **Host-side policy routing** with iptables packet marking
2. **SNAT (Source NAT)** to rewrite the source IP
3. **Container-side IP assignment** for proper identification

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HOST (85.121.241.162)                       │
│                                                                     │
│  ┌──────────────────┐         ┌─────────────────────────────────┐   │
│  │   Docker Bridge  │         │         GRE Tunnel              │   │
│  │  172.66.2.0/24   │◄────────│  gre1: 192.168.31.2/30          │   │
│  │                  │  route  │         5.230.168.0/32          │   │
│  │  172.66.2.1 (gw) │         │         peer: 5.230.205.35      │   │
│  └────────┬─────────┘         └─────────────────────────────────┘   │
│           │                                                         │
│  ┌────────▼──────────────────┐                                      │
│  │   Postfix Container       │                                      │
│  │   172.66.2.100 (eth0)     │                                      │
│  │   5.230.168.0/32 (lo)     │                                      │
│  └───────────────────────────┘                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Step-by-Step Implementation

#### Step 1: GRE Tunnel Setup (Already Done)
```bash
# Create GRE tunnel on host
ip tunnel add gre1 mode gre local 85.121.241.162 remote 5.230.205.35 ttl 255
ip addr add 192.168.31.2/30 dev gre1
ip addr add 5.230.168.0/32 dev gre1
ip link set gre1 up

# Add routing table for Noez IP
ip rule add from 5.230.168.0 table 20 prio 200
ip route add default via 192.168.31.1 dev gre1 table 20
```

#### Step 2: Add Noez IP to Container
```bash
# Get Postfix container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' billionmail-postfix-billionmail-1)

# Add Noez IP to container's loopback interface
nsenter -t $CONTAINER_PID -n ip addr add 5.230.168.0/32 dev lo

# Add container routing rule
nsenter -t $CONTAINER_PID -n ip rule add from 5.230.168.0 table 100
nsenter -t $CONTAINER_PID -n ip route add default via 172.66.2.1 table 100
```

#### Step 3: Host Policy Routing (The Key!)
```bash
# Mark packets from the container (mark 100)
iptables -t mangle -A PREROUTING -s 172.66.2.100 -j MARK --set-mark 100

# Route marked packets through table 20 (GRE tunnel)
ip rule add fwmark 100 table 20 prio 50

# SNAT to change source IP to 5.230.168.0
iptables -t nat -I POSTROUTING 1 -s 172.66.2.100 -j SNAT --to-source 5.230.168.0

# Allow forwarding for Noez IP
iptables -I FORWARD -s 5.230.168.0 -j ACCEPT
iptables -I FORWARD -d 5.230.168.0 -j ACCEPT
```

#### Step 4: Postfix Configuration

**conf/postfix/master.cf:**
```
# Noez IP for moescaleb2b.site (SNAT handled by host)
smtp_bind_ip_5_230_168_0 unix  -       -       n       -       -       smtp
```

**Database entry:**
```sql
INSERT INTO bm_domain_smtp_transport (atype, domain, smtp_name) 
VALUES ('dedicated_ip', '@moescaleb2b.site', 'smtp_bind_ip_5_230_168_0');
```

### Why This Works

1. **Container Identification**: The container has 5.230.168.0 on its loopback, so Postfix can identify itself with this IP in email headers

2. **Packet Marking**: Host iptables marks packets from 172.66.2.100 with mark 100

3. **Policy Routing**: The `fwmark 100` rule routes marked packets through table 20, which uses the GRE tunnel

4. **Source NAT**: Before packets leave the host, SNAT changes the source from 172.66.2.100 to 5.230.168.0

5. **Return Path**: Reply packets to 5.230.168.0 arrive via GRE tunnel, get DNATed back to 172.66.2.100, and reach the container

### Verification Commands

```bash
# Test connectivity from container
nsenter -t $CONTAINER_PID -n ping 8.8.8.8

# Check traffic goes through GRE
tcpdump -i gre1 -n icmp

# Send test email
docker exec -i billionmail-postfix-billionmail-1 sh -c 'cat << "EOF" | sendmail -v test@example.com
From: admin@moescaleb2b.site
To: test@example.com
Subject: Test from Noez IP

Test
EOF'

# Check mail logs
tail -f logs/postfix/mail.log
```

### Persistence After Reboot

To make this configuration persistent:

1. **GRE Tunnel**: Add to `/etc/rc.local` or create a systemd service
2. **Container Setup**: Run `setup_noez_email_routing.sh` after container starts
3. **Docker Compose**: Add to post-start hooks or use a sidecar container

### Files Created

- `setup_noez_email_routing.sh` - Automated setup script
- `NOEZ_GRE_TUNNEL_SETUP.md` - This documentation

---

**Setup Date**: 2026-03-30  
**Working**: ✅ Yes - Emails successfully sent from 5.230.168.0 via GRE tunnel

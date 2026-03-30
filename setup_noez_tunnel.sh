#!/bin/bash
# Permanent Noez GRE Tunnel Setup
# Run this on boot to restore the tunnel

LOCAL_IP="85.121.241.162"
REMOTE_IP="5.230.205.35"
TUNNEL_NAME="gre1"
INTERNAL_IP="192.168.31.2/30"
GATEWAY="192.168.31.1"

# Create tunnel if not exists
if ! ip tunnel show "$TUNNEL_NAME" 2>/dev/null | grep -q "$TUNNEL_NAME"; then
    ip tunnel add $TUNNEL_NAME mode gre local $LOCAL_IP remote $REMOTE_IP ttl 255
fi

# Configure interface
ip addr add $INTERNAL_IP dev $TUNNEL_NAME 2>/dev/null || true
ip link set $TUNNEL_NAME up

# Add Noez IPs (add your IPs here)
ip addr add 5.230.168.0/32 dev $TUNNEL_NAME 2>/dev/null || true
# Add more IPs as needed:
# ip addr add 5.230.168.1/32 dev $TUNNEL_NAME 2>/dev/null || true

# Add routing rules
ip rule add from 5.230.168.0 table 20 prio 1 2>/dev/null || true

# Add route
ip route add default via $GATEWAY dev $TUNNEL_NAME table 20 2>/dev/null || true

echo "Noez GRE tunnel restored"

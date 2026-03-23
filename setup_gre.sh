#!/bin/bash
# GRE Tunnel Setup for BillionMail

# Create GRE tunnel
ip tunnel add gre1 mode gre local 66.55.64.133 remote 5.230.205.35 ttl 255 2>/dev/null
ip addr add 192.168.31.2/30 dev gre1 2>/dev/null
ip link set gre1 up

# Add all 32 IPs
ip addr add 5.230.119.217/32 dev gre1 2>/dev/null
ip addr add 5.230.122.55/32 dev gre1 2>/dev/null
ip addr add 5.230.122.64/32 dev gre1 2>/dev/null
ip addr add 5.230.122.65/32 dev gre1 2>/dev/null

for i in $(seq 0 15); do
    ip addr add 5.230.168.$i/32 dev gre1 2>/dev/null
done

for i in $(seq 33 44); do
    ip addr add 5.230.168.$i/32 dev gre1 2>/dev/null
done

# Add routing rules
ip rule add from 5.230.119.217 table 20 prio 1 2>/dev/null
ip rule add from 5.230.122.55 table 20 prio 1 2>/dev/null
ip rule add from 5.230.122.64 table 20 prio 1 2>/dev/null
ip rule add from 5.230.122.65 table 20 prio 1 2>/dev/null

for i in $(seq 0 15); do
    ip rule add from 5.230.168.$i table 20 prio 1 2>/dev/null
done

for i in $(seq 33 44); do
    ip rule add from 5.230.168.$i table 20 prio 1 2>/dev/null
done

# Add default route
ip route add default via 192.168.31.1 dev gre1 table 20 2>/dev/null

echo "GRE tunnel setup complete"

# Add all GRE IPs to eth0 for BillionMail multi-IP support
ip addr add 5.230.119.217/32 dev eth0 2>/dev/null
ip addr add 5.230.122.55/32 dev eth0 2>/dev/null
ip addr add 5.230.122.64/32 dev eth0 2>/dev/null
ip addr add 5.230.122.65/32 dev eth0 2>/dev/null

for i in $(seq 0 15); do
    ip addr add 5.230.168.$i/32 dev eth0 2>/dev/null
done

for i in $(seq 33 44); do
    ip addr add 5.230.168.$i/32 dev eth0 2>/dev/null
done

echo "All GRE IPs added to eth0"

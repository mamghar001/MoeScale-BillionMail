#!/bin/bash
# Auto-setup all 32 Noez IPs on boot
CONTAINER_NAME="billionmail-postfix-billionmail-1"
DOCKER_BRIDGE_GW="172.66.2.1"
ALL_IPS="5.230.119.217 5.230.122.55 5.230.122.64 5.230.122.65 5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.3 5.230.168.4 5.230.168.5 5.230.168.6 5.230.168.7 5.230.168.8 5.230.168.9 5.230.168.10 5.230.168.11 5.230.168.12 5.230.168.13 5.230.168.14 5.230.168.15 5.230.168.33 5.230.168.34 5.230.168.35 5.230.168.36 5.230.168.37 5.230.168.38 5.230.168.39 5.230.168.40 5.230.168.41 5.230.168.42 5.230.168.43 5.230.168.44"

echo "Waiting for container $CONTAINER_NAME..."
for i in $(seq 1 60); do
    if [ "$(docker inspect -f '{{.State.Status}}' $CONTAINER_NAME 2>/dev/null)" = "running" ]; then break; fi
    sleep 1
done

CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null)
if [ -z "$CONTAINER_PID" ]; then echo "ERROR: Container not found!"; exit 1; fi
CONTAINER_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $CONTAINER_NAME 2>/dev/null | tr ' ' '\n' | grep '172.66.2' | head -1)
[ -z "$CONTAINER_IP" ] && CONTAINER_IP="172.66.2.100"

echo "Container PID=$CONTAINER_PID IP=$CONTAINER_IP"

for IP in $ALL_IPS; do
    echo "Setting up $IP..."
    # Bind to container loopback
    nsenter -t $CONTAINER_PID -n ip addr add $IP/32 dev lo 2>/dev/null || true
    # Policy routing inside container: source-based route via bridge GW
    nsenter -t $CONTAINER_PID -n ip rule add from $IP table 100 2>/dev/null || true
    nsenter -t $CONTAINER_PID -n ip route add default via $DOCKER_BRIDGE_GW table 100 2>/dev/null || true
    # Host-level: remove local route, add route via container
    ip route del local $IP 2>/dev/null || true
    ip route replace $IP/32 via $CONTAINER_IP 2>/dev/null || ip route add $IP/32 via $CONTAINER_IP 2>/dev/null || true
    # CRITICAL: Add host IP rule to route traffic from Noez IP through GRE tunnel
    ip rule add from $IP lookup 20 2>/dev/null || true
done
echo "All 32 Noez IPs configured!"

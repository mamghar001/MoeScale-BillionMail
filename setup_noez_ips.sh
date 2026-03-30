#!/bin/bash
# Auto-setup Noez IPs when Postfix container starts
# This script runs on boot to re-add IPs to container

CONTAINER_NAME="billionmail-postfix-billionmail-1"
ALL_IPS="5.230.168.0 5.230.168.1 5.230.168.2 5.230.168.10 5.230.168.11"

# Wait for container to be running
for i in {1..30}; do
    if [ "$(docker inspect -f '{{.State.Status}}' $CONTAINER_NAME 2>/dev/null)" == "running" ]; then
        break
    fi
    sleep 1
done

# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME 2>/dev/null)
if [ -z "$CONTAINER_PID" ]; then
    echo "Container not found"
    exit 1
fi

# Add all Noez IPs to container
for IP in $ALL_IPS; do
    nsenter -t $CONTAINER_PID -n ip addr add $IP/32 dev lo 2>/dev/null || true
done

echo "Noez IPs added to container: $ALL_IPS"

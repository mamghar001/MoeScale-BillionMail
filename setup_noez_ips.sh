#!/bin/bash
# Auto-setup Noez IPs when Postfix container starts
# Run this script after container restart to re-add IPs

CONTAINER_NAME="billionmail-postfix-billionmail-1"

# Wait for container to be running
while [ "$(docker inspect -f '{{.State.Status}}' $CONTAINER_NAME 2>/dev/null)" != "running" ]; do
    sleep 1
done

# Get container PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' $CONTAINER_NAME)

# Add all Noez IPs to container
nsenter -t $CONTAINER_PID -n ip addr add 5.230.168.0/32 dev lo 2>/dev/null || true
nsenter -t $CONTAINER_PID -n ip addr add 5.230.168.1/32 dev lo 2>/dev/null || true

echo "Noez IPs added to container"

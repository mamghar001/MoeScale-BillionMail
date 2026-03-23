# GRE Tunnel Setup Guide for BillionMail

This guide explains how to configure BillionMail with GRE tunnel IPs from noez.de for multi-IP email sending.

## Prerequisites

- VPS with GRE tunnel support
- noez.de GRE tunnel subscription
- Cloudflare DNS for your domains

## Setup Steps

### 1. Configure GRE Tunnel on Host

Run the setup script after each reboot:

```bash
sudo bash setup_gre.sh
```

This script:
- Creates the GRE tunnel interface (gre1)
- Adds all 32 IPs to the tunnel
- Configures routing rules
- Adds IPs to the host interface for BillionMail

### 2. Configure Multi-IP in BillionMail

Create `docker-compose_addnetwork.yml`:

```yaml
name: billionmail

services:
  postfix-billionmail:
    networks:
      billionmail-network:
        ipv4_address: 172.66.1.100
      billionmail-network-2:
        ipv4_address: 172.66.2.100

networks:
  billionmail-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.66.1.0/24

  billionmail-network-2:
    driver: bridge
    ipam:
      config:
        - subnet: 172.66.2.0/24
```

### 3. Apply Multi-IP Configuration

```bash
bash bm.sh multi_ip
```

This will:
- Configure Postfix with dedicated IPs for each domain
- Set up SMTP transport mappings
- Add IPs to the network interface

### 4. Add Domains with Dedicated IPs

Use the API to add domains:

```bash
curl -X POST "https://your-server/api/domains/create" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com","a_record":"mail.example.com"}'
```

Then configure dedicated IPs:

```bash
curl -X POST "https://your-server/api/multi_ip_domain/apply" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com","outbound_ip":"5.230.x.x"}'
```

### 5. Update DNS Records

Point `mail.{domain}` A records to the assigned GRE IPs in Cloudflare.

## Persistence

Add to crontab for automatic setup on reboot:

```bash
echo "@reboot /opt/billionmail/setup_gre.sh" | sudo crontab -
```

## Verification

```bash
# Check GRE tunnel
ip addr show gre1

# Check GRE IPs on interface
ip addr show eth0 | grep "5.230"

# Check domains
docker exec billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c "SELECT domain, outbound_ip FROM bm_multi_ip_domain;"
```

## Troubleshooting

### GRE Tunnel Not Working
```bash
# Test connectivity
ping -I 5.230.119.217 8.8.8.8
```

### Domains Not Sending from Dedicated IP
```bash
# Check SMTP transport mappings
docker exec billionmail-pgsql-billionmail-1 psql -U billionmail -d billionmail -c "SELECT * FROM bm_domain_smtp_transport;"
```

### DNS Issues
Ensure `mail.{domain}` A record points to the correct GRE IP.

## Notes

- Each domain should have a unique GRE IP for optimal deliverability
- Main server IP (66.55.64.133) is not used for sending
- GRE tunnel requires reconfiguration after server reboot

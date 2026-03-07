# BillionMail Namecheap DNS Auto-Sync

Automatically syncs DNS records between BillionMail and Namecheap when domains are added.

## What it does

1. Monitors BillionMail API for newly added domains
2. Automatically sets correct DNS records (A, MX, SPF, DKIM, DMARC) in Namecheap
3. Refreshes DNS validation in BillionMail

## Requirements

- BillionMail installed and running
- Namecheap API access with whitelisted IP
- Python 3 with requests module

## Setup

### 1. Namecheap API Whitelist

You MUST add your VPS IP to Namecheap API whitelist:
- Go to Namecheap → Profile → Tools → API Access
- Add your VPS IP: `85.121.241.162`
- Save settings

### 2. Install Script

```bash
sudo cp dns_auto_sync.py /opt/billionmail/
sudo cp billionmail-dns-sync.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable billionmail-dns-sync
sudo systemctl start billionmail-dns-sync
```

### 3. View Logs

```bash
sudo journalctl -u billionmail-dns-sync -f
```

## Configuration

Edit the script to set your credentials:

```python
# BillionMail API
BILLIONMAIL_TOKEN = "your_token_here"

# Namecheap API
NAMECHEAP_API_USER = "your_username"
NAMECHEAP_API_KEY = "your_api_key"

# VPS IP
VPS_IP = "85.121.241.162"
```

## How it works

When you add a domain in BillionMail (manually or via API), this script:

1. Detects the new domain within 60 seconds
2. Reads DKIM keys from BillionMail's rspamd-data folder
3. Sets DNS records in Namecheap:
   - A record: root domain → VPS IP
   - A record: mail subdomain → VPS IP
   - MX record: @ → mail.domain.com
   - SPF record: v=spf1 +ip4:85.121.241.162 ~all
   - DMARC record: v=DMARC1;p=quarantine;rua=mailto:dmarc@domain
   - DKIM default + short keys
4. Refreshes BillionMail DNS validation

## Important Notes

- When adding domains via BillionMail API, always include `hostname` field:
  ```json
  {
    "domain": "example.com",
    "hostname": "mail.example.com",
    ...
  }
  ```
- The script runs as a systemd service and auto-starts on boot
- Logs are available via `journalctl`

## License

MIT

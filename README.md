# BillionMail Namecheap DNS Auto-Sync

Automatically configure DNS records in Namecheap when you add domains to BillionMail.

## What It Does

This script monitors your BillionMail server for new domains and automatically sets up the required DNS records in Namecheap:

- **A Records**: Root domain and mail subdomain pointing to your VPS IP
- **MX Record**: Mail server configuration (priority 10)
- **SPF Record**: Email authentication
- **DMARC Record**: Email policy and reporting
- **DKIM Records**: Email signing (default + short keys)

## Requirements

- BillionMail server running
- Namecheap account with API access enabled
- VPS with the script running
- Your VPS IP whitelisted in Namecheap API settings

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/mamghar001/MoeScale-BillionMail.git
cd MoeScale-BillionMail
```

### 2. Configure Environment Variables

Create a `.env` file in the same directory as `dns_auto_sync.py`:

```bash
cp .env.example .env
nano .env
```

### 3. Edit the `.env` File

Fill in your actual credentials:

```env
# BillionMail API
BILLIONMAIL_BASE_URL=http://127.0.0.1/api
BILLIONMAIL_TOKEN=your_billionmail_api_token_here

# Namecheap API
NAMECHEAP_API_USER=your_namecheap_username
NAMECHEAP_API_KEY=your_namecheap_api_key
NAMECHEAP_CLIENT_IP=your_vps_ip_address

# VPS Configuration
VPS_IP=your_vps_ip
SECONDARY_IPS=ip1,ip2,ip3

# DKIM Keys path
DKIM_KEYS_PATH=/opt/billionmail/rspamd-data/dkim
```

**⚠️ IMPORTANT: Never commit the `.env` file to Git! It's already in `.gitignore`.**

### 4. Where to Put the Files

On your VPS:

```bash
# Create directory
sudo mkdir -p /opt/billionmail

# Copy files
sudo cp dns_auto_sync.py /opt/billionmail/
sudo cp .env /opt/billionmail/
sudo chmod 600 /opt/billionmail/.env  # Secure the credentials file
sudo chmod +x /opt/billionmail/dns_auto_sync.py
```

### 5. Install as Systemd Service

Create the service file `/etc/systemd/system/billionmail-dns-sync.service`:

```ini
[Unit]
Description=BillionMail Namecheap DNS Auto-Sync
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/billionmail
ExecStart=/usr/bin/python3 /opt/billionmail/dns_auto_sync.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable billionmail-dns-sync
sudo systemctl start billionmail-dns-sync
```

### 6. Check Status

```bash
# View service status
sudo systemctl status billionmail-dns-sync

# View logs
sudo journalctl -u billionmail-dns-sync -f
```

## How It Works

1. **Every 60 seconds**, the script checks BillionMail for domains
2. **When a new domain is found**, it automatically:
   - Disables Namecheap email forwarding (`EmailType=MX`)
   - Sets A records for root and mail subdomain
   - Configures MX record pointing to `mail.yourdomain.com`
   - Adds SPF, DMARC, and DKIM TXT records
   - Refreshes BillionMail DNS cache
3. **Logs everything** so you can monitor progress

## Getting Your Credentials

### BillionMail API Token

1. Login to your BillionMail admin panel
2. Go to Settings → API
3. Copy your API token

### Namecheap API Access

1. Login to Namecheap
2. Go to Profile → Tools → API Access
3. Enable API access
4. Whitelist your VPS IP address
5. Copy your API Key

## Troubleshooting

### MX Record Not Working

Make sure you included `EmailType=MX` in the Namecheap API calls (this script does it automatically). Without this, Namecheap uses email forwarding instead of custom MX.

### "Invalid request IP" Error

Your VPS IP is not whitelisted in Namecheap API settings. Add it at:
https://ap.www.namecheap.com/settings/tools/apiaccess/whitelist

### Script Not Detecting Domains

Check the logs:
```bash
sudo journalctl -u billionmail-dns-sync -f
```

## Security Notes

- **Never commit `.env` to Git** - it contains sensitive credentials
- **Keep `.env` permissions at 600** - only readable by root
- **Rotate API keys periodically** - good security practice

## License

GNU Affero General Public License v3.0

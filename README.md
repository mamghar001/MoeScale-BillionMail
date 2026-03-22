# BillionMail Namecheap DNS Auto-Sync

> **🚀 Quick Start - Use the Latest Stable Release:**
> ```bash
> # Clone the latest stable version (MoeScale V4.9.0)
> git clone -b MoeScale-V4.9.0 https://github.com/mamghar001/MoeScale-BillionMail.git
> 
> # Or checkout the tag after cloning
> git checkout MoeScale-V4.9.0
> ```
> 
> **Why use a tag?** Tags are permanent snapshots that never change - perfect for production!

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

## 📦 Releases & Tags

### Latest Stable Release: `MoeScale-V4.9.0`

To use the **stable version** (recommended for production):

```bash
# Clone directly to the stable tag
git clone -b MoeScale-V4.9.0 https://github.com/mamghar001/MoeScale-BillionMail.git

# Or checkout the tag after cloning
git checkout MoeScale-V4.9.0
```

### View All Available Tags

```bash
git tag -l
```

### Tag vs Branch - What's the Difference?

| Tag 🏷️ | Branch 🌿 |
|---------|-----------|
| **Frozen snapshot** - never changes | **Active timeline** - keeps moving |
| Perfect for releases | Perfect for development |
| Always points to same commit | Gets new commits |
| Safe backup point | Working area |

**Think of it like this:**
- 📸 **Tag** = A photograph (preserves a moment)
- 🛣️ **Branch** = A road (keeps extending)

### How to Go Back to Safe Version

If something breaks on the main branch:

```bash
# Switch to the safe tagged version
git checkout MoeScale-V4.9.0

# Or create a new branch from the tag
git checkout -b my-fix-branch MoeScale-V4.9.0
```

## License

MIT

---

# 🚀 Deploy & Rebuild Guide

Quick commands to deploy code changes to your running BillionMail.

## 📋 Command Summary

| Command | What it does | When to use |
|---------|--------------|-------------|
| `bm rebuild-frontend` | Builds frontend UI and copies to container | After editing `.vue` or frontend files |
| `sudo bash deploy-updated.sh --quick` | Patches JS files + restarts container | Quick deploy without building |
| `sudo bash deploy-updated.sh` | Full build + deploy | When you want clean build |
| `bm restart` | Restarts all containers | After config changes |
| `bm restart-service core` | Restarts only core container | Quick restart without rebuild |

---

## ⚡ Quick Deploy (Recommended for Development)

Fastest way - patches built files directly, no build needed:

```bash
cd /opt/billionmail
sudo bash deploy-updated.sh --quick
```

**What it does:**
1. Patches `rotate_senders: 0` → `1` in JS files
2. Restarts core container
3. ✅ Done in 10 seconds

---

## 🏗️ Full Frontend Rebuild

When you need clean build (uses Docker, no Node.js needed on host):

```bash
bm rebuild-frontend
```

**What it does:**
1. Builds frontend with `node:20-alpine` Docker image
2. Copies `core/frontend/dist/` → `core/public/dist/`
3. Copies files to running container
4. Restarts core container

---

## 🔧 Backend Rebuild (Go Binary)

Only needed if you modify `.go` files:

```bash
cd /opt/billionmail/core

# Build for x86 (most servers)
go build -ldflags="-s -w" -o billionmail-amd64 main.go

# Copy to main binary
cp billionmail-amd64 billionmail

# Restart to use new binary
cd /opt/billionmail
bm restart-service core
```

**Requires:** Go 1.22+ (if not available, use Docker method in `go-build.sh`)

---

## 🔄 Standard Management Commands

```bash
# Restart everything
bm restart

# Restart just the web UI (core)
bm restart-service core

# Check status
bm status

# View logs
bm log-container core

# Full rebuild all containers (⚠️ longer downtime)
bm rebuild
```

---

## 📝 When to Use What?

| Situation | Command |
|-----------|---------|
| Changed `.vue` or frontend | `bm rebuild-frontend` |
| Changed `.go` files | Build binary + `bm restart-service core` |
| Quick test (patched files) | `sudo bash deploy-updated.sh --quick` |
| Config changes only | `bm restart` |
| Everything broken | `bm rebuild` |

---

## 🐛 Troubleshooting

**Changes not showing?**
- Hard refresh browser: `Ctrl+Shift+R`
- Check: `bm log-container core | tail -20`

**Build fails?**
- Use `--quick` mode (patches files directly)
- Or use `bm rebuild-frontend` (uses Docker)

**Container won't start?**
- Check logs: `bm log-container core`
- Restore backup: `docker-compose -f docker-compose.yml.backup.XXX up -d`

---

## 📁 File Locations

| Type | Location |
|------|----------|
| Frontend source | `core/frontend/src/` |
| Frontend built | `core/public/dist/` |
| Backend source | `core/internal/` |
| Backend binary | `core/billionmail` |
| Configs | `conf/` |
| Deploy scripts | `deploy-updated.sh`, `bm` |

#!/usr/bin/env python3
"""
Fix DKIM DNS records in Cloudflare
Updates truncated DKIM keys with full keys from rspamd
"""

import os
import sys
import requests
import re
import time
import argparse

# Cloudflare API Token (set via environment variable or edit here)
CF_API_TOKEN = os.environ.get('CF_API_TOKEN', '')

HEADERS = {
    "Authorization": f"Bearer {CF_API_TOKEN}",
    "Content-Type": "application/json"
}

def get_zone_id(domain):
    """Get Cloudflare zone ID for domain"""
    url = f"https://api.cloudflare.com/client/v4/zones?name={domain}"
    resp = requests.get(url, headers=HEADERS)
    if resp.status_code == 200:
        data = resp.json()
        if data.get("success") and data.get("result"):
            return data["result"][0]["id"]
    return None

def get_dkim_key(domain, selector="default"):
    """Get DKIM key from rspamd"""
    key_path = f"/opt/billionmail/rspamd-data/dkim/{domain}/{selector}.pub"
    try:
        with open(key_path, 'r') as f:
            content = f.read()
            match = re.search(r'p=([A-Za-z0-9+/=]+)', 
                            content.replace('"', '').replace('\n', '').replace('\t', ''))
            if match:
                return match.group(1)
    except Exception as e:
        print(f"  Error reading DKIM for {domain}: {e}")
    return None

def delete_dkim_records(zone_id, selector):
    """Delete existing DKIM records"""
    record_name = f"{selector}._domainkey"
    url = f"https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records?name={record_name}"
    resp = requests.get(url, headers=HEADERS)
    
    if resp.status_code == 200:
        data = resp.json()
        if data.get("success") and data.get("result"):
            for record in data["result"]:
                del_url = f"https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records/{record['id']}"
                requests.delete(del_url, headers=HEADERS)

def create_dkim_record(zone_id, domain, selector, dkim_key):
    """Create DKIM TXT record in Cloudflare"""
    record_name = f"{selector}._domainkey"
    
    # Format: v=DKIM1; k=rsa; p=<key>
    content = f'v=DKIM1; k=rsa; p={dkim_key}'
    
    record_data = {
        "type": "TXT",
        "name": record_name,
        "content": content,
        "ttl": 1  # Auto
    }
    
    url = f"https://api.cloudflare.com/client/v4/zones/{zone_id}/dns_records"
    resp = requests.post(url, headers=HEADERS, json=record_data)
    
    if resp.status_code == 200:
        result = resp.json()
        return result.get("success", False)
    return False

def main():
    parser = argparse.ArgumentParser(description='Fix DKIM DNS records in Cloudflare')
    parser.add_argument('--token', help='Cloudflare API Token', default=CF_API_TOKEN)
    parser.add_argument('--domain', help='Specific domain to fix (default: all)')
    parser.add_argument('--delete-first', action='store_true', 
                       help='Delete existing records before creating new ones')
    args = parser.parse_args()
    
    if not args.token:
        print("Error: Cloudflare API token required. Set CF_API_TOKEN env var or use --token")
        sys.exit(1)
    
    global HEADERS
    HEADERS = {
        "Authorization": f"Bearer {args.token}",
        "Content-Type": "application/json"
    }
    
    dkim_dir = "/opt/billionmail/rspamd-data/dkim"
    
    if args.domain:
        domains = [args.domain] if os.path.isdir(os.path.join(dkim_dir, args.domain)) else []
    else:
        domains = sorted([d for d in os.listdir(dkim_dir) 
                         if os.path.isdir(os.path.join(dkim_dir, d))])
    
    print(f"Processing {len(domains)} domain(s)...\n")
    print("=" * 70)
    
    success_count = 0
    fail_count = 0
    
    for domain in domains:
        print(f"\nProcessing: {domain}")
        
        zone_id = get_zone_id(domain)
        if not zone_id:
            print(f"  ⚠️  Could not find zone ID (domain might not be in this CF account)")
            fail_count += 1
            continue
        
        if args.delete_first:
            delete_dkim_records(zone_id, "default")
            delete_dkim_records(zone_id, "short")
            time.sleep(0.5)
        
        # Create default DKIM
        dkim_key = get_dkim_key(domain, "default")
        if dkim_key:
            if create_dkim_record(zone_id, domain, "default", dkim_key):
                print(f"  ✅ default._domainkey")
                success_count += 1
            else:
                print(f"  ❌ default._domainkey")
                fail_count += 1
            time.sleep(0.3)
        
        # Create short DKIM
        short_key = get_dkim_key(domain, "short")
        if short_key:
            if create_dkim_record(zone_id, domain, "short", short_key):
                print(f"  ✅ short._domainkey")
                success_count += 1
            else:
                print(f"  ❌ short._domainkey (may already exist)")
            time.sleep(0.3)
    
    print("\n" + "=" * 70)
    print(f"SUMMARY: {success_count} records created/updated, {fail_count} failed")

if __name__ == "__main__":
    main()

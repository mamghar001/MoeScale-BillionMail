#!/usr/bin/env python3
"""
BillionMail + Namecheap DNS Auto-Configurator
Monitors BillionMail for new domains and automatically sets correct DNS records in Namecheap
"""

import json
import time
import os
import sys
import requests
from datetime import datetime

# Load configuration from environment variables or .env file
def load_config():
    """Load configuration from environment or .env file"""
    # Try to load from .env file if it exists
    env_file = os.path.join(os.path.dirname(os.path.abspath(__file__)), '.env')
    if os.path.exists(env_file):
        with open(env_file, 'r') as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    os.environ.setdefault(key, value)
    
    # Required configuration
    required_vars = [
        'BILLIONMAIL_TOKEN',
        'NAMECHEAP_API_USER',
        'NAMECHEAP_API_KEY',
        'NAMECHEAP_CLIENT_IP',
        'VPS_IP'
    ]
    
    missing = [var for var in required_vars if not os.environ.get(var)]
    if missing:
        print(f"ERROR: Missing required environment variables: {', '.join(missing)}")
        print("Please create a .env file with these variables or set them in your environment.")
        sys.exit(1)
    
    return {
        'BILLIONMAIL_BASE_URL': os.environ.get('BILLIONMAIL_BASE_URL', 'http://127.0.0.1/api'),
        'BILLIONMAIL_TOKEN': os.environ['BILLIONMAIL_TOKEN'],
        'NAMECHEAP_API_USER': os.environ['NAMECHEAP_API_USER'],
        'NAMECHEAP_API_KEY': os.environ['NAMECHEAP_API_KEY'],
        'NAMECHEAP_CLIENT_IP': os.environ['NAMECHEAP_CLIENT_IP'],
        'VPS_IP': os.environ['VPS_IP'],
        'SECONDARY_IPS': os.environ.get('SECONDARY_IPS', '').split(',') if os.environ.get('SECONDARY_IPS') else [],
        'DKIM_KEYS_PATH': os.environ.get('DKIM_KEYS_PATH', '/opt/billionmail/rspamd-data/dkim')
    }

# Load configuration
config = load_config()

BILLIONMAIL_BASE_URL = config['BILLIONMAIL_BASE_URL']
BILLIONMAIL_TOKEN = config['BILLIONMAIL_TOKEN']
NAMECHEAP_API_USER = config['NAMECHEAP_API_USER']
NAMECHEAP_API_KEY = config['NAMECHEAP_API_KEY']
NAMECHEAP_CLIENT_IP = config['NAMECHEAP_CLIENT_IP']
VPS_IP = config['VPS_IP']
SECONDARY_IPS = config['SECONDARY_IPS']
DKIM_KEYS_PATH = config['DKIM_KEYS_PATH']

def log(message):
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] {message}")
    sys.stdout.flush()

def get_billionmail_domains():
    """Fetch all domains from BillionMail API"""
    try:
        headers = {
            "Authorization": f"Bearer {BILLIONMAIL_TOKEN}",
            "Content-Type": "application/json"
        }
        response = requests.get(
            f"{BILLIONMAIL_BASE_URL}/domains/list?page=1&page_size=100",
            headers=headers,
            timeout=30
        )
        if response.status_code == 200:
            data = response.json()
            if data.get("success"):
                return data["data"]["list"]
        log(f"Error fetching domains: {response.text}")
        return []
    except Exception as e:
        log(f"Exception fetching domains: {e}")
        return []

def get_dkim_keys(domain):
    """Get DKIM keys from BillionMail rspamd-data"""
    try:
        domain_path = f"{DKIM_KEYS_PATH}/{domain}"
        if not os.path.exists(domain_path):
            return None, None
        
        # Read default DKIM
        default_pub_path = f"{domain_path}/default.pub"
        default_key = ""
        if os.path.exists(default_pub_path):
            with open(default_pub_path, 'r') as f:
                content = f.read()
                # Extract the p= value
                import re
                match = re.search(r'p=([A-Za-z0-9+/=]+)', content.replace('"', '').replace('\n', '').replace('\t', ''))
                if match:
                    default_key = match.group(1)
        
        # Read short DKIM
        short_pub_path = f"{domain_path}/short.pub"
        short_key = ""
        if os.path.exists(short_pub_path):
            with open(short_pub_path, 'r') as f:
                content = f.read()
                match = re.search(r'p=([A-Za-z0-9+/=]+)', content.replace('"', '').replace('\n', '').replace('\t', ''))
                if match:
                    short_key = match.group(1)
        
        return default_key, short_key
    except Exception as e:
        log(f"Error reading DKIM keys for {domain}: {e}")
        return None, None

def set_namecheap_dns(domain, default_dkim, short_dkim):
    """Set DNS records in Namecheap via API"""
    try:
        parts = domain.split('.')
        if len(parts) < 2:
            log(f"Invalid domain format: {domain}")
            return False
        
        sld = parts[0]
        tld = '.'.join(parts[1:])
        
        # Use VPS IP for Namecheap API
        actual_ip = VPS_IP
        
        # Build SPF record with all IPs
        spf_ips = f"+ip4:{VPS_IP}"
        for ip in SECONDARY_IPS:
            spf_ips += f" +ip4:{ip}"
        spf_value = f"v=spf1 {spf_ips} ~all"
        
        # Build API URL
        base_url = "https://api.namecheap.com/xml.response"
        
        # Namecheap API has issues saving MX when combined with many other records
        # We need to set records in 3 separate calls
        
        success = True
        
        # Step 1: Set A records only (with EmailType=MX to disable forwarding)
        a_params = {
            "ApiUser": NAMECHEAP_API_USER,
            "ApiKey": NAMECHEAP_API_KEY,
            "UserName": NAMECHEAP_API_USER,
            "Command": "namecheap.domains.dns.setHosts",
            "ClientIp": actual_ip,
            "SLD": sld,
            "TLD": tld,
            "EmailType": "MX",  # CRITICAL: Disables Namecheap email forwarding
            "HostName1": "@",
            "RecordType1": "A",
            "Address1": VPS_IP,
            "HostName2": "mail",
            "RecordType2": "A",
            "Address2": VPS_IP,
        }
        
        a_response = requests.get(base_url, params=a_params, timeout=30)
        if 'Status="OK"' in a_response.text:
            log(f"✅ A records set for {domain}")
        else:
            log(f"⚠️  A records failed for {domain}")
            success = False
        
        # Step 2: Set TXT records (SPF, DMARC, DKIM)
        txt_params = {
            "ApiUser": NAMECHEAP_API_USER,
            "ApiKey": NAMECHEAP_API_KEY,
            "UserName": NAMECHEAP_API_USER,
            "Command": "namecheap.domains.dns.setHosts",
            "ClientIp": actual_ip,
            "SLD": sld,
            "TLD": tld,
            "EmailType": "MX",  # CRITICAL: Keep custom MX enabled
            "HostName1": "@",
            "RecordType1": "A",
            "Address1": VPS_IP,
            "HostName2": "mail",
            "RecordType2": "A",
            "Address2": VPS_IP,
            "HostName3": "@",
            "RecordType3": "TXT",
            "Address3": spf_value,
            "HostName4": "_dmarc",
            "RecordType4": "TXT",
            "Address4": f"v=DMARC1;p=quarantine;rua=mailto:dmarc@{domain}",
            "HostName5": "default._domainkey",
            "RecordType5": "TXT",
            "Address5": f"v=DKIM1;k=rsa;p={default_dkim}",
            "HostName6": "short._domainkey",
            "RecordType6": "TXT",
            "Address6": f"v=DKIM1;k=rsa;p={short_dkim}",
        }
        
        txt_response = requests.get(base_url, params=txt_params, timeout=30)
        if 'Status="OK"' in txt_response.text:
            log(f"✅ TXT records set for {domain}")
        else:
            log(f"⚠️  TXT records failed for {domain}")
        
        # Step 3: Set MX record LAST (with EmailType=MX to ensure it works)
        mx_params = {
            "ApiUser": NAMECHEAP_API_USER,
            "ApiKey": NAMECHEAP_API_KEY,
            "UserName": NAMECHEAP_API_USER,
            "Command": "namecheap.domains.dns.setHosts",
            "ClientIp": actual_ip,
            "SLD": sld,
            "TLD": tld,
            "EmailType": "MX",  # CRITICAL: Enables custom MX instead of forwarding
            "HostName1": "@",
            "RecordType1": "A",
            "Address1": VPS_IP,
            "HostName2": "mail",
            "RecordType2": "A",
            "Address2": VPS_IP,
            "HostName3": "@",
            "RecordType3": "TXT",
            "Address3": spf_value,
            "HostName4": "_dmarc",
            "RecordType4": "TXT",
            "Address4": f"v=DMARC1;p=quarantine;rua=mailto:dmarc@{domain}",
            "HostName5": "default._domainkey",
            "RecordType5": "TXT",
            "Address5": f"v=DKIM1;k=rsa;p={default_dkim}",
            "HostName6": "short._domainkey",
            "RecordType6": "TXT",
            "Address6": f"v=DKIM1;k=rsa;p={short_dkim}",
            "HostName7": "@",
            "RecordType7": "MX",
            "Address7": f"mail.{domain}.",
            "MXPref7": "10",
        }
        
        mx_response = requests.get(base_url, params=mx_params, timeout=30)
        if 'Status="OK"' in mx_response.text:
            log(f"✅ MX record set for {domain}")
        else:
            log(f"⚠️  MX record failed for {domain}")
            success = False
        
        if success:
            log(f"✅ All DNS records set successfully for {domain}")
            return True
        elif 'Invalid request IP' in mx_response.text or '1011150' in mx_response.text:
            log(f"⚠️  Namecheap API Error: VPS IP {actual_ip} not whitelisted!")
            log(f"   To fix: Add IP {actual_ip} to your Namecheap API whitelist")
            log(f"   Go to Namecheap Profile → Tools → API Access → Whitelist IP")
            return False
        elif '1011105' in mx_response.text:
            log(f"⚠️  Namecheap API Error: ClientIP parameter invalid")
            log(f"   Current IP: {actual_ip}")
            log(f"   Make sure this IP is whitelisted in Namecheap API settings")
            return False
        else:
            log(f"❌ Failed to set DNS for {domain}")
            return False
            
    except Exception as e:
        log(f"Exception setting DNS for {domain}: {e}")
        return False

def refresh_billionmail_dns(domain):
    """Refresh DNS validation in BillionMail"""
    try:
        headers = {
            "Authorization": f"Bearer {BILLIONMAIL_TOKEN}",
            "Content-Type": "application/json"
        }
        response = requests.post(
            f"{BILLIONMAIL_BASE_URL}/domains/fresh_dns_records",
            headers=headers,
            json={"domain": domain},
            timeout=30
        )
        if response.status_code == 200 and response.json().get("success"):
            log(f"🔄 BillionMail DNS refreshed for {domain}")
            return True
        return False
    except Exception as e:
        log(f"Error refreshing DNS in BillionMail: {e}")
        return False

def process_domain(domain_data):
    """Process a single domain"""
    domain = domain_data.get("domain")
    if not domain:
        return
    
    log(f"Processing domain: {domain}")
    
    # Get DKIM keys from BillionMail
    default_dkim, short_dkim = get_dkim_keys(domain)
    
    if not default_dkim or not short_dkim:
        log(f"⚠️  DKIM keys not found for {domain}, skipping...")
        return
    
    # Set DNS in Namecheap
    if set_namecheap_dns(domain, default_dkim, short_dkim):
        # Wait a moment for DNS propagation
        time.sleep(2)
        # Refresh in BillionMail
        refresh_billionmail_dns(domain)

def main():
    """Main loop - check for domains and process them"""
    log("🚀 BillionMail + Namecheap DNS Auto-Configurator started")
    log(f"⏱️  Checking every 60 seconds...")
    
    processed_domains = set()
    
    while True:
        try:
            domains = get_billionmail_domains()
            
            for domain_data in domains:
                domain = domain_data.get("domain")
                
                if domain and domain not in processed_domains:
                    process_domain(domain_data)
                    processed_domains.add(domain)
            
            time.sleep(60)  # Check every minute
            
        except KeyboardInterrupt:
            log("👋 Shutting down...")
            break
        except Exception as e:
            log(f"Main loop error: {e}")
            time.sleep(60)

if __name__ == "__main__":
    main()

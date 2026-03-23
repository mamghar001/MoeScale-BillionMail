#!/usr/bin/env python3
"""
Add mailboxes to BillionMail domains via API
Uses American female names for firstname.lastname@domain format
"""

import requests
import json
import time
import random
import urllib3

# Disable SSL warnings
urllib3.disable_warnings()

# Configuration
API_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50SWQiOjEsInVzZXJuYW1lIjoiRmxxZGZ1VkgiLCJyb2xlcyI6WyJhZG1pbiJdLCJhcGlUb2tlbiI6dHJ1ZSwiaXNzIjoiYmlsbGlvbi1tYWlsIiwic3ViIjoiYXBpX3Rva2VuIiwibmJmIjoxNzc0MjA4OTYyLCJpYXQiOjE3NzQyMDg5NjIsImp0aSI6InY2YngwdDBvMDAwZGg5a2w1N3NlYm1mMTAwOWU4cmNndjZieDB0MG8wMDBkaDlrbDU3c2VqbngyMDAxbDVpdG8ifQ.QSnxA6E40wWzX6U--DRwN_2LUDjljTa2_if0SZKBloA"
API_BASE = "https://66.55.64.133/api"
PASSWORD = "mokanaBillionmailMoescale321890"
MAILBOXES_PER_DOMAIN = 10  # TEST MODE: 10 per domain

# American female names
FIRST_NAMES = [
    "Emma", "Olivia", "Ava", "Isabella", "Sophia", "Mia", "Charlotte", "Amelia", "Evelyn", "Abigail",
    "Harper", "Emily", "Elizabeth", "Avery", "Sofia", "Ella", "Madison", "Scarlett", "Victoria", "Aria",
    "Grace", "Chloe", "Camila", "Penelope", "Riley", "Layla", "Lillian", "Nora", "Zoey", "Mila",
    "Aubrey", "Hannah", "Lily", "Addison", "Eleanor", "Natalie", "Luna", "Savannah", "Brooklyn", "Leah",
    "Zoe", "Stella", "Hazel", "Ellie", "Paisley", "Audrey", "Skylar", "Violet", "Claire", "Bella",
    "Aurora", "Lucy", "Anna", "Samantha", "Caroline", "Genesis", "Aaliyah", "Kennedy", "Kinsley", "Allison",
    "Maya", "Sarah", "Madelyn", "Adeline", "Alexa", "Ariana", "Elena", "Gabriella", "Naomi", "Alice",
    "Sadie", "Hailey", "Eva", "Emilia", "Autumn", "Quinn", "Nevaeh", "Piper", "Ruby", "Serenity",
    "Willow", "Everly", "Cora", "Kaylee", "Lydia", "Aubree", "Arianna", "Eliana", "Peyton", "Melanie",
    "Gianna", "Isabelle", "Julia", "Valentina", "Nova", "Clara", "Vivian", "Reagan", "Mackenzie", "Madeline"
]

LAST_NAMES = [
    "Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
    "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
    "Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
    "Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
    "Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter", "Roberts",
    "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker", "Cruz", "Edwards", "Collins", "Reyes",
    "Stewart", "Morris", "Morales", "Murphy", "Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper",
    "Peterson", "Bailey", "Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
    "Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza", "Ruiz", "Hughes"
]

def get_domains():
    """Get all domains from BillionMail"""
    headers = {"Authorization": f"Bearer {API_TOKEN}"}
    try:
        response = requests.get(f"{API_BASE}/domains/list", headers=headers, verify=False, timeout=30)
        data = response.json()
        if data.get("success"):
            return [d["domain"] for d in data["data"]["list"]]
    except Exception as e:
        print(f"Error getting domains: {e}")
    return []

def generate_mailbox_name(used_names):
    """Generate unique firstname.lastname"""
    max_attempts = 1000
    for _ in range(max_attempts):
        first = random.choice(FIRST_NAMES)
        last = random.choice(LAST_NAMES)
        email_local = f"{first.lower()}.{last.lower()}"
        
        if email_local not in used_names:
            used_names.add(email_local)
            return first, last, email_local
    
    # If all combinations exhausted, add number
    first = random.choice(FIRST_NAMES)
    last = random.choice(LAST_NAMES)
    email_local = f"{first.lower()}.{last.lower()}{random.randint(1,999)}"
    return first, last, email_local

def create_mailbox(domain, first_name, last_name, email_local):
    """Create a mailbox via API"""
    headers = {
        "Authorization": f"Bearer {API_TOKEN}",
        "Content-Type": "application/json"
    }
    
    email = f"{email_local}@{domain}"
    display_name = f"{first_name} {last_name}"
    
    # API uses PascalCase field names
    data = {
        "Domain": domain,
        "LocalPart": email_local,
        "Password": PASSWORD,
        "Name": display_name
    }
    
    try:
        response = requests.post(
            f"{API_BASE}/mailbox/create",
            headers=headers,
            json=data,
            verify=False,
            timeout=30
        )
        result = response.json()
        return result.get("success", False), result.get("msg", "Unknown error")
    except Exception as e:
        return False, str(e)

def main():
    print("=" * 70)
    print("BillionMail Mailbox Creator - TEST MODE")
    print(f"Creating {MAILBOXES_PER_DOMAIN} mailboxes per domain")
    print(f"Password: {PASSWORD}")
    print("=" * 70)
    
    # Get all domains
    print("\nFetching domains...")
    domains = get_domains()
    print(f"Found {len(domains)} domains")
    print(f"Total mailboxes to create: {len(domains) * MAILBOXES_PER_DOMAIN}")
    print()
    
    print("Starting mailbox creation in 3 seconds...")
    time.sleep(3)
    print()
    
    total_created = 0
    total_failed = 0
    
    for domain_idx, domain in enumerate(domains, 1):
        print(f"\n{'='*70}")
        print(f"[{domain_idx}/{len(domains)}] Processing domain: {domain}")
        print(f"{'='*70}")
        
        used_names = set()
        created = 0
        failed = 0
        
        for i in range(MAILBOXES_PER_DOMAIN):
            first, last, email_local = generate_mailbox_name(used_names)
            success, msg = create_mailbox(domain, first, last, email_local)
            
            if success:
                created += 1
                email = f"{email_local}@{domain}"
                print(f"  ✅ {email} ({first} {last})")
            else:
                failed += 1
                print(f"  ❌ {email_local}@{domain}: {msg}")
            
            # Small delay to avoid overwhelming the API
            time.sleep(0.3)
        
        print(f"\nDomain {domain} complete: {created} created, {failed} failed")
        total_created += created
        total_failed += failed
        
        # Small pause between domains
        if domain_idx < len(domains):
            time.sleep(1)
    
    print(f"\n{'='*70}")
    print("SUMMARY")
    print(f"{'='*70}")
    print(f"Total domains processed: {len(domains)}")
    print(f"Total mailboxes created: {total_created}")
    print(f"Total failed: {total_failed}")
    print(f"Password for all mailboxes: {PASSWORD}")
    print(f"\nTest completed! To create 100 mailboxes per domain, change MAILBOXES_PER_DOMAIN to 100")
    print(f"and run the script again.")

if __name__ == "__main__":
    main()

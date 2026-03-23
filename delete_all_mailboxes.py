#!/usr/bin/env python3
"""
Delete all existing mailboxes via BillionMail API
"""

import requests
import json
import time
import urllib3

urllib3.disable_warnings()

API_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50SWQiOjEsInVzZXJuYW1lIjoiRmxxZGZ1VkgiLCJyb2xlcyI6WyJhZG1pbiJdLCJhcGlUb2tlbiI6dHJ1ZSwiaXNzIjoiYmlsbGlvbi1tYWlsIiwic3ViIjoiYXBpX3Rva2VuIiwibmJmIjoxNzc0MjA4OTYyLCJpYXQiOjE3NzQyMDg5NjIsImp0aSI6InY2YngwdDBvMDAwZGg5a2w1N3NlYm1mMTAwOWU4cmNndjZieDB0MG8wMDBkaDlrbDU3c2VqbngyMDAxbDVpdG8ifQ.QSnxA6E40wWzX6U--DRwN_2LUDjljTa2_if0SZKBloA"
API_BASE = "https://66.55.64.133/api"

def get_all_mailboxes():
    """Get all mailboxes from all domains"""
    headers = {"Authorization": f"Bearer {API_TOKEN}"}
    mailboxes = []
    
    try:
        # Get domains first
        response = requests.get(f"{API_BASE}/domains/list", headers=headers, verify=False, timeout=30)
        data = response.json()
        
        if data.get("success"):
            domains = [d["domain"] for d in data["data"]["list"]]
            
            # Get mailboxes for each domain
            for domain in domains:
                try:
                    resp = requests.get(
                        f"{API_BASE}/mailbox/list?domain={domain}",
                        headers=headers,
                        verify=False,
                        timeout=30
                    )
                    result = resp.json()
                    if result.get("success") and result["data"].get("list"):
                        for mb in result["data"]["list"]:
                            mailboxes.append({
                                "username": mb.get("username"),
                                "domain": domain
                            })
                except Exception as e:
                    print(f"  Error getting mailboxes for {domain}: {e}")
                time.sleep(0.2)
    except Exception as e:
        print(f"Error: {e}")
    
    return mailboxes

def delete_mailbox(username):
    """Delete a mailbox via API"""
    headers = {
        "Authorization": f"Bearer {API_TOKEN}",
        "Content-Type": "application/json"
    }
    
    data = {"Username": username}
    
    try:
        response = requests.post(
            f"{API_BASE}/mailbox/delete",
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
    print("=" * 60)
    print("DELETE ALL MAILBOXES")
    print("=" * 60)
    
    print("\nFetching all mailboxes...")
    mailboxes = get_all_mailboxes()
    print(f"Found {len(mailboxes)} mailboxes to delete")
    
    if len(mailboxes) == 0:
        print("No mailboxes found. Nothing to delete.")
        return
    
    print(f"\nThis will DELETE all {len(mailboxes)} mailboxes!")
    print("Starting in 5 seconds...")
    time.sleep(5)
    print()
    
    deleted = 0
    failed = 0
    
    for i, mb in enumerate(mailboxes, 1):
        username = mb["username"]
        success, msg = delete_mailbox(username)
        
        if success:
            deleted += 1
            print(f"  ✅ [{i}/{len(mailboxes)}] Deleted: {username}")
        else:
            failed += 1
            print(f"  ❌ [{i}/{len(mailboxes)}] Failed: {username} - {msg}")
        
        time.sleep(0.2)
        
        # Progress every 10
        if i % 10 == 0:
            print(f"  Progress: {i}/{len(mailboxes)} deleted")
    
    print(f"\n{'='*60}")
    print("SUMMARY")
    print(f"{'='*60}")
    print(f"Total mailboxes: {len(mailboxes)}")
    print(f"Deleted: {deleted}")
    print(f"Failed: {failed}")
    print("\nAll mailboxes deleted! Ready to create fresh ones.")

if __name__ == "__main__":
    main()

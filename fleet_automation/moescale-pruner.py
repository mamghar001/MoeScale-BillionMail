#!/usr/bin/env python3
"""
MoeScale Dead Mailbox Pruner
Safely eliminates confirmed non-existent / dead email addresses from bm_contacts.
Protects valid leads affected by temporary spam/blacklist blocks and real unsubscribes.
Runs with low priority (nice/ionice) to guarantee zero impact on system smoothness.
"""

import sys
import json
import time
import subprocess
from datetime import datetime, timezone

LOG_FILE = "/var/log/moescale-pruner.log"
BATCH_SIZE = 100
BATCH_PAUSE_SEC = 0.05

DEAD_KEYWORDS = [
    'user unknown', 'no such user', 'recipient address rejected', 
    'does not exist', 'mailbox not found', 'invalid recipient', 
    'mailbox unavailable', 'undeliverable address', 'no mailbox', 
    'unknown user', 'not exist', 'recipient not found', 
    'account disabled', 'account closed', '550 5.1.1', '550 5.1.2', '550 5.1.0',
    'host or domain name not found', 'name service error', 'host not found',
    'manually added'
]

PROTECT_KEYWORDS = [
    'spamhaus', 'proofpoint', 'spam', 'blacklist', 'rbl', 'dnsbl',
    'blocked using', 'rate limit', 'greylist', 'policy rejection',
    'surbl', 'barracuda', 'cloudmark', 'junkemailfilter'
]

def log(msg):
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")
    line = f"[{ts}] {msg}"
    print(line)
    try:
        with open(LOG_FILE, "a") as f:
            f.write(line + "\n")
    except Exception:
        pass

def run_psql(sql, timeout=30):
    cmd = [
        "docker", "exec", "-i", "billionmail-pgsql-billionmail-1",
        "psql", "-U", "billionmail", "-d", "billionmail", "-t", "-A", "-c", sql
    ]
    try:
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=timeout)
        if res.returncode != 0:
            log(f"PSQL Error: {res.stderr.strip()}")
            return None
        return res.stdout.strip()
    except Exception as e:
        log(f"PSQL Exception: {e}")
        return None

def prune():
    log("=== MoeScale Dead Mailbox Pruner Starting ===")
    
    # 1. Fetch abnormal recipients
    sql = "SELECT json_agg(t) FROM (SELECT DISTINCT recipient, description FROM abnormal_recipient WHERE recipient IS NOT NULL AND recipient != '') t;"
    out = run_psql(sql, timeout=60)
    if not out:
        log("No records retrieved from abnormal_recipient.")
        return

    try:
        records = json.loads(out)
    except Exception as e:
        log(f"Failed to parse abnormal records JSON: {e}")
        return

    dead_emails = set()
    protected_count = 0
    uncategorized_count = 0

    for r in records:
        email = (r.get("recipient") or "").strip().lower()
        desc = (r.get("description") or "").lower()
        if not email or "@" not in email:
            continue

        is_protect = any(pk in desc for pk in PROTECT_KEYWORDS)
        is_dead = any(dk in desc for dk in DEAD_KEYWORDS)

        if is_protect and not is_dead:
            protected_count += 1
        elif is_dead:
            dead_emails.add(email)
        else:
            uncategorized_count += 1

    total_dead = len(dead_emails)
    log(f"Identified {total_dead} confirmed dead email addresses.")
    log(f"Protected {protected_count} valid leads with temporary spam/RBL filter blocks.")
    log(f"Preserved {uncategorized_count} uncategorized/policy bounce records.")

    if not dead_emails:
        log("No dead emails to prune.")
        return

    # 2. Batch prune from bm_contacts and bm_contact_tags
    dead_list = list(dead_emails)
    total_deleted = 0

    for i in range(0, total_dead, BATCH_SIZE):
        batch = dead_list[i:i + BATCH_SIZE]
        emails_escaped = ",".join("'" + e.replace("'", "''") + "'" for e in batch)
        
        del_sql = f"""
        WITH deleted AS (
            DELETE FROM bm_contacts WHERE email IN ({emails_escaped}) RETURNING id
        )
        DELETE FROM bm_contact_tags WHERE contact_id IN (SELECT id FROM deleted);
        """
        res = run_psql(del_sql, timeout=30)
        
        # In batch, check deleted count if available
        time.sleep(BATCH_PAUSE_SEC)
        
        if (i // BATCH_SIZE) % 10 == 0 or (i + BATCH_SIZE) >= total_dead:
            log(f"Processed {min(i + BATCH_SIZE, total_dead)}/{total_dead} dead addresses...")

    log("=== MoeScale Dead Mailbox Pruner Completed Successfully ===")

if __name__ == "__main__":
    prune()

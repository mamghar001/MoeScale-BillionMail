#!/usr/bin/env python3
import requests
import json
import time
import urllib3

urllib3.disable_warnings()

API_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50SWQiOjEsInVzZXJuYW1lIjoiRmxxZGZ1VkgiLCJyb2xlcyI6WyJhZG1pbiJdLCJhcGlUb2tlbiI6dHJ1ZSwiaXNzIjoiYmlsbGlvbi1tYWlsIiwic3ViIjoiYXBpX3Rva2VuIiwibmJmIjoxNzc0MjA4OTYyLCJpYXQiOjE3NzQyMDg5NjIsImp0aSI6InY2YngwdDBvMDAwZGg5a2w1N3NlYm1mMTAwOWU4cmNndjZieDB0MG8wMDBkaDlrbDU3c2VqbngyMDAxbDVpdG8ifQ.QSnxA6E40wWzX6U--DRwN_2LUDjljTa2_if0SZKBloA"
API_BASE = "https://66.55.64.133/api"

# Try direct database deletion instead
def delete_via_sql():
    import subprocess
    result = subprocess.run([
        'docker', 'exec', 'billionmail-pgsql-billionmail-1', 
        'psql', '-U', 'billionmail', '-d', 'billionmail',
        '-c', 'DELETE FROM mailbox;'
    ], capture_output=True, text=True)
    return result.returncode == 0, result.stdout + result.stderr

print("Deleting all mailboxes via SQL...")
success, output = delete_via_sql()
if success:
    print("✅ All mailboxes deleted successfully")
else:
    print(f"Error: {output}")

# Also delete from aliases table
def delete_aliases():
    import subprocess
    result = subprocess.run([
        'docker', 'exec', 'billionmail-pgsql-billionmail-1', 
        'psql', '-U', 'billionmail', '-d', 'billionmail',
        '-c', 'DELETE FROM alias WHERE address LIKE \"%@%.%\";'
    ], capture_output=True, text=True)
    return result.returncode == 0, result.stdout + result.stderr

print("\nCleaning up aliases...")
success, output = delete_aliases()
if success:
    print("✅ Aliases cleaned")

print("\nDone! Ready to create new mailboxes.")

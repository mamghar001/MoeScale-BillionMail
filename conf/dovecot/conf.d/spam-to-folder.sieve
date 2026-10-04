require ["fileinto", "mailbox", "regex"];

# 1. Flagged as spam by Rspamd (X-Spam: Yes)
if header :is "X-Spam" "Yes" {
  fileinto :create "Junk";
  stop;
}

# 2. Known affiliate marketing tracking headers
if anyof (
  exists "x-campaign-name",
  exists "x-offer-id"
) {
  fileinto :create "Junk";
  stop;
}

# 3. Mass marketing blasts (List-Unsubscribe / Precedence: bulk) that are NOT replies
# Note: Real prospect replies always have In-Reply-To, References, or "Re:" in Subject
if allof (
  anyof (
    exists "List-Unsubscribe",
    header :contains "Precedence" "bulk"
  ),
  not anyof (
    exists "In-Reply-To",
    exists "References",
    header :regex :comparator "i;ascii-casemap" "Subject" "^(re|aw|r|sv)\\s*:"
  )
) {
  fileinto :create "Junk";
  stop;
}

# 4. Known recurring scam / affiliate blast subjects and senders
if anyof (
  header :contains "Subject" "Silver-infused fabric",
  header :contains "Subject" "Medicare Kit waiting for you",
  header :contains "Subject" "Dr. Oz",
  header :contains "Subject" "Losartan",
  header :contains "Subject" "arterial cement",
  header :contains "Subject" "blood pressure flush",
  header :contains "Subject" "arterial flush",
  header :contains "Subject" "clogged arteries",
  header :contains "Subject" "PolarVision",
  header :contains "Subject" "PolorVision",
  header :contains "From" "Miracle Made Partner",
  header :contains "From" "Cardiovascular Breakthrough",
  header :contains "From" "CBS Breaking Health"
) {
  fileinto :create "Junk";
  stop;
}

require ["fileinto", "mailbox", "body", "envelope"];

if anyof (
  exists "X-SP-WUP",
  exists "X-SP-RFS",
  header :contains "X-SP-WUP" "1",
  header :contains "Subject" "[ref:",
  header :contains "Subject" "Strachka",
  header :contains "Subject" "straska",
  body :contains "Strachka",
  body :contains "straska",
  exists "X-Warmy-ID",
  exists "X-Mail-Warmup",
  exists "X-Smartlead-Warmup",
  exists "X-Instantly-Warmup"
) {
  # Discard warmup copies delivered to all_replies catchall to protect disk I/O
  if anyof (
    envelope :is "to" "all_replies@b2bprosperity.com",
    header :contains "Delivered-To" "all_replies@b2bprosperity.com",
    header :contains "X-Original-To" "all_replies@b2bprosperity.com"
  ) {
    discard;
    stop;
  }
  fileinto :create "Warmup";
  stop;
}



#!/bin/bash
set -e

# Check if output file exists
if [[ ! -f /tmp/decoded-password.txt ]]; then
    echo "FAIL: Output file '/tmp/decoded-password.txt' not found"
    exit 1
fi

# Read and verify the decoded password
DECODED=$(cat /tmp/decoded-password.txt)

# Remove any trailing newline for comparison
DECODED=$(echo -n "$DECODED" | tr -d '\n')

if [[ "$DECODED" != "SuperSecret123!" ]]; then
    echo "FAIL: Decoded password is '$DECODED', expected 'SuperSecret123!'"
    exit 1
fi

echo "PASS: Secret password decoded correctly and saved to /tmp/decoded-password.txt"
exit 0

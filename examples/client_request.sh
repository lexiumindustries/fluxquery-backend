#!/bin/bash

# This script generates the signature and sends a secure request via cURL

SECRET="devsecret"
URL="http://localhost:8080/export"
PATH_URL="/export"
METHOD="POST"
BODY='{"query":"SELECT * FROM users LIMIT 10", "email":"admin@example.com", "format":"json"}'

# 1. Generate Timestamp
TIMESTAMP=$(date +%s)

# 2. Construct Payload: Method + Path + Body + Timestamp
PAYLOAD="${METHOD}${PATH_URL}${BODY}${TIMESTAMP}"

# 3. Generate HMAC-SHA256 Signature
# Note: uses openssl to generate binary hmac, then xxd to convert to hex
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

# 4. Send Request
echo "Sending signed request..."
echo "Timestamp: $TIMESTAMP"
echo "Signature: $SIGNATURE"

curl -X $METHOD "$URL" \
     -H "Content-Type: application/json" \
     -H "X-Timestamp: $TIMESTAMP" \
     -H "X-Signature: $SIGNATURE" \
     -d "$BODY"

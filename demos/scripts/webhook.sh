#!/usr/bin/env bash
# Demo script: webhook ingestion + signature verification
BASE=http://localhost:8080
SECRET=dev-mock-secret

# Create a charge to get its provider reference
REF=$(curl -s -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-wh-001' \
  -d '{"amount":424242,"currency":"usd"}' | jq -r '.provider_ref')

BODY='{"event_type":"charge.succeeded","charge_id":"'"$REF"'","status":"confirmed"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')

echo "==> Valid signed webhook -> 200 OK"
curl -s -o /dev/null -w "HTTP %{http_code}" -X POST "$BASE/v1/webhooks/mock" \
  -H 'Content-Type: application/json' \
  -H "X-Webhook-Signature: $SIG" \
  -d "$BODY"
echo
echo

echo "==> Tampered signature -> 400 rejected"
curl -s -X POST "$BASE/v1/webhooks/mock" \
  -H 'Content-Type: application/json' \
  -H 'X-Webhook-Signature: deadbeef' \
  -d "$BODY" | jq
echo

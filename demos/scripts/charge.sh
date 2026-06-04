#!/usr/bin/env bash
# Demo script: charge flow
BASE=http://localhost:8080

echo "==> Health check"
curl -s "$BASE/health" | jq
echo

echo "==> Create charge (success: 424242 cents)"
curl -s -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-key-001' \
  -d '{"amount":424242,"currency":"usd","metadata":{"order_id":"ord_42"}}' | jq
echo

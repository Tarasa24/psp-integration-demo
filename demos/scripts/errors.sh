#!/usr/bin/env bash
# Demo script: error cases
BASE=http://localhost:8080

echo "==> Declined card (400002) -> 402 Payment Required"
curl -s -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-err-001' \
  -d '{"amount":400002,"currency":"usd"}' | jq
echo

echo "==> Not found -> 404"
curl -s "$BASE/v1/charges/00000000-0000-0000-0000-000000000000" | jq
echo

echo "==> 3DS required charge -> requires_action"
curl -s -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-3ds-001' \
  -d '{"amount":300042,"currency":"usd"}' | jq '{status,three_ds_status}'
echo

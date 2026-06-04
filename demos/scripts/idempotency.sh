#!/usr/bin/env bash
# Demo script: idempotency replay
BASE=http://localhost:8080

echo "==> First request (creates charge)"
curl -si -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-key-001' \
  -d '{"amount":424242,"currency":"usd"}' \
  | grep -E 'HTTP/|X-Idempotent|"id"'
echo

echo "==> Replay (same key, different body) -> cached"
curl -si -X POST "$BASE/v1/charges" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-key-001' \
  -d '{"amount":999,"currency":"eur"}' \
  | grep -E 'HTTP/|X-Idempotent|"id"'
echo

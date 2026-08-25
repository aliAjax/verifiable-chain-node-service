#!/usr/bin/env bash
set -euo pipefail
base="${BASE_URL:-http://127.0.0.1:8080}"
curl -fsS "$base/healthz" >/dev/null
curl -fsS "$base/api/v1/networks" >/dev/null
curl -fsS -X POST "$base/api/v1/transactions" -H 'content-type: application/json' -d '{"from":"alice","to":"state","nonce":1,"fee":2,"gas":1000,"key":"demo","value":"ok"}' >/dev/null
curl -fsS "$base/api/v1/mempool" >/dev/null
curl -fsS -X POST "$base/api/v1/blocks/mine" >/dev/null
curl -fsS "$base/api/v1/blocks/1" >/dev/null
curl -fsS -X POST "$base/api/v1/snapshots" >/dev/null
curl -fsS "$base/api/v1/proofs/demo" >/dev/null
echo smoke-ok

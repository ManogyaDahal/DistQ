#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  if [[ -n "${API_PID:-}" ]] && kill -0 "$API_PID" 2>/dev/null; then kill "$API_PID" 2>/dev/null || true; fi
  if [[ -n "${BROKER_PID:-}" ]] && kill -0 "$BROKER_PID" 2>/dev/null; then kill "$BROKER_PID" 2>/dev/null || true; fi
  if [[ -n "${WORKER_PID:-}" ]] && kill -0 "$WORKER_PID" 2>/dev/null; then kill "$WORKER_PID" 2>/dev/null || true; fi
  if [[ -n "${SDK_DEMO_PID:-}" ]] && kill -0 "$SDK_DEMO_PID" 2>/dev/null; then kill "$SDK_DEMO_PID" 2>/dev/null || true; fi
}

trap cleanup EXIT INT TERM

echo "[1/4] Starting Redis..."
if (echo > /dev/tcp/127.0.0.1/6379) >/dev/null 2>&1; then
  echo "Redis already appears to be running on 127.0.0.1:6379; reusing it."
else
  docker compose -f "$ROOT_DIR/docker-compose.yml" up -d redis
fi

echo "[2/4] Building dashboard..."
pushd "$ROOT_DIR/dashboard" >/dev/null
if command -v pnpm >/dev/null 2>&1; then
  pnpm install
  pnpm run build
elif command -v npm >/dev/null 2>&1; then
  npm install
  npm run build
else
  echo "pnpm or npm is required to build the dashboard" >&2
  exit 1
fi
popd >/dev/null

echo "[3/4] Starting Go services..."
cd "$ROOT_DIR"
go run ./cmd/api > /tmp/distq-api.log 2>&1 &
API_PID=$!
go run ./cmd/broker > /tmp/distq-broker.log 2>&1 &
BROKER_PID=$!
go run ./cmd/worker > /tmp/distq-worker.log 2>&1 &
WORKER_PID=$!

echo "[4/4] Starting SDK demo in background..."
go run ./cmd/sdk-demo > /tmp/distq-sdk-demo.log 2>&1 &
SDK_DEMO_PID=$!

echo
echo "DistQ is starting. Logs:"
echo "  API:      /tmp/distq-api.log"
echo "  Broker:   /tmp/distq-broker.log"
echo "  Worker:   /tmp/distq-worker.log"
echo "  SDK demo: /tmp/distq-sdk-demo.log"
echo
echo "Dashboard: http://localhost:8080"
echo "SDK demo CLI is running in the background; open another terminal and run './distq-sdk-demo' to interact with it directly."

wait "$API_PID" "$BROKER_PID" "$WORKER_PID" "$SDK_DEMO_PID"
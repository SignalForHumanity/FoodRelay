#!/usr/bin/env bash
# Run registry and node locally for development.
# Requires Go 1.22+. Run from repo root.
set -e

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example — edit it before running in production."
fi

# Load env
set -a; source .env; set +a

echo "Starting registry on :8081 ..."
go run ./cmd/registry &
REGISTRY_PID=$!

sleep 1

echo "Starting node on :8080 ..."
go run ./cmd/node &
NODE_PID=$!

cleanup() {
  echo "Stopping..."
  kill $REGISTRY_PID $NODE_PID 2>/dev/null
}
trap cleanup EXIT INT TERM

wait

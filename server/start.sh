#!/bin/bash
cd "$(dirname "$0")"

# Load environment variables from .env
export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)

# Always rebuild so restarting the service cannot run a stale binary.
go build -o ./bin/api ./cmd/api || exit 1

# Start the API server
exec ./bin/api

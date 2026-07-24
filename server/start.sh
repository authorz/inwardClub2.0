#!/bin/bash
cd "$(dirname "$0")"

# Load environment variables from .env
export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)

# Start the API server
./bin/api

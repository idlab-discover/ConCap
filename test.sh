#!/bin/bash

set -e

echo "Running tests..."
go test -v ./internal/...
echo "Tests completed successfully." 
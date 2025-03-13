#!/bin/bash

set -e

echo "Building Concap..."
cd cmd/concap
go build -o ../../concap
cd ../..
echo "Build completed successfully. Binary is at ./concap" 
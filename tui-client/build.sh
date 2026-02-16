#!/bin/bash
set -e

echo "Building lumi TUI client..."
go build -o lumi main.go
echo "✓ Built: ./lumi"

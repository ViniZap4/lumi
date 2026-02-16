#!/bin/bash
set -e

echo "Building lumi server..."
go build -o lumi-server main.go
echo "✓ Built: ./lumi-server"

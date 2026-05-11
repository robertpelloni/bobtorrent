#!/bin/bash
set -e

echo "Building bobtorrent (Omni-Workspace)..."

VERSION=$(cat VERSION)

echo "Building Go Port..."
mkdir -p build

cd bobtorrent
go build -ldflags="-X main.Version=$VERSION" -buildvcs=false -o ../build/bobtorrent cmd/bobtorrent/main.go

echo "Build complete."

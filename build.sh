#!/bin/bash
set -e

echo "Building bobtorrent (Omni-Workspace)..."

VERSION=$(cat VERSION)

echo "Building Go Port..."
mkdir -p build

go build -ldflags="-X main.Version=$VERSION" -buildvcs=false -o build/supernode-go ./cmd/supernode-go/
go build -buildvcs=false -o build/lattice-go ./cmd/lattice-go/
go build -buildvcs=false -o build/dht-proxy ./cmd/dht-proxy/

echo "Building WASM Storage Bridge..."
GOOS=js GOARCH=wasm go build -buildvcs=false -o build/storage.wasm ./cmd/wasm/

echo "Build complete."

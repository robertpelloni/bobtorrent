@echo off
echo Building bobtorrent (Omni-Workspace)...
npm install
echo Building Go Port...
mkdir build 2>nul
go build -o build/dht-proxy cmd/dht-proxy/main.go
go build -o build/supernode-go cmd/supernode-go/main.go
echo Build complete.
pause

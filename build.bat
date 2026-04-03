@echo off
echo Building bobtorrent (Omni-Workspace)...
npm install
echo Building Go Port (DHT Proxy)...
mkdir build 2>nul
go build -o build/dht-proxy cmd/dht-proxy/main.go
echo Build complete.
pause

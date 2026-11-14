@echo off
setlocal
REM BobTorrent Omni-Workspace build (Windows).
REM Pure-Go build: CGO_ENABLED=0 (modernc sqlite + pure-Go stack; the local
REM mingw gcc emits objects Go cannot parse). Packages are built by path —
REM never `go build cmd\supernode-go\main.go`, which misses sibling files.
echo Building bobtorrent (Omni-Workspace)...
set CGO_ENABLED=0
mkdir build 2>nul
set /p VERSION=<VERSION

go build -buildvcs=false -o build\dht-proxy.exe ./cmd/dht-proxy/
if errorlevel 1 exit /b 1
go build -ldflags="-X main.Version=%VERSION%" -buildvcs=false -o build\supernode-go.exe ./cmd/supernode-go/
if errorlevel 1 exit /b 1
go build -buildvcs=false -o build\lattice-go.exe ./cmd/lattice-go/
if errorlevel 1 exit /b 1

echo Building WASM Storage Bridge...
set GOOS=js
set GOARCH=wasm
go build -buildvcs=false -o build\storage.wasm ./cmd/wasm/
if errorlevel 1 exit /b 1
set GOOS=
set GOARCH=

go env GOROOT > "%TEMP%\goroot.txt"
set /p GOROOT=<"%TEMP%\goroot.txt"
if exist "%GOROOT%\lib\wasm\wasm_exec.js" copy /Y "%GOROOT%\lib\wasm\wasm_exec.js" build\wasm_exec.js >nul

echo Build complete: build\supernode-go.exe build\lattice-go.exe build\dht-proxy.exe build\storage.wasm
endlocal

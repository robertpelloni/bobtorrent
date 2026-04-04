@echo off
setlocal

echo Building bobtorrent (Omni-Workspace)...
npm install

echo Building Go Port...
mkdir build 2>nul

go build -buildvcs=false -o build/dht-proxy cmd/dht-proxy/main.go
if errorlevel 1 exit /b 1

go build -buildvcs=false -o build/supernode-go cmd/supernode-go/main.go
if errorlevel 1 exit /b 1

go build -buildvcs=false -o build/lattice-go cmd/lattice-go/main.go
if errorlevel 1 exit /b 1

echo Building WASM Storage Bridge...
set GOOS=js
set GOARCH=wasm
go build -buildvcs=false -o build/storage.wasm cmd/wasm/main.go
if errorlevel 1 exit /b 1
set GOOS=
set GOARCH=

if exist "%GOROOT%\lib\wasm\wasm_exec.js" (
    copy /Y "%GOROOT%\lib\wasm\wasm_exec.js" build\wasm_exec.js >nul
) else if exist "%ProgramFiles%\Go\lib\wasm\wasm_exec.js" (
    copy /Y "%ProgramFiles%\Go\lib\wasm\wasm_exec.js" build\wasm_exec.js >nul
)

echo Build complete.
pause

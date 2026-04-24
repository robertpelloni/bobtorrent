@echo off
setlocal
title BobTorrent
cd /d "%~dp0"

echo [BobTorrent] Starting...
where go >nul 2>nul
if errorlevel 1 (
    echo [BobTorrent] go not found. Please install it.
    pause
    exit /b 1
)

go run .

if errorlevel 1 (
    echo [BobTorrent] Exited with error code %errorlevel%.
    pause
)
endlocal

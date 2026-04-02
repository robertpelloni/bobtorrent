#!/bin/bash
set -e

# Ensure we are in the right directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$SCRIPT_DIR/.."
QBT_DIR="$ROOT_DIR/qbittorrent"

echo "Applying Megatorrent Reference Implementation to qBittorrent submodule..."

if [ ! -d "$QBT_DIR" ]; then
    echo "Error: qbittorrent directory not found at $QBT_DIR"
    exit 1
fi

# Copy API files
cp "$SCRIPT_DIR/megatorrent-webui/api/megatorrentcontroller.h" "$QBT_DIR/src/webui/api/"
cp "$SCRIPT_DIR/megatorrent-webui/api/megatorrentcontroller.cpp" "$QBT_DIR/src/webui/api/"

# Copy WebUI Frontend
cp "$SCRIPT_DIR/megatorrent-webui/www/private/megatorrent.html" "$QBT_DIR/src/webui/www/private/"
cp "$SCRIPT_DIR/megatorrent-webui/www/private/scripts/megatorrent.js" "$QBT_DIR/src/webui/www/private/scripts/"

# Apply Patches (Overwrite files)
cp "$SCRIPT_DIR/megatorrent-webui/index.html.modified" "$QBT_DIR/src/webui/www/private/index.html"
cp "$SCRIPT_DIR/megatorrent-webui/client.js.modified" "$QBT_DIR/src/webui/www/private/scripts/client.js"
cp "$SCRIPT_DIR/megatorrent-webui/CMakeLists.txt.modified" "$QBT_DIR/src/webui/CMakeLists.txt"
cp "$SCRIPT_DIR/megatorrent-webui/webapplication.cpp.modified" "$QBT_DIR/src/webui/webapplication.cpp"

echo "Success! The qbittorrent submodule is now patched with Megatorrent features."
echo "WARNING: The submodule is now in a DIRTY state. Do not commit the submodule reference update."

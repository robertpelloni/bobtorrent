# Megatorrent C++ Reference Integration

This directory contains the canonical source code for the Megatorrent integration into qBittorrent.
Because the `qbittorrent/` directory is a git submodule, we cannot directly commit changes to it without breaking the submodule reference for other users.

Instead, we maintain the reference implementation files here. Developers wishing to work on the C++ integration should apply these files to the submodule.

## Structure

*   `megatorrent-webui/`: Contains the WebUI extensions.
    *   `api/`: C++ Controller classes (`MegatorrentController`).
    *   `www/private/`: HTML/JS frontend files.
    *   `CMakeLists.txt.modified`: Patched CMake build file.
    *   `webapplication.cpp.modified`: Patched application router.
    *   `client.js.modified`: Patched WebUI frontend logic.
    *   `index.html.modified`: Patched WebUI entry point.

## Applying the Reference Implementation

To apply these changes to the `qbittorrent/` submodule for development or testing:

1.  Run the installation script:
    ```bash
    ./install_webui_patches.sh
    ```

2.  Build qBittorrent normally (see `qbittorrent/README.md`).

## Reverting

To revert the submodule to its clean state (e.g. before committing the main repo):

```bash
cd ../qbittorrent
git checkout .
git clean -fd
```

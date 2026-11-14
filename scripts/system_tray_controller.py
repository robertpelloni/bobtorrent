#!/usr/bin/env python3
"""
scripts/system_tray_controller.py
==================================
Universal BobTorrent Sovereign Daemon & System Tray Lifecycle Controller.

Cross-platform (Windows / macOS / Linux):
- Monitors BobTorrent Go Supernode (:8000) and Go-Lattice daemon (:4000).
- Start / Stop / Restart / Status / Open Dashboard / Quit (Quit also stops
  the services it manages).
- GUI system tray via pystray+Pillow when a desktop session exists;
  interactive headless CLI fallback otherwise.

Windows notes: process control uses `taskkill` / PowerShell CIM queries
(never `lsof`), binaries resolve with the `.exe` suffix, and the build
step runs `build.bat`. All spawned children are created below-normal
priority on Windows so heavyweight nodes never hang the desktop.
"""

import os
import sys
import time
import socket
import subprocess
import threading
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
IS_WINDOWS = sys.platform.startswith("win")
EXE = ".exe" if IS_WINDOWS else ""

_SUPERNODE_PORT = int(os.environ.get("BOBTORRENT_PORT", "8000"))
SERVICES = {
    "supernode-go": {"port": _SUPERNODE_PORT, "bin": f"build/supernode-go{EXE}",
                     "args": ["-headless", "-port", str(_SUPERNODE_PORT)]},
    "lattice-go": {"port": 4000, "bin": f"build/lattice-go{EXE}", "args": []},
}

_children = {}  # name -> subprocess.Popen we spawned (only these are ours to stop)


def check_port(port: int, host: str = "127.0.0.1", timeout: float = 0.5) -> bool:
    for h in [host, "::1", "localhost"]:
        try:
            with socket.create_connection((h, port), timeout=timeout):
                return True
        except OSError:
            continue
    return False


def get_status():
    return {name: check_port(cfg["port"]) for name, cfg in SERVICES.items()}


def _creationflags_below_normal():
    # CREATE_NEW_PROCESS_GROUP | BELOW_NORMAL_PRIORITY_CLASS (Windows only)
    return 0x00000200 | 0x00004000 if IS_WINDOWS else 0


def _build_missing():
    print("[SystemTray] Building missing Go binaries...")
    if IS_WINDOWS:
        subprocess.run(["cmd", "/c", "build.bat"], cwd=str(REPO_ROOT),
                       creationflags=_creationflags_below_normal())
    else:
        subprocess.run(["bash", str(REPO_ROOT / "build.sh")], cwd=str(REPO_ROOT))


def start_services():
    if any(not (REPO_ROOT / cfg["bin"]).exists() for cfg in SERVICES.values()):
        _build_missing()
    for name, cfg in SERVICES.items():
        if check_port(cfg["port"]):
            print(f"[SystemTray] {name} already online on port {cfg['port']}.")
            continue
        binary = REPO_ROOT / cfg["bin"]
        if not binary.exists():
            print(f"[SystemTray] ERROR: {binary} missing and build did not produce it.")
            continue
        print(f"[SystemTray] Spawning {name} on port {cfg['port']} (below-normal priority)...")
        _children[name] = subprocess.Popen(
            [str(binary), *cfg["args"]], cwd=str(REPO_ROOT),
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            creationflags=_creationflags_below_normal())
    return True


def _pids_on_port(port: int):
    """PIDs listening on a TCP port, cross-platform, stdlib+OS tools only."""
    pids = set()
    try:
        if IS_WINDOWS:
            out = subprocess.check_output(
                ["powershell", "-NoProfile", "-Command",
                 f"(Get-NetTCPConnection -LocalPort {port} -State Listen "
                 f"-ErrorAction SilentlyContinue).OwningProcess"],
                stderr=subprocess.DEVNULL, timeout=10).decode()
            pids = {int(p) for p in out.split() if p.strip().isdigit()}
        else:
            out = subprocess.check_output(["lsof", "-t", f"-i:{port}"],
                                          stderr=subprocess.DEVNULL, timeout=10).decode()
            pids = {int(p) for p in out.split() if p.isdigit()}
    except Exception:
        pass
    return pids


def stop_services():
    print("[SystemTray] Stopping BobTorrent services...")
    # 1) Children we spawned ourselves — always safe to terminate.
    for name, proc in list(_children.items()):
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
            print(f"[SystemTray] Stopped {name} (PID {proc.pid})")
        _children.pop(name, None)
    # 2) Anything else listening on our ports: only stop it if its image
    #    name matches our binaries, so unrelated apps are never killed.
    for name, cfg in SERVICES.items():
        for pid in _pids_on_port(cfg["port"]):
            if any(p.pid == pid for p in _children.values()):
                continue
            try:
                if IS_WINDOWS:
                    img = subprocess.check_output(
                        ["powershell", "-NoProfile", "-Command",
                         f"(Get-Process -Id {pid} -ErrorAction SilentlyContinue).ProcessName"],
                        stderr=subprocess.DEVNULL, timeout=10).decode().strip().lower()
                    if name not in img:
                        print(f"[SystemTray] Port {cfg['port']} held by unrelated '{img}' (PID {pid}) — left running.")
                        continue
                    subprocess.run(["taskkill", "/PID", str(pid), "/T"], timeout=10)
                else:
                    os.kill(pid, 15)
                print(f"[SystemTray] Stopped {name} (PID {pid})")
            except Exception:
                pass
    print("[SystemTray] Stop complete.")


def restart_services():
    stop_services()
    time.sleep(1.5)
    start_services()


def open_dashboard():
    url = f"http://localhost:{_SUPERNODE_PORT}"
    print(f"[SystemTray] Opening dashboard at {url}")
    try:
        import webbrowser
        webbrowser.open(url)
    except Exception as e:
        print(f"[SystemTray] Unable to launch browser: {e}")


def create_tray_icon():
    try:
        from PIL import Image, ImageDraw
        img = Image.new("RGBA", (64, 64), color=(0, 0, 0, 0))
        d = ImageDraw.Draw(img)
        d.polygon([(32, 4), (60, 32), (32, 60), (4, 32)], outline=(88, 166, 255, 255), width=3)
        d.polygon([(32, 16), (48, 32), (32, 48), (16, 32)], fill=(46, 160, 67, 230))
        return img
    except Exception:
        return None


def _status_title():
    s = get_status()
    return "BobTorrent: " + ", ".join(
        f"{n} {'ON' if on else 'OFF'}" for n, on in s.items())


def run_gui_tray():
    import pystray
    from pystray import MenuItem as item

    icon_img = create_tray_icon()
    if not icon_img:
        raise RuntimeError("Icon creation unavailable (Pillow missing)")

    def refresh(icon, _item):
        icon.title = _status_title()
        icon.update_menu()

    menu = pystray.Menu(
        item(lambda _i: _status_title(), lambda icon, item: None, enabled=False),
        pystray.Menu.SEPARATOR,
        item("Open Command Center (Web UI)", lambda icon, item: open_dashboard()),
        item("Start All Services", lambda icon, item: threading.Thread(target=start_services, daemon=True).start()),
        item("Restart All Services", lambda icon, item: threading.Thread(target=restart_services, daemon=True).start()),
        item("Stop All Services", lambda icon, item: threading.Thread(target=stop_services, daemon=True).start()),
        item("Refresh Status", refresh),
        pystray.Menu.SEPARATOR,
        item("Quit (stops services)", lambda icon, item: (stop_services(), icon.stop())),
    )
    tray = pystray.Icon("bobtorrent_sovereign", icon_img, _status_title(), menu)
    print("[SystemTray] Graphical system tray initialized. Right-click the tray icon for controls; Quit stops the servers.")
    tray.run()


def run_cli_daemon():
    print("=" * 60)
    print("  BOBTORRENT SOVEREIGN NODE LIFECYCLE CONTROLLER (HEADLESS)")
    print("=" * 60)
    print("Commands: [s]tatus  [u]p/start  [r]estart  [k]ill/stop  [o]pen UI  [q]uit (stops services)")
    for svc, online in get_status().items():
        print(f"  * {svc:16}: {'ONLINE' if online else 'OFFLINE'} (Port {SERVICES[svc]['port']})")
    print("=" * 60)
    if not sys.stdin.isatty():
        print("[SystemTray] Non-interactive mode: status printed above, exiting. "
              "Install pystray+Pillow for the GUI tray, or run with a terminal for the CLI.")
        return
    while True:
        try:
            line = input("bobtorrent-ctl> ").strip().lower()
            if not line:
                continue
            if line == 'q':
                stop_services()
                break
            elif line == 's':
                for svc, online in get_status().items():
                    print(f"  * {svc:16}: {'ONLINE' if online else 'OFFLINE'} (Port {SERVICES[svc]['port']})")
            elif line == 'u':
                start_services()
            elif line == 'k':
                stop_services()
            elif line == 'r':
                restart_services()
            elif line == 'o':
                open_dashboard()
            else:
                print("Unknown command. Valid options: s, u, r, k, o, q")
        except (KeyboardInterrupt, EOFError):
            print("\nExiting controller.")
            break


def _has_desktop_session() -> bool:
    if IS_WINDOWS or sys.platform == "darwin":
        return True  # Explorer / Aqua session assumed when launched interactively
    return bool(os.environ.get("DISPLAY") or os.environ.get("WAYLAND_DISPLAY"))


def main():
    if _has_desktop_session():
        try:
            run_gui_tray()
            return
        except Exception as e:
            print(f"[SystemTray] GUI tray unavailable ({e}), falling back to CLI daemon. "
                  f"Tip: pip install pystray Pillow")
    run_cli_daemon()


if __name__ == "__main__":
    main()

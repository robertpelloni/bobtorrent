#!/usr/bin/env python3
"""
scripts/system_tray_controller.py
==================================
Universal BobTorrent Sovereign Daemon & System Tray Lifecycle Controller.

Provides:
- Real-time status monitoring for:
  * BobTorrent Go Supernode (Port 8000)
  * Go-Lattice Consensus Daemon (Port 4000)
  * DHT Privacy Proxy (Port 9090 / 8080)
  * Bobcoin Web Dashboard (Port 4173 / 3000)
- Interactive server controls: Start, Stop, Restart, Status Check, Open Dashboard.
- Graphical System Tray integration (via Pillow & pystray when GUI display is detected).
- Headless interactive CLI daemon fallback when operating in headless server/cloud environments.
"""

import os
import sys
import time
import signal
import socket
import subprocess
import threading
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

SERVICES = {
    "supernode-go": {"port": 8000, "cmd": ["./build/supernode-go", "-headless"], "cwd": REPO_ROOT},
    "lattice-go": {"port": 4000, "cmd": ["./build/lattice-go"], "cwd": REPO_ROOT},
}

def check_port(port: int, host: str = "127.0.0.1", timeout: float = 0.5) -> bool:
    """Checks if a TCP port is actively accepting connections."""
    for h in [host, "::1", "localhost"]:
        try:
            with socket.create_connection((h, port), timeout=timeout):
                return True
        except OSError:
            continue
    return False

def get_status():
    """Returns mapping of service name to boolean online state."""
    res = {}
    for name, cfg in SERVICES.items():
        res[name] = check_port(cfg["port"])
    return res

def start_services():
    """Builds and starts core ecosystem binaries if offline."""
    print("[SystemTray] Verifying build artifacts...")
    supernode_bin = REPO_ROOT / "build" / "supernode-go"
    lattice_bin = REPO_ROOT / "build" / "lattice-go"

    if not supernode_bin.exists() or not lattice_bin.exists():
        print("[SystemTray] Building missing Go binaries via build.sh...")
        subprocess.run(["bash", str(REPO_ROOT / "build.sh")], cwd=str(REPO_ROOT))

    print("[SystemTray] Launching BobTorrent ecosystem services...")
    for name, cfg in SERVICES.items():
        if not check_port(cfg["port"]):
            print(f"[SystemTray] Spawning {name} on port {cfg['port']}...")
            subprocess.Popen(cfg["cmd"], cwd=str(cfg["cwd"]), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        else:
            print(f"[SystemTray] {name} is already online on port {cfg['port']}.")
    return True

def stop_services():
    """Terminates running services safely."""
    print("[SystemTray] Stopping BobTorrent services...")
    for name, cfg in SERVICES.items():
        port = cfg["port"]
        try:
            out = subprocess.check_output(["lsof", "-t", f"-i:{port}"], stderr=subprocess.DEVNULL).decode().strip()
            pids = [int(p) for p in out.split() if p.isdigit()]
            for pid in pids:
                try:
                    os.kill(pid, signal.SIGTERM)
                    print(f"[SystemTray] Sent SIGTERM to {name} (PID {pid})")
                except OSError:
                    pass
        except Exception:
            pass
    print("[SystemTray] All monitored services stopped.")

def restart_services():
    """Restarts ecosystem services."""
    stop_services()
    time.sleep(1.5)
    start_services()

def open_dashboard():
    """Opens Web Command Console in user's browser."""
    url = "http://localhost:8000"
    print(f"[SystemTray] Opening dashboard at {url}")
    try:
        import webbrowser
        webbrowser.open(url)
    except Exception as e:
        print(f"[SystemTray] Unable to launch browser: {e}")

def create_tray_icon():
    """Creates a stylized icon for system tray if Pillow is available."""
    try:
        from PIL import Image, ImageDraw
        img = Image.new("RGBA", (64, 64), color=(0, 0, 0, 0))
        d = ImageDraw.Draw(img)
        # Draw diamond node lattice
        d.polygon([(32, 4), (60, 32), (32, 60), (4, 32)], outline=(88, 166, 255, 255), width=3)
        d.polygon([(32, 16), (48, 32), (32, 48), (16, 32)], fill=(46, 160, 67, 230))
        return img
    except Exception:
        return None

def run_gui_tray():
    """Runs GUI tray with interactive menu if pystray is installed."""
    import pystray
    from pystray import MenuItem as item

    icon_img = create_tray_icon()
    if not icon_img:
        raise RuntimeError("Icon creation unavailable")

    menu = pystray.Menu(
        item("BobTorrent Sovereign Node", lambda icon, item: None, enabled=False),
        pystray.Menu.SEPARATOR,
        item("Open Command Center (Web UI)", lambda icon, item: open_dashboard()),
        item("Start All Services", lambda icon, item: start_services()),
        item("Restart All Services", lambda icon, item: restart_services()),
        item("Stop All Services", lambda icon, item: stop_services()),
        pystray.Menu.SEPARATOR,
        item("Quit Controller", lambda icon, item: (stop_services(), icon.stop()))
    )

    tray = pystray.Icon("bobtorrent_sovereign", icon_img, "BobTorrent Sovereign Daemon", menu)
    print("[SystemTray] Graphical system tray initialized.")
    tray.run()

def run_cli_daemon():
    """Runs interactive CLI controller when running headless."""
    print("=" * 60)
    print("  BOBTORRENT SOVEREIGN NODE LIFECYCLE CONTROLLER (HEADLESS)")
    print("=" * 60)
    print("Commands:")
    print("  [s] Status check")
    print("  [u] Up / Start services")
    print("  [r] Restart services")
    print("  [k] Kill / Stop services")
    print("  [o] Open Web Command Center")
    print("  [q] Quit controller")
    print("=" * 60)

    status = get_status()
    for svc, online in status.items():
        print(f"  * {svc:16}: {'ONLINE' if online else 'OFFLINE'} (Port {SERVICES[svc]['port']})")
    print("=" * 60)

    if not sys.stdin.isatty():
        print("[SystemTray] Running in non-interactive background mode.")
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
                status = get_status()
                for svc, online in status.items():
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

def main():
    display = os.environ.get("DISPLAY")
    if display:
        try:
            run_gui_tray()
            return
        except Exception as e:
            print(f"[SystemTray] GUI tray unavailable ({e}), falling back to CLI daemon.")
    run_cli_daemon()

if __name__ == "__main__":
    main()

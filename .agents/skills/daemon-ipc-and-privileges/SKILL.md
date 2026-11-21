---
name: daemon-ipc-and-privileges
description: Implement privileged daemon communication, zero-friction Windows/Linux elevation, BOM-free nftables/hosts generation, and Android VpnService lifecycle.
---

# Daemon, IPC & Zero-Friction Privileges Playbook

Use this skill when working on `core/cmd/netportal-daemon/`, `core/api/`, `desktop/`, or `android/`.

## 1. Two-Process Desktop Security Model

1. **`netportal-daemon` (Privileged Background Process)**:
   - Owns TUN creation, `amneziawg-go/v3`, `sing-box`, `winws`/`nfqws`, `nftables`, and `/etc/hosts` mutations.
   - Exposes a local IPC transport:
     - **Windows**: Named Pipe (`\\.\pipe\netportal-daemon`) with DACL restricting access to SYSTEM, Administrators, and the authenticated user SID.
     - **Linux / macOS**: Unix Domain Socket (`/run/netportal/daemon.sock` for system service or `$XDG_RUNTIME_DIR/netportal-daemon.sock` for session mode) validated via `SO_PEERCRED` (UID check).
   - Optional loopback HTTP+WS bridge for `--web` mode MUST bind exclusively to `127.0.0.1:<ephemeral-or-configured-port>` and require a cryptographic per-session Bearer token.
2. **`desktop` UI (Unprivileged User Process)**:
   - Runs as normal desktop user so WebView2 (Windows) and WebKit2GTK 4.1 (Linux Wayland/X11) operate reliably without root display restrictions or Win32 `DropAllPrivileges` hacks.

## 2. Zero-Friction Elevation Per Platform

- **Windows**:
  - **Installed (NSIS)**: Installs `NetPortalService` (Windows Service) once during setup. Day-to-day UI launches require ZERO UAC prompts.
  - **Portable (`.exe`)**: Starts unprivileged immediately. Pre-connect probing, subscriptions, WARP config generation, `ByeDPI`, `Psiphon`, and `sing-box` system proxy run with ZERO UAC. Only enabling `TUN`, `Zapret` (`WinDivert`), or `Hosts` triggers a single UAC prompt to start a persistent privileged `netportal-daemon` session for the app lifetime.
- **Linux (Wayland / X11)**:
  - **`.deb` Package**: `DEBIAN/postinst` registers `netportal-daemon.service` (systemd) + Polkit policy, installs runtime tools (`nfqws`, `ciadpi`) under `/usr/lib/netportal/` with `0755`. Day-to-day launches require ZERO password prompts.
  - **Portable (`.tar.gz`)**: On startup, embedded helper assets are extracted to `~/.local/share/netportal/bin/` with `os.Chmod(..., 0755)` executed automatically by the binary itself (users MUST NOT run `chmod`/`setcap` manually). Privileged operations spawn one persistent `pkexec netportal-daemon session` process (single password prompt per session) communicating over stdin/stdout or UDS, cleanly torn down on `QUIT` or parent exit.
- **Android**:
  - Rootless `VpnService`. Single standard Android OS VPN consent dialog.
  - Native libraries (`libciadpi.so`, `netportal-core.aar` JNI libs) packaged in APK `jniLibs/` so Android extracts them with executable permissions automatically.

## 3. Linux `nftables` & System File Mandates

- Write all Linux files (`rules.nft`, systemd units, `/etc/hosts`) in strict **UTF-8 without BOM** (`0xEF 0xBB 0xBF` causes `nft: Error: syntax error, unexpected junk table`).
- Start every dynamic `nftables` ruleset with idempotent cleanup:
  ```nft
  add table inet netportal
  flush table inet netportal
  table inet netportal {
      ...
  }
  ```
- Always restore `/etc/hosts` markers and flush `nftables` table `inet netportal` in `defer` / signal handlers (`SIGTERM`, `SIGINT`, daemon crash recovery).

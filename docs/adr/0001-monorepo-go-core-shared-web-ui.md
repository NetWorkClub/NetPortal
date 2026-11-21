# ADR-0001: Monorepo with Unified Go Core and Shared Responsive Web UI

- **Status**: Accepted
- **Date**: 2026-11-22

## Context
NetPortal must run natively on Windows, Linux (Wayland / X11), and Android (rootless), with clear expansion paths to macOS and iOS, while keeping high development velocity across UI, protocol engines, and routing logic.

Evaluated alternatives:
1. **Separate repositories per component** (core, web UI, desktop, and mobile split across multiple repos).
2. **Flutter + Go core FFI** (similar to `hiddify-app`).
3. **Tauri v2 (Rust + Wry) everywhere + Go core**.
4. **Single monorepo (`NetPortal`) with Go core (`core/`), modular Angular 21 Web UI (`ui/`), thin Wails v2 shell (`desktop/`), and thin Kotlin shell (`android/`)**.

## Decision
Adopt **Option 4**: a single monorepo containing:
- `core/`: Go 1.25 networking library, parsers, prober, routing constructor compiler, WARP generator, privileged desktop daemon, and `gomobile` facade (`core/mobile`).
- `ui/`: Modular Angular 21 (Standalone + Signals + Zoneless) + TypeScript + Modular SCSS frontend (`src/app/{core,layout,shared,pages}`) talking to backends through an abstract `Transport` interface.
- `desktop/`: Thin Go + Wails v2 shell (Windows, Linux Wayland with WebKit2GTK 4.1, macOS) plus `--web` loopback fallback.
- `android/`: Thin rootless Kotlin shell (`VpnService`, `WebViewAssetLoader`, `netportal-core.aar`, `libciadpi.so`).

## Consequences
- **Positive**: One backend language (Go) matching all target network libraries (`amneziawg-go/v3`, `sing-box`, `mihomo`, `psiphon-tunnel-core`). Zero Rust-JNI glue layer on Android. Atomic cross-layer API changes in one repo without multi-repo version drift. Single UI codebase across Desktop, Android, Android TV, and `--web` mode.
- **Negative**: Requires strict discipline keeping `core/` free of desktop/Wails imports and `core/mobile/` compliant with `gomobile` type restrictions; Desktop tray relies on `libayatana-appindicator3` on Linux GNOME Wayland.

# NetPortal AI Agent Guide

Authoritative engineering instructions for any AI coding agent (Antigravity, Claude Code, OpenAI Codex, Cursor, Qwen, DeepSeek) working on **NetPortal**.

---

## 1. Project Identity & Core Philosophy

**NetPortal** is an open-source, cross-platform **Universal Connectivity Aggregator & Visual Routing Constructor** targeting **Windows**, **Linux (Wayland / X11)**, and **Android (rootless)**, with architecture ready for **macOS** and **iOS**.

Every implementation decision must preserve five foundational product principles:
1. **Universal Aggregator**: Unify AmneziaWG (2.0 / 3.0 / 3.1), multi-protocol proxy cores (`sing-box`, optional `mihomo`/MASQUE), Cloudflare WARP (AWG & MASQUE), Psiphon (`psiphon-tunnel-core`), DPI bypass (`byedpi`/`ciadpi` + `zapret` `winws`/`nfqws`), and Smart Hosts / Smart DNS (`smartdns`) inside one coherent platform.
2. **Zero Hidden Default Split-Routing [CRITICAL]**: Never enforce opinionated or silent default routing splits (e.g., do not silently route AI domains through hosts/WARP or bypass domains without explicit user opt-in). Activating a single node, tunnel, or standalone mode performs strictly what that selection defines.
3. **Opt-in Visual Routing Constructor**: Power users compose arbitrary ordered rules matching domains, suffixes, keywords, regexes, external/geo lists, IP/CIDR/IP pools, ports/protocols, and desktop processes / installed applications (or Android package IDs).
4. **Cross-Protocol Sticky-Primary Fallback**: Users can pin a primary node for IP stability (e.g., AI workflows that break on IP hopping) with an ordered cross-protocol fallback chain (e.g., `VLESS DE -> VLESS NL -> Psiphon -> WARP AWG 3.1`) that fails over only on healthcheck failure.
5. **Pre-Connect Inventory & Probing (Happ-style UX)**: Subscriptions and configs are parsed immediately into local storage on import. All nodes are visible, selectable, and testable (fast TCP/TLS handshake + ephemeral no-TUN in-memory real-delay URL-test + anti-captcha check) BEFORE clicking Connect.

---

## 2. Monorepo Structure

```text
NetPortal/
├── AGENTS.md                 # Authoritative agent rules (this file)
├── GEMINI.md / CLAUDE.md / .cursorrules  # Harness bridges to AGENTS.md
├── .agents/skills/           # Domain-specific agent playbooks
├── go.work                   # Go workspace linking ./core and ./desktop
├── core/                     # Unified Go 1.25 multi-engine library & daemon
│   ├── api/                  # Shared typed RPC/IPC contracts & event payloads
│   ├── engine/               # Pluggable protocol engines + conflict guard
│   ├── mobile/               # gomobile-safe exported facade (.aar / .xcframework)
│   ├── parser/               # Config & subscription parsers (AWG 2/3/3.1, VLESS, Clash, MASQUE)
│   ├── probe/                # Pre-connect TCP/TLS + no-TUN URL-test + anti-captcha detector
│   ├── router/               # Routing constructor IR, validation, core config compiler
│   ├── storage/              # Local profile/node/rule persistence
│   ├── warp/                 # Built-in WARP (AWG & MASQUE) generator + endpoint scanner
│   └── cmd/netportal-daemon/ # Privileged desktop daemon (Win Service / systemd / pkexec)
├── ui/                       # Single responsive Web UI (Angular 21 Standalone + Signals + Zoneless + Modular SCSS)
├── desktop/                  # Thin Go + Wails v2 shell + tray + --web browser fallback
├── android/                  # Thin rootless Kotlin shell (VpnService + WebViewAssetLoader)
├── ios/                      # Reserved for NEPacketTunnelProvider + WKWebView wrapper
└── docs/                     # VISION.md, ARCHITECTURE.md, ROADMAP.md, DESIGN_SYSTEM.md, adr/
```

---

## 3. Strict Engineering Rules [MUST]

### 3.1 Style, Git & Security
- **Zero-Emoji Policy**: Strictly ZERO emojis anywhere in code, UI components, icons (use inline SVG), commit messages, release notes, or documentation.
- **Commit Convention**: Use English Conventional Commits (`feat(scope): ...`, `fix(scope): ...`, `docs(scope): ...`, `chore(scope): ...`).
- **Release Standard**: Bilingual `<details open><summary><b>English</b></summary>` and `<details><summary><b>Русский</b></summary>`, 1-line executive summary at top, no duplicate `<h1>` inside release body.
- **No Secrets or Artifacts**: Never commit `.env*`, test VPN keys/configs (`.conf`, `.key`, `.pem`), or build outputs (`dist/`, `*.exe`, `*.apk`, `*.aar`). Use `.local/` for manual local fixtures.
- **Autonomous Git**: Never run interactive credential prompts; do not push to remote unless explicitly asked by the user.

### 3.2 Architectural Boundaries
- **`core/` Isolation**: `core/` MUST NEVER import Wails, GTK, WebView, or OS-UI packages. All platform hooks (TUN fd handoff, elevation helper, DNS setter, process list) pass through clean Go interfaces.
- **`core/mobile/` Compatibility**: All exported symbols intended for `gomobile bind` MUST use only basic Go types (`string`, `[]byte`, `int`, `int64`, `bool`, `error`, and simple callback interfaces). Pass complex DTOs as JSON strings/bytes matching `core/api` schemas.
- **`ui/` Transport Abstraction**: `ui/` MUST NOT call `window.go.*` directly inside components. All backend calls go through a single `Transport` adapter supporting `wails` (Desktop), `android` (`JavascriptInterface`), and `http-ws` (`desktop --web` browser fallback).
- **Explicit Conflict Guard**: Never silently ignore or override active conflicting network layers (`TUN` vs `/etc/hosts`, `WinDivert`/`NFQUEUE` vs `TUN`, duplicate system proxy). Surface conflicts via `ConflictReport` with deterministic resolution actions.

### 3.3 Platform-Specific Mandates (Linux Wayland/X11, Windows, Android)
- **Linux GUI & Wayland (`desktop/`)**:
  - Always build Wails on Linux with `-tags webkit2_41`.
  - Programmatically call `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")` on Linux before initializing the Wails window to prevent blank screens and Wayland `Error 71` under NVIDIA/Mesa DMA-BUF.
  - Package `.deb` dependencies as `libgtk-3-0 | libgtk-3-0t64, libwebkit2gtk-4.1-0, ca-certificates` (never `libwebkit2gtk-4.0-37`), plus StatusNotifierItem tray support (`libayatana-appindicator3`).
  - In `ui/` CSS, always declare `<meta name="color-scheme" content="dark light">` and reset `appearance: none` on `<select>`, `<input>`, and `<button>` so WebKitGTK does not render bright white native GTK inputs on dark themes.
  - Always provide a `--web` flag opening the UI in the user's browser on `127.0.0.1` with a per-session auth token.
- **Linux `nftables` & System Files**:
  - Write all Linux CLI/system files (`rules.nft`, `systemd` units, `/etc/hosts`) in UTF-8 strictly WITHOUT BOM.
  - Make `nftables` scripts idempotent by calling `add table inet <name>` followed by `flush table inet <name>` before redefining chains.
- **Zero-Friction Privileges**:
  - Users MUST NOT run `chmod` or `setcap` manually.
  - Embedded helper binaries unpacked on portable runs must have permissions set programmatically (`0755`) and run through a single cached `pkexec` session (Linux) or single UAC daemon session (Windows), while `.deb` / NSIS installers configure systemd/Windows Service for zero-prompt day-to-day usage.
  - Android MUST operate 100% rootless via `VpnService` (`protect(fd)` on engine sockets, native libraries packaged in `jniLibs`).

---

## 4. Progressive Disclosure Navigation

Load only the document or skill needed for your current task:

| Topic | Path |
| :--- | :--- |
| Product Vision & Feature Matrix | [docs/VISION.md](./docs/VISION.md) |
| Architecture, Engines & Domain Model | [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) |
| Phased Roadmap & DoD | [docs/ROADMAP.md](./docs/ROADMAP.md) |
| Architecture Decision Records | [docs/adr/](./docs/adr/) |
| Protocol Engine Skill | [.agents/skills/protocol-engine/SKILL.md](./.agents/skills/protocol-engine/SKILL.md) |
| Routing Constructor Skill | [.agents/skills/routing-constructor/SKILL.md](./.agents/skills/routing-constructor/SKILL.md) |
| Daemon, IPC & Privileges Skill | [.agents/skills/daemon-ipc-and-privileges/SKILL.md](./.agents/skills/daemon-ipc-and-privileges/SKILL.md) |
| UI Design Pipeline Skill | [.agents/skills/ui-design-pipeline/SKILL.md](./.agents/skills/ui-design-pipeline/SKILL.md) |
| Cross-Platform Release Skill | [.agents/skills/cross-platform-release/SKILL.md](./.agents/skills/cross-platform-release/SKILL.md) |

---

## 5. Current Milestone & Handoff Status

- **Completed**: Phase 0 (repository architecture, ADRs, skills) and **Phase 1 baseline** (`docs/DESIGN_SYSTEM.md` + interactive modular Angular 21 frontend in `ui/` using Standalone + Signals + Zoneless + Lazy Routes + Modular SCSS + JSON i18n `ru`/`en` + `MockTransport`). The UI baseline is verified across Desktop (`1200x760`, `960x760`) and Mobile (`390x844`) and remains intentionally open to iterative refinement once wired to real cores.
- **Next Milestone (Phase 2 — Core, Daemon & Native Shells)**:
  1. `core/api` and `core/storage` (typed JSON-safe commands/events and local storage).
  2. `core/parser` (WireGuard / AmneziaWG 2.0 / 3.0 / 3.1 `.conf` and `vpn://` codecs with full parameter coverage + unit tests).
  3. `core/engine/awg` + `core/cmd/netportal-daemon` (`amneziawg-go/v3`, Windows Named Pipe service, Linux systemd / single-session `pkexec`).
  4. `desktop/` (Wails v2 embedding compiled `ui/` bundle, tray, `-tags webkit2_41`, `--web` loopback fallback) and real `WailsTransport` / `HttpWsTransport` implementations replacing `MockTransport`.
  5. `core/mobile` (`gomobile` primitives facade) + `android/` (`VpnService`, `WebViewAssetLoader`, `AndroidBridgeTransport`).
- **Self-Containment Mandate**: Keep every file, comment, ADR, skill, and commit strictly self-contained inside NetPortal. Never add links, paths, or references to external personal workspaces or unrelated repositories.

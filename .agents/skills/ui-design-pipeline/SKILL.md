---
name: ui-design-pipeline
description: Build and maintain the shared modular Angular 21 + TypeScript + Modular SCSS UI across Desktop (Wails/WebKitGTK 4.1), Android WebView, and --web mode.
---

# UI Design Pipeline Playbook

Use this skill when authoring design tokens, modular SCSS sheets (`ui/src/styles/`), or Standalone Signal Angular components (`ui/src/app/{core,layout,shared,pages}`).

## 1. Visual & Structural Standards

- **Zero-Emoji Policy**: Use crisp inline monochrome/duotone SVGs (e.g., Lucide-style 1.5px stroke icons) for status badges, protocols, and country/region indicators instead of emoji flags or symbols.
- **Responsive Breakpoints & Automatic OS Context (Single Codebase)**:
  - Platform differences (Windows/Linux Desktop vs Android) MUST be resolved automatically from OS/runtime context (`navigator`/bridge/viewport) — NEVER expose manual "Desktop / Android" or artificial "Simple / Constructor" mode toggles in the user header.
  - **Desktop (`>= 900px`)**: Compact fixed left sidebar (`Главная`, `Узлы`, `Маршрутизация`, `Настройки`, `Логи`, `О программе`) + bottom live `Traffic` card + content workspace without heavy global topbars.
  - **Mobile / Narrow (`< 900px`, Android)**: Single-row `52px` top app bar (`← Back` on sub-screens + 1 contextual action) + full-width stacked content + fixed 4-tab bottom navigation (`Главная`, `Узлы`, `Маршруты`, `Настройки`) with `env(safe-area-inset-bottom)`. On mobile, `Логи` and `О программе` live inside the `Настройки` category menu; quick toggles open as a bottom sheet `<dialog>`.
  - **Android TV / Keyboard**: Clear `:focus-visible` outlines and D-pad/Tab traversal order on every interactive card and button.
- **Key-Based JSON Localization (i18n)**:
  - All visible labels, headings, settings descriptions, and feedback messages MUST be resolved by keys from dedicated locale JSON files (`locales/ru.json`, `locales/en.json`, etc.) via `t(key, params)` with 100% key parity across languages — never hardcode untranslated UI strings inside components.
- **Container-Query Adaptive Lists**:
  - Wrap the pre-connect node list in a named CSS container (`container: nodes / inline-size`) and use two compact card tiers (`@container nodes (inline-size < 760px)` 2-row compact layout, `@container nodes (inline-size < 500px)` phone layout) plus a single-row scrollable sources strip so rows never bloat vertically and action buttons (`Select`, `Parameters`) never clip at intermediate desktop/tablet widths.
- **Core VPN Settings & Progressive Disclosure**:
  - **Главная**: Calm connection center with bypass method filter pills (`All`, `AWG/WARP`, `VLESS/Proxy`, `MASQUE`, `Psiphon`, `Zero-VPN DPI/DNS`), active node/mode selector, tactile connect control, compact Signal Path Strip, and `Быстрые настройки` sheet trigger.
  - **Маршрутизация**: Segmented `Правила` (2-tier precedence: Tier 1 App/Process Whitelist/Blacklist -> Tier 2 Network Rules -> `Default Target`, opt-in disabled presets, full-page Rule Builder sub-view, collapsible Rule Hit Tester) and `Цепь` (vertical layer flow: Entry -> DPI/TLS -> Primary/Sticky-Fallback -> Extra Cascade).
  - **Настройки**: Drill-down category list (`Общие`, `Туннель и Входящие` [MTU, TUN/Proxy, IPv6, Mux, Local SOCKS5/HTTP, LAN Share, Android Per-App VPN], `Обфускация, TLS и DPI` [TLS fragmentation/padding/SNI case, AWG 2/3/3.1 defaults, ByeDPI, Desktop-only Zapret], `DNS и Smart Hosts`, `Контроль конфликтов`).

## 2. Happ-Style Pre-Connect UX Rules

- Never hide or disable the server list when disconnected.
- Show **Parsed Node Count**, **Last Sync Time**, and **Protocol Badges** (`AWG 3.1`, `VLESS`, `MASQUE`, `WARP`, `PSIPHON`) immediately after subscription/config import.
- Support **Batch Ping** ("Ping All" or "Ping Selected / Group") with live streaming updates per card:
  - Distinguish Fast TCP/TLS ping vs Real Delay (no-TUN HTTP 204 URL-test) vs Anti-Captcha status (Clean / Challenge).
- Clicking any node selects it immediately (or assigns it to a constructor target) without forcing auto-connect first.

## 3. Linux WebKitGTK 4.1 & Wayland CSS Mandates

- Include `<meta name="color-scheme" content="dark light">` in `ui/index.html`.
- Declare `color-scheme: dark;` on `:root` (and `light` on `[data-theme="light"]`).
- On `button, input, select, textarea`, enforce `color-scheme: inherit; appearance: none; -webkit-appearance: none;` and style custom SVG dropdown chevrons + explicit `select option` background/foreground colors so Linux WebKitGTK never paints unreadable white GTK widgets on dark UIs.
- Keep `desktop/main.go` calling `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")` on `runtime.GOOS == "linux"` prior to `wails.Run(...)`.

## 4. Transport Abstraction (`ui/src/app/core/transport/`)

All UI Signal services inject `NETPORTAL_TRANSPORT` implementing `TransportContract`, backed by interchangeable adapters:
1. `MockTransport` (Phase 1 interactive preview).
2. `WailsTransport` (Desktop `window.go`).
3. `AndroidBridgeTransport` (`window.NetPortalBridge` via Kotlin `JavascriptInterface` + `WebMessagePort`).
4. `HttpWsTransport` (`--web` browser mode).

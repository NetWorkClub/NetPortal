# NetPortal iOS Shell (`ios/`)

Reserved module for future Apple mobile/desktop parity (`NEPacketTunnelProvider` + `WKWebView` + `NetPortalCore.xcframework` built from `core/mobile/` via `gomobile`).

## Status
Deferred until Windows, Linux (Wayland / X11), and Android pipelines stabilize. All Go core contracts under `core/mobile/` remain mobile-safe from day one so iOS integration requires only the thin Swift extension shell.

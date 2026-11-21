# NetPortal UI (`ui/`)

Shared responsive web frontend built with **Angular 21 (Standalone + Signals + Zoneless + Lazy Routes) + TypeScript + Modular SCSS**.

## Architecture Layout

```text
ui/
├── public/locales/           # JSON i18n dictionaries (ru.json, en.json)
└── src/
    ├── styles/               # Modular SCSS design system (abstracts, base, components)
    └── app/
        ├── core/             # DTO models, Transport abstraction (MockTransport), Signal services, RuleEngine
        ├── layout/           # AppShell, DesktopRail, MobileChrome (Topbar + BottomNav)
        ├── shared/           # TranslatePipe, SVG Icon, ToggleSwitch, ConflictBanner, SignalStrip, Native <dialog> modals
        └── pages/            # Lazy-loaded feature routes (/home, /nodes, /routing, /routing/rule/:id, /settings/*, /logs, /about)
```

## Transport Backends

All state stores communicate through the `NETPORTAL_TRANSPORT` token (`TransportContract`), starting with `MockTransport` in Phase 1 and binding to Wails (`desktop/`), Kotlin `WebViewAssetLoader` (`android/`), or loopback HTTP/WS (`--web`) in Phase 2+.

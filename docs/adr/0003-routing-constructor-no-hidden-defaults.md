# ADR-0003: Opt-In Visual Routing Constructor with Zero Hidden Defaults & Cross-Protocol Sticky Fallback

- **Status**: Accepted
- **Date**: 2026-11-22

## Context
Users rely on diverse subscriptions, self-hosted VPN servers, WARP, Psiphon, DPI tools, and Smart DNS, each having different strengths and weaknesses (e.g., `hosts` requires admin rights and gets shadowed by TUN; DPI tools vary by ISP; WARP often fails AI region checks; standard VPN nodes offer good AI geo-IPs but can drop or trigger captchas if load-balanced across random IPs).

Imposing an opinionated default split (such as silently sending AI domains to `hosts` and video to `Zapret`) violates user expectations and makes debugging connectivity unpredictable.

## Decision
1. **Zero Hidden Default Rules**: Newly created profiles contain an empty rule list (`rules: []`) and perform strictly what the user explicitly selects. Optional presets (e.g., AI domains, RU Direct, OpenCCK service lists) are available strictly as opt-in templates in the Routing Constructor.
2. **Rich Opt-In Matchers**: Rules support domains, suffixes, keywords, regexes, external/geo lists, individual IPs, CIDR subnets, explicit IP pools (`start-end`), ports/protocols, Desktop processes/executables, and Android package IDs.
3. **Cross-Protocol `sticky_fallback` Groups**: Users can pin a stable primary server (preserving a single IP across requests) backed by an ordered cross-protocol fallback chain (e.g., `VLESS DE -> VLESS NL -> Psiphon -> WARP AWG 3.1`) controlled by explicit healthcheck thresholds and optional recovery hysteresis.
4. **Independent Pre-Connect Node Inventory**: Subscription and config parsing is decoupled from routing activation so every node can be viewed, selected, pinned, and latency-probed before connecting.

## Consequences
- **Positive**: Complete predictability for beginners (1-click means 100% what was clicked) and full composable power for advanced users without hand-editing raw JSON/YAML.
- **Negative**: Requires a clear, approachable visual rule editor and an explicit "Rule Hit Tester" + `ConflictGuard` in the UI so power configurations remain easy to verify.

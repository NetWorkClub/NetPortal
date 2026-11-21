---
name: routing-constructor
description: Design and compile NetPortal RoutingProfile rules, matchers, sticky cross-protocol fallback chains, and conflict detection without hidden defaults.
---

# Routing Constructor Playbook

Use this skill when modifying `core/router/`, `core/storage/`, or routing editor flows in `ui/`.

## 1. Non-Negotiable Invariants

1. **Zero Hidden Default Rules**:
   - A newly created `RoutingProfile` starts with `Rules: []` and an explicit `DefaultTarget` chosen by the user (or `ActiveSelection` when running a single node directly).
   - Never inject hidden AI domain splits, hidden WARP fallbacks, or hidden `iplist` splits unless the user explicitly enables a preset template or adds those rules in the Constructor.
2. **Pre-Connect Independence**:
   - Parsing subscriptions/configs populates `Source` and `Node` records independently of `RoutingProfile` activation.
   - Users MUST be able to inspect, select, pin, and probe any node before starting any tunnel.
3. **Two-Tier Deterministic Evaluation Order**:
   - **Tier 1 (`app`, highest precedence)**: Process / Application matching supporting both **Whitelist (`include`: match ONLY selected apps/processes)** and **Blacklist (`exclude`: match everything EXCEPT selected apps/processes)**. Optional `ContinueToNetwork` allows combining an app gate with Tier 2 network matchers cleanly without conflicts.
   - **Tier 2 (`network`, secondary precedence)**: Ordered first-match-wins evaluation across network matchers (`Domains`, `DomainSuffixes`, `DomainKeywords`, `DomainRegexes`, `ExternalLists`, `IPCIDRs`, `IPPools`, `GeoIP`, `Network`, `Ports`, sniffed `Protocols`, `IPVersion`, `Inbound`, `Invert`), ending in explicit `DefaultTarget`.
   - Each `Rule` has `Enabled bool`, `Tier string`, `Label string`, `Match RuleMatcher`, `Invert bool`, and `Target RuleTarget`.
4. **Opt-In Disabled Presets**:
   - Common splits (LAN / RFC 1918 bypass, Corporate / CGNAT `100.64.0.0/10`, QUIC UDP 443 block, Ads block, OpenCCK lists, AI services) MAY be listed as presets in the Constructor UI strictly **disabled (`Enabled: false`) by default**.

## 2. Supported Matcher Dimensions

A `RuleMatcher` combines conditions (AND across fields, OR inside slices):
- `AppMatchMode` (`off` | `include` | `exclude`), `ProcessNames []string` / `ProcessPaths []string` (Desktop Windows/Linux), `AndroidPackages []string` (Android `packageName`, e.g., `com.google.android.youtube`).
- `Domains []string`: Exact FQDN match.
- `DomainSuffixes []string`: Suffix match (e.g., `.openai.com`, `.anthropic.com`).
- `DomainKeywords []string`: Substring match.
- `DomainRegexes []string`: Compiled and validated at save time (reject invalid regex before applying).
- `IPCIDRs []string`: Single IPs, IPv4/IPv6 CIDR subnets.
- `IPPools []IPRange`: Explicit `start-end` IP ranges or named pool references.
- `GeoIP []string` & `ExternalLists []string`: Syncable remote lists (e.g., `iplist.opencck.org`, `geosite:*`, `geoip:*`) cached locally with manual/scheduled refresh.
- `Ports []string`, `Network []string` (`tcp`, `udp`), `Protocols []string` (`tls`, `http`, `quic`, `dns`, `bittorrent`, `ssh`), `IPVersion` (`any`, `ipv4`, `ipv6`), and `Inbound` (`any`, `tun`, `proxy`, `lan`).

## 3. Target Types & Sticky-Primary Cross-Protocol Fallback

`RuleTarget` supports:
- `direct` (bypass VPN/proxy cleanly).
- `block` (reject/blackhole).
- `node:<id>` (strictly pinned single node — zero unexpected IP hopping).
- `group:<id>` where `Group` has one of three strategies:
  - `selector`: Manual active node chosen by user.
  - `urltest`: Fastest healthy node by latency.
  - `sticky_fallback`: Ordered cross-protocol chain `[member_1, member_2, ...]` (e.g., `[vless-de-pinned, vless-nl, psiphon-de, warp-awg31]`). Stays locked on `member_1` until `HealthCheck` fails `FailThreshold` consecutive times; fails over cleanly to the next healthy member; supports optional `StickBackOnRecovery` with hysteresis cooldown to avoid flapping.
- `byedpi:<preset_id>` (route matched traffic to local `ciadpi` SOCKS5 listener inside TUN/VpnService).
- `dns:<resolver_id>` (override DNS resolution per rule).

## 4. Conflict Guard Rules

Before starting any session, `ConflictGuard.Evaluate(desiredPlan, systemState)` MUST check for:
- Active `TUN` (full tunnel) vs `hosts` overrides (`smartdns` hosts mode may be bypassed or overwritten if TUN forces remote DNS).
- Active `WinDivert` / `nfqws` (`Zapret`) vs `TUN` capturing overlapping interfaces/ports.
- External system VPN interfaces (`wg*`, `tun*`, `amnezia*`, `tailscale*`) or pre-existing `HTTP_PROXY`/`HTTPS_PROXY` variables.
Return structured `ConflictReport` warnings/blocks with concrete resolution actions rather than failing obscurely.

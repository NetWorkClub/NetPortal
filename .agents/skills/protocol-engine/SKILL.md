---
name: protocol-engine
description: Implement, wrap, or extend NetPortal protocol and bypass engines inside core/engine/ while preserving multi-platform and gomobile compatibility.
---

# Protocol Engine Playbook

Use this skill when adding or modifying engines in `core/engine/`, config parsers in `core/parser/`, or the WARP generator in `core/warp/`.

## 1. Engine Matrix & Responsibilities

| Engine ID | Upstream / Library | Protocols & Features | Platforms |
| :--- | :--- | :--- | :--- |
| `awg` | `github.com/amnezia-vpn/amneziawg-go/v3` | WireGuard, AmneziaWG 2.0, 3.0, 3.1 | Win, Linux, Android, macOS/iOS |
| `singbox` | `sing-box` (with MASQUE / AWG patches or adapter) | VLESS (+Reality/Vision/XHTTP), VMess, Trojan, Hysteria2, TUIC, SS, SOCKS5, HTTP | Win, Linux, Android, macOS/iOS |
| `mihomo` | `mihomo` (optional/adapter where needed) | Clash Meta YAML, native `type: masque` (RFC 9484 CONNECT-IP) | Win, Linux, Android |
| `psiphon` | `github.com/Psiphon-Labs/psiphon-tunnel-core` | Country selection, standalone SOCKS5 or chained hop | Win, Linux, Android |
| `byedpi` | `ciadpi` (`libciadpi.so` / embedded binary) | Rootless user-space TCP DPI desync via local SOCKS5 | Win, Linux, Android, macOS |
| `zapret` | `winws` (WinDivert) / `nfqws` (NFQUEUE + nftables) | Kernel/packet-queue DPI bypass + YouTube QUIC UDP 443 desync | Win, Linux |
| `smartdns` | Native Go module (`core/engine/smartdns`) | Atomic hosts patcher, local HTTP CONNECT proxy, DoH/DoT/DoQ probe + anti-captcha | Win, Linux (Smart DNS on Android) |

## 2. Standard Engine Interface Contract

Every engine MUST implement a uniform lifecycle interface so `core/router/` and `netportal-daemon` can compose or switch engines cleanly:
- `ID() string` and `Capabilities() EngineCapabilities` (supports TUN, supports SOCKS outbound, supports `DialContext` for no-TUN URL testing, requires elevation on Desktop).
- `Validate(cfg any) error`: Strict schema validation before startup.
- `Start(ctx context.Context, runtime EngineRuntime) error`: Receives TUN handle/fd (on Android `protect(fd)` callback), DNS endpoints, and local ports; never hardcodes system ports blindly.
- `Stop(ctx context.Context) error`: Idempotent, guaranteed cleanup of child processes, sockets, `nftables` tables, or `hosts` markers on failure or cancellation.
- `Health(ctx context.Context) EngineStatus`: Live state, active outbound, rx/tx bytes, last error.
- `TestDialer(node NodeConfig) (Dialer, error)`: Creates an isolated in-memory dialer WITHOUT touching system TUN, system proxy, or `/etc/hosts`, enabling Happ-style pre-connect URL tests.

## 3. AmneziaWG 2.0 / 3.0 / 3.1 Parameter Support

Parsers and validators MUST support all standard WireGuard fields plus:
- **AWG 2.0**: `Jc`, `Jmin`, `Jmax`, `S1`, `S2`, `S3`, `S4`, `H1`, `H2`, `H3`, `H4`, `I1`..`I5` (including custom `I1` chains from `warp-generation.github.io`).
- **AWG 3.0**: `HeaderProtectionKey`, `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`, and uint ranges (`min-max`) for `H1`..`H4` and `PersistentKeepalive`.
- **AWG 3.1**: `RandomTrailers`, `DisableCookies` (`on`/`off`/`0`/`1`).

## 4. Android & `gomobile` Rules

- Any package imported by `core/mobile/` MUST compile with `gomobile bind -target=android`.
- Never bind raw OS `exec.Cmd` paths to Android without NDK binary packaging (`jniLibs/arm64-v8a/libciadpi.so` etc., executable via app `nativeLibraryDir`).
- Socket loop prevention on Android: every outbound dialer (AWG, sing-box, Psiphon, ByeDPI) MUST protect raw file descriptors via the injected Android `VpnService.protect(fd)` callback.

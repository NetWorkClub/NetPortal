# ADR-0002: Multi-Engine Orchestration Over Single-Core Monoculture

- **Status**: Accepted
- **Date**: 2026-11-22

## Context
No single upstream networking kernel supports all required NetPortal capabilities cleanly and up to date:
- `amneziawg-go/v3` provides authoritative AmneziaWG 2.0, 3.0, and 3.1 support, including newly introduced parameters (`HeaderProtectionKey`, `RandomTrailers`, `DisableCookies`, ranges).
- `sing-box` excels at VLESS/Reality/Vision/XHTTP, VMess, Trojan, Hysteria2, TUIC, Shadowsocks, process/package split tunneling, and rule-based TUN routing, while MASQUE (RFC 9484 CONNECT-IP) and Clash Meta YAML compatibility may rely on `sing-box` extensions or `mihomo` adapters.
- `psiphon-tunnel-core` provides native Psiphon connectivity in Go.
- `byedpi` (`ciadpi`) enables rootless user-space DPI desynchronization across Android and Desktop.
- `zapret` (`winws` / `nfqws` + `nftables`) handles kernel/packet-queue DPI bypass and QUIC UDP 443 desync on Desktop.
- A native Go `smartdns` engine covers zero-RAM `hosts` patching, local CONNECT proxying, encrypted DNS, and anti-captcha TLS probing.

## Decision
Build `core/engine/` as a **pluggable Multi-Engine Orchestrator** behind a unified Go lifecycle and dialer interface rather than forcing every protocol into one forked proxy core.
- Standalone simple profiles (e.g., pure AWG 3.1 tunnel, pure Zapret, pure Smart Hosts) can run their dedicated engine directly without unnecessary indirection.
- Hybrid constructor profiles route through a unifying TUN/netstack router that attaches `awg`, `psiphon`, `byedpi`, `masque`, and proxy nodes as composable local outbounds.
- Every proxy/VPN engine provides an ephemeral `TestDialer(node)` path that bypasses OS TUN/proxy state for Happ-style pre-connect URL testing.

## Consequences
- **Positive**: Immediate Day-1 parity with upstream AWG 3.1, Psiphon, ByeDPI, and Zapret updates without maintaining fragile monolithic patches inside `sing-box`.
- **Negative**: Larger build/asset matrix (Go modules plus native `ciadpi`/`winws`/`nfqws` helpers) and need for explicit `ConflictGuard` coordination between engines.

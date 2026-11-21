# NetPortal

<details open>
<summary><b>English</b></summary>

## Overview

**NetPortal** unifies fragmented anti-censorship and tunneling workflows into a single open-source platform for **Windows**, **Linux (Wayland / X11)**, and **Android** (with architecture ready for macOS and iOS).

Unlike clients that lock server selection behind an active connection or enforce opinionated default split-routing, NetPortal separates **Node Inventory** from **Routing Policies**:
- **Zero Hidden Defaults**: Connecting a single tunnel or engine routes exactly what you configured, without silent third-party overrides.
- **Pre-Connect Probing**: Parse subscriptions and configs on import, inspect all endpoints, and run fast TCP/TLS or no-TUN real-delay URL-tests before connecting.
- **Visual Routing Constructor**: Compose ordered rules by domain, domain suffix, keyword/regex, external IP/domain lists, CIDR subnets/IP pools, ports/protocols, and desktop processes or installed apps (including Android package names).
- **Cross-Protocol Sticky-Primary Fallback**: Pin a primary node for IP stability (e.g., for AI services) and define ordered cross-protocol backups (e.g., `VLESS DE -> VLESS NL -> Psiphon -> WARP AWG 3.1`) that trigger only on healthcheck failure.

## Supported Protocols & Engines

| Category | Technologies |
| :--- | :--- |
| **Tunnel & VPN** | AmneziaWG 2.0 / 3.0 / 3.1 (`amneziawg-go/v3`), WireGuard, MASQUE (RFC 9484 CONNECT-IP) |
| **Proxy Cores** | `sing-box` / `mihomo` (VLESS + Reality/Vision/XHTTP, VMess, Trojan, Hysteria2, TUIC, Shadowsocks, SOCKS5/HTTP) |
| **Built-in Generators & Networks** | Cloudflare WARP (AWG 2/3/3.1 & MASQUE) + clean endpoint scanner, Psiphon (`psiphon-tunnel-core` standalone & chained) |
| **DPI Bypass** | ByeDPI (`ciadpi` rootless user-space SOCKS5 across Android/Desktop), Zapret (`winws` / `nfqws` + `nftables` on Desktop) |
| **Smart Hosts & DNS** | Atomic hosts patcher, local HTTP CONNECT proxy, encrypted DNS (DoH/DoT/DoQ), anti-captcha probing |

## Architecture Overview

```text
+-------------------------------------------------------------------+
|                   Shared Responsive Web UI (ui/)                  |
|        Angular 21 + Signals + Modular SCSS (Transport API)          |
+-----------------+-------------------------------+-----------------+
                  |                               |
     Desktop (Wails v2 / --web)      Android (Kotlin WebView + Bridge)
                  |                               |
+-----------------v---------------+ +-------------v-----------------+
|   Privileged netportal-daemon   | |   Rootless VpnService + TUN   |
|  Win Service / systemd / pkexec | | netportal-core.aar + ciadpi   |
+-----------------+---------------+ +-------------+-----------------+
                  |                               |
                  +---------------+---------------+
                                  |
+---------------------------------v---------------------------------+
|                        Unified Go Core (core/)                    |
|   parser | probe | router | warp | api | engine (awg3/sing-box/   |
|               psiphon/byedpi/zapret/smartdns)                     |
+-------------------------------------------------------------------+
```

## Repository Layout

- `core/` - Unified Go multi-engine library, subscription parsers, pre-connect prober, routing constructor compiler, WARP generator, privileged desktop daemon, and `gomobile` bindings.
- `ui/` - Single responsive modular Angular 21 + TypeScript + Modular SCSS web interface (`src/app/{core,layout,shared,pages}`) for Desktop, Android, Android TV, and browser `--web` mode.
- `desktop/` - Thin Go + Wails v2 desktop shell (Windows NSIS/Portable, Linux WebKit2GTK 4.1 with Wayland DMA-BUF mitigation, system tray).
- `android/` - Thin Kotlin shell (`VpnService`, `WebViewAssetLoader`, Quick Settings Tile, `libciadpi.so`).
- `ios/` - Reserved directory for future `NEPacketTunnelProvider` + `WKWebView` wrapper.
- `docs/` - Vision, architecture, roadmap, design system, and ADRs.
- `.agents/` - AI coding agent instructions and domain skills.

## Development Status

**Phase 0 (Foundation)** and **Phase 1 baseline (Design System & Modular Angular 21 UI in `ui/`)** are complete and open to iterative refinement during real usage. Moving next to **Phase 2 (Go Core, Privileged Daemon, Desktop Wails v2 & Android VpnService)**. See [docs/ROADMAP.md](./docs/ROADMAP.md) for the milestone plan.

## License

MIT License (c) 2026 NetWorkClub.

</details>

<details>
<summary><b>Русский</b></summary>

## Обзор

**NetPortal** объединяет разрозненные инструменты туннелирования и обхода блокировок в едином открытом клиенте для **Windows**, **Linux (Wayland / X11)** и **Android** (с архитектурой, готовой к macOS и iOS).

В отличие от клиентов, где выбор серверов открывается только после подключения или навязаны скрытые правила раздельного туннелирования, NetPortal разделяет **Инвентарь нод** и **Конструктор маршрутизации**:
- **Никаких скрытых правил по умолчанию**: при включении одного профиля или движка работает ровно то, что выбрал пользователь.
- **Проверка и выбор до подключения**: подписки и конфиги разбираются сразу при импорте — все ноды видны и доступны для массового или выборочного пинга (TCP/TLS и эфемерный URL-test без поднятия TUN) до нажатия кнопки подключения.
- **Визуальный конструктор маршрутизации**: упорядоченные правила по доменам, суффиксам, ключевым словам/regex, внешним спискам, IP/CIDR/пулам адресов, портам и процессам/приложениям (включая пакеты Android).
- **Кросс-протокольный Sticky-Primary Fallback**: закрепление основного сервера (стабильный IP без случайных прыжков, например для нейросетей) с упорядоченным резервом across-protocols (например, `VLESS DE -> VLESS NL -> Psiphon -> WARP AWG 3.1`), который срабатывает только при отказе healthcheck.

## Поддерживаемые протоколы и движки

| Категория | Технологии |
| :--- | :--- |
| **Туннели и VPN** | AmneziaWG 2.0 / 3.0 / 3.1 (`amneziawg-go/v3`), WireGuard, MASQUE (RFC 9484 CONNECT-IP) |
| **Прокси-ядра** | `sing-box` / `mihomo` (VLESS + Reality/Vision/XHTTP, VMess, Trojan, Hysteria2, TUIC, Shadowsocks, SOCKS5/HTTP) |
| **Генераторы и сети** | Cloudflare WARP (AWG 2/3/3.1 и MASQUE) + сканер чистых эндпоинтов, Psiphon (`psiphon-tunnel-core` отдельно и в цепочках) |
| **DPI-обход** | ByeDPI (`ciadpi` user-space SOCKS5 без root для Android и Desktop), Zapret (`winws` / `nfqws` + `nftables` для Desktop) |
| **Smart Hosts и DNS** | Атомарное управление hosts, локальный HTTP CONNECT прокси, DoH/DoT/DoQ и антибот-проверка |

## Структура репозитория

- `core/` - единое Go-ядро, парсеры подписок, пре-коннект пингер, компилятор правил маршрутизации, генератор WARP, привилегированный демон и биндинги `gomobile`.
- `ui/` - единый адаптивный модульный интерфейс на Angular 21 (Standalone + Signals + Zoneless) и TypeScript/SCSS (`src/app/{core,layout,shared,pages}`) для Desktop, Android и браузерного режима `--web`.
- `desktop/` - тонкая desktop-оболочка на Go + Wails v2 (Windows NSIS/Portable, Linux WebKit2GTK 4.1 с защитой от багов Wayland, трей).
- `android/` - тонкая Kotlin-оболочка (`VpnService` без root, `WebViewAssetLoader`, Quick Settings Tile, `libciadpi.so`).
- `ios/` - зарезервировано под будущую оболочку `NEPacketTunnelProvider` + `WKWebView`.
- `docs/` - видение продукта, архитектура, роадмап, дизайн-система и ADR.
- `.agents/` - инструкции и проектные скиллы для AI-агентов.

## Статус разработки

**Фаза 0 (Фундамент)** и **базовый этап Фазы 1 (Дизайн-система и модульный интерфейс Angular 21 в `ui/`)** завершены и открыты для доработки по мере практического использования. Следующий этап — **Фаза 2 (Go-ядро, системный демон, Desktop Wails v2 и Android VpnService)**. План этапов зафиксирован в [docs/ROADMAP.md](./docs/ROADMAP.md).

## Лицензия

MIT License (c) 2026 NetWorkClub.

</details>

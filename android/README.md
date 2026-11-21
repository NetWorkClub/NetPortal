# NetPortal Android Shell (`android/`)

Thin rootless Kotlin shell wrapping Android `VpnService`, `WebViewAssetLoader`, Quick Settings Tile, and native runtime libraries (`netportal-core.aar` compiled via `gomobile bind` from `core/mobile/` and `libciadpi.so` via NDK).

## Purpose
Provides zero-root single-tunnel VPN and DPI bypass orchestration by feeding the Android TUN `ParcelFileDescriptor` directly into the unified Go core while rendering the shared `ui/` frontend.

## Status
Project scaffolding and CI APK builds start in **Phase 2** (following Phase 1 UI/UX design completion).

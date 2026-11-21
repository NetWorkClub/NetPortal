# ADR-0004: Zero-Friction Privileges and Linux Wayland / WebKit2GTK 4.1 Hardening

- **Status**: Accepted
- **Date**: 2026-11-22

## Context
Legacy Windows-only WireGuard GUI clients rely on Win32 SCM/handle-inheritance patterns that do not generalize to Linux/Android, and modern Linux desktop environments (Wayland / GTK3) introduce well-known WebKitGTK and firewall pitfalls:
- `libwebkit2gtk-4.0` removal in modern Linux `.deb` releases (`webkit2gtk-4.1` and `libgtk-3-0t64` transition).
- Blank windows / Wayland `Error 71` under WebKit2GTK DMA-BUF rendering on Wayland/NVIDIA.
- Light native GTK form widgets (`<select>`, `<input>`) breaking dark themes in WebKitGTK.
- `.NET` UTF-8 BOM breaking `nftables` parser (`unexpected junk table`).
- User frustration from manually running `chmod +x`, `setcap`, or entering root passwords repeatedly.

## Decision
1. **Process Split on Desktop**: Run `netportal-daemon` as a privileged background service (Windows Service / Linux `systemd`) or persistent on-demand elevated session (single UAC / single `pkexec`), communicating over ACL-protected Named Pipes / `SO_PEERCRED`-verified Unix sockets. Keep `desktop` UI completely unprivileged.
2. **Automatic Asset Permissions**: Portable builds extract embedded binaries (`winws`, `nfqws`, `ciadpi`) programmatically and apply `0755` permissions via Go (`os.Chmod`) without manual terminal commands.
3. **Linux Wayland Guarantees**:
   - Build Wails with `-tags webkit2_41`.
   - Call `os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")` on Linux before window initialization.
   - Use `libayatana-appindicator3` for GNOME Wayland tray support.
   - Enforce `<meta name="color-scheme" content="dark light">` and `appearance: none` CSS resets on form controls.
   - Always ship `--web` fallback opening the identical UI in the default browser via token-authenticated `127.0.0.1` HTTP+WS.
4. **BOM-Free Idempotent `nftables`**: Write all Linux system/firewall files in UTF-8 without BOM and start `nftables` rulesets with `add table inet netportal` + `flush table inet netportal`.
5. **Rootless Android**: Package `netportal-core.aar` and `libciadpi.so` inside `jniLibs/` and protect outbound sockets with `VpnService.protect(fd)`.

## Consequences
- **Positive**: Installed `.exe`/`.deb`/`.apk` builds work immediately out of the box with zero manual terminal commands and zero repeated admin prompts; portable builds prompt at most once per privileged session and work without prompts for user-space engines.
- **Negative**: Requires maintaining NSIS service installer scripts on Windows, systemd/Polkit packaging in `.deb`, and NDK multi-ABI builds for `libciadpi.so` on Android.

---
name: cross-platform-release
description: Maintain GitHub Actions CI/CD pipelines, packaging (.exe NSIS, .deb, .tar.gz, .apk, .dmg), and bilingual NetWorkClub releases.
---

# Cross-Platform Build & Release Playbook

Use this skill when modifying `.github/workflows/`, packaging scripts, or release documentation.

## 1. Target Artifact Matrix

| Target | Build Toolchain | Artifacts | Notes |
| :--- | :--- | :--- | :--- |
| **Windows x64** | Go + Wails v2 + NSIS | `NetPortal_Windows_Installer.exe`, `NetPortal_Windows_Portable.exe` | Append `C:\Program Files (x86)\NSIS` to `$env:GITHUB_PATH` after `choco install nsis -y` before `wails build -nsis`. |
| **Linux amd64** (`.deb` / `.tar.gz`) | Go + Wails v2 (`-tags webkit2_41`) + `dpkg-deb` | `NetPortal_Linux_amd64.deb`, `NetPortal_Linux_amd64.tar.gz` | Build with `libgtk-3-dev libwebkit2gtk-4.1-dev`; `.deb` `Depends: libgtk-3-0 \| libgtk-3-0t64, libwebkit2gtk-4.1-0, libayatana-appindicator3-1, ca-certificates`. Include `systemd` unit + `.desktop` + `postinst`. |
| **Android (arm64-v8a / armeabi-v7a / x86_64)** | Go (`gomobile bind`) + Android NDK (`ciadpi`) + Gradle | `NetPortal_Android_universal.apk` (and split ABIs if needed) | Signed with release keystore secret (or debug keystore on snapshot workflow_dispatch). |
| **macOS (arm64 / amd64)** | Go + Wails v2 | `NetPortal_macOS.dmg` | Added in later polish phase once core stabilizes. |

## 2. Linux Packaging Critical Rules

- Never depend on `libwebkit2gtk-4.0-37` (dropped in modern Linux `.deb` distributions).
- Always use `libgtk-3-0 | libgtk-3-0t64` to cover both `t64` and legacy GTK3 package naming.
- In `.deb` `postinst`, ensure `chmod 755` on daemon/helper binaries under `/usr/lib/netportal/`, reload `systemd`, and enable socket/service cleanly without failing inside non-systemd container installs (`[ -d /run/systemd/system ] && systemctl daemon-reload || true`).

## 3. Release Notes Standard

Follow NetPortal release discipline:
- Zero emojis in release title or body.
- Start body directly with a concise 1-line summary (NO duplicate `<h1>` repeating version number).
- Use `<details open><summary><b>English</b></summary>` followed by `<details><summary><b>Русский</b></summary>`.
- List SHA256 checksums for every uploaded binary/package.

# NetPortal Instructions (Claude Code)

Follow [AGENTS.md](./AGENTS.md) as the authoritative engineering specification for NetPortal.

Key reminders:
- Monorepo layout: Go core (`core/`), modular Angular 21 UI (`ui/`), Wails v2 desktop (`desktop/`), Kotlin Android (`android/`).
- Zero hidden default split-routing; explicit user opt-in via the Routing Constructor.
- Zero emojis in code, UI, commits, or docs; use English Conventional Commits.
- Consult `.agents/skills/*/SKILL.md` on demand when implementing specific subsystems.

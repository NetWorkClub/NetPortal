# NetPortal Instructions (Gemini / Antigravity)

Read [AGENTS.md](./AGENTS.md) as the single source of truth for NetPortal architecture, rules, and platform mandates before planning or editing code.

Load domain playbooks from `.agents/skills/` only when working on that subsystem:
- Protocol wrappers & `gomobile` core: `.agents/skills/protocol-engine/SKILL.md`
- Routing rules, matchers & conflict guard: `.agents/skills/routing-constructor/SKILL.md`
- Privileged daemon, IPC & zero-friction elevation: `.agents/skills/daemon-ipc-and-privileges/SKILL.md`
- Angular 21 UI, design tokens, modular SCSS & WebKitGTK fixes: `.agents/skills/ui-design-pipeline/SKILL.md`
- CI builds & packaging (Win/Linux/Android): `.agents/skills/cross-platform-release/SKILL.md`

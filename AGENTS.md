# lazyado — Agent Instructions (root)

One-line: A terminal UI for Azure DevOps (Boards and pull requests) using Bubble Tea and Go.

Package manager: Go modules (go.mod)

Essentials (quick):

- Build: `go build -o lazyado .` or `task build`
- Run: `./lazyado`
- Config path: `~/.config/lazyado/config.yaml` (Viper + env overrides)
- File naming convention: snake_case (example: `pbi_arent_modal`)

Quick links (table of contents)

- User docs: [README.md](README.md) • [docs/guide/](docs/guide) (getting started, configuration, boards, pull requests, keybindings, troubleshooting)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md) • docs index: [docs/README.md](docs/README.md)
- Architecture: [docs/development/architecture.md](docs/development/architecture.md)
- API: [docs/development/api-client.md](docs/development/api-client.md)
- Styling & UI: [docs/development/styling.md](docs/development/styling.md) • [docs/development/bubbletea-guide.md](docs/development/bubbletea-guide.md)
- Keys: [docs/development/keymap.md](docs/development/keymap.md)
- Workflow & build: [docs/development/workflow.md](docs/development/workflow.md)
- Skills & references: [docs/development/skills.md](docs/development/skills.md)
- Agent rules & conventions (detailed): [docs/development/agents-guidelines.md](docs/development/agents-guidelines.md)

Detailed conventions and non‑negotiable rules live in [docs/development/agents-guidelines.md](docs/development/agents-guidelines.md) so they can be referenced, edited, and linked from focused docs.

Documentation rule: keep README.md short (pitch, install, quick start, links). User-facing behaviour goes in [docs/guide/](docs/guide), key changes in [docs/guide/keybindings.md](docs/guide/keybindings.md), and change history in [CHANGELOG.md](CHANGELOG.md).

Last updated: November 17, 2026


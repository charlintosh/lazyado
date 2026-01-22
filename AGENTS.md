# lazyado — Agent Instructions (root)

One-line: A terminal UI for Azure DevOps Boards using Bubble Tea and Go.

Package manager: Go modules (go.mod)

Essentials (quick):

- Build: `go build -o lazyado .` or `task build`
- Run: `./lazyado`
- Config path: `~/.config/lazyado/config.yaml` (Viper + env overrides)
- File naming convention: snake_case (example: `pbi_arent_modal`)

Quick links (table of contents)

- Project docs: [docs/architecture.md](docs/architecture.md) • [docs/patterns.md](docs/patterns.md) • [docs/ui.md](docs/ui.md)
- API: [docs/api-client.md](docs/api-client.md)
- Styling & UI: [docs/styling.md](docs/styling.md)
- Workflow & build: [docs/workflow.md](docs/workflow.md)
- Common tasks & integration: [docs/common-tasks.md](docs/common-tasks.md) • [docs/integration.md](docs/integration.md)
- Skills & references: [docs/skills.md](docs/skills.md)
- Agent rules & conventions (detailed): [docs/agents-guidelines.md](docs/agents-guidelines.md)

Detailed conventions and non‑negotiable rules live in [docs/agents-guidelines.md](docs/agents-guidelines.md) so they can be referenced, edited, and linked from focused docs.

Last updated: January 24, 2026


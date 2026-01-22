# Development Workflow

Build & run

```bash
go build -o lazyado .
./lazyado
```

Config setup

- First run creates default config at `~/.config/lazyado/config.yaml`.
- Requires Azure DevOps PAT with `Work Items (Read)` and `Project and Team (Read)` scopes.

Dependencies

- Uses Bubble Tea, Bubbles, Lipgloss, Viper (see `go.mod`).

Notes

- Do not modify production builder scripts or factory patterns unless explicitly requested.

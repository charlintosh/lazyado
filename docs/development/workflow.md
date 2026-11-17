# Development Workflow

Build & run

```bash
go build -o lazyado .
./lazyado
```

Config setup

- First run creates default config at `~/.config/lazyado/config.yaml`.
- Token scopes are listed in [docs/guide/configuration.md](../guide/configuration.md#personal-access-token).

Dependencies

- Uses Bubble Tea, Bubbles, Lipgloss, Viper (see `go.mod`).

Notes

- Do not modify production builder scripts or factory patterns unless explicitly requested.

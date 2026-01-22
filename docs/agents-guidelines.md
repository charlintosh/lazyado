# Agent Guidelines and Conventions

This document contains the detailed conventions, rules and patterns referenced from [AGENTS.md](AGENTS.md).

## Non‑negotiable rules

- Shared UI colors and style constants MUST be defined and exported from the [internal/styles](internal/styles) package. Do not duplicate color literals across files; import and reuse [internal/styles](internal/styles) values instead.
- Define all color variables in [internal/styles/styles.go](internal/styles/styles.go) as exported `Color*` vars (example: `ColorPrimary = lipgloss.Color("#7C3AED")`) and reference them from components via the [internal/styles](internal/styles) package (example: `styles.ColorPrimary`).
- Do not add code comments in new code. Prefer small, self-explanatory functions and clear names.
- Whenever keybindings or keyboard behavior change, update [README.md](README.md) and the in-app help component ([internal/components/panels/help.go](internal/components/panels/help.go)) with a short note describing the change and the affected files.
- All keyboard input handling MUST use the centralized [internal/keys/keymap.go](internal/keys/keymap.go). Do not use raw string matching in `Update` handlers. Instead:
  - Add new bindings to the `KeyMap` struct.
  - Provide defaults in `DefaultKeyMap()`.
  - Use `key.Matches(msg, keymap.KeyName)` for matching.
- Components MUST derive status/help labels from the centralized `KeyMap` (via `KeyMap.Help()` or `components.ShortHelp()`) or expose a `GetAvailableActions()` method; never hardcode key strings in views.

## Logging convention

- Use the scoped logger API via `debug.Scope("<scope>")` in each file that emits logs. Do not call `debug.Log` directly.
- Create one package-level scoped logger per file (unique variable name). Example:

  ```go
  var logClient = debug.Scope("client")
  ```

- Prefer `Debugf` for formatted messages. Example:

  ```go
  logClient.Debugf("fetching %s %s", method, url)
  ```

- Scope strings should be short and lower_snake_case (typically the filename without extension). Example values: `comment_modal`, `client`.
- Avoid reusing the same scope string or logger variable name in multiple files.

## Skills & Auto-invoke mapping

- Local skills live under `.github/skills/` and provide focused guidance.
  - `bubbletea` — Bubble Tea UI patterns and components: [.github/skills/bubbletea/SKILL.md](.github/skills/bubbletea/SKILL.md)
  - `go-uber-style` — Go style, testing, concurrency: [.github/skills/go-uber-style/SKILL.md](.github/skills/go-uber-style/SKILL.md)
  - `references/` — concurrency, error handling, performance, testing patterns: [.github/skills/go-uber-style/references](.github/skills/go-uber-style/references)

- Auto-invoke guidance (where to consult a skill):
  - When editing `internal/components` (UI components), consult `bubbletea` first.
  - For general Go code, tests, and concurrency, consult `go-uber-style`.
  - For implementation guides (concurrency, testing, perf), consult `references`.

## Where to put detailed rules

- Component-level UI patterns: [docs/patterns.md](docs/patterns.md) and [.github/skills/bubbletea/SKILL.md](.github/skills/bubbletea/SKILL.md).
- KeyMap / keybinding rules: [docs/keymap.md](docs/keymap.md) (and [internal/keys/keymap.go](internal/keys/keymap.go)).
- API client guidance: [docs/api-client.md](docs/api-client.md) and [internal/api](internal/api).
- Styling and shared constants: [docs/styling.md](docs/styling.md) and [internal/styles/styles.go](internal/styles/styles.go).
- Logging and scoped logger rules: this file and examples in [debug/logger.go](debug/logger.go).

## Quick checklist for PRs that change keybindings or inputs

- Update [internal/keys/keymap.go](internal/keys/keymap.go) with the new binding and default.
- Update [internal/components/panels/help.go](internal/components/panels/help.go) (in-app help) to include the binding.
- Add a short note to [README.md](README.md) describing the change and affected files.
- Run unit tests for components affected by input handling.

## Change log

- Moved detailed agent rules from root [AGENTS.md](AGENTS.md) into [docs/agents-guidelines.md](docs/agents-guidelines.md) (Jan 24, 2026).

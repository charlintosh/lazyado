# Agent Guidelines and Conventions

This document contains the detailed conventions, rules and patterns referenced from [AGENTS.md](../../AGENTS.md).

## Non‑negotiable rules

- Shared UI colors and style constants MUST be defined and exported from the [internal/styles](../../internal/styles) package. Do not duplicate color literals across files; import and reuse [internal/styles](../../internal/styles) values instead.
- Define all color variables in [internal/styles/styles.go](../../internal/styles/styles.go) as exported `Color*` vars (example: `ColorPrimary = lipgloss.Color("#7C3AED")`) and reference them from components via the [internal/styles](../../internal/styles) package (example: `styles.ColorPrimary`).
- Do not add code comments in new code. Prefer small, self-explanatory functions and clear names.
- Whenever keybindings or keyboard behavior change, update `KeyMap.FullHelp()` (rendered by [internal/components/modals/help_modal.go](../../internal/components/modals/help_modal.go)), the key reference in [docs/guide/keybindings.md](../guide/keybindings.md), the user guide page for that screen under [docs/guide](../guide), and add an entry to [CHANGELOG.md](../../CHANGELOG.md). Keep the README short: it links to the guides and never carries change notes or "affected files" lists.
- All keyboard input handling MUST use the centralized [internal/keys/keymap.go](../../internal/keys/keymap.go). Do not use raw string matching in `Update` handlers. Instead:
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
  - `bubbletea` — Bubble Tea UI patterns and components: [.agents/skills/bubbletea/SKILL.md](../../.agents/skills/bubbletea/SKILL.md)
  - `go-uber-style` — Go style, testing, concurrency: [.agents/skills/go-uber-style/SKILL.md](../../.agents/skills/go-uber-style/SKILL.md)
  - `references/` — concurrency, error handling, performance, testing patterns: [.agents/skills/go-uber-style/references](../../.agents/skills/go-uber-style/references)

- Auto-invoke guidance (where to consult a skill):
  - When editing `internal/components` (UI components), consult `bubbletea` first.
  - For general Go code, tests, and concurrency, consult `go-uber-style`.
  - For implementation guides (concurrency, testing, perf), consult `references`.

## Where to put detailed rules

- Component-level UI patterns: [docs/development/architecture.md](architecture.md) and [.agents/skills/bubbletea/SKILL.md](../../.agents/skills/bubbletea/SKILL.md).
- KeyMap / keybinding rules: [docs/development/keymap.md](keymap.md) (and [internal/keys/keymap.go](../../internal/keys/keymap.go)).
- API client guidance: [docs/development/api-client.md](api-client.md) and [internal/api](../../internal/api).
- Styling and shared constants: [docs/development/styling.md](styling.md) and [internal/styles/styles.go](../../internal/styles/styles.go).
- Logging and scoped logger rules: this file and examples in [internal/debug/logger.go](../../internal/debug/logger.go).

## Quick checklist for PRs that change keybindings or inputs

- Update [internal/keys/keymap.go](../../internal/keys/keymap.go) with the new binding and default.
- Add the binding to `KeyMap.FullHelp()` so the in-app help ([internal/components/modals/help_modal.go](../../internal/components/modals/help_modal.go)) shows it.
- Update [docs/guide/keybindings.md](../guide/keybindings.md) and the guide page for the affected screen.
- Add an `[Unreleased]` entry to [CHANGELOG.md](../../CHANGELOG.md).
- Run unit tests for components affected by input handling.

## Change log

- Moved detailed agent rules from root [AGENTS.md](../../AGENTS.md) into [docs/development/agents-guidelines.md](agents-guidelines.md) (Jan 24, 2026).

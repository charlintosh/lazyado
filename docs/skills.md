# Skills and References — LLM Agent Quick Prompts

This file is a compact, actionable prompt for LLM agents and contributors. Use the short patterns below when performing tasks.

- If you need to implement or modify a Bubble Tea UI component, use: [.github/skills/bubbletea/SKILL.md](.github/skills/bubbletea/SKILL.md) and follow its message, model, and view patterns.
- If you need Go style, testing, or concurrency guidance, use: [.github/skills/go-uber-style/SKILL.md](.github/skills/go-uber-style/SKILL.md).
- If you need implementation references (concurrency, error handling, perf, testing), use: [.github/skills/go-uber-style/references](.github/skills/go-uber-style/references).

Examples:

- "If you need to change keybindings: update [internal/keys/keymap.go](internal/keys/keymap.go); add defaults in `DefaultKeyMap()`; update in-app help [internal/components/panels/help.go](internal/components/panels/help.go) and [README.md](README.md)."
- "If you need shared colors/styles: define exported `Color*` vars in [internal/styles/styles.go](internal/styles/styles.go) and import [internal/styles](internal/styles) from components."
- "If you need logging: create a scoped logger with `debug.Scope(\"<scope>\")` at file scope and use `logX.Debugf(...)`. Keep the scope short and lower_snake_case."

Auto-invoke rules for agents

- When editing UI components: invoke `bubbletea` first.
- For general Go code, tests, and concurrency: invoke `go-uber-style` first.
- When working with Azure DevOps HTML rich-text fields (descriptions, acceptance criteria, repro steps): invoke `html-to-markdown` first.
- For detailed implementation patterns (WIQL, API, etc.): consult `docs/` (e.g., [docs/api-client.md](docs/api-client.md)).

Where to add new guidance

- Add short, focused examples to `.github/skills/*` and link them from this file.

See also: [AGENTS.md](AGENTS.md) (top-level TOC) and [docs/agents-guidelines.md](docs/agents-guidelines.md) (detailed agent rules).

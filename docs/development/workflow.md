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

## Screenshots

The README's screenshots in `docs/images/screenshots/` are real frames, not mock-ups. Regenerate them after a visible change with:

```bash
task screenshots
```

`TestRenderScreenshots` (in `internal/app/screenshots_test.go`, skipped unless `LAZYADO_SCREENSHOTS_DIR` is set) opens every view at 120x34 with the test fixtures and writes the raw frame. `scripts/screenshots/render.mjs` then draws each frame in xterm.js on headless Chromium, with JetBrains Mono, and saves a PNG. It needs Node.js and Playwright's Chromium (`npx playwright install chromium`). To add a view, add an entry to `screenshots()` and an image to the README.

# Contributing to lazyado

Thanks for your interest in improving lazyado! Bug reports, ideas, documentation fixes and code are all welcome.

- [Reporting bugs and suggesting features](#reporting-bugs-and-suggesting-features)
- [Development setup](#development-setup)
- [Making a change](#making-a-change)
- [Project conventions](#project-conventions)

## Reporting bugs and suggesting features

Open an issue on [GitHub](https://github.com/charlintosh/lazyado/issues). For bugs, include your version, OS and terminal, and the relevant part of a [debug log](docs/guide/troubleshooting.md#debug-logging).

## Development setup

You need the Go version listed in [`go.mod`](go.mod) and an Azure DevOps project to test against (see [Configuration](docs/guide/configuration.md)).

```bash
git clone https://github.com/charlintosh/lazyado.git
cd lazyado
go run .            # run from source
go test ./...       # run the tests
git config core.hooksPath .githooks   # turn on the git hooks (task build/dev do it for you)
```

The git hooks are required, like Husky in a Node.js project. They need [golangci-lint](https://golangci-lint.run) v2.15.0 (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`):

| Hook | Runs | When |
| ---- | ---- | ---- |
| `pre-commit` | `golangci-lint run ./...` | the commit touches Go files, `go.mod`/`go.sum` or `.golangci.yml` |
| `pre-push` | `golangci-lint run ./...` and `go test ./...`; for a `v*` tag, also checks the tag's `CHANGELOG.md` section and README version badge | every push |

Don't skip them with `--no-verify`: the Lint workflow and the Release workflow run the same checks, and a release whose lint or tests fail is never published.

The repository includes a [Taskfile](Taskfile.yml) for [Task](https://taskfile.dev) (the binary is called `task`, or `go-task` on some distributions):

| Task | Does |
| ---- | ---- |
| `task build` | Build the `lazyado` binary |
| `task dev` | Run from source |
| `task debug` | Run from source with `-debug` logging |
| `task lint` | Run [golangci-lint](https://golangci-lint.run) |
| `task check` | Lint and test, as the `pre-push` hook and CI do |
| `task hooks` | Turn on the git hooks (`build`, `dev` and `debug` do it automatically) |
| `task install` | Move the built binary to `/usr/local/bin` (uses `sudo`) |

`-debug` writes every API request to `lazyado-debug.log` in your temporary directory, which helps when you are investigating behaviour.

## Making a change

1. Fork the repository and create a branch from `main`.
2. Make your change, with tests for new behaviour. The suite runs without network access: API calls go through a stubbed HTTP transport, and the UI is driven by sending messages to the root model. [`internal/app/pull_requests_test.go`](internal/app/pull_requests_test.go) has examples of both.
3. Run `task check` (lint and tests). The hooks run the same checks on commit and push, and the Lint workflow on every pull request and push to `main`.
4. If you changed what users see or which keys do what, update:
   - the relevant page under [docs/guide](docs/guide), and [docs/guide/keybindings.md](docs/guide/keybindings.md) for key changes;
   - `KeyMap.FullHelp()` in [`internal/keys/keymap.go`](internal/keys/keymap.go), so the in-app help (`?`) matches;
   - the `[Unreleased]` section of [CHANGELOG.md](CHANGELOG.md).
5. Open a pull request describing what changed and why.

## Project conventions

These are the rules most likely to come up in review. The full list is in [docs/development/agents-guidelines.md](docs/development/agents-guidelines.md).

- **Keys** are defined once in [`internal/keys/keymap.go`](internal/keys/keymap.go) and matched with `key.Matches`. Never compare raw key strings, and never hardcode key labels in views: derive them from the `KeyMap`.
- **Colors and layout constants** live in [`internal/styles`](internal/styles). Don't repeat color literals elsewhere.
- **Logging** uses a scoped logger per file: `var logFoo = debug.Scope("foo")`.
- **No I/O in `Update` or `View`.** API calls run in `tea.Cmd`s and come back as messages. See [Architecture](docs/development/architecture.md).
- **Files** use `snake_case` names. New code doesn't add comments; it prefers small, well-named functions.

New to Bubble Tea? The [Bubble Tea guide](docs/development/bubbletea-guide.md) covers the patterns used throughout the code. All developer documentation is indexed in [docs/README.md](docs/README.md#for-contributors).

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE.md).

# LazyADO - An Azure DevOps Boards TUI

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
![Go Report Card](https://goreportcard.com/badge/github.com/charlintosh/lazyado)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
![Repo Size](https://img.shields.io/github/repo-size/charlintosh/lazyado)

A terminal user interface (TUI) for Azure DevOps Boards.

## Features
- **Work items** — Browse, search and filter by sprint, state, assigned, area, tags; sort by ID / Type / State.
- **Hierarchical view** — PBIs / Bugs with expandable child tasks.
- **CRUD** — Create, edit, delete PBIs, Bugs and child tasks via modal forms.
- **Assignments** — Assign or unassign work items to team members using a searchable list.
- **Comments** — Add, edit, delete comments (supports @mentions).
- **Integration helpers** — Create a git branch from a work item (branch-name helper).
- **Detail view** — Fullscreen view showing metadata, description, acceptance criteria, tags and comments.
- **Keyboard-driven UX** — Modals for search/assign/branch/change-state/parent-selection; Vim-like navigation (j/k/g/G); panel switching and contextual help.
- **Cross-platform** — Works on Linux, macOS and Windows.

# Installation

### Install with Go

```bash
go install github.com/charlintosh/lazyado@latest
```

Then run with:

```bash
lazyado
```

### Or build from source (Optional)

If you prefer to build from source:

```bash
go build -o lazyado .
sudo mv lazyado /usr/local/bin/
```

# Get started

Create a config file at `~/.config/lazyado/config.yaml`:

```yaml
# Azure DevOps connection
organization: "my-organization"
project: "my-project"
team: "my-team"

# Authentication (or use AZURE_DEVOPS_PAT env variable)
pat: "your-personal-access-token"

# UI settings
theme: "default"

# Default filters at startup
defaults:
  sprint: "current"
  state: "all"
  assigned: "me"
```

### Environment Variables

| Variable               | Description                         |
| ---------------------- | ----------------------------------- |
| `AZURE_DEVOPS_PAT`     | Personal Access Token (recommended) |
| `AZURE_DEVOPS_ORG`     | Organization (overrides config)     |
| `AZURE_DEVOPS_PROJECT` | Project (overrides config)          |
| `AZURE_DEVOPS_TEAM`    | Team (overrides config)             |

### PAT Permissions

Your Personal Access Token needs these scopes:

- `Work Items (Read)` - Read work items
- `Project and Team (Read)` - List sprints/iterations

## Keyboard Shortcuts


### Keyboard Shortcuts (canonical)

These shortcuts reflect the canonical `internal/keys/keymap.go` bindings used by the application.

#### Global / Panel

| Key | Description |
| ---- | ----------- |
| `1` | Panel 1 / Sort by ID |
| `2` | Panel 2 / Sort by Type |
| `3` | Panel 3 / Sort by State |
| `4` | Panel 4 |
| `5` | Panel 5 |
| `6` | Panel 6 |
| `Tab` | Switch to next panel |
| `Shift+Tab` | Switch to previous panel |
| `?` | Show/hide help |
| `Ctrl+r` | Reload data |
| `q` / `Ctrl+c` | Quit |

Note: keys `1`, `2`, `3` are also used for sorting in some contexts (see Sort section).

#### Navigation

| Key | Description |
| --- | ----------- |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `h` / `←` | Left / collapse |
| `l` / `→` | Right / expand |
| `g` | Go to first item |
| `G` | Go to last item |

#### Actions

| Key | Description |
| --- | ----------- |
| `enter` | Select / Open |
| `v` | View fullscreen details |
| `/` | Search in filter panels |
| `s` | Change work item state |
| `n` | Create new parent item (PBI/Bug) |
| `t` | Create new child task (on PBI) |
| `e` | Edit work item |
| `d` | Delete work item |
| `c` | Add comment (with @mentions) |
| `b` | Create git branch |
| `a` | Assign to user |

#### Modals & Confirmation

| Key | Description |
| --- | ----------- |
| `esc` | Back / Close |
| `ctrl+s` | Save / Submit in modals |
| `y` / `Y` | Confirm |
| `left` / `right` | Modal-only arrow navigation |

#### Sorting

| Key | Description |
| --- | ----------- |
| `1` | Sort by ID (contextual) |
| `2` | Sort by Type (contextual) |
| `3` | Sort by State (contextual) |

If a key has multiple meanings, the active context determines the action (panel vs list sort).

### Work Item Creation/Editing

| Key | Description |
| --- | ----------- |
| `Tab` | Switch input field |
| `Ctrl+S` | Save/Submit form in modals |

### Comments (when Comments panel is focused)

| Key | Description                        |
| --- | ---------------------------------- |
| `e` | Edit selected comment              |
| `d` | Delete selected comment            |
| `c` | Add comment (from Work Items view) |

## Status Messages

| Key | Description |
| --- | ----------- |
| `@` | Trigger user suggestions |
| `↑` / `↓` | Navigate suggestions |
| `Enter` / `Tab` | Select user from suggestions |
| `Ctrl+S` | Submit comment |
| `Esc` | Cancel / Close suggestions |

### Detail View

| Key         | Description                    |
| ----------- | ------------------------------ |
| `Esc` / `q` | Back to main view              |
| `c`         | Add comment                    |
| `j` / `k`   | Scroll (description, comments) |

The detail view displays:

- Work item metadata (type, state, ID, dates, etc.)
- Parent item (if any)
- Description
- Acceptance criteria (if present)
- Tags
- Comments with author and timestamp
# Contributing
### Taskfile

This repository includes a `Taskfile.yml` with convenient tasks for building, running, and debugging the app. Use the `task` CLI (https://github.com/go-task/task) to run them.

Examples:

```bash
# Build release binary
task build

# Run in dev mode
task dev

# Build with debug flags (runtime debug log)
task debug

# Install to PATH (requires sudo)
task install
```

See [Taskfile.yml](Taskfile.yml) for details.

### Debug logging

Enable debug logging with the `-debug` flag when running the binary. On Linux the debug output is written to a temporary file (for example `/tmp/lazyado-debug.log`):

```bash
lazyado -debug
```

Use this when reporting issues or investigating runtime behaviour.


## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Viper](https://github.com/spf13/viper) - Configuration

> ### Acknowledgements
>
> This project is a fork of [shcizo/devops-tui](https://github.com/shcizo/devops-tui). Thanks to the original author(s) for their work and inspiration.

## License

MIT

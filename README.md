# LazyADO - An Azure DevOps Boards TUI

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
![Go Report Card](https://goreportcard.com/badge/github.com/charlintosh/lazyado)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
![Repo Size](https://img.shields.io/github/repo-size/charlintosh/lazyado)
![version](https://img.shields.io/badge/version-0.1.6-purple)

A terminal user interface (TUI) for Azure DevOps Boards.

## Features

- **Azure DevOps layout** — Top bar with `org / project / Boards / hub` breadcrumb, a Boards navigation rail with the filters as dropdowns, and a status bar with the active mode and contextual keys — laid out like the Azure DevOps web UI.
- **Three Boards hubs** — Backlog grid (`L`), Kanban board (`B`) and sprint taskboard (`S`); move cards between workflow columns with `<` / `>`.
- **Work items** — Browse, search and filter by sprint, state, assigned, area, tags; sort by ID / Type / State.
- **Quick search** — Instantly jump to any work item by ID using Ctrl+F.
- **Hierarchical view** — PBIs / Bugs with expandable child tasks.
- **CRUD** — Create, edit, delete PBIs, Bugs and child tasks via modal forms.
- **Assignments** — Assign or unassign work items to team members using a searchable list.
- **Comments** — Add, edit, delete comments (supports @mentions).
- **Work item info** — Quickly copy work item ID or URL to clipboard via info modal.
- **Integration helpers** — Create a git branch from a work item (branch-name helper).
- **Work item form** — Fullscreen form modelled on the Azure DevOps work item form: type stripe, title, State/Area/Iteration row, Description, Acceptance Criteria and Discussion on the left; Planning, Development and Related Work on the right.
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

| Key            | Description              |
| -------------- | ------------------------ |
| `1`            | Iteration filter         |
| `2`            | State filter             |
| `3`            | Assigned-to filter       |
| `4`            | Area filter              |
| `5`            | Main view (grid / board) |
| `6`            | Details tab              |
| `7`            | Discussion tab           |
| `L`            | Backlogs hub             |
| `B`            | Boards hub (Kanban)      |
| `S`            | Sprints hub (taskboard)  |
| `Tab`          | Switch to next panel     |
| `Shift+Tab`    | Switch to previous panel |
| `?`            | Show/hide help           |
| `Ctrl+r`       | Reload data              |
| `q` / `Ctrl+c` | Quit                     |

Note: keys `1`, `2`, `3` are also used for sorting in some contexts (see Sort section).

#### Navigation

| Key       | Description      |
| --------- | ---------------- |
| `j` / `↓` | Move down        |
| `k` / `↑` | Move up          |
| `h` / `←` | Left / collapse  |
| `l` / `→` | Right / expand   |
| `g`       | Go to first item |
| `G`       | Go to last item  |
| `Ctrl+u`  | Half page up     |
| `Ctrl+d`  | Half page down   |

#### Board & taskboard

| Key       | Description                                    |
| --------- | ---------------------------------------------- |
| `h` / `l` | Previous / next column (board) or card (lane)  |
| `j` / `k` | Next / previous card (board) or lane (sprint)  |
| `<`       | Move the card to the previous workflow state   |
| `>`       | Move the card to the next workflow state       |

#### Actions

| Key      | Description                       |
| -------- | --------------------------------- |
| `enter`  | Select / Open                     |
| `v`      | View fullscreen details           |
| `/`      | Search in filter panels           |
| `Ctrl+f` | Quick search by work item ID      |
| `s`      | Change work item state            |
| `n`      | Create new parent item (PBI/Bug)  |
| `t`      | Create new child task (on PBI)    |
| `e`      | Edit work item                    |
| `d`      | Delete work item                  |
| `c`      | Add comment (with @mentions)      |
| `b`      | Create git branch                 |
| `a`      | Assign to user                    |
| `i`      | Show work item info (copy URL/ID) |

#### Modals & Confirmation

| Key              | Description                 |
| ---------------- | --------------------------- |
| `esc`            | Back / Close                |
| `ctrl+s`         | Save / Submit in modals     |
| `y` / `Y`        | Confirm                     |
| `left` / `right` | Modal-only arrow navigation |

#### Sorting

| Key | Description                |
| --- | -------------------------- |
| `1` | Sort by ID (contextual)    |
| `2` | Sort by Type (contextual)  |
| `3` | Sort by State (contextual) |

If a key has multiple meanings, the active context determines the action (panel vs list sort).

### Work Item Creation/Editing

| Key      | Description                |
| -------- | -------------------------- |
| `Tab`    | Switch input field         |
| `Ctrl+S` | Save/Submit form in modals |

### Comments (when Comments panel is focused)

| Key | Description                        |
| --- | ---------------------------------- |
| `e` | Edit selected comment              |
| `d` | Delete selected comment            |
| `c` | Add comment (from Work Items view) |

## Status Messages

| Key             | Description                  |
| --------------- | ---------------------------- |
| `@`             | Trigger user suggestions     |
| `↑` / `↓`       | Navigate suggestions         |
| `Enter` / `Tab` | Select user from suggestions |
| `Ctrl+S`        | Submit comment               |
| `Esc`           | Cancel / Close suggestions   |

### Work Item Form (`v`)

| Key         | Description                         |
| ----------- | ----------------------------------- |
| `Esc` / `q` | Back to the hub                     |
| `Tab`       | Switch between Details / Discussion |
| `c`         | Add comment                         |
| `j` / `k`   | Scroll, or move between comments    |

Keybinding change note: the Boards hubs (`L`, `B`, `S`), card moves (`<`, `>`) and half-page scrolling (`Ctrl+u`, `Ctrl+d`) were added with the Azure DevOps layout; panels `1`–`7` keep their numbers but are now labelled by what they show. Affected files: `internal/keys/keymap.go`, `internal/components/panels/help.go`, `internal/app/app.go`, `internal/components/panels/board.go`, `internal/components/panels/taskboard.go`.

The work item form displays:

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

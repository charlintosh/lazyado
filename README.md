# LazyADO - An Azure DevOps Boards TUI

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
![Go Report Card](https://goreportcard.com/badge/github.com/charlintosh/lazyado)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
![Repo Size](https://img.shields.io/github/repo-size/charlintosh/lazyado)
![version](https://img.shields.io/badge/version-1.0.0-purple)

A terminal user interface (TUI) for Azure DevOps Boards.

## Features

- **Azure DevOps layout** — Top bar with `org / project / Boards / hub` breadcrumb, a left navigation rail (project, then Boards › Boards / Backlogs / Sprints) that collapses to hub keys on narrow terminals, a hub header with view pivots on the left and the scope pickers and filter funnel on the right, and a status bar with the active mode and contextual keys — laid out like the Azure DevOps web UI.
- **Filters where Azure DevOps puts them** — Sprints shows `Sprint ▾` and `Person ▾` in the header; every other filter lives in the filter bar under the pivots (`f`), with a keyword box (`/`) and a count on the funnel when filters are active. The Person / Assigned to list is built from the people actually assigned in the loaded sprint, team and area (with counts), plus team members with nothing assigned, `@Me` and `Unassigned`.
- **Three Boards hubs** — Backlog grid (`L`), Kanban board (`B`) and sprint taskboard (`S`); move cards between workflow columns with `<` / `>`.
- **Work items** — Browse, search and filter by sprint, state, assigned, area, tags; sort by ID / Type / State.
- **Quick search** — Instantly jump to any work item by ID using Ctrl+F.
- **Hierarchical view** — PBIs / Bugs with expandable child tasks.
- **CRUD** — Create, edit, delete PBIs, Bugs and child tasks via modal forms.
- **Assignments** — Assign or unassign work items to team members using a searchable list.
- **Comments** — Add comments (supports @mentions), and edit or delete your own from the Discussion tab or the work item view; your comments are marked "you". Comments render with full Markdown formatting (code blocks, lists, tables, links).
- **Work item info** — Quickly copy work item ID or URL to clipboard via info modal.
- **Work item form** — Fullscreen view (`v`) laid out like the Azure DevOps work item form: type stripe, type and ID, title, assignee / comment count / tags with the item actions on the right, a State · Reason · Area · Iteration band with the Details / Discussion tabs at its bottom-right, then Description, Acceptance Criteria and Discussion on the left and Planning and Related Work on the right. The header paints no background of its own, so it blends into any terminal theme; under 100 columns the band and columns stack.
- **Keyboard-driven UX** — Modals for search/assign/change-state/parent-selection drawn over the board; type-to-filter pickers with fuzzy matching; Vim-like navigation (j/k/g/G); panel switching and contextual help.
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

lazyado identifies you as the user the PAT belongs to, by asking Azure DevOps (`_apis/connectionData`, no extra scope needed). That identity decides which comments you can edit or delete and what `@Me` means.

## Keyboard Shortcuts

### Keyboard Shortcuts (canonical)

These shortcuts reflect the canonical `internal/keys/keymap.go` bindings used by the application.

#### Global / Panel

| Key            | Description              |
| -------------- | ------------------------ |
| `1`            | Iteration / Sprint filter |
| `2`            | Person (assigned to) filter |
| `3`            | State filter             |
| `4`            | Area filter              |
| `/`            | Keyword filter (opens the filter bar) |
| `f`            | Show / hide the filter bar |
| `x`            | Clear keyword, person, state and area filters |
| `z`            | Full screen (hide the navigation rail) |
| `[` / `]`      | Previous / next view (Sprints: Taskboard ↔ Backlog) |
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

Filters `3` and `4` live in the filter bar on Sprints; on Boards and Backlogs all four do. Pressing a filter key opens the bar if needed and drops the picker down under its field.

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
| `/`      | Filter by keyword (ID, title, tag) |
| `Ctrl+f` | Quick search by work item ID      |
| `s`      | Change work item state            |
| `n`      | Create new parent item (PBI/Bug)  |
| `t`      | Create new child task (on PBI)    |
| `e`      | Edit work item                    |
| `d`      | Delete work item                  |
| `c`      | Add comment (with @mentions)      |
| `a`      | Assign to user                    |
| `i`      | Show work item info (copy URL/ID) |

#### Modals & Confirmation

| Key       | Description                                         |
| --------- | --------------------------------------------------- |
| `esc`     | Clear the filter, then back / close                 |
| `ctrl+s`  | Save / Submit in modals                             |
| `y` / `Y` | Confirm                                             |
| `↑` / `↓` | Move in pickers (also `Ctrl+p` / `Ctrl+n`)          |
| _type_    | Filter pickers (assign, filters, forms)             |
| `enter`   | Choose the highlighted option                       |

#### Sorting

| Key | Description                |
| --- | -------------------------- |
| `1` | Sort by ID (contextual)    |
| `2` | Sort by Type (contextual)  |
| `3` | Sort by State (contextual) |

If a key has multiple meanings, the active context determines the action (panel vs list sort).

### Work Item Creation/Editing

| Key                 | Description                                         |
| ------------------- | --------------------------------------------------- |
| `Tab` / `Shift+Tab` | Next / previous field                               |
| `Enter`             | Next field (new line inside Description / Criteria) |
| `↑` / `↓`           | Choose in the Assign to and Sprint pickers          |
| `Ctrl+S`            | Save/Submit form in modals                          |

### Comments (when Comments panel is focused)

| Key | Description                        |
| --- | ---------------------------------- |
| `e` | Edit selected comment (yours only) |
| `d` | Delete selected comment (yours only) |
| `c` | Add comment (from Work Items view) |

The same keys work in the work item view (`v`) once Discussion is focused with `Tab`. Ownership is decided by your Azure DevOps identity ID (the user behind the PAT), falling back to your sign-in name — never by display name — so someone with the same name can't edit your comments, and you can't edit theirs. If lazyado can't resolve your identity, comments stay read-only and `Ctrl+r` retries.

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
| `s`         | Change state                        |
| `a`         | Assign                              |
| `e`         | Edit (edits the comment when Discussion is focused) |
| `t`         | New child task (on PBI / Bug)       |
| `d`         | Delete (deletes the comment when Discussion is focused) |
| `i`         | Info (copy ID / URL)                |
| `c`         | Add comment                         |
| `j` / `k`   | Scroll, or move between comments    |

The work item view runs the same actions as the backlog, board and taskboard. It refreshes in place after a change and closes when the item is deleted.

Keybinding change note: the Boards hubs (`L`, `B`, `S`), card moves (`<`, `>`) and half-page scrolling (`Ctrl+u`, `Ctrl+d`) were added with the Azure DevOps layout; panels `1`–`7` keep their numbers but are now labelled by what they show. Affected files: `internal/keys/keymap.go`, `internal/components/modals/help_modal.go`, `internal/app/update_keys.go`, `internal/components/panels/board.go`, `internal/components/panels/taskboard.go`.

Behaviour change note (comment ownership): `e` / `d` on a comment only act on comments you wrote; on anyone else's comment they show a notice instead, and the hints hide them. `Ctrl+r` now also works from the work item view and retries resolving your identity when it is unknown. Affected files: `internal/api/teams.go`, `internal/api/work_items.go`, `internal/models/team.go`, `internal/models/comment.go`, `internal/components/panels/comment_thread.go`, `internal/components/panels/comment_list.go`, `internal/app/update_results.go`, `internal/app/update_keys.go`.

Layout change note (work item view redesign): the work item view follows the Azure DevOps form and its header no longer sets a background colour. The Reason field is now fetched (`System.Reason`) and shown under State. Affected files: `internal/components/panels/detail_view.go`, `internal/components/panels/comment_list.go`, `internal/api/work_items.go`, `internal/models/work_item.go`, `internal/styles/layout.go`.

Keybinding change note (work item view actions): `s`, `a`, `e`, `t`, `d` and `i` now work in the work item view (`v`), the same as in the lists; with Discussion focused, `e` / `d` still edit and delete the selected comment. Affected files: `internal/app/update_keys.go`, `internal/app/update_results.go`, `internal/components/panels/detail_view.go`.

Keybinding change note (filters in Azure DevOps positions): the filters left the navigation rail. `1`–`4` now open dropdown pickers anchored to the header or filter bar and are ordered like Azure DevOps — `1` Iteration/Sprint, `2` Person (was State), `3` State (was Assigned to), `4` Area. New keys: `/` keyword filter, `f` filter bar, `x` clear filters, `z` full screen, `[` / `]` switch views. The Person filter is applied locally over the loaded items, so changing it no longer reloads. Affected files: `internal/keys/keymap.go`, `internal/app/update_keys.go`, `internal/app/header.go`, `internal/components/panels/page_header.go`, `internal/components/panels/nav_rail.go`, `internal/components/modals/filter_picker_modal.go`, `internal/components/widgets/dropdown.go`, `internal/models/filter.go`.

Keybinding change note (bubbles components refactor): the `b` "create git branch" action was removed. The modal-only `←` / `→` bindings are gone; sprint, type and assignee selection now use pickers driven by `↑` / `↓` (or `Ctrl+p` / `Ctrl+n`) with type-to-filter, and `Enter` advances to the next form field. `s`, `a`, `e`, `c` and `v` also work from the Details tab. Affected files: `internal/keys/keymap.go`, `internal/components/widgets/picker.go`, `internal/components/modals/work_item_form.go`, `internal/app/update_keys.go`.

The work item form displays:

- Work item metadata (type, state, ID, dates, etc.)
- Parent item (if any)
- Description
- Acceptance criteria (if present)
- Tags
- Comments with author and timestamp

# Contributing

### Taskfile

This repository includes a `Taskfile.yml` with convenient tasks for building, running, and debugging the app. Use the `go-task` CLI (https://taskfile.dev/docs/installation) to run them.

Examples:

```bash
# Build release binary
go-task build

# Run in dev mode
go-task dev

# Build with debug flags (runtime debug log)
go-task debug

# Install to PATH (requires sudo)
go-task install
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

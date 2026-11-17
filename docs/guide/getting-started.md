# Getting started

This guide takes you from nothing to a running lazyado connected to your Azure DevOps project.

- [Installation](#installation)
- [First run](#first-run)
- [A tour of the screen](#a-tour-of-the-screen)
- [Moving around](#moving-around)
- [Next steps](#next-steps)

## Installation

### Prebuilt binary

Each release on [GitHub Releases](https://github.com/charlintosh/lazyado/releases) has archives for Linux, macOS and Windows (x86_64, i386 and arm64). Download the one for your platform, extract it, and put `lazyado` (or `lazyado.exe`) somewhere on your `PATH`.

### With Go

You need the Go version listed in [`go.mod`](../../go.mod) or newer.

```bash
go install github.com/charlintosh/lazyado@latest
```

This puts `lazyado` in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

### From source

```bash
git clone https://github.com/charlintosh/lazyado.git
cd lazyado
go build -o lazyado .
sudo mv lazyado /usr/local/bin/
```

## First run

lazyado needs four things to connect: your **organization**, **project**, **team**, and a **Personal Access Token** (PAT).

1. Create a PAT in Azure DevOps (*User settings → Personal access tokens*) with the scopes listed in [Configuration › Personal access token](configuration.md#personal-access-token).
2. Run `lazyado` once. If it finds no usable configuration, it writes a template to `~/.config/lazyado/config.yaml` and exits.
3. Fill in the template:

   ```yaml
   organization: "my-organization"   # from https://dev.azure.com/my-organization
   project: "my-project"
   team: "my-team"
   pat: "your-personal-access-token"
   ```

   You can leave `pat` empty and set the `AZURE_DEVOPS_PAT` environment variable instead. This keeps the token out of the file.
4. Run `lazyado` again. It loads your iterations, areas and work items and opens the **Backlogs** hub.

[Configuration](configuration.md) explains every setting and environment variable.

## A tour of the screen

lazyado is laid out like the Azure DevOps web UI.

| Area | What it shows |
| ---- | ------------- |
| **Top bar** | `organization / project / section / hub`, the quick search hint, and whether data is syncing. Notifications appear on the right. |
| **Navigation rail** (left) | *Boards* → Boards (`B`), Backlogs (`L`), Sprints (`S`), and *Repos* → Pull requests (`P`). On narrow terminals it shrinks to the hub keys; `z` hides it. |
| **Hub header** | The team (and sprint, on Sprints), the view pivots on the left, and the pickers and filter funnel on the right. |
| **Content** | The backlog grid, board, taskboard or pull request list, with a details or preview pane on wide terminals. |
| **Status bar** (bottom) | The current mode and the keys that work right now. |

The status bar only lists keys that apply where you are. Press `?` for the full list.

## Moving around

| Key | Does |
| --- | ---- |
| `L` `B` `S` `P` | Backlogs, Boards, Sprints, Pull requests |
| `j` `k` / `↓` `↑` | Move down / up |
| `h` `l` / `←` `→` | Move left / right, collapse / expand |
| `g` / `G` | First / last item |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `enter` | Open |
| `esc` | Back, or close a dialog |
| `Ctrl+f` | Quick search |
| `Ctrl+r` | Reload from Azure DevOps |
| `?` | Help |
| `q` | Quit |

### Quick search

`Ctrl+f` jumps straight to a work item or pull request from anywhere:

| You type | lazyado opens |
| -------- | ------------- |
| `1234` | The work item or pull request with that ID. If both exist, you choose. |
| `!1234` | Pull request 1234 |
| `#1234` | Work item 1234 |
| A link | The work item or pull request it points to. Paste any `dev.azure.com/…` or `….visualstudio.com/…` work item or pull request URL. |

Links to another organization or project are refused with a message naming the project lazyado is connected to.

## Next steps

- [Boards](boards.md): backlogs, boards, sprints and work items.
- [Pull requests](pull-requests.md): reviewing code.
- [Keybindings](keybindings.md): the complete key reference.

# lazyado

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/charlintosh/lazyado)](https://goreportcard.com/report/github.com/charlintosh/lazyado)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)
![version](https://img.shields.io/badge/version-1.3.0-purple)

**A keyboard-driven terminal UI for Azure DevOps.** Plan work on your boards and review pull requests without leaving the terminal.

lazyado follows the layout of the Azure DevOps web UI (a navigation rail, hub pivots, filters where you expect them), so it feels familiar from the first launch.

## Features

**Boards**
- Backlogs, Kanban boards and Sprint taskboards, with the same filters as Azure DevOps: sprint, person, state, area and keyword.
- Create, edit, assign, move and delete Product Backlog Items, Bugs and tasks.
- Drag tasks between state columns on the Sprints taskboard.
- A full-screen work item form, with Markdown-rendered descriptions and a discussion that supports @mentions.

**Pull requests**
- Mine / Active / Completed / Abandoned views for one repository or all of them, with a search bar that filters as you type and searches Azure DevOps by author, reviewer or branch.
- Create pull requests: pick two branches, see how far ahead and behind they are, and start from the repository's pull request template.
- A review screen with the description, policies, status checks, reviewer votes and linked work items.
- A file browser with an inline diff. Comment on a line or a range of lines, reply, like comments, resolve threads and vote.

**Pipelines**
- Recently run pipelines, every pipeline by folder, and the latest runs, with server-side search by name, state, branch, pipeline and requester.
- Run details with stages, jobs, errors and step logs that follow a running job.
- Run a pipeline from a branch or tag, with its YAML parameters, the stages to run and queue-time variables.
- Classic releases: starred release pipelines first, stage progress, approvals waiting on you, deploying a stage, and creating a release with chosen artifact versions, stages and variables.

**Everywhere**
- Quick search (`Ctrl+F`) for a work item or pull request by ID or by pasted link.
- Vim-style navigation, contextual key hints in the status bar, and full help on `?`.
- Runs on Linux, macOS and Windows.

## Installation

Download a prebuilt binary for your platform from [GitHub Releases](https://github.com/charlintosh/lazyado/releases), or install with Go:

```bash
go install github.com/charlintosh/lazyado@latest
```

Other options are covered in [Getting started](docs/guide/getting-started.md#installation).

## Quick start

1. Create a [Personal Access Token](https://learn.microsoft.com/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate) with the [scopes lazyado needs](docs/guide/configuration.md#personal-access-token).
2. Create `~/.config/lazyado/config.yaml`:

   ```yaml
   organization: "my-organization"
   project: "my-project"
   team: "my-team"
   pat: "your-personal-access-token"   # or set AZURE_DEVOPS_PAT
   ```

3. Run `lazyado`. Press `?` at any time to see every key.

## Documentation

| Guide | What it covers |
| ----- | -------------- |
| [Getting started](docs/guide/getting-started.md) | Installing, first run, and a tour of the screen |
| [Configuration](docs/guide/configuration.md) | Config file, environment variables, token scopes, saved state |
| [Boards](docs/guide/boards.md) | Backlogs, boards, sprints, filters and work items |
| [Pull requests](docs/guide/pull-requests.md) | Browsing, reviewing, commenting and voting on pull requests |
| [Pipelines](docs/guide/pipelines.md) | Pipelines, runs, logs, and running a pipeline |
| [Releases](docs/guide/releases.md) | Releases, approvals, deploying stages and creating releases |
| [Keybindings](docs/guide/keybindings.md) | Every key, by screen |
| [Troubleshooting](docs/guide/troubleshooting.md) | Common problems and debug logging |

Changes in each release are listed in the [changelog](CHANGELOG.md).

## Contributing

Bug reports, ideas and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) explains how to build, test and debug lazyado, and links to the [developer documentation](docs/README.md#for-contributors).

## Acknowledgements

lazyado started as a fork of [shcizo/devops-tui](https://github.com/shcizo/devops-tui). Thanks to its authors for the foundation. It is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## License

[MIT](LICENSE.md)

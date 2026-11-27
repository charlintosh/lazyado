<h1 align="center">
  <img src="docs/images/wordmark.png" alt="lazyado" width="420">
</h1>

[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](LICENSE.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/charlintosh/lazyado)](https://goreportcard.com/report/github.com/charlintosh/lazyado)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)
![version](https://img.shields.io/badge/version-1.8.0-purple)

**A keyboard-driven terminal UI for Azure DevOps.** Plan work on your boards and review pull requests without leaving the terminal.

lazyado follows the layout of the Azure DevOps web UI (a navigation rail, hub pivots, filters where you expect them), so it feels familiar from the first launch.

<img src="docs/images/screenshots/backlog.png" alt="lazyado showing a backlog with an expanded product backlog item and its tasks" width="100%">

## Features

**Boards**
- Backlogs, Kanban boards and Sprint taskboards, with the same filters as Azure DevOps: sprint, person, state, area and keyword.
- Create, edit, assign, move and delete Product Backlog Items, Bugs and tasks.
- Drag tasks between state columns on the Sprints taskboard.
- A full-screen work item form, with Markdown-rendered descriptions and a discussion that supports @mentions.

<details>
<summary>Screenshots</summary>

**Kanban board**

<img src="docs/images/screenshots/board.png" alt="Kanban board" width="100%">

**Sprint taskboard**

<img src="docs/images/screenshots/sprint.png" alt="Sprint taskboard" width="100%">

**Work item**

<img src="docs/images/screenshots/work-item.png" alt="Work item" width="100%">

</details>

**Repos**
- Browse a repository's files on any branch or tag, with the last commit behind each item, rendered READMEs and Markdown, and copy its HTTPS or SSH clone URL.
- Commit history for a branch, a tag, a file or a folder, with each commit's changed files.
- Branches with ahead and behind counts and their pull requests; create and delete branches.
- Tags, annotated or lightweight; create annotated tags.

<details>
<summary>Screenshots</summary>

**Files**

<img src="docs/images/screenshots/files.png" alt="Files" width="100%">

**Commits**

<img src="docs/images/screenshots/commits.png" alt="Commits" width="100%">

**Branches**

<img src="docs/images/screenshots/branches.png" alt="Branches" width="100%">

**Tags**

<img src="docs/images/screenshots/tags.png" alt="Tags" width="100%">

</details>

**Pull requests**
- Mine / Active / Completed / Abandoned views for one repository or all of them, with a search bar that filters as you type and searches Azure DevOps by author, reviewer or branch.
- Create pull requests: pick two branches, see how far ahead and behind they are, and start from the repository's pull request template.
- A review screen with the description, policies, status checks, reviewer votes and linked work items.
- A file browser with an inline diff. Comment on a line or a range of lines, reply, like comments, resolve threads and vote.

<details>
<summary>Screenshots</summary>

**Pull requests**

<img src="docs/images/screenshots/pull-requests.png" alt="Pull requests" width="100%">

**Review overview**

<img src="docs/images/screenshots/pull-request-review.png" alt="Review overview" width="100%">

**Files and inline comments**

<img src="docs/images/screenshots/pull-request-files.png" alt="Files and inline comments" width="100%">

</details>

**Pipelines**
- Recently run pipelines, every pipeline by folder, and the latest runs, with server-side search by name, state, branch, pipeline and requester.
- Run details with stages, jobs, errors and step logs that follow a running job.
- Run a pipeline from a branch or tag, with its YAML parameters, the stages to run and queue-time variables.
- Classic releases: starred release pipelines first, stage progress, approvals waiting on you, deploying a stage, creating a release with chosen artifact versions, stages and variables, and seeing or editing a release pipeline's tasks, settings and variables.

<details>
<summary>Screenshots</summary>

**Pipelines**

<img src="docs/images/screenshots/pipelines.png" alt="Pipelines" width="100%">

**Pipeline run**

<img src="docs/images/screenshots/pipeline-run.png" alt="Pipeline run" width="100%">

**Releases**

<img src="docs/images/screenshots/releases.png" alt="Releases" width="100%">

**Release**

<img src="docs/images/screenshots/release.png" alt="Release" width="100%">

</details>

**Everywhere**
- Quick search (`Ctrl+F`) for a work item or pull request by ID or by pasted link.
- Vim-style navigation, contextual key hints in the status bar, and full help on `?`.
- Runs on Linux, macOS and Windows.

<details>
<summary>Screenshots</summary>

**Keyboard shortcuts**

<img src="docs/images/screenshots/help.png" alt="Keyboard shortcuts" width="100%">

</details>

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
| [Repos](docs/guide/repos.md) | Files, commits, branches, tags and cloning |
| [Pull requests](docs/guide/pull-requests.md) | Browsing, reviewing, commenting and voting on pull requests |
| [Pipelines](docs/guide/pipelines.md) | Pipelines, runs, logs, and running a pipeline |
| [Releases](docs/guide/releases.md) | Releases, approvals, deploying stages, creating releases and editing release pipelines |
| [Keybindings](docs/guide/keybindings.md) | Every key, by screen |
| [Troubleshooting](docs/guide/troubleshooting.md) | Common problems and debug logging |

Changes in each release are listed in the [changelog](CHANGELOG.md).

## Contributing

Bug reports, ideas and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) explains how to build, test and debug lazyado, and links to the [developer documentation](docs/README.md#for-contributors).

## Acknowledgements

lazyado started as a fork of [shcizo/devops-tui](https://github.com/shcizo/devops-tui). Thanks to its authors for the foundation. It is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## License

[MIT](LICENSE.md)

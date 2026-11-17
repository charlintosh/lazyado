# Configuration

- [Config file](#config-file)
- [Environment variables](#environment-variables)
- [Large files in diffs](#large-files-in-diffs)
- [Personal access token](#personal-access-token)
- [Who you are](#who-you-are)
- [What lazyado remembers](#what-lazyado-remembers)

## Config file

lazyado reads `config.yaml` from the first of these places that has one:

1. `~/.config/lazyado/config.yaml`
2. `./config.yaml`, in the directory you start lazyado from

```yaml
organization: "my-organization"
project: "my-project"
team: "my-team"
pat: "your-personal-access-token"
```

| Key | Required | Description |
| --- | :------: | ----------- |
| `organization` | yes | The organization name, as in `https://dev.azure.com/<organization>` |
| `project` | yes | The project to open |
| `team` | yes | The team whose iterations, areas, board and members are used |
| `pat` | yes | A Personal Access Token. Prefer `AZURE_DEVOPS_PAT` so the token stays out of the file. |

If a required value is missing, lazyado stops and names it, for example `organization is required (set in config or AZURE_DEVOPS_ORG)`.

## Environment variables

Environment variables override the config file. They are handy for keeping the token out of the file, or for switching projects in a single command.

| Variable | Overrides |
| -------- | --------- |
| `AZURE_DEVOPS_PAT` | `pat` |
| `AZURE_DEVOPS_ORG` | `organization` |
| `AZURE_DEVOPS_PROJECT` | `project` |
| `AZURE_DEVOPS_TEAM` | `team` |

```bash
AZURE_DEVOPS_PROJECT=other-project lazyado
```

## Large files in diffs

Lockfiles, generated bundles and other very large files make diffs slow and hard to read. Like GitHub and Azure DevOps, lazyado shows **This file is too large to show** for them instead. You can still press `enter` to load the diff, or `o` to open the file in the browser.

```yaml
diff:
  max_file_lines: 3000      # files longer than this aren't rendered
  max_changed_lines: 1000   # diffs that change more lines than this aren't rendered
  max_file_kb: 1024         # files bigger than this aren't downloaded at all (browser only)
  collapse:                 # extra paths to collapse, using .gitattributes patterns
    - "**/*.snap"
```

| Key | Default | Description |
| --- | ------- | ----------- |
| `diff.max_file_lines` | `3000` | The longer of the two versions is over this many lines. This catches `package-lock.json`, `yarn.lock` and similar files. |
| `diff.max_changed_lines` | `1000` | The change itself is bigger than this, even in a shorter file. |
| `diff.max_file_kb` | `1024` | The file is bigger than this. It isn't downloaded, so it can only be opened in the browser. |
| `diff.collapse` | none | Extra patterns to collapse, written like `.gitattributes` paths (`*.lock`, `dist/**`). |

lazyado also reads the repository's root `.gitattributes`, as GitHub does. Files marked `linguist-generated` are collapsed as *generated*, and files marked `-diff` or `binary` are collapsed with their diff turned off. Size limits are checked first, so a large lockfile always says it is too large.

## Personal access token

Create the token under *User settings → Personal access tokens* in Azure DevOps. Give it these scopes:

| Scope | Needed for |
| ----- | ---------- |
| **Work Items** · Read | Browsing backlogs, boards, sprints and work items |
| **Work Items** · Read & write | Creating, editing, assigning, moving and deleting work items, and commenting on them |
| **Project and Team** · Read | Iterations, areas and team members |
| **Code** · Read | Repositories, pull requests, threads, policies and file contents |
| **Code** · Read & write | Voting, commenting, replying, liking and resolving threads on pull requests |

You can skip the *Code* scopes if you only use the Boards hubs, and the *Read & write* scopes if you only want to browse.

## Who you are

lazyado asks Azure DevOps who owns the token (`_apis/connectionData`, no extra scope needed). That identity decides:

- what `@Me` means in the Person filter, and which pull requests the **Mine** view shows (ones you created or are reviewing);
- which comments are yours. Only you can edit or delete them, and you can't like your own;
- your vote on a pull request.

Ownership is matched by your identity ID, falling back to your sign-in name. It is never matched by display name, so someone with the same name can't edit your comments. If lazyado can't confirm your identity, comments become read-only and voting is disabled. `Ctrl+r` retries.

## What lazyado remembers

Some choices are saved to `~/.config/lazyado/state.json` and restored the next time you start:

- the sprint, person, state and area filters;
- the repository chosen in the Pull requests hub (or *All repositories*);
- the repositories you starred locally (`Ctrl+t` in the repository picker).

Delete the file to start fresh. Repositories you starred in Azure DevOps are read from your account each time and are not stored here.

# Keybindings

Every key in lazyado, grouped by where it works. Press `?` in the app for the same list, and watch the status bar: it only shows keys that apply to what is on screen.

Some keys mean different things in different places. For example, `a` assigns a work item on the Boards hubs but votes in the Pull requests hub. The tables below say which applies where.

- [Everywhere](#everywhere)
- [Navigation](#navigation)
- [Boards hubs](#boards-hubs)
- [Work item form](#work-item-form)
- [Files hub](#files-hub)
- [Commits hub](#commits-hub)
- [Branches hub](#branches-hub)
- [Tags hub](#tags-hub)
- [Pull requests list](#pull-requests-list)
- [Pull request review](#pull-request-review)
- [Builds hub](#builds-hub)
- [Pipeline run](#pipeline-run)
- [Releases hub](#releases-hub)
- [Release](#release)
- [Dialogs and pickers](#dialogs-and-pickers)

## Everywhere

| Key | Action |
| --- | ------ |
| `L` | Backlogs hub |
| `B` | Boards hub (Kanban) |
| `S` | Sprints hub (taskboard) |
| `E` | [Repos › Files hub](repos.md#files) |
| `H` | [Repos › Commits hub](repos.md#commits) (in the Files hub: the selected item's history; in Branches: the branch's commits) |
| `V` | [Repos › Branches hub](repos.md#branches) |
| `T` | [Repos › Tags hub](repos.md#tags) |
| `P` | Pull requests hub |
| `D` | [Pipelines › Builds hub](pipelines.md) |
| `R` | [Pipelines › Releases hub](releases.md) |
| `Ctrl+f` | [Quick search](getting-started.md#quick-search): `1234`, `!1234`, `#1234` or a link |
| `Ctrl+r` | Reload from Azure DevOps (also retries identifying you) |
| `z` | Full screen (hide the navigation rail) |
| `,` | Dismiss the notification |
| `?` | Help |
| `q` / `Ctrl+c` | Quit |

## Navigation

| Key | Action |
| --- | ------ |
| `j` / `↓` | Down |
| `k` / `↑` | Up |
| `h` / `←` | Left, or collapse |
| `l` / `→` | Right, or expand |
| `g` / `G` | First / last |
| `Ctrl+d` / `Ctrl+u` | Half page down / up (also `PgDn` / `PgUp`) |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `[` / `]` | Previous / next view |
| `enter` | Open / select |
| `esc` | Back |

## Boards hubs

Backlogs, Boards and Sprints. See [Boards](boards.md) for details.

### Filters and panes

| Key | Action |
| --- | ------ |
| `1` | Iteration / sprint filter |
| `2` | Person filter |
| `3` | State filter |
| `4` | Area filter |
| `/` | Keyword filter |
| `f` | Show / hide the filter bar |
| `x` | Clear keyword, person, state and area filters |
| `5` | Main view (grid, board or taskboard) |
| `6` | Details tab |
| `7` | Discussion tab |

### Work items

| Key | Action |
| --- | ------ |
| `v` | Open the work item form |
| `n` | New backlog item (PBI or Bug) |
| `t` | New child task |
| `e` | Edit (in Discussion: edit your comment) |
| `+` | In Discussion: like / unlike someone else's comment |
| `d` | Delete (in Discussion: delete your comment) |
| `s` | Change state |
| `a` | Assign |
| `c` | Add a comment |
| `i` | Info: copy ID or URL |

### Board and taskboard

| Key | Action |
| --- | ------ |
| `<` / `>` | Move the card to the previous / next state |
| `h` / `l` | Previous / next column (board), or collapse / expand a row (taskboard) |
| `C` | Collapse / expand every taskboard row |
| Drag and drop | Move a task to another state column (Sprints taskboard) |

## Work item form

Opened with `v`.

| Key | Action |
| --- | ------ |
| `Tab` | Switch between Details and Discussion |
| `j` / `k` | Scroll, or move between comments |
| `s` `a` `e` `t` `d` `i` `c` | Same as in the lists. With Discussion focused, `e` / `d` act on your selected comment and `+` likes someone else's. |
| `esc` / `q` | Back to the hub |

## Files hub

See [Repos](repos.md) for details.

| Key | Action |
| --- | ------ |
| `l` / `enter` | Expand a folder, or open a file's contents |
| `h` | Collapse a folder, or go to its parent |
| `Tab` | Switch between the tree and the contents |
| `esc` | Back to the tree |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `c` | [Clone](repos.md#cloning): copy the HTTPS or SSH URL, or a `git clone` command |
| `H` | History of the selected file or folder |
| `y` | Copy the selected item's path |
| `o` | Open in the browser |

## Commits hub

See [Repos](repos.md#commits) for details.

| Key | Action |
| --- | ------ |
| `enter` | Browse the files at that commit |
| `/` | [Filter](repos.md#filtering): text, or `author:` and `path:` |
| `esc` | Clear the filter |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `y` | Copy the commit ID |
| `o` | Open the commit in the browser |

## Branches hub

See [Repos](repos.md#branches) for details.

| Key | Action |
| --- | ------ |
| `1` – `3` | Mine / All / Stale |
| `Tab` / `Shift+Tab`, `h` / `l`, `[` / `]` | Next / previous view |
| `n` | [New branch](repos.md#creating-a-branch) |
| `d` | [Delete the branch](repos.md#deleting-a-branch) |
| `enter` | Browse the branch's files |
| `H` | The branch's commits |
| `p` | New pull request from the branch |
| `/` | Search by name |
| `y` | Copy the branch name |
| `o` | Open in the browser |

## Tags hub

See [Repos](repos.md#tags) for details.

| Key | Action |
| --- | ------ |
| `n` | [New tag](repos.md#creating-a-tag) |
| `enter` | Browse the files at the tag |
| `H` | The commits up to the tag |
| `/` | Search by name |
| `y` | Copy the tag name |
| `o` | Open in the browser |

## Pull requests list

| Key | Action |
| --- | ------ |
| `1` – `4` | Mine / Active / Completed / Abandoned |
| `Tab` / `Shift+Tab`, `h` / `l`, `[` / `]` | Next / previous view |
| `r` | Choose the repository |
| `n` | [New pull request](pull-requests.md#creating-a-pull-request) |
| `/` | [Search](pull-requests.md#searching): text, or `author:`, `reviewer:`, `source:`, `target:`, `is:draft` |
| `enter` | Open the review |
| `a` | Vote |
| `c` | Comment on the pull request |
| `b` | Open the build behind a policy or status check |
| `o` | Open in the browser |

## Pull request review

See [Pull requests](pull-requests.md) for details.

| Key | Action |
| --- | ------ |
| `1` / `2` | Overview / Files tab |
| `[` / `]` | Switch tabs (`Tab` also works on Overview) |
| `a` | Vote |
| `c` | Comment on the pull request, or on the current line or selection in Files |
| `w` | Open a linked work item |
| `b` | Open the build behind a policy or status check; `esc` comes back |
| `o` | Open the pull request, or the focused thread, in the browser |
| `esc` | Back to the list |

### Comments

These keys act on the focused comment, on Overview and in the Files diff.

| Key | Action |
| --- | ------ |
| `j` / `k` | Next / previous comment (in Files: line or comment) |
| `r` | Reply |
| `e` / `d` | Edit / delete, on your own comments |
| `+` | Like / unlike, on other people's comments |
| `s` | Resolve / reopen the thread |
| `enter` | Expand a folded thread; on Overview, jump to the thread's line in Files |

### Files tab

| Key | Action |
| --- | ------ |
| `J` / `K` | Next / previous file |
| `n` / `N` | Next / previous change |
| `Tab` | Switch between the file list and the diff |
| `v` | Select lines; `j` / `k` extend, `c` comments, `esc` cancels |
| `enter` | On a file that is too large or generated: load its diff anyway |
| `o` | Open the selected file in the browser |

## Builds hub

See [Pipelines](pipelines.md) for details.

| Key | Action |
| --- | ------ |
| `1` – `3` | Recent / All / Runs. On a pipeline's runs: `1` Runs, `2` Branches |
| `Tab` / `Shift+Tab`, `h` / `l`, `[` / `]` | Next / previous view |
| `/` | [Search](pipelines.md#searching): pipeline names, or `state:`, `branch:`, `pipeline:`, `by:@me` for runs |
| `enter` | Open the pipeline's runs, or the run |
| `v` | Open the pipeline's latest run |
| `n` | [Run the pipeline](pipelines.md#running-a-pipeline) |
| `o` | Open in the browser |
| `esc` | From a pipeline's runs: back to the hub |

## Pipeline run

| Key | Action |
| --- | ------ |
| `1` / `2` | Summary / Logs tab (`[` / `]` also switch) |
| `j` / `k` | Summary: next / previous job. Logs: next / previous job or step, or scroll the log |
| `enter` | Summary: open the job's logs. Logs: go to the next pane |
| `Tab` / `Shift+Tab` | Logs: move between jobs, steps and the log |
| `F` | Follow the log of a running step |
| `n` | Run the pipeline again on the same branch |
| `o` | Open the run, or the selected job and step, in the browser |
| `esc` | Back a pane, then back to the list |

## Releases hub

See [Releases](releases.md) for details.

| Key | Action |
| --- | ------ |
| `1` / `2` | Releases / Approvals (`[` / `]` also switch) |
| `Tab` / `Shift+Tab`, `h` / `l` | Move between release pipelines and releases |
| `/` | [Search](releases.md#searching-release-pipelines) release pipelines by name. In the bar: `↑` / `↓` move, `Enter` goes into the match's releases, `Esc` clears |
| `Ctrl+t` | Star / unstar a release pipeline |
| `enter` | Release pipeline: show its releases. Release or approval: open the release |
| `v` | Release pipeline: open its latest release. Release: open it |
| `a` | Approve or reject |
| `n` | Create a release |
| `o` | Open in the browser |
| `esc` | From the releases: back to the release pipelines. In the release pipelines: clear the filter |

## Release

| Key | Action |
| --- | ------ |
| `1` / `2` | Pipeline / Logs tab (`[` / `]` also switch) |
| `h` / `l` | Previous / next stage |
| `a` | Approve or reject the stage's pending approval |
| `d` | Deploy or redeploy the stage |
| `x` | Cancel the stage's deployment |
| `enter` | Open the stage's logs |
| `Tab` / `Shift+Tab`, `j` / `k`, `F` | In Logs: as in a [pipeline run](#pipeline-run) |
| `n` | Create another release |
| `o` | Open in the browser |
| `esc` | Back a pane, then back to the hub |

## Dialogs and pickers

| Key | Action |
| --- | ------ |
| `↑` / `↓` (or `Ctrl+p` / `Ctrl+n`) | Move in a picker |
| *type* | Filter the picker |
| `enter` | Choose / next field |
| `Tab` / `Shift+Tab` | Next / previous form field |
| `@` | Mention a team member in a comment |
| `Ctrl+s` | Save / submit |
| `y` | Confirm |
| `Ctrl+t` | Star / unstar a repository (repository picker) |
| `Ctrl+s` | In the new branch and new tag forms: create it |
| `Tab` | In the pull request search bar: complete the suggested filter |
| `enter` | In a reviewer list: add or remove the highlighted person. On *Create as draft*: toggle it |
| `enter` / `space` | In *Stages to run* or *Deploy automatically*: toggle the highlighted stage. On a checkbox: toggle it |
| `h` / `l` | In the approval dialog: choose Approve or Reject |
| `esc` | Clear the filter, then close |

# Keybindings

Every key in lazyado, grouped by where it works. Press `?` in the app for the same list, and watch the status bar: it only shows keys that apply to what is on screen.

Some keys mean different things in different places. For example, `a` assigns a work item on the Boards hubs but votes in the Pull requests hub. The tables below say which applies where.

- [Everywhere](#everywhere)
- [Share menu](#share-menu)
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
| `y` | [Share](sharing.md) the selected item, or the selected lines: copy its link, ID, path, Markdown and more |
| `Y` | Copy the selected item's link right away |
| `o` | Open the selected item in the browser |
| `Ctrl+f` | [Quick search](getting-started.md#quick-search): `1234`, `!1234`, `#1234` or a link |
| `Ctrl+r` | Reload from Azure DevOps (also retries identifying you) |
| `z` | Full screen (hide the navigation rail) |
| `,` | Dismiss the notification |
| `?` | Help |
| `q` / `Ctrl+c` | Quit |

## Share menu

Opened with `y`. Each format keeps its letter in every hub; a letter that doesn't apply to the item does nothing. See [Sharing](sharing.md).

| Key | Copies |
| --- | ------ |
| `u` | Link (the first row; `Y` copies it too) |
| `p` | Permalink, pinned to a commit (files, lines) |
| `i` | ID or number (work item ID, `!1234`, build number, definition ID, commit ID) |
| `n` | Name (branch, tag, release, pipeline, a pull request's source branch) |
| `f` | Path, with the line range when lines are selected |
| `c` | Code snippet as a Markdown code block, with its link |
| `m` | Markdown link |
| `t` | Title and link |
| `h` | Work item mention: `#1234` |
| `g` | GitHub commit link: `AB#1234` |
| `r` | Run ID of a build |
| `l` | Link to the selected job's logs |
| `d` | Link to the pipeline or release pipeline |
| `j` / `k`, `enter` | Move, then copy the highlighted row |
| `o` | Open the link in the browser |
| `esc` | Close |

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
| `y` | Share: link, ID, `#mention`, `AB#` link, Markdown |

### Board and taskboard

| Key | Action |
| --- | ------ |
| `<` / `>` | Move the card to the previous / next state |
| `h` / `l` | Previous / next column (board), or collapse / expand a row (taskboard) |
| `C` | Collapse / expand every taskboard row |
| Drag and drop | Move a card to another column (Board) or a task to another state column (Sprints taskboard); `esc` cancels |
| Double click | Open a Board card's work item form |

## Work item form

Opened with `v`.

| Key | Action |
| --- | ------ |
| `Tab` | Switch between Details and Discussion |
| `j` / `k` | Scroll, or move between comments |
| `s` `a` `e` `t` `d` `c` `y` | Same as in the lists. With Discussion focused, `e` / `d` act on your selected comment and `+` likes someone else's. |
| `esc` / `q` | Back to the hub |

## Files hub

See [Repos](repos.md) for details.

| Key | Action |
| --- | ------ |
| `Ctrl+p` | [Go to file](repos.md#go-to-file): fuzzy-find any file; in the finder `↑` / `↓` or `Ctrl+p` / `Ctrl+n` move, `enter` opens, `Ctrl+r` rebuilds the index, `esc` closes |
| `l` / `enter` | Expand a folder, or open a file's contents |
| `h` | Collapse a folder, or go to its parent |
| `Tab` | Switch between the tree and the contents |
| `v` | In the contents: select lines; `j` / `k` extend, `y` copies the code (like vim's yank), `s` shares, `Y` copies their link, `esc` cancels |
| `1` / `2` | Markdown files: File (source, to select lines) / Preview tab (`[` / `]` also switch) |
| `esc` | Clear the selection, then back to the tree |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `c` | [Clone](repos.md#cloning): copy the HTTPS or SSH URL, or a `git clone` command |
| `H` | History of the selected file or folder |
| `y` | Share the file or folder, or the selected lines (link, permalink, path, snippet) |
| `o` | Open in the browser, at the cursor line or selection |

## Commits hub

See [Repos](repos.md#commits) for details.

| Key | Action |
| --- | ------ |
| `enter` | Browse the files at that commit |
| `/` | [Filter](repos.md#filtering): text, or `author:` and `path:` |
| `esc` | Clear the filter |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `y` | Share the commit (link, full ID, Markdown) |
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
| `y` | Share the branch (link, name, Markdown) |
| `o` | Open in the browser |

## Tags hub

See [Repos](repos.md#tags) for details.

| Key | Action |
| --- | ------ |
| `n` | [New tag](repos.md#creating-a-tag) |
| `enter` | Browse the files at the tag |
| `H` | The commits up to the tag |
| `/` | Search by name |
| `y` | Share the tag (link, name, Markdown) |
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
| `y` | Share the pull request (link, `!1234`, Markdown, source branch) |
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
| `y` | Share the pull request; in Files, the file or the current line or selection |
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
| `v` | Select lines; `j` / `k` extend, `y` copies the code (like vim's yank), `s` shares, `Y` copies their link, `c` comments, `esc` cancels |
| `enter` | On a file that is too large or generated: load its diff anyway |
| `o` | Open the selected file in the browser |

## Builds hub

See [Pipelines](pipelines.md) for details.

| Key | Action |
| --- | ------ |
| `1` – `3` | Recent / All / Runs. On a pipeline's runs: `1` Runs, `2` Branches |
| `Tab` / `Shift+Tab`, `h` / `l`, `[` / `]` | Next / previous view |
| `/` | [Search](pipelines.md#searching) as you type (Azure DevOps is asked after a short pause): pipeline names, or `state:`, `branch:`, `pipeline:`, `by:@me` for runs. `Enter` closes the bar |
| `enter` | Open the pipeline's runs, or the run |
| `v` | Open the pipeline's latest run |
| `e` | Open the run's logs at its first error |
| `n` | [Run the pipeline](pipelines.md#running-a-pipeline) |
| `y` | Share the pipeline (link, definition ID, name) or the run (link, build number, run ID) |
| `o` | Open in the browser |
| `esc` | From a pipeline's runs: back to the hub |

## Pipeline run

| Key | Action |
| --- | ------ |
| `1` / `2` | Summary / Logs tab (`[` / `]` also switch) |
| `j` / `k` | Summary: next / previous stage or job, in the focused pane. Logs: next / previous job or step in the tree, or scroll the log |
| `Tab` / `Shift+Tab` | Summary: switch between Stages and Jobs. Logs: switch between the jobs tree and the log |
| `h` / `l` | Summary: go to Stages / Jobs. Logs tree: collapse / expand a job (or go to its job / first step); in the log, `h` goes back to the tree |
| `enter` | Summary: open the stage's or job's logs. Logs tree: expand the job, or go to the log |
| `e` / `E` | Next / previous error. Summary: open the logs at the first error |
| `C` | Logs: fold or unfold every group in the log |
| `F` | Follow the log of a running step |
| `n` | Run the pipeline again on the same branch |
| `y` | Share the run: link, build number, run ID, the selected job's logs, the pipeline |
| `o` | Open the run, or the selected job and step, in the browser |
| `esc` | Back a pane, then back to the list |

## Releases hub

See [Releases](releases.md) for details.

| Key | Action |
| --- | ------ |
| `1` / `2` | Releases / Approvals (`[` / `]` also switch) |
| `Tab` / `Shift+Tab`, `h` / `l` | Move between release pipelines and releases (or approvals) |
| `/` | [Search](releases.md#searching-release-pipelines) release pipelines by name. In the bar: `↑` / `↓` move, `Enter` goes into the match's releases, `Esc` clears |
| `Ctrl+t` | Star / unstar a release pipeline |
| `enter` | Release pipeline: show its releases. Release or approval: open the release |
| `v` | Release pipeline: open its latest release. Release: open it |
| `a` | Approve or reject |
| `A` | Approvals: switch between the selected pipeline's approvals and every pipeline's |
| `n` | Create a release |
| `y` | Share the release pipeline, release or approval's release |
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
| `<` / `>` | Previous / next stage, also in Logs |
| `enter` | Open the stage's logs |
| `Tab` / `Shift+Tab`, `j` / `k`, `F` | In Logs: as in a [pipeline run](#pipeline-run) |
| `n` | Create another release |
| `y` | Share the release: link, name, stages, its release pipeline |
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
| `y` | Confirm (in confirmation dialogs) |
| `Ctrl+t` | Star / unstar a repository (repository picker) |
| `Ctrl+s` | In the new branch and new tag forms: create it |
| `Tab` | In the pull request search bar: complete the suggested filter |
| `enter` | In a reviewer list: add or remove the highlighted person. On *Create as draft*: toggle it |
| `enter` / `space` | In *Stages to run* or *Deploy automatically*: toggle the highlighted stage. On a checkbox: toggle it |
| `h` / `l` | In the approval dialog: choose Approve or Reject |
| `esc` | Clear the filter, then close |

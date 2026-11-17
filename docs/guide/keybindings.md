# Keybindings

Every key in lazyado, grouped by where it works. Press `?` in the app for the same list, and watch the status bar: it only shows keys that apply to what is on screen.

Some keys mean different things in different places. For example, `a` assigns a work item on the Boards hubs but votes in the Pull requests hub. The tables below say which applies where.

- [Everywhere](#everywhere)
- [Navigation](#navigation)
- [Boards hubs](#boards-hubs)
- [Work item form](#work-item-form)
- [Pull requests list](#pull-requests-list)
- [Pull request review](#pull-request-review)
- [Dialogs and pickers](#dialogs-and-pickers)

## Everywhere

| Key | Action |
| --- | ------ |
| `L` | Backlogs hub |
| `B` | Boards hub (Kanban) |
| `S` | Sprints hub (taskboard) |
| `P` | Pull requests hub |
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
| `Tab` | In the pull request search bar: complete the suggested filter |
| `enter` | In a reviewer list: add or remove the highlighted person. On *Create as draft*: toggle it |
| `esc` | Clear the filter, then close |

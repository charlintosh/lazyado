# Boards

The *Boards* section has three hubs, matching Azure DevOps:

| Hub | Key | What it is |
| --- | --- | ---------- |
| **Backlogs** | `L` | A grid of backlog items (PBIs, user stories, bugs) with their child tasks |
| **Boards** | `B` | The Kanban board: one column per workflow state |
| **Sprints** | `S` | The sprint taskboard, plus a sprint backlog view (`[` / `]` switch between them) |

Switch hubs with `L`, `B` and `S`. On Sprints, `[` / `]` switch between the taskboard and the sprint backlog; the pane title names the other view.

- [Filtering](#filtering)
- [Working with work items](#working-with-work-items)
- [The work item form](#the-work-item-form)
- [Discussion and comments](#discussion-and-comments)
- [Moving cards](#moving-cards)
- [The Sprints taskboard](#the-sprints-taskboard)

## Filtering

Filters sit where Azure DevOps puts them. On **Sprints**, the sprint and person pickers are in the hub header. Everything else is in the **filter bar** under the hub header, which `f` shows and hides.

| Key | Filter |
| --- | ------ |
| `1` | Iteration / sprint |
| `2` | Person (assigned to) |
| `3` | State |
| `4` | Area |
| `/` | Keyword: matches ID, title and tags |
| `x` | Clear the keyword, person, state and area filters |

Pressing a filter key opens the filter bar if needed and drops the picker down under its field. Type to narrow the list, then press `enter` to choose. The funnel icon shows how many filters are active.

The **Person** list is built from the people actually assigned in the loaded sprint, team and area (with counts), plus team members with nothing assigned, `@Me` and *Unassigned*. Changing the person filter is instant: it filters what is already loaded.

A query loads at most the 1,000 most recently changed items. When it reaches that cap lazyado tells you, so you can narrow the filters or [raise the cap](configuration.md#large-boards).

Your filter choices are [remembered between runs](configuration.md#what-lazyado-remembers).

## Working with work items

Select an item, then:

| Key | Action |
| --- | ------ |
| `enter` | Open / select |
| `v` | Open the full-screen [work item form](#the-work-item-form) |
| `n` | New backlog item (choose PBI or Bug) |
| `t` | New child task under the selected PBI or Bug |
| `e` | Edit |
| `d` | Delete (asks first) |
| `s` | Change state |
| `a` | Assign to someone, or unassign |
| `c` | Add a comment |
| `y` | [Share](sharing.md): copy the link, ID, `#mention`, `AB#` GitHub link or a Markdown link (`Y` copies the link right away) |
| `h` / `l` | Collapse / expand an item's child tasks |

On wide terminals a side pane shows the selected item: `6` for **Details**, `7` for **Discussion**, and `5` returns to the grid or board.

Your changes update the item in place as soon as Azure DevOps saves them, without reloading the list. When a change moves an item out of the current filters, the list reloads. Changes other people make appear when you press `Ctrl+r` or change a filter.

### Create and edit forms

| Key | Action |
| --- | ------ |
| `Tab` / `Shift+Tab` | Next / previous field |
| `enter` | Next field (a new line in Description and Acceptance Criteria) |
| `↑` / `↓` | Choose in the *Assign to* and *Sprint* pickers; type to filter them |
| `Ctrl+s` | Save |
| `esc` | Cancel |

## The work item form

`v` opens the selected item full screen, laid out like the Azure DevOps form:

- a header with the type, ID, title, assignee, comment count and tags, with the item actions on the right;
- a *State · Reason · Area · Iteration* band, with the **Details** / **Discussion** tabs;
- **Description**, **Acceptance Criteria** and **Discussion** on the left, and **Planning** and **Related work** on the right. Under 100 columns these stack.

Descriptions and comments are rendered as Markdown (headings, lists, code blocks, tables and links). The form supports the same actions as the lists: `s`, `a`, `e`, `t`, `d`, `c` and `y`. It refreshes in place after a change and closes if the item is deleted.

| Key | Action |
| --- | ------ |
| `Tab` | Switch between Details and Discussion |
| `j` / `k` | Scroll, or move between comments in Discussion |
| `esc` / `q` | Back to the hub |

## Discussion and comments

`c` adds a comment. Type `@` to mention a team member: pick with `↑` / `↓` and press `enter` or `Tab`, then `Ctrl+s` to post.

Coming back to an item shows its comments at once. If they were loaded more than 30 seconds ago they refresh in the background, so a new comment from someone else appears a moment later. After you add, edit, delete or like a comment, the discussion always reloads.

Comments look the same as [pull request comments](pull-requests.md#comments): each one is a rounded card with the author, date and likes in the top edge. Your own comments are marked **(you)**, and edited ones say **edited**. Under about 36 columns the cards give way to a thin line on the left so the text keeps its width.

With Discussion focused (`7`, or `Tab` in the form), `j` / `k` move between comments. The focused card's border turns blue, and its bottom edge lists what you can do with it:

| Key | Action | On |
| --- | ------ | -- |
| `e` | Edit | your comments |
| `d` | Delete (asks first) | your comments |
| `+` | Like or unlike | other people's comments |

Likes show as `♥ 2`, or `♥ 2 · you` when one of them is yours. On anyone else's comment, `e` and `d` show a notice instead. See [Who you are](configuration.md#who-you-are).

## Moving cards

On the **Boards** and **Sprints** hubs, `<` and `>` move the selected card to the previous or next workflow state. On the board, `h` / `l` move between columns and `j` / `k` between cards.

You can also **drag a card with the mouse** onto another column, on the Board and on the Sprints taskboard:

- Under the pointer a card lightens and shows a ⠿ grip.
- While you drag, the card's place shows a dotted outline (*moving…*), the column under the pointer is tinted with a dashed preview of the card at its top (*↳ drop to set Committed*), the column counts show the change (`2 → 3`), and a label follows the pointer. The status bar reads **MOVING**.
- Releasing over the card's own column, outside any column, or over a state the item's type doesn't have (for example a Bug over a column only Product Backlog Items use) does nothing; the label says why.
- On the Board, when not every column fits, hold the pointer at the left or right edge for half a second to scroll to the hidden columns.
- `esc` cancels a drag. On the Board, a double click opens the work item form.

The card moves right away and shows *Saving…*, then *✓ Moved*. If Azure DevOps rejects the change, the card goes back, marked *✕ Not saved*, and the error appears in the top bar.

Board cards show the ID and type with the effort, the title on two lines, up to two tags, and the assignee with the task progress. The column already names the state, so the card doesn't repeat it.

## The Sprints taskboard

The taskboard has one row per backlog item and one column per task state.

### Collapsing rows

Rows collapse like they do in Azure DevOps. A row starts open if any of its tasks is still pending, and collapsed if every task (or the backlog item itself) is done. A collapsed row is a single line: the backlog item, its done count, and a task count under each state column.

- `h` / `l` on a backlog item collapses or expands its row; `C` toggles every row.
- With the mouse, click `▸` / `▾`, a collapsed row, or the *Collapse all* header.

Your choices last for the session, and a row never collapses on its own when its last task finishes.

### Drag and drop

You can drag a task card with the mouse and drop it on another state column (any row of that column, or its header) to change the task's state.

- **Hover:** the card under the pointer lifts and shows a grip `⠿`.
- **While dragging:** the original slot stays as a dashed *moving…* ghost. A chip by the pointer names the destination, a dashed phantom card waits in the target cell, and the column header previews the new counts (`Done 2 → 3`). `esc` cancels.
- **Drops that change nothing** (the same column, outside the board) show no phantom. The chip says why, and releasing cancels.
- **After the drop**, the card shows the same *Saving… / ✓ Moved / ✕ Not saved* feedback as `<` / `>` on the taskboard.

A plain click just selects a card. Dragging works between the state columns on screen, and you can drop onto a collapsed row's column. Hover effects need a terminal that reports mouse motion; most do.

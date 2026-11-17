# Pull requests

The *Repos › Pull requests* hub (`P`) lists the pull requests of a repository. You can open any of them to review it: read the description and policies, browse the changed files, comment, and vote.

- [The pull request list](#the-pull-request-list)
- [Reviewing a pull request](#reviewing-a-pull-request)
- [Comments](#comments)
- [The Files tab](#the-files-tab)
- [Voting](#voting)

## The pull request list

### Choosing a repository

Pull requests are listed one repository at a time. The first time you open the hub, lazyado asks which repository to show. A project with a single repository picks it for you.

Press `r` to switch. The picker is searchable, includes an explicit **All repositories** option, and lists favorites first: repositories you starred in Azure DevOps, plus ones you star locally with `Ctrl+t`. Your choice is [remembered between runs](configuration.md#what-lazyado-remembers).

### Views

| Key | View |
| --- | ---- |
| `1` | **Mine**: pull requests you created or are a reviewer on |
| `2` | **Active** |
| `3` | **Completed** |
| `4` | **Abandoned** |

`Tab` / `Shift+Tab`, `h` / `l` and `[` / `]` also cycle through the views. Each row shows the reviewers' votes. On wide terminals a preview pane shows the selected pull request's policies, reviewers and description.

| Key | Action |
| --- | ------ |
| `enter` | Open the review |
| `a` | Vote |
| `c` | Comment on the pull request |
| `o` | Open in the browser |

## Reviewing a pull request

The review has two tabs: `1` **Overview** and `2` **Files**. `[` / `]` switch between them, and `esc` goes back to the list.

The header shows the ID and title, the status, who wants to merge which branch into which, and the number of active comments.

**Overview** shows:

- whether the pull request has merge conflicts and whether its required policies are met;
- **Policies**: build validation, required reviewers and custom status checks, split into required and optional;
- **Reviewers** and their votes;
- **Work items** linked to the pull request. Press `w` to open one in the [work item form](boards.md#the-work-item-form); `esc` brings you back to the review;
- **Tags**, the **Description** (rendered as Markdown), and the **Activity**: every comment thread, plus system events such as votes and updates.

`o` opens the pull request in the browser.

## Comments

Every comment can be focused on its own, both in Overview and in the Files diff. Press `j` / `k` to move between them. Each comment sits in its own rounded box, with the author, date and likes in the top edge, and replies are indented under the comment they answer. The focused comment's border turns blue, and its bottom edge lists the actions you can take on it:

| Key | Action | On |
| --- | ------ | -- |
| `r` | Reply to the comment | any comment |
| `e` | Edit | your comments |
| `d` | Delete (asks first) | your comments |
| `+` | Like or unlike | other people's comments |
| `s` | Resolve or reopen the thread | any thread |
| `o` | Open the thread in the browser | any comment |
| `c` | Start a new comment on the pull request | — |

Likes show as `♥ 2`, or `♥ 2 · you` (highlighted) when one of them is yours. Type `@` while writing to mention a team member. Mentions in existing comments show the person's name.

Each thread is labelled by where it is: `file.go:12`, `Line 12` or `Lines 12-18` in the diff, or *General comment*. Resolved and closed threads fold to a single bordered row (status, location, comment count and the last author). `enter` expands a folded thread. On terminals narrower than about 60 columns, comments drop the boxes and are grouped by a thin line on the left instead, so the text keeps its width.

On Overview, `enter` on a comment in a file thread jumps to that line in the Files tab.

## The Files tab

The Files tab shows the changed files on the left and an inline diff of the selected file on the right.

| Key | Action |
| --- | ------ |
| `J` / `K` | Next / previous file |
| `n` / `N` | Next / previous change |
| `Tab` | Switch between the file list and the diff |
| `j` / `k` | Move through the diff, one code line or comment at a time |
| `g` / `G`, `Ctrl+d` / `Ctrl+u` | Top / bottom, half page down / up |

The file list flags files with active comments. Comment threads appear under the line they belong to. Threads on the whole file, or on lines that are no longer in the diff, are listed above it. Positions follow the latest update to the pull request.

### Commenting on code

- `c` on a code line comments on that line.
- `v` starts a selection. Extend it with `j` / `k` (within the same hunk), then press `c` to comment on the whole range, or `esc` to cancel. Range threads appear under their last line, with the covered lines marked.

When the cursor is on a comment rather than a code line, the [comment keys](#comments) apply instead.

### Large and generated files

Very large files, such as `package-lock.json`, show **This file is too large to show** instead of a diff, and the file list marks them *too large*. Press `enter` to load the diff anyway, or `o` to open the file in Azure DevOps. Files the repository's `.gitattributes` marks as generated are collapsed the same way. Comments on a collapsed file are still listed and can be focused. The limits are [configurable](configuration.md#large-files-in-diffs).

## Voting

`a` opens the vote picker: **Approve**, **Approve with suggestions**, **Wait for author**, **Reject**, or **Reset vote**. If required policies are not met yet, the picker tells you. You can only vote on active pull requests, and only when lazyado knows [who you are](configuration.md#who-you-are).

# Pull requests

The *Repos › Pull requests* hub (`P`) lists the pull requests of a repository. You can open any of them to review it: read the description and policies, browse the changed files, comment, and vote.

- [The pull request list](#the-pull-request-list)
- [Searching](#searching)
- [Creating a pull request](#creating-a-pull-request)
- [Reviewing a pull request](#reviewing-a-pull-request)
- [Builds](#builds)
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
| `b` | [Open its build](#builds) |
| `y` | [Share](sharing.md): link, `!1234`, Markdown, title, source branch (`Y` copies the link) |
| `o` | Open in the browser |

## Searching

Press `/` to open the search bar above the list. What you type filters the list as you type, matching the ID, title, branches, author and repository. Matches are highlighted, and the bar shows how many of the loaded pull requests match.

Azure DevOps can't search pull requests by text, so text only filters what's loaded: up to the 200 most recent pull requests in the current view. These filters go further, because Azure DevOps can search by them:

| Filter | Finds pull requests | Searched |
| ------ | ------------------- | -------- |
| `author:name` or `author:@me` | created by someone | on the server |
| `reviewer:name` or `reviewer:@me` | with someone as a reviewer | on the server |
| `source:branch` | from a branch | on the server |
| `target:branch` | into a branch | on the server |
| `is:draft` | that are drafts | in the loaded list |

When the bar is empty, or you start typing a filter, suggestions appear under it, and `Tab` completes the first one, including team members' full names and branch names. `Enter` sends the filters to Azure DevOps and the bar says the results came from the *server*. Press `Enter` again to open the selected pull request. Filters combine with the Mine / Active / Completed / Abandoned views.

If you type an ID that isn't in the current view, `Enter` opens that pull request anyway. `Esc` clears the search, and a second `Esc` closes the bar.

## Creating a pull request

Press `n` (or *+ New pull request* in the header) to open the form. If the hub shows all repositories, choose the repository first.

1. **Pick the branches.** The target starts on the repository's default branch. Branches that already have an open pull request are marked `!1234 open`.
2. **Check the line under them.** It compares the branches the way Azure DevOps does:
   - `✓ 3 commits ahead, 0 behind main · 7 files changed`: ready to go.
   - `! … 12 behind main`: allowed, but you may want to update your branch first.
   - `✕ An active pull request already exists: !4826 …`: `Ctrl+s` opens that one instead.
   - `✕ Nothing to merge` or the same branch on both sides: the form won't create it.
3. **Write it up.** The title is filled in from the commit message when there is a single commit, otherwise from the branch name. The description starts from the repository's pull request template, if it has one (`.azuredevops/pull_request_template.md` and the other places Azure DevOps looks, including branch-specific templates). A work item ID in the branch name, as in `feature/1043-guest-checkout`, is linked for you. Anything you type yourself is never overwritten.
4. **Add reviewers, work items, tags, or mark it as a draft.** In a reviewer list, type to filter and press `Enter` to add or remove someone. Required reviewers and build policies from branch policies are still added by Azure DevOps.
5. **Press `Ctrl+s`.** The pull request is created and its review opens. If Azure DevOps refuses, the error is shown over the form and everything you typed is kept.

## Reviewing a pull request

The review has two tabs: `1` **Overview** and `2` **Files**. `[` / `]` switch between them, and `esc` goes back to the list.

The header shows the ID and title, the status, who wants to merge which branch into which, and the number of active comments.

**Overview** shows:

- whether the pull request has merge conflicts and whether its required policies are met;
- **Policies**: build validation, required reviewers and custom status checks, split into required and optional. When one of them points to a build, the title shows `b` ([Builds](#builds));
- **Reviewers** and their votes;
- **Work items** linked to the pull request. Press `w` to open one in the [work item form](boards.md#the-work-item-form); `esc` brings you back to the review;
- **Tags**, the **Description** (rendered as Markdown), and the **Activity**: every comment thread, plus system events such as votes and updates.

`o` opens the pull request in the browser, and `y` [shares](sharing.md) it.

## Builds

A build validation policy, or a status check posted by a pipeline, points to the build that produced it. Press `b`, in the review or on the selected pull request in the list, to open that build in the [run view](pipelines.md#a-run): its stages, jobs, errors and logs. With several builds, a picker lists them with their state. `esc` in the run view goes back to the pull request.

Status checks from other tools (SonarQube, for example) don't point to a build and aren't listed.

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

The Files tab shows the changed files on the left and an inline diff of the selected file on the right. Code is syntax highlighted by file type, with the same colors as code blocks in descriptions and comments.

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

### Sharing code

`y` [shares](sharing.md#sharing-lines-of-code) the file, or the line under the cursor, or the selected lines: a link to the file in the pull request, a permalink to the lines at the pull request's latest commit, the path with its line range, or the code as a Markdown snippet.

### Large and generated files

Very large files, such as `package-lock.json`, show **This file is too large to show** instead of a diff, and the file list marks them *too large*. Press `enter` to load the diff anyway, or `o` to open the file in Azure DevOps. Files the repository's `.gitattributes` marks as generated are collapsed the same way. Comments on a collapsed file are still listed and can be focused. The limits are [configurable](configuration.md#large-files-in-diffs).

## Voting

`a` opens the vote picker: **Approve**, **Approve with suggestions**, **Wait for author**, **Reject**, or **Reset vote**. If required policies are not met yet, the picker tells you. You can only vote on active pull requests, and only when lazyado knows [who you are](configuration.md#who-you-are).

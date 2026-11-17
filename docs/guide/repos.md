# Repos

The *Repos* group in the navigation rail has these hubs, next to [Pull requests](pull-requests.md) (`P`):

- **Files** (`E`): browse a repository's code. Read-only.
- **Commits** (`H`): the history of a branch, tag, file or folder. Read-only.
- **Branches** (`V`): every branch with how far ahead and behind the default branch it is. Create and delete branches.
- **Tags** (`T`): the repository's tags. Create tags.

Contents:

- [Choosing a repository](#choosing-a-repository)
- [Files](#files)
- [Cloning](#cloning)
- [Commits](#commits)
- [Branches](#branches)
- [Tags](#tags)

## Choosing a repository

The Repos hubs show one repository at a time. The first time you open one, lazyado picks the repository you last used there, then the one chosen in the Pull requests hub, then the project's only repository. If none of those apply, it asks.

Press `r` to switch. The picker lists your starred repositories first, the same way as in the Pull requests hub. Choosing a repository here doesn't change the Pull requests hub, and the choice is [remembered between runs](configuration.md#what-lazyado-remembers).

## Files

The hub has two panes: the folder tree on the left and the selected item on the right. On narrow terminals only the focused pane is shown.

- **A folder** shows its files and subfolders with the last commit that changed each one: when, the short commit ID and the commit message. If the folder has a `README.md`, it is rendered below the list.
- **A file** shows its contents with line numbers and syntax highlighting, the same way as the [pull request diff](pull-requests.md#the-files-tab): `j` / `k` move a line cursor, `v` selects lines, `y` [shares](sharing.md#sharing-lines-of-code) the line or selection, and `o` opens the file in the browser at that line. Markdown files have two tabs: **Preview** (rendered, the default) and **File** (the highlighted source, where you can select and share lines). `1` / `2` or `[` / `]` switch between them, and the tab you pick stays for the next Markdown file. Binary files and files over the [`diff.max_file_kb` limit](configuration.md#large-files-in-diffs) aren't shown; `o` opens them in the browser.

Folders load when you open them, and a file loads once the cursor rests on it.

The header shows the branch you're browsing. It starts on the repository's default branch; press `b` to pick another branch or a tag. Typing filters the list.

| Key | Action |
| --- | ------ |
| `j` / `k` | Move in the tree, or move the line cursor in a file |
| `g` / `G`, `Ctrl+d` / `Ctrl+u` | Top / bottom, half page down / up |
| `l` / `enter` | Expand a folder, or open a file's contents |
| `h` | Collapse a folder, or go to its parent |
| `Tab` | Switch between the tree and the contents |
| `v` | In a file's contents: select lines; `j` / `k` extend |
| `1` / `2` | Markdown files: File (source) / Preview tab (`[` / `]` also switch) |
| `esc` | Clear the selection, then back to the tree |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `H` | [History](#commits) of the selected file or folder |
| `y` | [Share](sharing.md) the selected item, or the current line or selected lines: link, permalink, path, code snippet, Markdown |
| `o` | Open the selected item in the browser, at the branch or tag you're browsing (and at the cursor line or selection, from a file's contents) |
| `Ctrl+r` | Reload |

## Cloning

Press `c` (or *Clone* in the header) to copy one of these:

- the **HTTPS** clone URL;
- the **SSH** clone URL (it needs a public key in Azure DevOps under *User settings › SSH public keys*);
- a ready-to-paste `git clone` command.

## Commits

The Commits hub lists the commits of the branch or tag in the header, newest first, 100 at a time; more load as you scroll. Each row shows the message, the short commit ID, the author and when.

On wide terminals a preview pane shows the selected commit's full message, author, date, committer and parents (two for a merge), and the files it changed (`A` added, `M` edited, `D` deleted, `R` renamed).

Press `H` in the Files hub to open the history of the selected file or folder. The pane title shows `· /path:…`, and `esc` goes back to every commit.

Azure DevOps doesn't draw a commit graph through its API, so lazyado lists commits without one.

### Filtering

Press `/` to filter. What you type filters the loaded commits as you type, matching the message, author and commit ID. These go to Azure DevOps when you press `Enter`:

| Filter | Finds commits |
| ------ | ------------- |
| `author:name` | by an author |
| `path:/src/cart.go` | that changed a file or anything in a folder |

`esc` in the list clears the filter.

| Key | Action |
| --- | ------ |
| `enter` | Browse the files at that commit, in the Files hub |
| `/` | Filter |
| `esc` | Clear the filter |
| `b` | Choose a branch or tag |
| `r` | Choose the repository |
| `y` | [Share](sharing.md) the commit: link, full commit ID, Markdown |
| `o` | Open the commit, with its diff, in the browser |

## Branches

The Branches hub lists the repository's branches in three views:

| Key | View |
| --- | ---- |
| `1` | **Mine**: branches you created or starred, plus the default branch |
| `2` | **All** |
| `3` | **Stale**: branches without a commit in the last 3 months |

`Tab` / `Shift+Tab`, `h` / `l` and `[` / `]` also switch views. Each row shows the last commit, its author and when, how many commits the branch is **behind | ahead** of the default branch, and its active pull request. The default branch is marked *Default*, and locked branches *Locked*.

`/` filters by name as you type. `Enter` keeps the filter and `esc` clears it.

| Key | Action |
| --- | ------ |
| `n` | [New branch](#creating-a-branch) |
| `d` | [Delete the branch](#deleting-a-branch) |
| `enter` | Browse the branch's files |
| `H` | The branch's commits |
| `p` | New pull request from the branch |
| `y` | [Share](sharing.md) the branch: link, name, Markdown |
| `o` | Open the branch in the browser |
| `r` | Choose the repository |

### Creating a branch

Press `n` (or *+ New branch* in the header). The form has three fields:

1. **Name**, for example `feature/1102-saved-carts`. lazyado checks it against Git's rules and the existing branches before sending it. A work item ID in the name is linked when you open a pull request from the branch.
2. **Based on**: a branch or tag. It starts on the branch highlighted in the list; type to filter.
3. **Or a commit**: a commit ID to start from instead.

`Tab` moves between fields and `Ctrl+s` creates the branch. If Azure DevOps refuses (for example, you lack the *Create branch* permission), the error is shown over the form and what you typed is kept. The new branch is selected in the list. Fetch it locally with `git fetch`, then `git switch` to it.

### Deleting a branch

Press `d` and confirm with `y`. The dialog warns you when the branch has an active pull request, which can't be completed without its source branch. You can't delete the default branch, or a locked branch until it's unlocked in Azure DevOps. Azure DevOps can restore a deleted branch from its Branches page.

Locking, branch policies and branch security stay in the browser (`o`).

## Tags

The Tags hub lists the repository's tags by name, each with the commit it points to and whether it's **annotated** (it has a message, a tagger and a date) or **lightweight** (just a name for a commit). On wide terminals a preview pane shows the selected annotated tag's message, tagger and date.

`/` filters by name as you type. `Enter` keeps the filter and `esc` clears it.

| Key | Action |
| --- | ------ |
| `n` | [New tag](#creating-a-tag) |
| `enter` | Browse the files at the tag |
| `H` | The commits up to the tag |
| `y` | [Share](sharing.md) the tag: link, name, Markdown |
| `o` | Open the tag in the browser |
| `r` | Choose the repository |

### Creating a tag

Press `n` (or *+ New tag* in the header). The form asks for:

1. **Name**, for example `v1.4.0`. It's checked against Git's rules and the existing tags.
2. **Based on**: a branch or tag, starting on the default branch.
3. **Or a commit**: a commit ID to tag instead.
4. **Description**: required. lazyado creates an annotated tag, the same kind the Azure DevOps web UI creates.

`Tab` moves between fields, `Enter` adds a line in the description, and `Ctrl+s` creates the tag. Errors are shown over the form and keep what you typed. The new tag is selected in the list. Fetch it locally with `git fetch --tags`.

Deleting tags stays in the browser (`o`).

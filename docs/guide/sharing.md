# Sharing

One key shares anything in lazyado: **`y`**. Select a work item, pull request, file, lines of code, commit, branch, tag, pipeline, run or release, press `y`, and pick what to copy.

- [The Share menu](#the-share-menu)
- [What each item offers](#what-each-item-offers)
- [Sharing lines of code](#sharing-lines-of-code)
- [When there is no clipboard](#when-there-is-no-clipboard)

## The Share menu

`y` opens a menu with every format the selected item has. Each format shows its letter, its name and a preview of what will be copied.

| Key | Action |
| --- | ------ |
| a format's letter | Copy that format |
| `j` / `k`, `enter` | Move, then copy the highlighted row |
| `Y` | Copy the link |
| `o` | Open the link in the browser |
| `esc` | Close |

Outside the menu, **`Y`** copies the link straight away, and **`o`** opens the same link in the browser.

A notification confirms what was copied, for example *Permalink to lines 4–6 copied for client.go:4-6*.

## What each item offers

The letter for a format is the same in every hub. Rows only appear when they apply.

| Letter | Format | Offered for |
| ------ | ------ | ----------- |
| `u` | Link | everything |
| `p` | Permalink, pinned to a commit so the lines never move | files, lines in a file, lines in a pull request |
| `i` | ID or number | work items (`1234`), pull requests (`!1234`), commits (full ID), build numbers (`20261117.3`), pipeline and release pipeline definition IDs |
| `n` | Name | branches, tags, releases, pipelines, a pull request's source branch |
| `f` | Path, with the line range when lines are selected | files and lines (`src/cart.go:42-57`) |
| `c` | Code snippet: a Markdown code block followed by its permalink | lines |
| `m` | Markdown link | everything |
| `t` | Title and link | work items, pull requests, commits, runs (with their result), releases (with their stages) |
| `h` | Mention: `#1234` | work items |
| `g` | GitHub commit link: `AB#1234` | work items |
| `r` | Run ID | build runs |
| `l` | Link to the selected job's logs | a run's Logs tab, with a job selected |
| `d` | Link to the pipeline or release pipeline | build runs, releases |

Where to press `y`:

| Hub | Shares |
| --- | ------ |
| Backlogs, Boards, Sprints, the work item form | The selected work item |
| Pull requests | The selected pull request |
| Pull request review | Overview: the pull request. Files: the file, or the current line or selected lines |
| Files | The selected file or folder, or the current line or selected lines in a file |
| Commits, Branches, Tags | The selected commit, branch or tag |
| Builds | The selected pipeline or run |
| Pipeline run | The run; on the Logs tab, also the selected job's logs |
| Releases | The selected release pipeline, release or approval's release |
| Release | The release |

## Sharing lines of code

In the Files hub and in a pull request's Files tab, `y` on the line cursor shares that line. To share several lines, press `v`, extend the selection with `j` / `k`, then press `s` for the menu or `Y` to copy their link at once. `esc` cancels the selection.

To copy just the code, press `y` while lines are selected: like vim's yank, it copies the selected lines as plain text, without a menu, and ends the selection.

- **Link** opens the file at those lines on the branch or tag you're browsing. In a pull request, it opens the file in the pull request's Files tab.
- **Permalink** pins the lines to a commit, so the link keeps pointing at the same code after the branch moves on. In the Files hub that is the last commit that changed the file; in a pull request it is the pull request's latest commit (or its base, for deleted lines).
- **Snippet** copies the code as a Markdown code block, followed by the permalink.

## When there is no clipboard

lazyado copies with `pbcopy` on macOS, `clip` on Windows, and `wl-copy`, `xclip`, `xsel` or WSL's `clip.exe` on Linux. When none of them is available, for example over SSH, it asks your terminal to copy the text instead (OSC 52, supported by most modern terminals and by tmux with `set-clipboard on`) and shows the text in a dialog so you can select it yourself if your terminal doesn't support that.

# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and adheres to Semantic Versioning.

## [Unreleased]

### Added

- feat: Pipelines › Builds hub (`D`, a new *Pipelines* group in the navigation rail) — **Recent** lists the 10 pipelines that ran last with their latest run, **All** lists every pipeline grouped by folder, and **Runs** lists the latest runs of every pipeline; long lists load more as you scroll, a preview pane shows the selected pipeline's last 10 runs, pass rate, median duration and the stages of its latest run, and lists with a queued or running run refresh every 15 seconds
- feat: search pipelines and runs (`/` in the Builds hub) — text filters the loaded list as you type, `Enter` searches every pipeline name on Azure DevOps (with `*` wildcards), and `state:`, `branch:`, `pipeline:` and `by:@me` search runs on Azure DevOps
- feat: a pipeline's runs (`enter` on a pipeline) with **Runs** and **Branches** (latest run per branch) views
- feat: run details (`enter` on a run, `v` for a pipeline's latest run) — **Summary** shows stages and jobs with progress and durations, errors and warnings with where they came from, run details and the commits it includes; **Logs** shows the jobs by stage, the steps of the selected job and the step's log with Azure DevOps' error, warning and section markers, opens on the failed or running step, and follows a running step (`F`); a running run refreshes every 5 seconds
- feat: run a pipeline (`n` in the Builds hub, `n` on a run to run it again on the same branch) — choose a branch or tag and an optional commit, set the parameters read from the pipeline's YAML at that branch or tag, untick the stages to skip, override queue-time variables and turn on system diagnostics; classic pipelines queue a build with the branch and variables, and the new run's logs open once it is queued

- feat: Pipelines › Releases hub (`R`) — release pipelines with an always-visible **Starred** group first (`Ctrl+t`, saved in `state.json`, plus any Azure DevOps favorites), searchable by name (typing filters, `↑` / `↓` move through matches, `Enter` goes into the match's releases, the filter stays as `· /text` in the pane title and `Esc` clears it; Azure DevOps is asked 300 ms after typing only when more pipelines exist than are loaded), `v` opens a pipeline's latest release, and each pipeline's releases with a chip per stage (deployed, deploying, waiting for approval, queued, failed, rejected, partial, canceled, not deployed); loads more as you scroll and refreshes while a release deploys
- feat: Approvals view (`2` in the Releases hub) — pending approvals waiting on you (including group approvals) and on others, with a preview of the release; `a` approves or rejects
- feat: release screen — stages as boxes (`h` / `l`), what the selected stage waits on and who can approve it, its trigger, approvers and last deployment attempt; `a` approves or rejects with a comment (required to reject), `d` deploys or redeploys a stage with warnings about approvals and failed attempts, and the **Logs** tab shows deployment jobs, tasks and logs; refreshes every 5 seconds while deploying
- feat: open a pull request's build (`b` in the review and in the pull request list) — build validation policies (`context.buildId`) and pipeline status checks (`targetUrl` to `_build/results?buildId=`) open in the run view with stages, jobs and logs; several builds show a picker, and `esc` returns to the pull request
- feat: cancel a deployment (`x` on the release screen) — for a stage that is deploying, queued, scheduled or waiting for approval; the dialog explains what stops, takes an optional comment, and the stage can be deployed again afterwards
- feat: create a release (`n`) — pick a branch and build version for each build artifact (succeeded and partially succeeded builds of that branch), a branch for Azure Repos artifacts, which automatic stages should wait for a manual deploy, release-time variables and a description

### Changed

- change: pane titles show the `5` key only on the Boards hubs, where it focuses the main view; the Pull requests, Builds and Releases panes no longer show it

### Fixed

- fix: the selected pull request row is highlighted edge to edge; the spaces between reviewer vote chips no longer break the highlight

## [v1.2.1] - 2026-11-17

Pull requests release: create pull requests from the TUI and search the pull request list.

### Added

- feat: create pull requests (`n` in the Pull requests hub) — pick the source and target branches (the target starts on the default branch, and branches with an open pull request are marked), see commits ahead and behind and the files changed, and write the title, description, required and optional reviewers, linked work items, tags and draft flag; the title comes from the commit or branch name, the description from the repository's pull request template, and a work item ID in the branch name is linked. An existing pull request for the same branches is offered instead of a duplicate, and branches with nothing to merge are refused before Azure DevOps would (1ebe9e5)
- feat: search pull requests (`/` in the Pull requests hub) — text filters the loaded list as you type with matches highlighted; `author:`, `reviewer:`, `source:` and `target:` (with `@me`, completion of names and branches, and `Tab`) search Azure DevOps beyond the loaded list, `is:draft` keeps drafts, and an ID that isn't in the view can still be opened (1ebe9e5)

### Changed

- change: the Pull requests hub's header action is now *+ New pull request*; `o` still opens the selected pull request in the browser (1ebe9e5)

## [v1.2.0] - 2026-11-17

Pull requests release: a Repos › Pull requests hub with full reviews, bordered comment cards with likes across pull requests and work items, pull request quick search, and "too large" handling for lockfiles and other big diffs.

### Added

- feat: Repos › Pull requests hub (`P`) — Mine / Active / Completed / Abandoned views with reviewer votes per row and a preview pane (policies, reviewers, description) (512906b)
- feat: the Pull requests hub lists one repository at a time — `r` opens a searchable repository picker (with an explicit "All repositories" option), the choice is saved in `state.json`, and a project with a single repository selects it automatically (512906b)
- feat: favorite repositories — repositories starred in Azure DevOps and local favorites (`Ctrl+t` in the picker, saved in `state.json`) are listed first (512906b)
- feat: pull request review screen — description, merge and policy status, policies and custom status checks, reviewers, linked work items, tags and comment threads (512906b)
- feat: Files tab — changed-file list and an inline diff computed locally from the two file versions, with threads shown under the lines they belong to (512906b)
- feat: multi-line comments in the Files tab — `v` starts a selection, `j` / `k` extend it within the hunk, `c` comments on the range; range threads show under their last line with the covered lines marked (512906b)
- feat: `@` mentions in pull request comments and replies, with the same team member picker as work item comments; mentions in threads show the person's name (512906b)
- feat: `w` in the review opens a linked work item in the work item view and `Esc` returns to the review (512906b)
- feat: edit (`e`) and delete (`d`) your own pull request comments; `j` / `k` on the Overview now move comment by comment and mark the focused one (512906b)
- feat: vote, comment (on the pull request or on a diff line), reply, and resolve or reopen threads (512906b)
- feat: `P`, `o`, `a`, `r`, `s`, `J` / `K` and `n` / `N` in the help screen and status bar hints (512906b)
- feat: quick search (`Ctrl+f`) finds pull requests too — `1234` looks up a work item and a pull request in parallel and lets you choose when both exist, `!1234` and `#1234` pick one kind, and pasted `dev.azure.com` / `visualstudio.com` links open the work item or pull request (links from another organization or project are refused with an explanation) (512906b)
- feat: every pull request comment is its own focus stop, on Overview and in the Files diff (`j` / `k` step from a code line into the comments under it and back out); the focused comment lists only the actions you can take on it (512906b)
- feat: like or unlike other people's pull request comments with `+`; likes show as `♥ n` (512906b)
- feat: `r` replies to the focused comment, and `o` with a comment focused opens its thread in the browser (512906b)
- feat: resolved and closed threads fold to a single bordered row until `enter` expands them (512906b)
- feat: work item comments use the same bordered card as pull request comments, in the Discussion pane and the work item view; edited comments say `edited` in the card's top edge (512906b)
- feat: like or unlike other people's work item comments with `+`; likes show as `♥ n` (`♥ n · you` when one is yours) (512906b)
- feat: very large files in a pull request (`package-lock.json` and the like) show "This file is too large to show" instead of a diff, with `enter` to load it anyway and `o` to open the file in the browser; the file list marks them "too large". The limits live in `config.yaml` (`diff.max_file_lines`, `diff.max_changed_lines`, `diff.max_file_kb`) with no file names hardcoded (512906b)
- feat: files the repository's `.gitattributes` marks `linguist-generated`, `-diff` or `binary` are collapsed in the Files tab, and `diff.collapse` in `config.yaml` adds your own patterns (512906b)

### Changed

- change: threads are labelled by their anchor (`file:line`, `Line 42`, `Lines 42-48` or `General comment`) instead of "Thread", and every comment gets its own rounded border instead of one box per thread: the author, date and likes sit in the top edge, the focused comment's border turns blue with its actions in the bottom edge, and replies are indented; under 36 columns comments fall back to a thin rail, so cards also fit the Discussion side pane (512906b)
- change: pull request loading states (list, comments, policies, files, diff) use the shared Bubbles spinner, and the app uses one spinner style (`MiniDot`) everywhere (512906b)
- docs: the README is now a short overview; the user guide moved to `docs/guide/` (getting started, configuration, boards, pull requests, keybindings, troubleshooting), contributor docs to `docs/development/`, and build, test and review conventions to `CONTRIBUTING.md` (512906b)

### Fixed

- fix: every thread on a file shows in the Files tab — the diff now includes the lines that carry a thread even when they are far from a change, thread positions are tracked to the latest update, and file-level or stale threads are listed above the diff instead of after its last line (512906b)
- fix: `e` / `d` in the Files diff act on the focused comment instead of your latest comment in the thread (512906b)
- fix: files over the download limit are no longer silently cut off, which produced a wrong diff (512906b)

## [v1.1.0] - 2026-11-16

Sprints taskboard release: drag and drop, optimistic task moves and collapsible rows.

### Added

- feat: drag and drop on the Sprints taskboard — drag a task card onto another state column (any row of it, or its header) to change the task's state; `Esc` cancels (84219aa)
- feat: drag-and-drop hints — hover lifts the card with a grip `⠿`, a dashed ghost keeps its original slot, a chip beside the pointer names the destination, a dashed phantom card waits in the target cell, the target column is tinted and its header previews the counts (`▸ Done 2 → 3`); drops that change nothing say why (84219aa)
- feat: optimistic task moves — a dropped or `<` / `>` moved task moves at once and shows `Saving…`, then `✓ Moved`, or returns in red with the error in the top bar if Azure DevOps rejects it (84219aa)
- feat: collapsible Sprints rows, like Azure DevOps — rows with pending tasks start open, rows whose tasks are all done (or whose backlog item is done) start collapsed to one line with per-column task counts; `h` / `l` on a backlog item, `C` for every row, or click `▸` / `▾` / the header (84219aa)
- feat: `drag`, `release` and `C` in the help screen and status bar hints (84219aa)

### Changed

- perf: one drag motion event through the whole app went from ~10.4 ms / 10.8 MB to ~3.3 ms / 1.1 MB — panels share one `*styles.Styles` (the model shrank from 227 KB to 66 KB), read-only `App` helpers take `*App`, layout and the page header are computed once per message, `Block` / `Pane` skip Lip Gloss wrapping for content that already fits, and the drag chip is spliced into the rows it covers (`widgets.Splice`) (84219aa)
- change: hover uses all-motion mouse tracking, only while the Sprints taskboard is focused with no dialog open (84219aa)
- change: the first column header of the Sprints taskboard is now the `⊟ Collapse all` / `⊞ Expand all` control (84219aa)
- change: a rejected task move on the Sprints taskboard shows a top-bar notice and returns the card instead of opening the error dialog; the Kanban board still uses the dialog (84219aa)

## [v1.0.0] - 2026-11-16

First major release: a new terminal UI laid out like Azure DevOps Boards, and a reworked keyboard UX around it.

### Added

- feat: Azure DevOps style shell — top bar with `org / project / Boards / hub` breadcrumb and sync status, framed panes, and a status bar with a mode pill and contextual hints (3b8f67b)
- feat: Kanban board hub (`B`) with one column per workflow state and `<` / `>` to move cards (3b8f67b)
- feat: sprint taskboard hub (`S`) with one lane per backlog item and task state columns (3b8f67b)
- feat: Details / Discussion preview pane beside the backlog on wide terminals (3b8f67b)
- feat: `Ctrl+u` / `Ctrl+d` half-page scrolling (3b8f67b)
- feat: left navigation rail modelled on Azure DevOps — project, then Boards › Boards / Backlogs / Sprints; collapses to hub keys between 70 and 99 columns, and `z` hides it (55ef8ec)
- feat: hub header with view pivots on the left and controls on the right; Sprints shows `Sprint ▾` (`1`) and `Person ▾` (`2`) (55ef8ec)
- feat: filter bar under the pivots (`f`) with a keyword filter (`/`), dropdown pickers anchored to their field, an active-filter count on the funnel, and clear (`x`) (55ef8ec)
- feat: Person / Assigned to options built from the people assigned in the loaded sprint, team and area, with counts, team members with nothing assigned, `@Me` and `Unassigned` (55ef8ec)
- feat: Sprints › Backlog view next to the Taskboard, switched with `[` / `]` (55ef8ec)
- feat: the work item view (`v`) runs the same actions as the lists — change state, assign, edit, new child task, info, delete — refreshes in place after a change and closes when the item is deleted (55ef8ec)
- feat: Reason field fetched and shown under State (55ef8ec)
- test: layout tests at many terminal sizes, plus behaviour tests for filters, pickers, the work item view and comment ownership (3b8f67b, 55ef8ec)

### Changed

- BREAKING: filter keys follow the Azure DevOps order — `1` Iteration / Sprint, `2` Person (was State), `3` State (was Assigned to), `4` Area; the filter rail is replaced by dropdowns in the header and filter bar (55ef8ec)
- BREAKING: Person and keyword filters apply locally to the loaded items, so changing them no longer reloads; `@Me` no longer narrows the server query when the signed-in user is known, so Iteration = All with Area = All loads the whole project (55ef8ec)
- feat: work item view redesigned after the Azure DevOps form — type stripe, State / Reason / Area / Iteration band with Details / Discussion tabs, Description, Acceptance Criteria and Discussion beside Planning and Related Work; the header sets no background, so it blends into any terminal theme (55ef8ec)
- feat: comments can only be edited or deleted by their author; the signed-in user is the identity behind the PAT, read from `_apis/connectionData` and matched by identity ID, never display name; your comments are marked "you" (55ef8ec)
- feat: backlog grid with Azure DevOps columns (ID, Title tree, State, Assigned To, Effort, Iteration, Tags) that drop responsively on narrow terminals (3b8f67b)
- chore: migrate to Bubble Tea v2, Bubbles v2, Lip Gloss v2 and Glamour v2 (`charm.land/*` modules) (ba96e3c)
- refactor: UI built from Charm components — bubbles `table`, `list`, `viewport`, `help` and `progress`, Lip Gloss `table`, `list` and `tree` (7e4cbbe)
- refactor: modals rebuilt as shared choice, confirm and form dialogs on Bubbles pickers; the app model split into commands, layout, view and update files (55ef8ec)
- style: dark palette based on Azure DevOps; work item type colors and live state colors from the process metadata; rounded borders instead of decorative symbols (3b8f67b, 7e4cbbe)

### Fixed

- fix: responsive at every terminal size — the breadcrumb shortens, toolbar buttons and status hints drop whole, and the board and taskboard show a window of columns (e5bd5ff)
- fix: the @mention picker no longer jumps back to the first person while the cursor blinks (55ef8ec)
- fix: the backlog keeps its selection when the table is resized (55ef8ec)

### Removed

- the navigation rail's filter section, replaced by the header and filter bar dropdowns (55ef8ec)
- the `b` "create git branch" action (55ef8ec)

## [v0.1.6] - 2026-02-25

### Added

- feat: HTML→Markdown→glamour pipeline for rich-text fields (Description, Acceptance Criteria) using `html-to-markdown/v2` and `glamour`
- feat: shared `RenderCommentList` component extracted from CommentsPanel — used by both CommentsPanel and DetailView
- feat: comment focus mode in DetailView — press `Tab` to navigate, select, edit and delete comments inline
- feat: `html-to-markdown` agent skill documenting the rich-text rendering pipeline

### Fixed

- fix: preserve whitespace between @mentions in comment rendering
- fix: comment boxes now shown consistently in both CommentsPanel and DetailView

### Changed

- refactor: ~190 lines of duplicated comment rendering code removed from CommentsPanel in favour of shared `comment_list.go`
- refactor: WorkItem model stores raw HTML (`DescriptionHTML`, `AcceptanceCriteriaHTML`) alongside plain-text fallback

## [v0.1.5] - 2026-02-20

### Added

- feat: add harmonica dependency and implement animated splash screen with loading effects (c9e1bf4)
- feat: update dependencies and enhance WorkItemsPanel styling with improved color scheme and item counter (6fca0f5)
- feat: enhance comments and details panels with improved rendering and scrolling (75a747a)
- feat: enhance error handling in QuickSearchModal and API response parsing (444313d)

### Fixed

- fix: re-release as v0.1.5 — v0.1.4 was cached by the Go module proxy with stale content due to tag reuse

## [v0.1.3] - 2026-02-19

### Added

- refactor: details and filter panels to use viewport for improved rendering (eab0bab)
- feat: add release-bump skill documentation for patch versioning process (66bf6c4)
- feat: add scripts to sync .agents/skills/ to supported folders via symlinks (37e0f68)
- feat: add quick search feature for work items by ID with modal interface (4adf942)
- feat: info modal (9113421)
- docs: add bubbletea skill (63349fc)
- feat: add goreleaser configuration and CHANGELOG.md (35343f1)
- feat: enhance comments functionality with HTML rendering, mentions support, and improved logging (fbf0f8f)
- feat: add notification system with auto-dismiss functionality and integrate into header bar (d2468ac)
- feat: enhance splash screen loading animation (be452d0)
- feat: enhance comments panel styling and structure (f5ffcce)
- feat: add tasks to bug work item types (24c0d1d)
- Refactor documentation: consolidate agent guidelines, update API client patterns, and remove outdated files (a81aed3)

### Fixed

- fix: remove invalid symlink causing go install to fail (d31ad33)
- fix: improve formatting in README for keyboard shortcuts and descriptions (1900f27)

## [v0.1.2] - 2026-02-17

## Added

- **Quick search feature** — Press `Ctrl+F` to open a modal where you can enter a work item ID and instantly jump to its full details. The feature includes:
  - Numeric-only input validation
  - Loading state with spinner during API fetch
  - Error handling with user-friendly messages (not found, network errors, etc.)
  - Automatic navigation to DetailView with comments loaded
  - Works globally from any panel in main view
  - `Esc` to cancel at any time

### Fixed

- N/A

## [v0.1.1] - 2026-02-10

### Added

- **Info modal** — Press `i` on a selected work item to open a modal with options to copy the work item URL or ID to clipboard. Uses `xclip` on Linux for clipboard integration.

### Fixed

- N/A

## [v0.1.0] - 2026-01-26

### Added

- Initial release of `lazyado` — a terminal UI for Azure DevOps Boards using Bubble Tea and Go.
- Command-line entrypoint and TUI located under `cmd/lazyado` and `app/` respectively.
- Core features:
  - List and filter work items
  - View and edit work item details
  - Create common work item types (PBI, Task, Bug)
  - Commenting and basic navigation panels
- Documentation added: `docs/` including architecture and UI guidelines.

### Fixed

- N/A — initial public release.

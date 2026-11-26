# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and adheres to Semantic Versioning.

## [Unreleased]

### Added

- docs: the README shows a screenshot of every view. `task screenshots` regenerates them: a test renders each view with demo data and `scripts/screenshots` draws the frames in xterm.js on Chromium

## [v1.7.0] - 2026-11-20

Releases and focus release: an Azure DevOps-style release list with the deployment that's live on each stage, latest deployments and starred pipelines that always show, Ctrl+t that works while searching and from releases, and a status bar that names what has focus, colored by area, with its keys split into actions and navigation.

### Changed

- feat: the status bar names what has focus, like a game's mode label: the screen, then the focused item by name (`RELEASES › Release-64`, `BOARD › TASK #1051`, `REVIEW !4821 › retry.go`, `RELEASE-64 › STAGE QA`, `EDIT › Resource group`, `› 3 LINES`), colored by area (Boards blue, Repos violet, Pipelines teal) or by what you're doing (editing orange, searching yellow, dragging pink). The keys that act on the focus come first, then, after a `│` and dimmer, the keys that move around (1de247d)
- feat: the releases pane reads like Azure DevOps' release list: **Releases**, **Created** and **Stages** columns, and a chip per stage with its state glyph and full name. The release deployed on a stage now is the bright green chip; earlier deployments, failures, deployments in progress, approvals and canceled or not-deployed stages each have their own color and glyph (c54b7ad)
- feat: release pipelines carry their latest deployment on the second line (`✓ Dev1 · Release-64 · 2h ago`), from one request for the project's recent deployments; starred pipelines show a gold `★`, pending approvals `◐ n` badges, and the Starred, All and Found on Azure DevOps groups their own colored tags (c54b7ad)
- perf: release, approval and pipeline lists render only the selected row on a cursor move; the other rows are kept until the list or its width changes (c54b7ad)

### Fixed

- fix: starred release pipelines' `★` could look out of line where the terminal's font draws it wider than a cell; it now has a two-cell slot on every row, so stars and approval badges keep one column (d5d3c69)
- fix: `Ctrl+t` did nothing in the Releases hub while the search bar was open or once `Enter` had moved to a pipeline's releases. It now stars the highlighted match while searching, and the pipeline whose releases you're looking at from the releases pane, with a notice either way (3e4b254)
- fix: a release pipeline starred from *Found on Azure DevOps*, or starred earlier but beyond the first 200 loaded (Azure DevOps favorites too), never showed in Starred. Starred pipelines that aren't loaded are now fetched by ID in one request, and pages that load later don't list them twice (3e4b254)

## [v1.6.0] - 2026-11-20

Release pipeline editor: see a classic release pipeline's stages, jobs, tasks, settings, variables and history as in Azure DevOps; edit tasks, agent jobs and variables with color-coded changes, pickers for everything Azure fills (never typed), a reviewed diff and a conflict-safe save; and edit a single release's stages that haven't deployed yet.

### Added

- feat: see and edit a release pipeline, like the Azure DevOps release pipeline editor. `e` on a release pipeline opens it read-only with **Pipeline** (artifacts, triggers, stages with their approvals, gates, jobs, variables and retention), **Tasks** (each stage's jobs and tasks as a tree, and the selected job's or task's settings, named and grouped as in Azure DevOps, with settings hidden by other settings left out), **Variables** (value, scope, settable at release time, variable groups) and **History** (revisions and their comments). `e` again starts editing: change settings with pickers for lists and service connections, turn tasks on or off with `space`, change variable values and secrets, undo with `u`, then `Ctrl+s` to review every change as before and after and save it as the next revision with a comment. If someone else saved meanwhile nothing is overwritten, and `r` reloads their revision and reapplies your changes, dropping and naming any they also changed (05a9ee5)
- feat: edit a job's agent settings, as in Azure DevOps: select the job in **Tasks** to change its agent pool (picked from the project's pools), agent specification (the Microsoft-hosted images), demands, parallelism, timeouts, artifact download, OAuth token access and run condition (05a9ee5)
- feat: settings Azure fills (service connections, subscriptions, resource groups, locations, app names and the like) are tagged `from Azure` and never typed: their values are listed by Azure through the task's service connection, as Azure DevOps' editor does, with the saved value marked and the pipeline's variables one `Tab` away. When Azure can't list them the dialog shows Azure's error, with retry, a pipeline variable, the browser or keeping the saved value as the only ways out. Changing a setting others depend on marks them `!` to pick again, and saving waits until they are (05a9ee5)
- feat: the editor uses color and glyphs to tell things apart: orange for unsaved changes (`▎`, counts on tasks, stages, tabs and the header) with the saved value struck through underneath, red `!` for settings to pick again, `●` / `○` for on and off, `▾` for picked and `›` for typed settings, a colored tag per settings group, `$(variables)` in violet, connections, pools and images as chips, `■` per variable scope and `⊘` for read-only (05a9ee5)
- feat: edit a single release (`e` on a release, or on the release screen), like *Edit release* in Azure DevOps: stages that have already deployed are read-only, and the release pipeline stays as it is (05a9ee5)
- feat: `y` in the release pipeline editor copies a task as YAML, or a variable's value (05a9ee5)

## [v1.5.1] - 2026-11-20

Performance and releases patch: lazyado follows Azure DevOps rate limits, sends fewer requests and uses less memory; release logs show one stage at a time and Approvals follow the selected release pipeline.

### Added

- perf: lazyado follows Azure DevOps rate limits. It reads the `X-RateLimit-*` headers, waits as long as `Retry-After` asks (at most 30 s), retries a read up to twice when Azure DevOps blocks it (`429`, `503`), never retries a write, and shows a one-time notice when Azure DevOps starts slowing your requests; `-debug` logs each request's TSTU cost and the session total (a6158c8)
- feat: `boards.max_work_items` (default `1000`) caps a Boards query at the most recently changed items, with a notice when the cap is reached; before, *All* filters could load up to 20,000 items in 100 sequential requests (a6158c8)

### Changed

- perf: at most six requests are in flight at once, and a load replaced by a newer one (Boards filters, pull request views, Builds lists and search, release search, Commits) is cancelled instead of finished and thrown away (a6158c8)
- perf: moving a card, changing state, assigning or editing a work item updates that item from Azure DevOps's reply instead of reloading the whole list. The list still reloads when the item no longer fits the query, and for creating or deleting. Other people's changes show up on `Ctrl+r` or the next filter change (a6158c8)
- perf: Boards start with fewer, parallel requests: the states of every work item type come with the type list (one request instead of one per type, with a per-type fallback), the startup requests run side by side, and work items load in parallel batches (a6158c8)
- perf: Discussion, pull request policies and pull request lists you have already seen show at once when you come back, then refresh in the background if they are older than 30 seconds. Pending or running checks always refresh, your own changes are never served from the cache, and a failed background refresh keeps what is on screen (a6158c8)
- perf: lower memory use. The Go to file index is about five times smaller (34 MB to 6 MB for 50,000 files); the Files hub keeps up to 32 MB of file text instead of every file opened; highlighted code (64 MB) and rendered Markdown (16 MB) are capped by size and drop the least recently used entry instead of starting over; resizing keeps three Markdown renderers instead of one per width; the Builds stage cache is capped (a6158c8)

### Fixed

- fix: a release's Logs tab listed the jobs of every stage at once. It now shows only the stage selected in Pipeline, with a stage strip on top; `<` / `>` switch stages from either tab (0a5c35d)
- fix: the Releases hub's Approvals view listed every release pipeline's approvals together. The release pipeline list now stays on the left, each pipeline shows how many approvals wait on you and on others, the view lists only the selected pipeline's approvals, and `A` switches to every pipeline's (0a5c35d)
- fix: parent titles went missing on boards with more than 200 distinct parents, because they were asked for in one request over the API's 200-ID limit; parents already in the list are no longer fetched again (a6158c8)

## [v1.5.0] - 2026-11-17

Builds and files release: redesigned Builds runs, run screen and logs with stages that scale to any pipeline, Builds search as you type, Go to file (`Ctrl+p`) in the Files hub, and vim-like yank of selected lines.

### Added

- feat: Go to file (`Ctrl+p`) in the Files hub — fuzzy-find any file by name or path, with highlighted matches, recent files, a preview of the highlighted file and `enter` to open it in the tree; one request lists the repository (cached per branch or tag, last three kept), typing sends no requests, previews load once you rest on a file and share the Files pane's cache, and very large repositories are indexed up to 50,000 files (bb5da12)
- feat: vim-like yank in the Files hub and a pull request's Files tab — select lines with `v`, then `y` copies them as plain text at once ("Copied 3 lines") and ends the selection; with lines selected, `s` opens the share menu for them (it was `y`) and `Y` still copies their link (70dfb94)
- feat: Builds search asks Azure DevOps as you type, 0.4 s after the last keystroke, instead of waiting for `Enter`; typing on cancels the pending request, the same text is never searched twice, the bar shows *searching* while it loads, and the list stays on screen until the results arrive; `Enter` now just closes the bar (8370b5b)
- feat: Builds runs are grouped under Today, Yesterday, Last 7 days and Older, and a Stages column shows one coloured cell per stage (`█░░·`, `+N` past 12), loaded for the rows on screen (8370b5b)
- feat: `e` on a failed run opens its logs at the first error; in a run, `e` / `E` jump between errors in the log, or to the next failed step (8370b5b)
- feat: run logs fold `##[group]` sections, showing how many lines they hide; groups with errors or warnings, or still running, stay open, and `C` folds or unfolds them all (8370b5b)
- feat: the Builds preview shows the run's stages as a strip (one cell per stage, or counts when they don't fit) and at most four of them — failed, running, last finished, next — with `+ N more` for the rest; then Where it failed (the first error) or Running now, and the last 10 runs of the run's pipeline (8370b5b)

### Changed

- feat: Builds lists, stages, jobs and steps show one state symbol (`✓ ✗ ! ◐ ○ ⊘ –`) instead of labels; the run header spells the state out (`✗ FAILED`), followed by the number, title and run details, with underlined Summary / Logs tabs (8370b5b)
- feat: the run Summary is two panes: every stage, one line each, and the selected stage's jobs with where they failed or what they are running; `Tab` or `h` / `l` switch panes, and errors and warnings are tinted cards (8370b5b)
- feat: run and release logs use a jobs tree (stages, jobs, expandable steps) and the log, in two framed panes; the focused pane has a blue border and the status bar names it (`LOGS · JOBS` / `LOGS · LOG`); the log header shows `Stage › Job › Step`, the agent, and error and warning counts (8370b5b)
- feat: a selected row in a pane without focus keeps a dimmer tint, so the selection never disappears (8370b5b)
- perf: dialogs draw faster — rows style runs of letters instead of each letter, blocks are fitted without word-wrapping, and dialogs are framed and placed over the screen without lipgloss's wrapper and compositor; moving through Go to file drops from 16 to 7 ms per key on a 50,000-file index (bb5da12)

## [v1.4.2] - 2026-11-17

Boards release: drag and drop on the Board like on Sprints, redesigned Board cards, and simpler Boards hub headers.

### Added

- feat: drag and drop on the Board (`B`), the same as on the Sprints taskboard — hover shows the ⠿ grip; while dragging, the card's place shows a dotted ghost, the target column is tinted with a dashed preview at its top and counts like `2 → 3`, a label follows the pointer and the status bar reads MOVING; the card moves at once and shows saving, then saved or not saved (it goes back on failure); `esc` cancels (430cf3e)
- feat: Board drags refuse a state the item's type doesn't have ("Bug has no Approved state") and, when columns are scrolled out of view, holding the pointer at the left or right edge scrolls to them (430cf3e)
- feat: double click a Board card to open its work item form (430cf3e)

### Changed

- feat: Board cards redesigned — bordered columns with a header and count, and five-row cards with padding on both sides: ID, type and effort; the title on two word-wrapped lines; up to two tag chips; the assignee's initials and name with task progress. The state is no longer repeated on the card (430cf3e)
- feat: the Backlogs, Boards and Sprints headers drop their view tabs (*Backlog*, *Board*, *Taskboard* / *Backlog*) and the *View as Board* / *View as Backlog* links — `L`, `B` and `S` switch hubs, and on Sprints `[` / `]` still switch views, named in the pane title; the main pane no longer shows the `5` badge (430cf3e)
- feat: `<` / `>` on the Board move the card at once with the same saving and saved feedback as dragging, and skip states the item's type doesn't have (430cf3e)

## [v1.4.1] - 2026-11-17

Share release: `y` shares anything from any hub — links, IDs, permalinks, paths and code snippets — and `Y` copies the link at once.

### Added

- feat: Share (`y`) everywhere — one key opens a Share menu for whatever is selected, with a letter per format that is the same in every hub: `u` link, `p` permalink pinned to a commit, `i` ID or number, `n` name, `f` path and lines, `c` Markdown code snippet, `m` Markdown link, `t` title and link, `h` `#mention`, `g` `AB#` GitHub link, `r` run ID, `l` job logs link, `d` pipeline or release pipeline link; `enter` copies the highlighted row and `o` opens the link. Covers work items (lists, board, taskboard, work item form), pull requests, a pull request's files and lines, files, folders and lines in the Files hub, commits, branches, tags, pipelines, runs (with their build number), release pipelines, releases and approvals. See [Sharing](docs/guide/sharing.md) (05bb51f)
- feat: `Y` copies the selected item's link without opening the menu (05bb51f)
- feat: line selection in the Files hub (`v`, then `j` / `k`), shared with `y`; `o` opens the browser at the selection; `esc` clears the selection before going back to the tree (05bb51f)
- feat: Markdown files in the Files hub have a **File** tab (highlighted source with the line cursor, so lines can be selected and shared) and a **Preview** tab (rendered, the default); `1` / `2` or `[` / `]` switch, and the choice carries over to the next Markdown file (05bb51f)
- feat: when no clipboard command is available (for example over SSH), copying falls back to the terminal clipboard (OSC 52) and shows the text in a dialog to select by hand (05bb51f)

### Changed

- feat: `y` in the Files, Commits, Branches and Tags hubs opens the Share menu instead of copying one value; the path, commit ID or name is one of its rows (05bb51f)
- fix: the clipboard tries `wl-copy` (Wayland), `xclip`, `xsel` and WSL's `clip.exe` on Linux, and Windows pipes the text into `clip` instead of `cmd /c echo`, which broke on links containing `&` (05bb51f)

### Removed

- feat: `i` (Info: copy ID or URL) on the Boards hubs and the work item form — `y` shares instead (05bb51f)

## [v1.4.0] - 2026-11-17

Repos release: browse a repository's files, commit history, branches and tags, create and delete branches, create annotated tags, copy clone URLs, and syntax highlighting in file views and pull request diffs.

### Added

- feat: Repos › Files hub (`E`, next to Pull requests in the navigation rail) — a folder tree beside the selected folder's files and subfolders with the last commit that changed each one (when, short ID, message) and its rendered `README.md`, or the selected file's contents with line numbers (Markdown rendered; binary files and files over `diff.max_file_kb` open in the browser); folders load as you open them and files once the cursor rests on them; `b` browses another branch or tag, `y` copies a path and `o` opens it in the browser at that branch or tag (0f8e053)
- feat: Repos › Commits hub (`H`) — the commits of a branch or tag, 100 at a time with more as you scroll, with a preview of the full message, author, committer, parents and changed files; `/` filters the loaded commits by text and searches Azure DevOps with `author:` and `path:`, `enter` browses the files at a commit, `y` copies its ID and `o` opens it in the browser (0f8e053)
- feat: file and folder history (`H` in the Files hub) — opens the Commits hub filtered to the selected path; `esc` clears it (0f8e053)
- feat: Repos › Branches hub (`V`) — **Mine** (yours, starred and the default branch), **All** and **Stale** (no commit in 3 months) views with each branch's last commit, behind | ahead counts against the default branch (`stats/branches`), *Default* and *Locked* marks and its active pull request; `/` filters by name, `enter` browses its files, `H` shows its commits, `p` opens a new pull request from it, `y` copies its name (0f8e053)
- feat: create a branch (`n` in the Branches hub) — name checked against Git's rules and existing branches, based on a branch, tag or commit ID; errors keep the form open and the new branch is selected (0f8e053)
- feat: delete a branch (`d` in the Branches hub) — confirmation that warns about an active pull request from it; the default branch and locked branches are refused (0f8e053)
- feat: Repos › Tags hub (`T`) — tags by name with their commit and whether they're annotated or lightweight, and a preview of an annotated tag's message, tagger and date; `/` filters by name, `enter` browses the files at the tag, `H` shows the commits up to it, `y` copies its name (0f8e053)
- feat: create a tag (`n` in the Tags hub) — an annotated tag (as the Azure DevOps web UI creates) on a branch, tag or commit ID, with a required description; the name is checked against Git's rules and existing tags, errors keep the form open and the new tag is selected (0f8e053)
- feat: clone a repository (`c` in the Files hub) — copy the HTTPS or SSH clone URL, or a `git clone` command (0f8e053)
- feat: the Repos hubs keep their own repository (`r`, without *All repositories*), remembered in `state.json` as `code_repository`; it starts from the Pull requests hub's repository or the project's only one (0f8e053)

### Changed

- feat: syntax highlighting in the pull request diff and the Files hub — one shared highlighter (`components.HighlightCode`) runs Chroma, the engine glamour uses, with lazyado's glamour code block theme, so diffs, file contents and Markdown code blocks share the same colors; each side of a diff is highlighted as a whole so multi-line comments and strings stay correct, and colors sit on the diff's add, delete and cursor backgrounds (0f8e053)
- feat: the Files hub shows a file like the pull request diff — a line-number gutter and a line cursor (`j` / `k`, `g` / `G`, `Ctrl+d` / `Ctrl+u`) with the focus bar; `o` opens the file in the browser at the cursor line (0f8e053)
- feat: confirmation dialogs fit short terminals by trimming the message instead of overflowing (0f8e053)
- refactor: hub keys are matched in one place (`hubFor` / `goToHub`) instead of once per hub (0f8e053)

## [v1.3.1] - 2026-11-17

Security awareness release: lazyado relies only on your Personal Access Token and keeps it, your identity and Azure DevOps content off disk and out of logs.

### Security

- security: the debug log moved from the shared temp directory to `lazyado/debug.log` in your user cache directory, readable only by you, and no longer records query strings, API error bodies or raw comment responses (which held names, emails and comment text) (be89af4)
- security: `config.yaml` is read only from `~/.config/lazyado/`; a `./config.yaml` in the directory you start lazyado from is no longer used, so a token can't be picked up from (or left in) a project folder (be89af4)
- security: `~/.config/lazyado/` is created readable only by you (`0700`) (be89af4)
- security: release binaries are built with `-trimpath`, so they don't embed the build machine's paths (be89af4)
- test: fixtures use reserved `example.com` addresses, and a repository check fails the tests on real-looking emails, token-shaped strings, encoded credentials or a non-placeholder `pat:` (be89af4)

## [v1.3.0] - 2026-11-17

Pipelines release: a Builds hub with run details, logs and running pipelines with their YAML options, a Releases hub with stage deployments, approvals, canceling and creating releases, and opening a pull request's build in the run view.

### Added

- feat: Pipelines › Builds hub (`D`, a new *Pipelines* group in the navigation rail) — **Recent** lists the 10 pipelines that ran last with their latest run, **All** lists every pipeline grouped by folder, and **Runs** lists the latest runs of every pipeline; long lists load more as you scroll, a preview pane shows the selected pipeline's last 10 runs, pass rate, median duration and the stages of its latest run, and lists with a queued or running run refresh every 15 seconds (19f80fd)
- feat: search pipelines and runs (`/` in the Builds hub) — text filters the loaded list as you type, `Enter` searches every pipeline name on Azure DevOps (with `*` wildcards), and `state:`, `branch:`, `pipeline:` and `by:@me` search runs on Azure DevOps (19f80fd)
- feat: a pipeline's runs (`enter` on a pipeline) with **Runs** and **Branches** (latest run per branch) views (19f80fd)
- feat: run details (`enter` on a run, `v` for a pipeline's latest run) — **Summary** shows stages and jobs with progress and durations, errors and warnings with where they came from, run details and the commits it includes; **Logs** shows the jobs by stage, the steps of the selected job and the step's log with Azure DevOps' error, warning and section markers, opens on the failed or running step, and follows a running step (`F`); a running run refreshes every 5 seconds (19f80fd)
- feat: run a pipeline (`n` in the Builds hub, `n` on a run to run it again on the same branch) — choose a branch or tag and an optional commit, set the parameters read from the pipeline's YAML at that branch or tag, untick the stages to skip, override queue-time variables and turn on system diagnostics; classic pipelines queue a build with the branch and variables, and the new run's logs open once it is queued (19f80fd)
- feat: Pipelines › Releases hub (`R`) — release pipelines with an always-visible **Starred** group first (`Ctrl+t`, saved in `state.json`, plus any Azure DevOps favorites), searchable by name (typing filters, `↑` / `↓` move through matches, `Enter` goes into the match's releases, the filter stays as `· /text` in the pane title and `Esc` clears it; Azure DevOps is asked 300 ms after typing only when more pipelines exist than are loaded), `v` opens a pipeline's latest release, and each pipeline's releases with a chip per stage (deployed, deploying, waiting for approval, queued, failed, rejected, partial, canceled, not deployed); loads more as you scroll and refreshes while a release deploys (7d8ff5a)
- feat: Approvals view (`2` in the Releases hub) — pending approvals waiting on you (including group approvals) and on others, with a preview of the release; `a` approves or rejects (7d8ff5a)
- feat: release screen — stages as boxes (`h` / `l`), what the selected stage waits on and who can approve it, its trigger, approvers and last deployment attempt; `a` approves or rejects with a comment (required to reject), `d` deploys or redeploys a stage with warnings about approvals and failed attempts, and the **Logs** tab shows deployment jobs, tasks and logs; refreshes every 5 seconds while deploying (7d8ff5a)
- feat: cancel a deployment (`x` on the release screen) — for a stage that is deploying, queued, scheduled or waiting for approval; the dialog explains what stops, takes an optional comment, and the stage can be deployed again afterwards (7d8ff5a)
- feat: create a release (`n`) — pick a branch and build version for each build artifact (succeeded and partially succeeded builds of that branch), a branch for Azure Repos artifacts, which automatic stages should wait for a manual deploy, release-time variables and a description (7d8ff5a)
- feat: open a pull request's build (`b` in the review and in the pull request list) — build validation policies (`context.buildId`) and pipeline status checks (`targetUrl` to `_build/results?buildId=`) open in the run view with stages, jobs and logs; several builds show a picker, and `esc` returns to the pull request (7d8ff5a)

### Changed

- change: pane titles show the `5` key only on the Boards hubs, where it focuses the main view; the Pull requests, Builds and Releases panes no longer show it (7d8ff5a)

### Fixed

- fix: the selected pull request row is highlighted edge to edge; the spaces between reviewer vote chips no longer break the highlight (19f80fd)

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

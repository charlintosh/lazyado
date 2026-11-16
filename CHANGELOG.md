# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and adheres to Semantic Versioning.

## [Unreleased]

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

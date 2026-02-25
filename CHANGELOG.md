# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and adheres to Semantic Versioning.

## [Unreleased]

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

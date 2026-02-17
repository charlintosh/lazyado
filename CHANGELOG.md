# Changelog

All notable changes to this project will be documented in this file.

The format is based on "Keep a Changelog" and adheres to Semantic Versioning.

## [Unreleased]

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

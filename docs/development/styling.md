# Styling (Lipgloss)

- All styles and color maps live in [internal/styles/styles.go](../../internal/styles/styles.go).
- Type and state color maps: `typeColors`, `stateColors`.
- Use badge helper functions like `styles.TypeBadge("Task")` to keep UI consistent.
- Layout helpers: `lipgloss.JoinHorizontal/Vertical()`.


## Azure DevOps palette

- The palette in `internal/styles/styles.go` follows the Azure DevOps dark theme: `ColorSurface` (top and status bars), `ColorSurfaceAlt` (navigation rail), `ColorHighlight` (selected rows and cards), `ColorPrimary` (focus and accent).
- Work item type colors: `ColorTypePBI`, `ColorTypeBug`, `ColorTypeTask`, `ColorTypeFeature`, `ColorTypeEpic`, read through `styles.TypeColor()`.
- State colors come from Azure DevOps at startup (`styles.RegisterStateColor`) and fall back to the built-in map; read them with `styles.StateColor()`.
- UI is built from Charm components: bubbles `table` (backlog grid), `list` with custom delegates (board columns, pickers), `widgets.Dropdown` (anchored filter pickers), `viewport` (preview pane, form, taskboard), `help` (status bar and help overlay), `progress` (child completion) and `spinner`; Lip Gloss `table` (sprint taskboard), `list` (child items) and `tree` enumerators (backlog hierarchy).
- Shared component styles and key maps live in `internal/components/panels/theme.go`; pane chrome (`Pane`, `Section`, `PaneTitle`, `Tabs`, `Badge`, `Chip`, `Button`) lives in `internal/components/panels/frame.go` and is drawn only with Lip Gloss borders, padding and backgrounds.
- Do not use decorative Unicode symbols or emoji in the UI: many are ambiguous-width and misalign in some terminals. Use color, background badges, borders and text labels instead.
- The exception is a fixed set of single-cell glyphs, each paired with a color and never the only signal: box drawing (`─ │ ├ └`), `● ○` (on / off, yes / no), `▎` (changed, orange), `▌` (cursor, blue), `!` (needs a value, red), `▾` (picked from a list), `›` (typed), `⊘` (read-only), `◌` (hidden), `■` (a scope's color) and `−` (the saved value under a change). The release pipeline editor defines them in `internal/components/panels/release_editor_view.go`; reuse those names instead of adding new symbols.
- In the release pipeline editor, orange always means an unsaved change, red a setting that must be picked again, violet `$(variables)` and task kinds, teal the `from Azure` tag, and each settings group has its own tag color (`groupColors`).
- Lip Gloss v2 `Width`/`Height` include borders and padding; size boxes by their outer dimensions.

# Styling (Lipgloss)

- All styles and color maps live in [internal/styles/styles.go](internal/styles/styles.go).
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
- Lip Gloss v2 `Width`/`Height` include borders and padding; size boxes by their outer dimensions.

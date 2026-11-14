# Styling (Lipgloss)

- All styles and color maps live in [internal/styles/styles.go](internal/styles/styles.go).
- Type and state color maps: `typeColors`, `stateColors`.
- Use badge helper functions like `styles.TypeBadge("Task")` to keep UI consistent.
- Layout helpers: `lipgloss.JoinHorizontal/Vertical()`.


## Azure DevOps palette

- The palette in `internal/styles/styles.go` follows the Azure DevOps dark theme: `ColorSurface` (top and status bars), `ColorSurfaceAlt` (navigation rail), `ColorHighlight` (selected rows and cards), `ColorPrimary` (focus and accent).
- Work item type colors: `ColorTypePBI`, `ColorTypeBug`, `ColorTypeTask`, `ColorTypeFeature`, `ColorTypeEpic`, read through `styles.TypeColor()`.
- State colors come from Azure DevOps at startup (`styles.RegisterStateColor`) and fall back to the built-in map; read them with `styles.StateColor()`.
- Pane chrome lives in `internal/components/panels/frame.go`: `RenderTitledBox` (title in the top border, exact size), `FitBlock` / `FillBlock`, `JoinBar`, `Hint`, `Chip`, `StateDot`, `TypeGlyph`.
- Lip Gloss v2 `Width`/`Height` include borders and padding; size boxes by their outer dimensions.

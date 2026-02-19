package panels

import (
	"strings"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// maxVisibleOptions aliases styles.FilterGroupVisibleOptions for local readability.
const maxVisibleOptions = styles.FilterGroupVisibleOptions

// FilterPanel is the filter panel component
type FilterPanel struct {
	filterState *models.FilterState
	viewport    viewport.Model
	styles      styles.Styles
	keys        keys.KeyMap
	width       int
	height      int
	focused     bool
}

// NewFilterPanel creates a new filter panel
func NewFilterPanel(filterState *models.FilterState, styles styles.Styles, keys keys.KeyMap) FilterPanel {
	return FilterPanel{
		viewport:    viewport.New(0, 0),
		filterState: filterState,
		styles:      styles,
		keys:        keys,
	}
}

// Init initializes the filter panel
func (f FilterPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the filter panel
func (f FilterPanel) Update(msg tea.Msg) (FilterPanel, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		return f.handleKey(msg)
	}
	return f, nil
}

// handleKey processes keyboard input for the filter panel.
func (f FilterPanel) handleKey(msg tea.KeyMsg) (FilterPanel, tea.Cmd) {
	switch {
	case key.Matches(msg, f.keys.Search):
		return f, func() tea.Msg { return OpenSearchMsg{} }
	case key.Matches(msg, f.keys.Up):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveUp(); f.adjustOffset(g) })
	case key.Matches(msg, f.keys.Down):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveDown(); f.adjustOffset(g) })
	case key.Matches(msg, f.keys.Top):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveToTop(); g.Offset = 0 })
	case key.Matches(msg, f.keys.Bottom):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveToBottom(); f.adjustOffset(g) })
	case key.Matches(msg, f.keys.Select):
		if group := f.filterState.ActiveFilterGroup(); group != nil {
			group.SelectCurrent()
		}
		f.rebuildViewport()
		return f, func() tea.Msg { return FilterChangedMsg{} }
	case key.Matches(msg, f.keys.Left):
		f.filterState.PrevGroup()
		f.rebuildViewport()
	case key.Matches(msg, f.keys.Right):
		f.filterState.NextGroup()
		f.rebuildViewport()
	}
	return f, nil
}

// moveCursor applies fn to the active filter group then rebuilds the viewport.
func (f *FilterPanel) moveCursor(fn func(*models.FilterGroup)) {
	if group := f.filterState.ActiveFilterGroup(); group != nil {
		fn(group)
		f.rebuildViewport()
	}
}

// adjustOffset ensures the cursor is visible within the scroll window
func (f *FilterPanel) adjustOffset(group *models.FilterGroup) {
	if group.Cursor < group.Offset {
		group.Offset = group.Cursor
	}
	if group.Cursor >= group.Offset+maxVisibleOptions {
		group.Offset = group.Cursor - maxVisibleOptions + 1
	}
}

// View renders the filter panel
func (f FilterPanel) View() string {
	panelStyle := f.styles.PanelInactive
	if f.focused {
		panelStyle = f.styles.PanelActive
	}
	return panelStyle.
		Width(f.width).
		Height(f.height).
		Render(f.viewport.View())
}

// SetSize sets the size of the filter panel
func (f *FilterPanel) SetSize(width, height int) {
	f.width = width
	f.height = height
	vpWidth := f.width - styles.PanelBorderOffset
	if vpWidth < styles.FilterMinViewportWidth {
		vpWidth = styles.FilterMinViewportWidth
	}
	f.viewport.Width = vpWidth
	f.viewport.Height = f.height
	f.rebuildViewport()
}

// SetFocused sets whether the panel is focused
func (f *FilterPanel) SetFocused(focused bool) {
	f.focused = focused
	f.rebuildViewport()
}

// FilterState returns the current filter state
func (f *FilterPanel) FilterState() *models.FilterState {
	return f.filterState
}

// SetFilterState updates the filter state
func (f *FilterPanel) SetFilterState(state *models.FilterState) {
	f.filterState = state
	f.rebuildViewport()
}

// SetActiveGroup sets the active filter group
func (f *FilterPanel) SetActiveGroup(index int) {
	if f.filterState != nil && index >= 0 && index < len(f.filterState.Groups) {
		f.filterState.ActiveGroup = index
	}
}

// rebuildViewport builds the full content string and auto-scrolls to the active cursor.
func (f *FilterPanel) rebuildViewport() {
	if f.viewport.Width <= 0 {
		return
	}
	f.viewport.SetContent(f.buildContent())
	f.scrollToActive()
}

// buildContent assembles all filter groups into a single scrollable string.
func (f *FilterPanel) buildContent() string {
	var b strings.Builder
	for i, group := range f.filterState.Groups {
		b.WriteString(f.renderGroup(i, group))
		if i < len(f.filterState.Groups)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderGroup renders a single filter group section.
func (f *FilterPanel) renderGroup(i int, group *models.FilterGroup) string {
	var b strings.Builder
	isActive := i == f.filterState.ActiveGroup && f.focused

	// Title line
	numberStyle := f.styles.TextMuted
	if isActive {
		numberStyle = f.styles.AccentBold
	}
	titleStyle := f.styles.FilterGroupTitle
	if isActive {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	title := group.Title
	if len(group.Options) > maxVisibleOptions {
		title += f.styles.TextMuted.Render(" (" + components.Itoa(len(group.Options)) + ")")
	}
	b.WriteString(numberStyle.Render(components.Itoa(i+1)+". ") + titleStyle.Render(title))
	b.WriteString("\n")

	// Separator
	sep := strings.Repeat("─", min(f.width-styles.FilterSepPadOffset, styles.FilterSepMaxWidth))
	b.WriteString(f.styles.Subtitle.Render(sep))
	b.WriteString("\n")

	f.renderGroupOptions(&b, group, isActive)
	return b.String()
}

// renderGroupOptions writes scroll indicators and visible options for a group.
func (f *FilterPanel) renderGroupOptions(b *strings.Builder, group *models.FilterGroup, isActive bool) {
	if group.Offset > 0 {
		b.WriteString(f.styles.TextMuted.Render("  ▲ more"))
		b.WriteString("\n")
	}

	startIdx := group.Offset
	endIdx := startIdx + maxVisibleOptions
	if endIdx > len(group.Options) {
		endIdx = len(group.Options)
	}

	for j := startIdx; j < endIdx; j++ {
		b.WriteString(f.renderOption(group, j, isActive))
		b.WriteString("\n")
	}

	if endIdx < len(group.Options) {
		b.WriteString(f.styles.TextMuted.Render("  ▼ more"))
		b.WriteString("\n")
	}
}

// renderOption renders a single filter option line.
func (f *FilterPanel) renderOption(group *models.FilterGroup, j int, isActive bool) string {
	opt := group.Options[j]
	indicator := "○"
	if opt.Selected {
		indicator = "●"
	}
	optStyle := f.styles.FilterOption
	if opt.Selected {
		optStyle = f.styles.FilterSelected
	}
	if j == group.Cursor && isActive {
		optStyle = f.styles.AccentBold
	}
	return optStyle.Render(" " + indicator + " " + opt.Label)
}

// scrollToActive adjusts the viewport Y offset so the active cursor line stays visible.
func (f *FilterPanel) scrollToActive() {
	if f.viewport.Height <= 0 {
		return
	}
	y := f.computeActiveLineY()
	if y < f.viewport.YOffset {
		f.viewport.SetYOffset(y)
	} else if y >= f.viewport.YOffset+f.viewport.Height {
		f.viewport.SetYOffset(y - f.viewport.Height + 1)
	}
}

// computeActiveLineY returns the 0-indexed line of the active cursor in the full content.
func (f *FilterPanel) computeActiveLineY() int {
	y := 0
	for i, g := range f.filterState.Groups {
		if i == f.filterState.ActiveGroup {
			y += 2 // title + separator
			if g.Offset > 0 {
				y++ // ▲ more
			}
			y += g.Cursor - g.Offset
			return y
		}
		y += f.groupLineCount(g)
	}
	return y
}

// groupLineCount returns the total number of rendered lines for a non-active group.
func (f *FilterPanel) groupLineCount(g *models.FilterGroup) int {
	lines := 2 // title + separator
	if g.Offset > 0 {
		lines++ // ▲ more
	}
	end := g.Offset + maxVisibleOptions
	if end > len(g.Options) {
		end = len(g.Options)
	}
	lines += end - g.Offset // visible options
	if end < len(g.Options) {
		lines++ // ▼ more
	}
	lines++ // blank line between groups
	return lines
}

// FilterChangedMsg is sent when a filter selection changes
type FilterChangedMsg struct{}

// OpenSearchMsg is sent when search should be opened
type OpenSearchMsg struct{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

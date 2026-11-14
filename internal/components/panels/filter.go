package panels

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const maxVisibleOptions = styles.RailFilterOptions

var filterLabels = map[models.FilterType]string{
	models.FilterTypeSprint:   "Iteration",
	models.FilterTypeState:    "State",
	models.FilterTypeAssigned: "Assigned to",
	models.FilterTypeArea:     "Area",
}

type FilterPanel struct {
	filterState *models.FilterState
	styles      styles.Styles
	keys        keys.KeyMap
	hub         Hub
	width       int
	height      int
	focused     bool
}

// NewFilterPanel creates a new filter panel
func NewFilterPanel(filterState *models.FilterState, styles styles.Styles, keys keys.KeyMap) FilterPanel {
	return FilterPanel{
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
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		return f.handleKey(msg)
	}
	return f, nil
}

func (f FilterPanel) handleKey(msg tea.KeyPressMsg) (FilterPanel, tea.Cmd) {
	switch {
	case key.Matches(msg, f.keys.Search):
		return f, func() tea.Msg { return OpenSearchMsg{} }
	case key.Matches(msg, f.keys.Up):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveUp(); adjustOffset(g) })
	case key.Matches(msg, f.keys.Down):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveDown(); adjustOffset(g) })
	case key.Matches(msg, f.keys.Top):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveToTop(); g.Offset = 0 })
	case key.Matches(msg, f.keys.Bottom):
		f.moveCursor(func(g *models.FilterGroup) { g.MoveToBottom(); adjustOffset(g) })
	case key.Matches(msg, f.keys.Select):
		if group := f.filterState.ActiveFilterGroup(); group != nil {
			group.SelectCurrent()
		}
		return f, func() tea.Msg { return FilterChangedMsg{} }
	case key.Matches(msg, f.keys.Left):
		f.filterState.PrevGroup()
	case key.Matches(msg, f.keys.Right):
		f.filterState.NextGroup()
	}
	return f, nil
}

func (f *FilterPanel) moveCursor(fn func(*models.FilterGroup)) {
	if group := f.filterState.ActiveFilterGroup(); group != nil {
		fn(group)
	}
}

func adjustOffset(group *models.FilterGroup) {
	if group.Cursor < group.Offset {
		group.Offset = group.Cursor
	}
	if group.Cursor >= group.Offset+maxVisibleOptions {
		group.Offset = group.Cursor - maxVisibleOptions + 1
	}
}

func (f FilterPanel) View() string {
	lines, activeLine := f.buildLines()
	if len(lines) > f.height && f.height > 0 {
		start := 0
		if activeLine >= f.height {
			start = activeLine - f.height + 2
		}
		start = min(start, len(lines)-f.height)
		lines = lines[start : start+f.height]
	}
	bg := styles.ColorSurfaceAlt
	body := FillBlock(strings.Join(lines, "\n"), max(f.width-1, 0), f.height, bg)
	edge := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render("│")
	out := strings.Split(body, "\n")
	for i := range out {
		out[i] += edge
	}
	return strings.Join(out, "\n")
}

func (f FilterPanel) buildLines() ([]string, int) {
	bg := styles.ColorSurfaceAlt
	w := max(f.width-1, 0)
	heading := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(bg).Bold(true)
	var lines []string
	activeLine := 0

	lines = append(lines, heading.Render(" BOARDS"))
	for _, hub := range []Hub{HubBacklog, HubBoard, HubSprint} {
		lines = append(lines, f.renderHub(hub, w))
	}
	lines = append(lines, "", heading.Render(" FILTERS"))

	if f.filterState == nil {
		return lines, activeLine
	}
	for i, group := range f.filterState.Groups {
		isActive := f.focused && i == f.filterState.ActiveGroup
		if isActive {
			activeLine = len(lines)
		}
		lines = append(lines, f.renderGroupLabel(i, group, isActive))
		lines = append(lines, f.renderField(group, isActive, w))
		if isActive {
			lines = append(lines, f.renderOptions(group, w)...)
			activeLine = len(lines) - 1
		}
	}
	return lines, activeLine
}

func (f FilterPanel) renderHub(hub Hub, width int) string {
	bg := styles.ColorSurfaceAlt
	fg := styles.ColorLight
	iconFg := styles.ColorTextMuted
	bar := lipgloss.NewStyle().Background(bg).Render(" ")
	if hub == f.hub {
		bg = styles.ColorHighlight
		fg = styles.ColorText
		iconFg = styles.ColorPrimary
		bar = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Background(bg).Render("▌")
	}
	shortcut := f.hubKey(hub).Help().Key
	label := lipgloss.NewStyle().Foreground(iconFg).Background(bg).Render(hub.Icon()+" ") +
		lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(hub == f.hub).Render(hub.Name())
	keyText := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(bg).Render(shortcut + " ")
	gap := width - 1 - lipgloss.Width(label) - lipgloss.Width(keyText)
	return bar + label + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", max(gap, 1))) + keyText
}

func (f FilterPanel) hubKey(hub Hub) key.Binding {
	switch hub {
	case HubBoard:
		return f.keys.HubBoard
	case HubSprint:
		return f.keys.HubSprint
	default:
		return f.keys.HubBacklog
	}
}

func (f FilterPanel) renderGroupLabel(i int, group *models.FilterGroup, isActive bool) string {
	bg := styles.ColorSurfaceAlt
	numFg := styles.ColorTextMuted
	labelFg := styles.ColorTextMuted
	if isActive {
		numFg = styles.ColorPrimary
		labelFg = styles.ColorText
	}
	label := filterLabels[group.Type]
	if label == "" {
		label = group.Title
	}
	if len(group.Options) > maxVisibleOptions {
		label += " (" + components.Itoa(len(group.Options)) + ")"
	}
	return lipgloss.NewStyle().Background(bg).Render(" ") +
		lipgloss.NewStyle().Foreground(numFg).Background(bg).Render("["+components.Itoa(i+1)+"] ") +
		lipgloss.NewStyle().Foreground(labelFg).Background(bg).Render(label)
}

func (f FilterPanel) renderField(group *models.FilterGroup, isActive bool, width int) string {
	fieldWidth := max(width-2, styles.RailFieldMinWidth)
	value := "All"
	if opt := group.SelectedOption(); opt != nil {
		value = opt.Label
	}
	caret := "▾"
	borderFg := styles.ColorBorder
	if isActive {
		caret = "▴"
		borderFg = styles.ColorPrimary
	}
	fieldBg := styles.ColorOnAccent
	text := lipgloss.NewStyle().Foreground(styles.ColorText).Background(fieldBg).Render(" " + value)
	caretText := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(fieldBg).Render(caret + " ")
	inner := fieldWidth - 1
	text = FitWidth(text, max(inner-lipgloss.Width(caretText), 1))
	field := lipgloss.NewStyle().Foreground(borderFg).Background(fieldBg).Render("▏") + text + caretText
	return lipgloss.NewStyle().Background(styles.ColorSurfaceAlt).Render(" ") + field
}

func (f FilterPanel) renderOptions(group *models.FilterGroup, width int) []string {
	bg := styles.ColorSurfaceAlt
	more := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(bg)
	var lines []string
	if group.Offset > 0 {
		lines = append(lines, more.Render("    ▲ more"))
	}
	end := min(group.Offset+maxVisibleOptions, len(group.Options))
	for j := group.Offset; j < end; j++ {
		opt := group.Options[j]
		rowBg := bg
		fg := styles.ColorLight
		if j == group.Cursor {
			rowBg = styles.ColorHighlight
			fg = styles.ColorText
		}
		indicator := "○"
		indFg := styles.ColorTextMuted
		if opt.Selected {
			indicator = "●"
			indFg = styles.ColorSuccess
		}
		row := lipgloss.NewStyle().Background(bg).Render("  ") +
			lipgloss.NewStyle().Foreground(indFg).Background(rowBg).Render(" "+indicator+" ") +
			lipgloss.NewStyle().Foreground(fg).Background(rowBg).Render(opt.Label)
		lines = append(lines, FillBlock(row, width, 1, rowBg))
	}
	if end < len(group.Options) {
		lines = append(lines, more.Render("    ▼ more"))
	}
	hint := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(bg).
		Render("    " + f.keys.Select.Help().Key + " apply · " + f.keys.Search.Help().Key + " find")
	lines = append(lines, hint)
	return lines
}

func (f *FilterPanel) SetSize(width, height int) {
	f.width = width
	f.height = height
}

func (f *FilterPanel) SetHub(hub Hub) {
	f.hub = hub
}

// SetFocused sets whether the panel is focused
func (f *FilterPanel) SetFocused(focused bool) {
	f.focused = focused
}

// FilterState returns the current filter state
func (f *FilterPanel) FilterState() *models.FilterState {
	return f.filterState
}

// SetFilterState updates the filter state
func (f *FilterPanel) SetFilterState(state *models.FilterState) {
	f.filterState = state
}

// SetActiveGroup sets the active filter group
func (f *FilterPanel) SetActiveGroup(index int) {
	if f.filterState != nil && index >= 0 && index < len(f.filterState.Groups) {
		f.filterState.ActiveGroup = index
	}
}

func (f *FilterPanel) GetAvailableActions() []string {
	return Hints(styles.ColorSurface, f.keys.Up, f.keys.Select, f.keys.Search, f.keys.Left, f.keys.Right)
}

// FilterChangedMsg is sent when a filter selection changes
type FilterChangedMsg struct{}

// OpenSearchMsg is sent when search should be opened
type OpenSearchMsg struct{}

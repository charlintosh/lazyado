package panels

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

var filterLabels = map[models.FilterType]string{
	models.FilterTypeSprint:   "Iteration",
	models.FilterTypeState:    "State",
	models.FilterTypeAssigned: "Assigned to",
	models.FilterTypeArea:     "Area",
}

type hubItem struct {
	hub      Hub
	shortcut string
}

func (h hubItem) FilterValue() string { return h.hub.Name() }

type hubDelegate struct{}

func (hubDelegate) Height() int                             { return 1 }
func (hubDelegate) Spacing() int                            { return 0 }
func (hubDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (hubDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	h, ok := li.(hubItem)
	if !ok {
		return
	}
	width := m.Width()
	st := lipgloss.NewStyle().Width(width).PaddingLeft(1).Foreground(styles.ColorLight).
		BorderStyle(lipgloss.ThickBorder()).BorderLeft(true).BorderForeground(styles.ColorSurfaceAlt)
	shortcut := Muted(h.shortcut)
	if index == m.Index() {
		st = st.Background(styles.ColorHighlight).Foreground(styles.ColorText).Bold(true).
			BorderForeground(styles.ColorPrimary).BorderBackground(styles.ColorHighlight)
		shortcut = lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Background(styles.ColorHighlight).Render(h.shortcut)
	}
	name := h.hub.Name()
	gap := max(width-styles.CardChromeWidth-lipgloss.Width(name)-lipgloss.Width(h.shortcut)-1, 1)
	_, _ = io.WriteString(w, st.Render(name+strings.Repeat(" ", gap)+shortcut))
}

type optionItem struct {
	option models.FilterOption
}

func (o optionItem) FilterValue() string { return o.option.Label }

type optionDelegate struct{}

func (optionDelegate) Height() int                             { return 1 }
func (optionDelegate) Spacing() int                            { return 0 }
func (optionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (optionDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	o, ok := li.(optionItem)
	if !ok {
		return
	}
	st := lipgloss.NewStyle().Width(m.Width()).MaxWidth(m.Width()).Padding(0, 1).Foreground(styles.ColorLight)
	if o.option.Selected {
		st = st.Foreground(styles.ColorSuccess).Bold(true)
	}
	if index == m.Index() {
		st = st.Background(styles.ColorHighlight).Foreground(styles.ColorText)
	}
	_, _ = io.WriteString(w, st.Render(o.option.Label))
}

type FilterPanel struct {
	filterState *models.FilterState
	styles      styles.Styles
	keys        keys.KeyMap
	hubs        list.Model
	options     list.Model
	hub         Hub
	width       int
	height      int
	focused     bool
}

func NewFilterPanel(filterState *models.FilterState, st styles.Styles, k keys.KeyMap) FilterPanel {
	hubs := newComponentList([]list.Item{
		hubItem{HubBacklog, k.HubBacklog.Help().Key},
		hubItem{HubBoard, k.HubBoard.Help().Key},
		hubItem{HubSprint, k.HubSprint.Help().Key},
	}, hubDelegate{}, k, "Boards")
	hubs.Styles = listStyles(styles.ColorTextMuted)
	hubs.SetShowPagination(false)
	options := newComponentList(nil, optionDelegate{}, k, "")
	options.SetShowTitle(false)
	return FilterPanel{
		filterState: filterState,
		styles:      st,
		keys:        k,
		hubs:        hubs,
		options:     options,
	}
}

func (f FilterPanel) Init() tea.Cmd {
	return nil
}

func (f FilterPanel) Update(msg tea.Msg) (FilterPanel, tea.Cmd) {
	if !f.focused || f.filterState == nil {
		return f, nil
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, nil
	}
	switch {
	case key.Matches(k, f.keys.Search):
		return f, func() tea.Msg { return OpenSearchMsg{} }
	case key.Matches(k, f.keys.Select):
		if group := f.filterState.ActiveFilterGroup(); group != nil {
			group.Cursor = f.options.Index()
			group.SelectCurrent()
		}
		f.syncOptions()
		return f, func() tea.Msg { return FilterChangedMsg{} }
	case key.Matches(k, f.keys.Left):
		f.filterState.PrevGroup()
		f.syncOptions()
		return f, nil
	case key.Matches(k, f.keys.Right):
		f.filterState.NextGroup()
		f.syncOptions()
		return f, nil
	}
	var cmd tea.Cmd
	f.options, cmd = f.options.Update(msg)
	if group := f.filterState.ActiveFilterGroup(); group != nil {
		group.Cursor = f.options.Index()
	}
	return f, cmd
}

func (f *FilterPanel) syncOptions() {
	group := f.activeGroup()
	if group == nil {
		f.options.SetItems(nil)
		return
	}
	items := make([]list.Item, len(group.Options))
	for i, o := range group.Options {
		items[i] = optionItem{option: o}
	}
	paged := len(items) > styles.RailFilterOptions
	height := min(len(items), styles.RailFilterOptions)
	if paged {
		height++
	}
	f.options.SetShowPagination(paged)
	f.options.SetItems(items)
	f.options.SetSize(max(f.width-styles.RailDropdownChrome, 1), max(height, 1))
	f.options.Select(group.Cursor)
}

func (f *FilterPanel) activeGroup() *models.FilterGroup {
	if f.filterState == nil {
		return nil
	}
	return f.filterState.ActiveFilterGroup()
}

func (f FilterPanel) View() string {
	if f.width <= 0 || f.height <= 0 {
		return ""
	}
	inner := max(f.width-1, 1)
	sections := []string{f.hubs.View(), "", Heading("Filters")}
	if f.filterState != nil {
		for i, group := range f.filterState.Groups {
			isActive := f.focused && i == f.filterState.ActiveGroup
			sections = append(sections, f.renderGroup(i, group, isActive, inner))
		}
	}
	body := lipgloss.NewStyle().Padding(1, 1, 0, 1).Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
	return lipgloss.NewStyle().
		Background(styles.ColorSurfaceAlt).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(styles.ColorBorder).
		Width(f.width).Height(f.height).
		MaxWidth(f.width).MaxHeight(f.height).
		Render(body)
}

func (f FilterPanel) renderGroup(i int, group *models.FilterGroup, isActive bool, width int) string {
	label := filterLabels[group.Type]
	if label == "" {
		label = group.Title
	}
	numberBg, numberFg := styles.ColorChip, styles.ColorLight
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	if isActive {
		numberBg, numberFg = styles.ColorPrimary, styles.ColorOnAccent
		labelStyle = labelStyle.Foreground(styles.ColorText).Bold(true)
	}
	heading := Badge(components.Itoa(i+1), numberBg, numberFg) + " " + labelStyle.Render(label)

	value := "All"
	if opt := group.SelectedOption(); opt != nil {
		value = opt.Label
	}
	fieldWidth := max(width-styles.RailFieldInset, styles.RailFieldMinWidth)
	field := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1).
		Width(fieldWidth)
	if !isActive {
		return lipgloss.JoinVertical(lipgloss.Left, heading, field.Render(Line(value, fieldWidth-styles.BoxPaddedSize)))
	}
	dropdown := field.BorderForeground(styles.ColorPrimary).Padding(0).Render(f.options.View())
	hint := Muted(f.keys.Select.Help().Key + " apply   " + f.keys.Search.Help().Key + " find")
	return lipgloss.JoinVertical(lipgloss.Left, heading, dropdown, hint)
}

func (f *FilterPanel) SetSize(width, height int) {
	f.width = width
	f.height = height
	f.hubs.SetSize(max(width-styles.RailHubInset, 1), styles.RailHubListHeight)
	f.syncOptions()
}

func (f *FilterPanel) SetHub(hub Hub) {
	f.hub = hub
	f.hubs.Select(int(hub))
}

func (f *FilterPanel) SetFocused(focused bool) {
	f.focused = focused
	f.syncOptions()
}

func (f *FilterPanel) FilterState() *models.FilterState {
	return f.filterState
}

func (f *FilterPanel) SetFilterState(state *models.FilterState) {
	f.filterState = state
	f.syncOptions()
}

func (f *FilterPanel) SetActiveGroup(index int) {
	if f.filterState != nil && index >= 0 && index < len(f.filterState.Groups) {
		f.filterState.ActiveGroup = index
	}
	f.syncOptions()
}

func (f *FilterPanel) KeyBindings() []key.Binding {
	return []key.Binding{f.keys.Up, f.keys.Select, f.keys.Search, f.keys.Left, f.keys.Right}
}

type FilterChangedMsg struct{}

// OpenSearchMsg is sent when search should be opened
type OpenSearchMsg struct{}

func (f *FilterPanel) Refresh() {
	f.syncOptions()
}

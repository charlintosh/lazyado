package ui

import (
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	keywordPrompt      = "⌕ "
	keywordPlaceholder = "Filter by keyword"
	caret              = " ▾"
	sprintIcon         = "↻ "
	filterIdleIcon     = "▽"
	filterSetIcon      = "▼"
	fullscreenIcon     = "⤢"
	viewAsIcon         = "⇄ "
	clearIcon          = "✕"
	newIcon            = "+ "
	titleSeparator     = "  /  "
	allIterations      = "All iterations"
)

var (
	sprintBarFilters = []models.FilterType{models.FilterTypeState, models.FilterTypeArea}
	allFilters       = []models.FilterType{models.FilterTypeSprint, models.FilterTypeAssigned, models.FilterTypeState, models.FilterTypeArea}
	sprintFilterName = map[models.FilterType]string{models.FilterTypeSprint: "Sprint", models.FilterTypeAssigned: "Person"}
	barFilterName    = map[models.FilterType]string{
		models.FilterTypeSprint:   "Iteration",
		models.FilterTypeAssigned: "Assigned to",
		models.FilterTypeState:    "State",
		models.FilterTypeArea:     "Area",
	}
)

func (a App) filterKeys() []key.Binding {
	return []key.Binding{a.keys.Panel1, a.keys.Panel2, a.keys.Panel3, a.keys.Panel4}
}

func (a App) barFilters() []models.FilterType {
	if a.hub == panels.HubSprint {
		return sprintBarFilters
	}
	return allFilters
}

func (a App) inBar(t models.FilterType) bool {
	for _, b := range a.barFilters() {
		if b == t {
			return true
		}
	}
	return false
}

func (a App) filterName(t models.FilterType) string {
	if name, ok := sprintFilterName[t]; ok && a.hub == panels.HubSprint {
		return name
	}
	return barFilterName[t]
}

func (a App) groupIndex(t models.FilterType) int {
	for i, g := range a.filters.Groups {
		if g.Type == t {
			return i
		}
	}
	return -1
}

func (a App) pickerOpen(group int) bool {
	m, ok := a.modals.top()
	if !ok {
		return false
	}
	p, ok := m.(modals.FilterPickerModal)
	return ok && p.Group() == group
}

func (a App) filterControl(t models.FilterType, prefix string) panels.HeaderControl {
	i := a.groupIndex(t)
	if i < 0 || i >= len(a.filterKeys()) {
		return panels.HeaderControl{Group: -1}
	}
	g := a.filters.Groups[i]
	label := g.SelectedOption().Label + caret
	if t == models.FilterTypeSprint {
		label = a.sprintName() + caret
	}
	return panels.HeaderControl{
		Prefix: prefix,
		Label:  label,
		Short:  label,
		Key:    a.filterKeys()[i],
		Set:    g.IsSet(),
		Open:   a.pickerOpen(i),
		Group:  i,
	}
}

func (a App) activeFilterCount() int {
	n := a.filters.ActiveCount(a.barFilters()...)
	if a.keyword.Value() != "" {
		n++
	}
	return n
}

func (a App) funnelControl() panels.HeaderControl {
	c := panels.HeaderControl{Label: filterIdleIcon + " Filter", Short: filterIdleIcon, Key: a.keys.FilterBar, Open: a.filterBar, Group: -1}
	if n := a.activeFilterCount(); n > 0 {
		count := strconv.Itoa(n)
		c.Label, c.Short, c.Set = filterSetIcon+" Filter "+count, filterSetIcon+" "+count, true
	}
	return c
}

func (a App) plainControl(label string, b key.Binding) panels.HeaderControl {
	return panels.HeaderControl{Label: label, Key: b, Group: -1}
}

func (a App) pageHeader(width int) panels.PageHeader {
	p := panels.PageHeaderProps{
		Width:      width,
		Title:      a.headerTitle(),
		ShowBar:    a.filterBar,
		Keyword:    panels.Line(a.keyword.View(), styles.KeywordInputWidth),
		KeywordKey: a.keys.Search,
		Clear:      panels.HeaderControl{Label: clearIcon + " Clear", Short: clearIcon, Key: a.keys.ClearFilters, Group: -1},
	}
	fullscreen := a.plainControl(fullscreenIcon, a.keys.Fullscreen)
	fullscreen.Open = a.fullscreen
	for _, t := range a.barFilters() {
		p.Bar = append(p.Bar, a.filterControl(t, a.filterName(t)+": "))
	}
	switch a.hub {
	case panels.HubSprint:
		p.Action = a.plainControl(newIcon+"New task", a.keys.CreateTask)
		if a.sprintView == sprintBacklog {
			p.Action = a.plainControl(newIcon+"New Work Item", a.keys.CreateParent)
		}
		p.Pivots, p.Pivot = sprintPivots, a.sprintView
		sprint := a.filterControl(models.FilterTypeSprint, "")
		sprint.Label, sprint.Set = sprintIcon+sprint.Label, false
		p.Controls = []panels.HeaderControl{sprint, a.filterControl(models.FilterTypeAssigned, "Person: "), a.funnelControl(), fullscreen}
	case panels.HubBoard:
		p.Action = a.plainControl(newIcon+"New item", a.keys.CreateParent)
		p.Pivots = []string{"Board"}
		link := a.plainControl(viewAsIcon+"View as Backlog", a.keys.HubBacklog)
		p.Link = &link
		p.Controls = []panels.HeaderControl{a.funnelControl(), fullscreen}
	default:
		p.Action = a.plainControl(newIcon+"New Work Item", a.keys.CreateParent)
		p.Pivots = []string{"Backlog"}
		link := a.plainControl(viewAsIcon+"View as Board", a.keys.HubBoard)
		p.Link = &link
		p.Controls = []panels.HeaderControl{a.funnelControl(), fullscreen}
	}
	return panels.RenderPageHeader(p)
}

func (a App) headerTitle() string {
	team := a.client.Team()
	if team == "" {
		team = a.client.Project()
	}
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	title := bold.Render(team)
	if a.hub != panels.HubSprint {
		return title
	}
	return title + panels.Muted(titleSeparator) + bold.Render(a.sprintName()) + panels.Muted(a.sprintDates())
}

func (a App) pickerAnchor(group int) (int, int, bool) {
	l := a.layout()
	anchor, ok := a.pageHeader(l.mainWidth).Anchor(group)
	if !ok {
		return 0, 0, false
	}
	return l.railWidth + anchor.X, styles.TopBarHeight + anchor.Y + 1, true
}

func (a *App) openFilterPicker(group int) tea.Cmd {
	if group < 0 || group >= len(a.filters.Groups) {
		return nil
	}
	g := *a.filters.Groups[group]
	if a.inBar(g.Type) {
		a.filterBar = true
	}
	return a.openModal(modals.NewFilterPickerModal(a.env(), group, g, a.filterName(g.Type), a.sprintName()))
}

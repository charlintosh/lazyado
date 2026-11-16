package modals

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	filterPickerBorder  = 2
	filterPickerPadding = 1
	idleSectionLabel    = "Team, nothing assigned"
	youNote             = "you"
)

type FilterOptionChosenMsg struct {
	Group int
	Value string
}

type FilterPickerModal struct {
	env      Env
	group    int
	title    string
	scope    string
	dropdown widgets.Dropdown
}

func NewFilterPickerModal(env Env, index int, group models.FilterGroup, title, scope string) FilterPickerModal {
	items := make([]widgets.DropdownItem, 0, len(group.Options))
	initial := ""
	for _, o := range group.Options {
		if o.Selected {
			initial = o.Value
		}
		items = append(items, dropdownItem(o, group.ShowCounts, scope))
	}
	m := FilterPickerModal{
		env:      env,
		group:    index,
		title:    title,
		scope:    scope,
		dropdown: widgets.NewDropdown(env.Keys, items, styles.FilterPickerRows, "Type to filter"),
	}
	m.layout()
	m.dropdown.Select(initial)
	m.dropdown.Focus()
	return m
}

func dropdownItem(o models.FilterOption, counts bool, scope string) widgets.DropdownItem {
	it := widgets.DropdownItem{Label: o.Label, Value: o.Value, Marked: o.Selected}
	if counts && o.Count != models.UnknownCount {
		it.Count = strconv.Itoa(o.Count)
	}
	if o.You {
		it.Note = youNote
	}
	switch o.Section {
	case models.SectionActive:
		it.Group = "Active in " + scope
	case models.SectionIdle:
		it.Group = idleSectionLabel
		it.Muted = true
	default:
		return it
	}
	it.Badge = widgets.Initials(o.Label)
	it.BadgeColor = styles.AvatarColor(o.Label)
	return it
}

func (m FilterPickerModal) Group() int {
	return m.group
}

func (m FilterPickerModal) Init() tea.Cmd {
	return m.dropdown.Init()
}

func (m FilterPickerModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			if m.dropdown.Query() != "" {
				m.dropdown.SetQuery("")
				return m, nil
			}
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Select):
			if it, ok := m.dropdown.Selected(); ok {
				return m, send(FilterOptionChosenMsg{Group: m.group, Value: it.Value})
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.dropdown, cmd = m.dropdown.Update(msg)
	return m, cmd
}

func (m FilterPickerModal) width() int {
	return max(min(styles.FilterPickerWidth, m.env.Width-filterPickerBorder), filterPickerBorder+2*filterPickerPadding+1)
}

func (m FilterPickerModal) innerWidth() int {
	return max(m.width()-filterPickerBorder-2*filterPickerPadding, 1)
}

func (m *FilterPickerModal) layout() {
	m.dropdown.SetWidth(m.innerWidth())
	m.dropdown.SetRows(m.env.Height - styles.FilterPickerChrome - styles.TopBarHeight - styles.PageHeaderHeight)
}

func (m FilterPickerModal) keyLabel() string {
	k := m.env.Keys
	bindings := []key.Binding{k.Panel1, k.Panel2, k.Panel3, k.Panel4}
	if m.group < 0 || m.group >= len(bindings) {
		return ""
	}
	return bindings[m.group].Help().Key
}

func (m FilterPickerModal) View() string {
	inner := m.innerWidth()
	badge := lipgloss.NewStyle().Background(styles.ColorPrimary).Foreground(styles.ColorOnAccent).Bold(true).Padding(0, 1).Render(m.keyLabel())
	title := badge + " " + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(m.title)
	scope := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(ansi.Truncate(m.scope, max(inner-lipgloss.Width(title)-1, 0), "…"))
	gap := max(inner-lipgloss.Width(title)-lipgloss.Width(scope), 1)
	header := lipgloss.NewStyle().MaxWidth(inner).Render(title + strings.Repeat(" ", gap) + scope)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", inner))
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Combine("move", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "apply"), widgets.Relabel(k.Back, "close"))
	body := lipgloss.JoinVertical(lipgloss.Left, header, m.dropdown.View(), rule, footer)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, filterPickerPadding).
		Width(m.width()).
		Render(body)
}

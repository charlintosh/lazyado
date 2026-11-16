package modals

import (
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
)

const helpDialogMaxWidth = 160

type HelpModal struct {
	env      Env
	width    int
	viewport viewport.Model
}

func NewHelpModal(env Env) HelpModal {
	m := HelpModal{env: env, viewport: widgets.NewViewport(env.Keys)}
	m.layout()
	return m
}

func (m HelpModal) Init() tea.Cmd { return nil }

func (m HelpModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, m.env.Keys.Help) || key.Matches(msg, m.env.Keys.Back) {
			return m, closeModal
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *HelpModal) layout() {
	available := widgets.DialogInnerWidth(widgets.DialogWidth(helpDialogMaxWidth, m.env.Width))
	content := helpContent(m.env.Keys.FullHelp(), available)
	m.width = widgets.DialogWidth(widgets.DialogOuterWidth(lipgloss.Width(content)), m.env.Width)
	m.viewport.SetWidth(widgets.DialogInnerWidth(m.width))
	m.viewport.SetHeight(min(lipgloss.Height(content), widgets.DialogBodyHeight(m.env.Height, false)))
	m.viewport.SetContent(content)
}

func helpContent(groups [][]key.Binding, width int) string {
	h := widgets.NewHelp()
	var content string
	for perRow := len(groups); perRow >= 1; perRow-- {
		var rows []string
		for chunk := range slices.Chunk(groups, perRow) {
			rows = append(rows, h.FullHelpView(chunk))
		}
		content = lipgloss.JoinVertical(lipgloss.Left, rows...)
		if lipgloss.Width(content) <= width {
			break
		}
	}
	return content
}

func (m HelpModal) View() string {
	k := m.env.Keys
	bindings := []key.Binding{widgets.Combine("close", k.Help, k.Back)}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		bindings = append([]key.Binding{widgets.Combine("scroll", k.Up, k.Down)}, bindings...)
	}
	return widgets.Dialog{
		Title:  "Keyboard shortcuts",
		Body:   m.viewport.View(),
		Footer: widgets.Hints(widgets.DialogInnerWidth(m.width), bindings...),
		Width:  m.width,
	}.View()
}

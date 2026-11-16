package modals

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/styles"
)

type ErrorModal struct {
	env      Env
	text     string
	viewport viewport.Model
}

func NewErrorModal(env Env, err error) ErrorModal {
	text := "Unknown error"
	if err != nil {
		text = err.Error()
	}
	m := ErrorModal{env: env, text: text, viewport: widgets.NewViewport(env.Keys)}
	m.layout()
	return m
}

func (m ErrorModal) Init() tea.Cmd { return nil }

func (m ErrorModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		k := m.env.Keys
		if key.Matches(msg, k.Back) || key.Matches(msg, k.Select) || key.Matches(msg, k.Quit) {
			return m, closeModal
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ErrorModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	content := lipgloss.NewStyle().Foreground(styles.ColorText).Width(inner).Render(m.text)
	m.viewport.SetWidth(inner)
	m.viewport.SetHeight(min(lipgloss.Height(content), widgets.DialogBodyHeight(m.env.Height, false)))
	m.viewport.SetContent(content)
}

func (m ErrorModal) width() int {
	return widgets.DialogWidth(styles.ModalWidthXXL, m.env.Width)
}

func (m ErrorModal) View() string {
	k := m.env.Keys
	bindings := []key.Binding{widgets.Combine("close", k.Select, k.Back)}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		bindings = append([]key.Binding{widgets.Relabel(k.Down, "scroll")}, bindings...)
	}
	return widgets.Dialog{
		Title:  "Error",
		Body:   m.viewport.View(),
		Footer: widgets.Hints(widgets.DialogInnerWidth(m.width()), bindings...),
		Width:  m.width(),
		Accent: styles.ColorError,
	}.View()
}

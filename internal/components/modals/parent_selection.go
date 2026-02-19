package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ParentSelectionModal lets the user pick between PBI and Bug when creating a parent
type ParentSelectionModal struct {
	visible bool
	cursor  int
	options []string
	styles  styles.Styles
	keys    keys.KeyMap
	width   int
	height  int
}

// NewParentSelectionModal creates the selection modal
func NewParentSelectionModal(styles styles.Styles, keys keys.KeyMap) ParentSelectionModal {
	return ParentSelectionModal{
		options: []string{"Product Backlog Item", "Bug"},
		styles:  styles,
		keys:    keys,
	}
}

func (m ParentSelectionModal) Init() tea.Cmd { return nil }

// ParentSelectionChosenMsg is sent when user picks a parent type
type ParentSelectionChosenMsg struct {
	Type string
}

func (m ParentSelectionModal) Update(msg tea.Msg) (ParentSelectionModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Select):
			sel := m.options[m.cursor]
			m.visible = false
			return m, func() tea.Msg { return ParentSelectionChosenMsg{Type: sel} }
		case key.Matches(msg, m.keys.Back):
			m.visible = false
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}
	return m, nil
}

func (m ParentSelectionModal) View() string {
	if !m.visible {
		return ""
	}
	var b strings.Builder
	title := m.styles.ModalTitle.Render("Create Parent Work Item")
	b.WriteString(title + "\n\n")
	for i, o := range m.options {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = "▸ "
			style = style.Bold(true).Foreground(styles.ColorPrimary)
		}
		b.WriteString(cursor + style.Render(o) + "\n")
	}
	sel := m.keys.Select.Help()
	back := m.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s  %s: %s", sel.Key, sel.Desc, back.Key, back.Desc)
	b.WriteString("\n" + m.styles.ModalInstructions.Render(instr))

	content := m.styles.ModalBox.Width(styles.ModalWidthSM).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *ParentSelectionModal) SetSize(w, h int) { m.width = w; m.height = h }
func (m *ParentSelectionModal) SetVisible(v bool) {
	m.visible = v
	if v {
		m.cursor = 0
	}
}
func (m *ParentSelectionModal) IsVisible() bool { return m.visible }

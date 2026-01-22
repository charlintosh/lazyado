package modals

import (
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorModal displays API errors in a copyable modal
type ErrorModal struct {
	visible bool
	errMsg  string
	styles  styles.Styles
	keys    keys.KeyMap
	width   int
	height  int
}

// NewErrorModal creates a new error modal
func NewErrorModal(st styles.Styles, k keys.KeyMap) ErrorModal {
	return ErrorModal{
		styles: st,
		keys:   k,
	}
}

// Init initializes the modal
func (m ErrorModal) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ErrorModal) Update(msg tea.Msg) (ErrorModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Select) || key.Matches(msg, m.keys.Quit) {
			m.visible = false
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}

	return m, nil
}

// View renders the modal
func (m ErrorModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := 80
	modalHeight := 20

	// Build content
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorError).
		Render("⚠ Error")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Error message with wrapping
	errorStyle := lipgloss.NewStyle().
		Foreground(styles.ColorText).
		Width(modalWidth - 6).
		MaxWidth(modalWidth - 6)

	b.WriteString(errorStyle.Render(m.errMsg))
	b.WriteString("\n\n")

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Render("Press Enter/Esc to close • Select text to copy")
	b.WriteString(instructions)

	// Style the modal
	content := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorError).
		Render(b.String())

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// SetSize sets the modal container size
func (m *ErrorModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetVisible sets the modal visibility
func (m *ErrorModal) SetVisible(visible bool) {
	m.visible = visible
}

// IsVisible returns whether the modal is visible
func (m *ErrorModal) IsVisible() bool {
	return m.visible
}

// SetError sets the error message to display
func (m *ErrorModal) SetError(err error) {
	if err != nil {
		m.errMsg = err.Error()
	} else {
		m.errMsg = ""
	}
}

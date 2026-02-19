package modals

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// QuickSearchModal is a modal for quickly searching work items by ID
type QuickSearchModal struct {
	visible      bool
	searchInput  textinput.Model
	loading      bool
	errorMessage string
	spinner      spinner.Model
	styles       styles.Styles
	keys         keys.KeyMap
	width        int
	height       int
}

// NewQuickSearchModal creates a new quick search modal
func NewQuickSearchModal(st styles.Styles, k keys.KeyMap) QuickSearchModal {
	ti := textinput.New()
	ti.Placeholder = "Enter work item ID..."
	ti.CharLimit = styles.InputCharLimitSM
	ti.Prompt = "🔍 "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.ColorText)
	ti.Validate = func(s string) error {
		// Only allow numeric input
		for _, r := range s {
			if r < '0' || r > '9' {
				return fmt.Errorf("only numbers allowed")
			}
		}
		return nil
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)

	return QuickSearchModal{
		styles:      st,
		keys:        k,
		searchInput: ti,
		spinner:     sp,
	}
}

// Init initializes the modal
func (m QuickSearchModal) Init() tea.Cmd {
	return nil
}

// Update handles messages for the quick search modal
func (m QuickSearchModal) Update(msg tea.Msg) (QuickSearchModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If loading, ignore input except Esc
		if m.loading {
			if msg.Type == tea.KeyEsc {
				m.visible = false
				m.loading = false
				return m, func() tea.Msg { return ModalClosedMsg{} }
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEsc:
			// Cancel search
			m.visible = false
			m.errorMessage = ""
			m.searchInput.SetValue("")
			return m, func() tea.Msg { return ModalClosedMsg{} }

		case tea.KeyEnter:
			// Validate and submit search
			value := strings.TrimSpace(m.searchInput.Value())
			if value == "" {
				m.errorMessage = "Please enter a work item ID"
				return m, nil
			}

			// Parse ID
			id, err := strconv.Atoi(value)
			if err != nil {
				m.errorMessage = "ID must be a number"
				return m, nil
			}

			if id <= 0 {
				m.errorMessage = "ID must be a positive number"
				return m, nil
			}

			// Clear error and emit request
			m.errorMessage = ""
			return m, func() tea.Msg {
				return QuickSearchRequestMsg{ID: id}
			}

		default:
			// Update search input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			// Clear error when user starts typing
			if m.errorMessage != "" {
				m.errorMessage = ""
			}
			return m, cmd
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the quick search modal
func (m QuickSearchModal) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Quick Search Work Item"))
	b.WriteString("\n\n")

	// Show loading state or input
	if m.loading {
		loadingText := m.spinner.View() + " Fetching work item..."
		loadingStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary)
		b.WriteString(loadingStyle.Render(loadingText))
	} else {
		// Search input
		b.WriteString(m.searchInput.View())
	}

	b.WriteString("\n")

	// Error message
	if m.errorMessage != "" {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().
			Background(styles.ColorError).
			Foreground(styles.ColorText).
			Bold(true).
			Padding(0, 2)
		b.WriteString(errorStyle.Render("✗ " + m.errorMessage))
	}

	// Help text
	b.WriteString("\n\n")
	if m.loading {
		instr := "Esc: cancel"
		b.WriteString(m.styles.ModalInstructions.Render(instr))
	} else {
		sel := m.keys.Select.Help()
		back := m.keys.Back.Help()
		instr := fmt.Sprintf("%s: search  %s: cancel", sel.Key, back.Key)
		b.WriteString(m.styles.ModalInstructions.Render(instr))
	}

	content := b.String()

	// Modal styling
	modalWidth := styles.ModalWidthMD
	modalHeight := styles.ModalHeightLG

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(styles.ModalPaddingV, styles.ModalPaddingH).
		Width(modalWidth).
		Height(modalHeight).
		Render(content)

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
	)
}

// SetVisible sets the visibility of the modal
func (m *QuickSearchModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.errorMessage = ""
		m.loading = false
	} else {
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.errorMessage = ""
		m.loading = false
	}
}

// IsVisible returns whether the modal is visible
func (m QuickSearchModal) IsVisible() bool {
	return m.visible
}

// SetSize sets the size of the modal container
func (m *QuickSearchModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetLoading sets the loading state
func (m *QuickSearchModal) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.searchInput.Blur()
	}
}

func (m *QuickSearchModal) SetError(err error) {
	m.loading = false
	m.searchInput.Focus()
	if err != nil {
		msg := err.Error()
		if idx := strings.Index(msg, "TF"); idx >= 0 {
			if end := strings.Index(msg[idx:], ":"); end > 0 {
				msg = strings.TrimSpace(msg[idx+end+1:])
			}
		}
		if strings.HasPrefix(msg, "API error") {
			msg = "Work item not found or you don't have permissions"
		}
		m.errorMessage = msg
	} else {
		m.errorMessage = ""
	}
}

// GetSpinnerCmd returns the spinner tick command when loading
func (m QuickSearchModal) GetSpinnerCmd() tea.Cmd {
	if m.loading && m.visible {
		return m.spinner.Tick
	}
	return nil
}

// Message types

// QuickSearchRequestMsg is emitted when user submits a work item ID
type QuickSearchRequestMsg struct {
	ID int
}

// QuickSearchSuccessMsg is emitted when work item is fetched successfully
type QuickSearchSuccessMsg struct {
	Item *models.WorkItem
}

// QuickSearchErrorMsg is emitted when work item fetch fails
type QuickSearchErrorMsg struct {
	Err error
}

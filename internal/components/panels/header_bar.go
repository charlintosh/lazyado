package panels

import (
	"fmt"

	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HeaderBar renders the top bar with title on left and notification on right
type HeaderBar struct {
	organization string
	project      string
	loading      bool
	notification *notification.Notification
	styles       styles.Styles
	width        int
	spinner      spinner.Model
}

// NewHeaderBar creates a new header bar component
func NewHeaderBar(styles styles.Styles) HeaderBar {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")) // Vibrant magenta
	sp.Spinner = spinner.Points

	return HeaderBar{
		styles:  styles,
		spinner: sp,
	}
}

// SetOrganization sets the organization name
func (h *HeaderBar) SetOrganization(org string) {
	h.organization = org
}

// SetProject sets the project name
func (h *HeaderBar) SetProject(project string) {
	h.project = project
}

// SetLoading sets the loading state
func (h *HeaderBar) SetLoading(loading bool) {
	h.loading = loading
}

// SetNotification sets the notification component
func (h *HeaderBar) SetNotification(notif *notification.Notification) {
	h.notification = notif
}

// SetWidth sets the width of the header bar
func (h *HeaderBar) SetWidth(width int) {
	h.width = width
}

// Init returns the initial command for the header bar
func (h HeaderBar) Init() tea.Cmd {
	return h.spinner.Tick
}

// Update handles messages for the header bar
func (h HeaderBar) Update(msg tea.Msg) (HeaderBar, tea.Cmd) {
	var cmd tea.Cmd
	h.spinner, cmd = h.spinner.Update(msg)
	return h, cmd
}

// View renders the header bar
func (h HeaderBar) View() string {
	var content string

	// If notification is visible, show ONLY the notification (centered)
	if h.notification != nil && h.notification.IsVisible() {
		content = h.notification.View()
	} else {
		// Otherwise show title: "org/project boards" with optional loading spinner
		titleText := fmt.Sprintf("%s/%s boards", h.organization, h.project)
		if h.loading {
			titleText += " " + h.spinner.View()
		}
		content = h.styles.Title.Render(titleText)
	}

	// Center the content in the full width
	return lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center).
		Render(content)
}

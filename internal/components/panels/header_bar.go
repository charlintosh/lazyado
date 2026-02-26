package panels

import (
	"fmt"

	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AppTab int

const (
	TabBoards AppTab = iota
	TabPullRequests
)

type TabChangedMsg struct {
	Tab AppTab
}

type HeaderBar struct {
	organization string
	project      string
	loading      bool
	notification *notification.Notification
	styles       styles.Styles
	width        int
	spinner      spinner.Model
	activeTab    AppTab
}

func NewHeaderBar(styles styles.Styles) HeaderBar {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
	sp.Spinner = spinner.Points

	return HeaderBar{
		styles:    styles,
		spinner:   sp,
		activeTab: TabBoards,
	}
}

func (h *HeaderBar) SetOrganization(org string) {
	h.organization = org
}

func (h *HeaderBar) SetProject(project string) {
	h.project = project
}

func (h *HeaderBar) SetLoading(loading bool) {
	h.loading = loading
}

func (h *HeaderBar) SetNotification(notif *notification.Notification) {
	h.notification = notif
}

func (h *HeaderBar) SetWidth(width int) {
	h.width = width
}

func (h *HeaderBar) SetActiveTab(tab AppTab) {
	h.activeTab = tab
}

func (h HeaderBar) ActiveTab() AppTab {
	return h.activeTab
}

func (h HeaderBar) Init() tea.Cmd {
	return h.spinner.Tick
}

func (h HeaderBar) Update(msg tea.Msg) (HeaderBar, tea.Cmd) {
	var cmd tea.Cmd
	h.spinner, cmd = h.spinner.Update(msg)
	return h, cmd
}

func (h HeaderBar) View() string {
	var content string

	if h.notification != nil && h.notification.IsVisible() {
		content = h.notification.View()
	} else {
		activeStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorText).
			Background(styles.ColorPrimary).
			Padding(0, 1)

		inactiveStyle := lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Padding(0, 1)

		prefix := fmt.Sprintf("%s/%s", h.organization, h.project)
		prefixRendered := h.styles.Title.Render(prefix)

		boardsTab := inactiveStyle.Render("Boards")
		prTab := inactiveStyle.Render("Pull Requests")
		if h.activeTab == TabBoards {
			boardsTab = activeStyle.Render("Boards")
		} else {
			prTab = activeStyle.Render("Pull Requests")
		}

		separator := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(" │ ")
		tabs := boardsTab + separator + prTab

		spinnerView := ""
		if h.loading {
			spinnerView = " " + h.spinner.View()
		}
		content = prefixRendered + "  " + tabs + spinnerView
	}

	return lipgloss.NewStyle().
		Width(h.width).
		Align(lipgloss.Center).
		Render(content)
}

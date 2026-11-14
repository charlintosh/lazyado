package panels

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

type HeaderBar struct {
	organization string
	project      string
	crumbs       []string
	loading      bool
	notification *notification.Notification
	styles       styles.Styles
	keys         keys.KeyMap
	width        int
	spinner      spinner.Model
}

// NewHeaderBar creates a new header bar component
func NewHeaderBar(st styles.Styles, k keys.KeyMap) HeaderBar {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Background(styles.ColorSurface)
	sp.Spinner = spinner.MiniDot

	return HeaderBar{
		styles:  st,
		keys:    k,
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

func (h *HeaderBar) SetCrumbs(crumbs ...string) {
	h.crumbs = crumbs
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
	bg := styles.ColorSurface
	on := func(fg color.Color, bold bool) lipgloss.Style {
		return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(bold)
	}
	sep := on(styles.ColorMuted, false).Render(" / ")

	parts := []string{
		on(styles.ColorPrimary, true).Render(" ◆ lazyado"),
		on(styles.ColorBorder, false).Render(" │ "),
	}
	trail := append([]string{h.organization, h.project}, h.crumbs...)
	var crumbParts []string
	for i, c := range trail {
		if c == "" {
			continue
		}
		fg := styles.ColorTextMuted
		bold := false
		switch {
		case i == len(trail)-1:
			fg = styles.ColorText
			bold = true
		case i == 1:
			fg = styles.ColorText
		}
		crumbParts = append(crumbParts, on(fg, bold).Render(c))
	}
	left := strings.Join(parts, "") + strings.Join(crumbParts, sep)

	var right []string
	if h.notification != nil && h.notification.IsVisible() {
		right = append(right, h.notification.View())
	} else {
		search := h.keys.QuickSearch.Help()
		right = append(right,
			on(styles.ColorTextMuted, false).Render("⌕ Search work items ")+on(styles.ColorLight, false).Render(search.Key))
		if h.loading {
			right = append(right, h.spinner.View()+on(styles.ColorTextMuted, false).Render(" syncing"))
		} else {
			right = append(right, on(styles.ColorSuccess, false).Render("● synced"))
		}
	}
	return JoinBar([]string{left}, right, h.width, bg)
}

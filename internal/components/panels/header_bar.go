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
	sp.Spinner = spinner.Line

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
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Background(bg).Foreground(fg)
	}
	product := lipgloss.NewStyle().Background(styles.ColorPrimary).Foreground(styles.ColorOnAccent).Bold(true).Padding(0, 1).Render("lazyado")

	trail := append([]string{h.organization, h.project}, h.crumbs...)
	switch {
	case h.width < styles.HeaderCompactWidth:
		trail = []string{"", "", trail[len(trail)-1]}
	case h.width < styles.HeaderFullWidth:
		trail = append([]string{"", h.project}, trail[len(trail)-1])
	}
	var crumbs []string
	for i, c := range trail {
		if c == "" {
			continue
		}
		st := on(styles.ColorTextMuted)
		if i == 1 {
			st = on(styles.ColorText)
		}
		if i == len(trail)-1 {
			st = on(styles.ColorText).Bold(true)
		}
		crumbs = append(crumbs, st.Render(c))
	}
	left := product + on(styles.ColorText).Render(" ") + strings.Join(crumbs, on(styles.ColorMuted).Render(" / "))

	var right string
	if h.notification != nil && h.notification.IsVisible() {
		right = h.notification.View()
	} else {
		search := on(styles.ColorTextMuted).Render("Search ") + on(styles.ColorPrimary).Bold(true).Render(h.keys.QuickSearch.Help().Key)
		status := on(styles.ColorSuccess).Render("Synced")
		if h.loading {
			status = h.spinner.View() + on(styles.ColorTextMuted).Render(" Syncing")
		}
		right = status + on(styles.ColorText).Render(" ")
		if h.width >= styles.HeaderFullWidth {
			right = search + on(styles.ColorText).Render("   ") + right
		}
	}
	right = on(styles.ColorText).Render("  ") + right
	leftWidth := max(h.width-lipgloss.Width(right), 0)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Background(bg).Width(leftWidth).MaxWidth(leftWidth).Inline(true).Render(left),
		right,
	)
}

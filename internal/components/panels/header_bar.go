package panels

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
)

type HeaderProps struct {
	Width        int
	Organization string
	Project      string
	Crumbs       []string
	Loading      bool
	Spinner      string
	Notice       string
	SearchKey    string
}

func RenderHeader(p HeaderProps) string {
	bg := styles.ColorSurface
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Background(bg).Foreground(fg)
	}
	product := lipgloss.NewStyle().Background(styles.ColorPrimary).Foreground(styles.ColorOnAccent).Bold(true).Padding(0, 1).Render("lazyado")
	left := product + on(styles.ColorText).Render(" ") + strings.Join(headerCrumbs(p, on), on(styles.ColorMuted).Render(" / "))

	right := p.Notice
	if right == "" {
		status := on(styles.ColorSuccess).Render("Synced")
		if p.Loading {
			status = on(styles.ColorPrimary).Render(p.Spinner) + on(styles.ColorTextMuted).Render(" Syncing")
		}
		right = status + on(styles.ColorText).Render(" ")
		if p.Width >= styles.HeaderFullWidth {
			search := on(styles.ColorTextMuted).Render("Search ") + on(styles.ColorPrimary).Bold(true).Render(p.SearchKey)
			right = search + on(styles.ColorText).Render("   ") + right
		}
	}
	right = on(styles.ColorText).Render("  ") + right
	leftWidth := max(p.Width-lipgloss.Width(right), 0)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Background(bg).Width(leftWidth).MaxWidth(leftWidth).Inline(true).Render(left),
		right,
	)
}

func headerCrumbs(p HeaderProps, on func(color.Color) lipgloss.Style) []string {
	trail := append([]string{p.Organization, p.Project}, p.Crumbs...)
	switch {
	case p.Width < styles.HeaderCompactWidth:
		trail = []string{"", "", trail[len(trail)-1]}
	case p.Width < styles.HeaderFullWidth:
		trail = []string{"", p.Project, trail[len(trail)-1]}
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
	return crumbs
}

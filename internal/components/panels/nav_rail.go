package panels

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	railBar       = "┃"
	railGroupIcon = "▾"
)

type RailProps struct {
	Width   int
	Height  int
	Project string
	Hub     Hub
	Compact bool
	Keys    keys.KeyMap
}

type railEntry struct {
	hub     Hub
	binding key.Binding
}

type railGroup struct {
	name    string
	entries []railEntry
}

func railGroups(k keys.KeyMap) []railGroup {
	return []railGroup{
		{HubBoard.Section(), []railEntry{{HubBoard, k.HubBoard}, {HubBacklog, k.HubBacklog}, {HubSprint, k.HubSprint}}},
		{HubPullRequests.Section(), []railEntry{{HubPullRequests, k.HubPullRequests}}},
	}
}

func RenderRail(p RailProps) string {
	if p.Width <= 0 || p.Height <= 0 {
		return ""
	}
	inner := max(p.Width-1, 1)
	lines := railFull(p, inner)
	if p.Compact {
		lines = railCompact(p, inner)
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(styles.ColorBorder).
		Width(p.Width).Height(p.Height).
		MaxWidth(p.Width).MaxHeight(p.Height).
		Render(strings.Join(lines, "\n"))
}

func railFull(p RailProps, inner int) []string {
	badge := Badge(widgets.Initials(p.Project), styles.ColorPrimary, styles.ColorOnAccent)
	name := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).
		Render(ansi.Truncate(p.Project, max(inner-lipgloss.Width(badge)-3, 1), "…"))
	lines := []string{"", " " + badge + " " + name}
	for _, g := range railGroups(p.Keys) {
		title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(g.name)
		lines = append(lines, "", " "+Muted(railGroupIcon)+" "+title)
		for _, e := range g.entries {
			lines = append(lines, " "+railItem(e.hub.Name(), e.binding.Help().Key, e.hub == p.Hub, inner-2))
		}
	}
	return lines
}

func railCompact(p RailProps, inner int) []string {
	initials := lipgloss.NewStyle().Background(styles.ColorPrimary).Foreground(styles.ColorOnAccent).Bold(true).
		Render(widgets.Initials(p.Project))
	lines := []string{"", lipgloss.PlaceHorizontal(inner, lipgloss.Center, initials)}
	for _, g := range railGroups(p.Keys) {
		lines = append(lines, "")
		for _, e := range g.entries {
			lines = append(lines, railKey(e.binding.Help().Key, e.hub == p.Hub, inner))
		}
	}
	return lines
}

func railItem(name, shortcut string, selected bool, width int) string {
	bg := lipgloss.NewStyle()
	bar := " "
	fg := styles.ColorLight
	if selected {
		bg = bg.Background(styles.ColorHighlight)
		bar = railBar
		fg = styles.ColorText
	}
	gap := max(width-2-lipgloss.Width(name)-lipgloss.Width(shortcut)-1, 1)
	return bg.Foreground(styles.ColorPrimary).Render(bar) +
		bg.Render(" ") +
		bg.Foreground(fg).Bold(selected).Render(name) +
		bg.Render(strings.Repeat(" ", gap)) +
		bg.Foreground(styles.ColorTextMuted).Render(shortcut) +
		bg.Render(" ")
}

func railKey(shortcut string, selected bool, width int) string {
	bg := lipgloss.NewStyle()
	bar := " "
	fg := styles.ColorLight
	if selected {
		bg = bg.Background(styles.ColorHighlight)
		bar = railBar
		fg = styles.ColorText
	}
	pad := max(width-2-lipgloss.Width(shortcut), 0)
	return bg.Foreground(styles.ColorPrimary).Render(bar) +
		bg.Render(" ") +
		bg.Foreground(fg).Bold(true).Render(shortcut) +
		bg.Render(strings.Repeat(" ", pad))
}

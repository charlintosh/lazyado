package widgets

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	dialogBorder   = 2
	dialogPaddingV = 1
	dialogPaddingH = 2
	dialogMargin   = 2
	dialogMinWidth = 24
)

type Dialog struct {
	Title    string
	Subtitle string
	Body     string
	Footer   string
	Width    int
	Accent   color.Color
}

func DialogWidth(preferred, screen int) int {
	return max(min(preferred, screen-dialogMargin), min(dialogMinWidth, screen))
}

func DialogInnerWidth(width int) int {
	return max(width-dialogBorder-2*dialogPaddingH, 1)
}

func DialogOuterWidth(inner int) int {
	return inner + dialogBorder + 2*dialogPaddingH
}

func DialogChromeHeight(subtitle bool) int {
	h := dialogBorder + 2*dialogPaddingV + 4
	if subtitle {
		h++
	}
	return h
}

func DialogBodyHeight(screen int, subtitle bool) int {
	return max(screen-dialogMargin-DialogChromeHeight(subtitle), 1)
}

func (d Dialog) View() string {
	accent := d.Accent
	if accent == nil {
		accent = styles.ColorPrimary
	}
	inner := DialogInnerWidth(d.Width)
	parts := []string{lipgloss.NewStyle().Bold(true).Foreground(accent).Render(ansi.Truncate(d.Title, inner, "..."))}
	if d.Subtitle != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(ansi.Truncate(d.Subtitle, inner, "...")))
	}
	parts = append(parts, "", d.Body)
	if d.Footer != "" {
		parts = append(parts, "", d.Footer)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(dialogPaddingV, dialogPaddingH).
		Width(d.Width).
		Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func Overlay(base, box string, width, height int) string {
	box = lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(box)
	x := max((width-lipgloss.Width(box))/2, 0)
	y := max((height-lipgloss.Height(box))/2, 0)
	return OverlayAt(base, box, x, y, width, height)
}

func OverlayAt(base, box string, x, y, width, height int) string {
	box = lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(box)
	x = max(min(x, width-lipgloss.Width(box)), 0)
	y = max(min(y, height-lipgloss.Height(box)), 0)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	).Render()
}

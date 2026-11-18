package widgets

import (
	"image/color"
	"strings"

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
	if framed, ok := frameDialog(strings.Join(parts, "\n"), d.Width, accent); ok {
		return framed
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(dialogPaddingV, dialogPaddingH).
		Width(d.Width).
		Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func frameDialog(content string, width int, accent color.Color) (string, bool) {
	inner := DialogInnerWidth(width)
	body, ok := fitBlock(content, inner, strings.Count(content, "\n")+1)
	if !ok || width < dialogBorder+2*dialogPaddingH+1 {
		return "", false
	}
	border := lipgloss.RoundedBorder()
	edge := lipgloss.NewStyle().Foreground(accent)
	side := edge.Render(border.Left)
	pad := strings.Repeat(" ", dialogPaddingH)
	blank := side + strings.Repeat(" ", width-dialogBorder) + side
	var b strings.Builder
	b.WriteString(edge.Render(border.TopLeft + strings.Repeat(border.Top, width-dialogBorder) + border.TopRight))
	for range dialogPaddingV {
		b.WriteString("\n" + blank)
	}
	for line := range strings.SplitSeq(body, "\n") {
		b.WriteString("\n" + side + pad + line + pad + side)
	}
	for range dialogPaddingV {
		b.WriteString("\n" + blank)
	}
	b.WriteString("\n" + edge.Render(border.BottomLeft+strings.Repeat(border.Bottom, width-dialogBorder)+border.BottomRight))
	return b.String(), true
}

func fitBox(box string, width, height int) string {
	if lipgloss.Height(box) <= height && lipgloss.Width(box) <= width {
		return box
	}
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(box)
}

func Overlay(base, box string, width, height int) string {
	box = fitBox(box, width, height)
	x := max((width-lipgloss.Width(box))/2, 0)
	y := max((height-lipgloss.Height(box))/2, 0)
	return OverlayAt(base, box, x, y, width, height)
}

func Splice(base, box string, x, y, width, height int) string {
	boxWidth, boxHeight := lipgloss.Width(box), lipgloss.Height(box)
	x = max(min(x, width-boxWidth), 0)
	y = max(min(y, height-boxHeight), 0)
	lines := strings.Split(base, "\n")
	for i, row := range strings.Split(box, "\n") {
		n := y + i
		if n >= len(lines) {
			break
		}
		left := ansi.Truncate(lines[n], x, "")
		if gap := x - ansi.StringWidth(left); gap > 0 {
			left += strings.Repeat(" ", gap)
		}
		right := ansi.TruncateLeft(lines[n], x+ansi.StringWidth(row), "")
		lines[n] = left + ansi.ResetStyle + row + ansi.ResetStyle + right
	}
	return strings.Join(lines, "\n")
}

func OverlayAt(base, box string, x, y, width, height int) string {
	return Splice(base, fitBox(box, width, height), x, y, width, height)
}

package panels

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type Hub int

const (
	HubBacklog Hub = iota
	HubBoard
	HubSprint
)

func (h Hub) Name() string {
	switch h {
	case HubBoard:
		return "Boards"
	case HubSprint:
		return "Sprints"
	default:
		return "Backlogs"
	}
}

func (h Hub) Mode() string {
	switch h {
	case HubBoard:
		return "BOARD"
	case HubSprint:
		return "SPRINT"
	default:
		return "BACKLOG"
	}
}

func (h Hub) Icon() string {
	switch h {
	case HubBoard:
		return "▦"
	case HubSprint:
		return "◷"
	default:
		return "☰"
	}
}

func FitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > width {
		s = ansi.Truncate(s, width, "…")
	}
	if pad := width - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func FitBlock(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, l := range lines {
		lines[i] = FitWidth(l, width)
	}
	return strings.Join(lines, "\n")
}

func FillBlock(s string, width, height int, bg color.Color) string {
	fill := lipgloss.NewStyle().Background(bg)
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > width {
			l = ansi.Truncate(l, width, "…")
		}
		if pad := width - ansi.StringWidth(l); pad > 0 {
			l += fill.Render(strings.Repeat(" ", pad))
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}

func BoxTitle(number, title string, focused bool) string {
	numberColor := styles.ColorTextMuted
	titleColor := styles.ColorLight
	if focused {
		numberColor = styles.ColorPrimary
		titleColor = styles.ColorPrimary
	}
	label := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render(title)
	if number == "" {
		return label
	}
	return lipgloss.NewStyle().Foreground(numberColor).Render("["+number+"] ") + label
}

func RenderTitledBox(title, body string, width, height int, focused bool) string {
	if width < styles.BoxPaddedSize || height < styles.BoxBorderSize {
		return FitBlock("", width, height)
	}
	borderColor := styles.ColorBorder
	if focused {
		borderColor = styles.ColorPrimary
	}
	border := lipgloss.NewStyle().Foreground(borderColor)
	innerWidth := width - styles.BoxBorderSize

	label := ""
	if title != "" {
		label = " " + title + " "
	}
	if ansi.StringWidth(label) > innerWidth-1 {
		label = ansi.Truncate(label, innerWidth-1, "…")
	}
	rest := innerWidth - 1 - ansi.StringWidth(label)
	top := border.Render("┌─") + label + border.Render(strings.Repeat("─", rest)+"┐")

	contentWidth := width - styles.BoxPaddedSize
	contentHeight := height - styles.BoxBorderSize
	content := FitBlock(body, contentWidth, contentHeight)

	lines := make([]string, 0, height)
	lines = append(lines, top)
	side := border.Render("│")
	for _, l := range strings.Split(content, "\n") {
		lines = append(lines, side+" "+l+" "+side)
	}
	lines = append(lines, border.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return strings.Join(lines, "\n")
}

func BoxContentSize(width, height int) (int, int) {
	return max(width-styles.BoxPaddedSize, 0), max(height-styles.BoxBorderSize, 0)
}

func Hint(b key.Binding, bg color.Color) string {
	h := b.Help()
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(styles.ColorLight)
	if bg != nil {
		keyStyle = keyStyle.Background(bg)
		descStyle = descStyle.Background(bg)
	}
	return keyStyle.Render(h.Key) + descStyle.Render(" "+h.Desc)
}

func Hints(bg color.Color, bindings ...key.Binding) []string {
	out := make([]string, 0, len(bindings))
	for _, b := range bindings {
		out = append(out, Hint(b, bg))
	}
	return out
}

func ToolbarItem(icon, label string, b key.Binding) string {
	text := lipgloss.NewStyle().Foreground(styles.ColorText).Render(icon + " " + label)
	shortcut := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(" " + b.Help().Key)
	return text + shortcut
}

func Chip(text string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorLight).Background(styles.ColorChip).Render(" " + text + " ")
}

func StateDot(state string, bg color.Color) string {
	dot := lipgloss.NewStyle().Foreground(styles.StateColor(state))
	text := lipgloss.NewStyle().Foreground(styles.ColorText)
	if bg != nil {
		dot = dot.Background(bg)
		text = text.Background(bg)
	}
	return dot.Render("●") + text.Render(" "+state)
}

func TypeGlyph(workItemType string, bg color.Color) string {
	st := lipgloss.NewStyle().Foreground(styles.TypeColor(workItemType))
	if bg != nil {
		st = st.Background(bg)
	}
	return st.Render("■")
}

func SectionLabel(text string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(strings.ToUpper(text))
}

func ProgressBar(done, total, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	filled := done * width / total
	return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(styles.ColorChip).Render(strings.Repeat("█", width-filled))
}

func JoinBar(left, right []string, width int, bg color.Color) string {
	fill := lipgloss.NewStyle().Background(bg)
	sep := fill.Render("  ")
	r := strings.Join(right, sep)
	rw := ansi.StringWidth(r)
	if rw > 0 {
		r = fill.Render(" ") + r + fill.Render(" ")
		rw += 2
	}
	if rw > width {
		r, rw = "", 0
	}
	avail := width - rw
	l := ""
	for i, seg := range left {
		candidate := seg
		if i > 0 {
			candidate = l + sep + seg
		}
		if ansi.StringWidth(candidate) > avail {
			if i == 0 {
				l = ansi.Truncate(seg, avail, "")
			}
			break
		}
		l = candidate
	}
	gap := avail - ansi.StringWidth(l)
	return l + fill.Render(strings.Repeat(" ", max(gap, 0))) + r
}

func DoneCount(children []*models.WorkItem) (int, int) {
	done := 0
	for _, c := range children {
		if models.IsCompletedState(string(c.State)) {
			done++
		}
	}
	return done, len(children)
}

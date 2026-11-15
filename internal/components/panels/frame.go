package panels

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
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

func Block(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(s)
}

func Line(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Inline(true).Width(width).MaxWidth(width).Render(s)
}

func paneBorderColor(focused bool) color.Color {
	if focused {
		return styles.ColorPrimary
	}
	return styles.ColorBorder
}

func Pane(title, body string, width, height int, focused bool) string {
	if width < styles.BoxPaddedSize || height < styles.PaneChromeHeight {
		return Block("", width, height)
	}
	innerWidth, innerHeight := PaneContentSize(width, height)
	header := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.ColorBorder).
		Render(Line(title, innerWidth))
	content := Block(body, innerWidth, innerHeight)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(paneBorderColor(focused)).
		Padding(0, 1).
		Width(width).Height(height).
		MaxWidth(width).MaxHeight(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, content))
}

func PaneContentSize(width, height int) (int, int) {
	return max(width-styles.BoxPaddedSize, 0), max(height-styles.PaneChromeHeight, 0)
}

func Section(title, body string, width int, focused bool) string {
	innerWidth := max(width-styles.BoxPaddedSize, 1)
	body = lipgloss.NewStyle().Width(innerWidth).Render(body)
	return Pane(title, body, width, lipgloss.Height(body)+styles.PaneChromeHeight, focused)
}

func PaneTitle(number, title string, focused bool) string {
	titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	badgeBg := styles.ColorChip
	badgeFg := styles.ColorLight
	if focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
		badgeBg = styles.ColorPrimary
		badgeFg = styles.ColorOnAccent
	}
	if number == "" {
		return titleStyle.Render(title)
	}
	return Badge(number, badgeBg, badgeFg) + " " + titleStyle.Render(title)
}

func Tabs(labels []string, active int, focused bool) string {
	activeColor := styles.ColorText
	if focused {
		activeColor = styles.ColorPrimary
	}
	tabs := make([]string, 0, len(labels))
	for i, l := range labels {
		st := lipgloss.NewStyle().Padding(0, 1).Foreground(styles.ColorTextMuted)
		if i == active {
			st = st.Foreground(activeColor).Bold(true).Underline(true)
		}
		tabs = append(tabs, st.Render(l))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func Badge(text string, bg, fg color.Color) string {
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(true).Padding(0, 1).Render(text)
}

func TypeBadge(workItemType string, label string) string {
	return Badge(label, styles.TypeColor(workItemType), styles.ColorOnAccent)
}

func TypeText(item *models.WorkItem) string {
	return lipgloss.NewStyle().Foreground(styles.TypeColor(string(item.Type))).Bold(true).Render(item.ShortType())
}

func StateText(state string) string {
	return lipgloss.NewStyle().Foreground(styles.StateColor(state)).Render(state)
}

func Chip(text string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorLight).Background(styles.ColorChip).Padding(0, 1).Render(text)
}

func Chips(tags []string) string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, Chip(t))
	}
	return strings.Join(out, " ")
}

func Button(label string, b key.Binding) string {
	text := lipgloss.NewStyle().Foreground(styles.ColorText).Background(styles.ColorChip).Padding(0, 1).Render(label)
	shortcut := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(b.Help().Key)
	return text + " " + shortcut
}

func Muted(s string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(s)
}

func Heading(s string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(strings.ToUpper(s))
}

func FieldRows(rows [][2]string, labelWidth, width int) string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(labelWidth)
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, label.Render(r[0]), Line(r[1], max(width-labelWidth, 1))))
	}
	return strings.Join(lines, "\n")
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

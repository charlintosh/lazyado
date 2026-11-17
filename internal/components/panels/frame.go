package panels

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/help"
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
	HubPullRequests
	HubBuilds
	HubReleases
	HubFiles
	HubCommits
	HubBranches
	HubTags
)

func (h Hub) Section() string {
	switch h {
	case HubPullRequests, HubFiles, HubCommits, HubBranches, HubTags:
		return "Repos"
	case HubBuilds, HubReleases:
		return "Pipelines"
	}
	return "Boards"
}

func (h Hub) Name() string {
	switch h {
	case HubBoard:
		return "Boards"
	case HubSprint:
		return "Sprints"
	case HubPullRequests:
		return "Pull requests"
	case HubBuilds:
		return "Builds"
	case HubReleases:
		return "Releases"
	case HubFiles:
		return "Files"
	case HubCommits:
		return "Commits"
	case HubBranches:
		return "Branches"
	case HubTags:
		return "Tags"
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
	case HubPullRequests:
		return "PULL REQUESTS"
	case HubBuilds:
		return "BUILDS"
	case HubReleases:
		return "RELEASES"
	case HubFiles:
		return "FILES"
	case HubCommits:
		return "COMMITS"
	case HubBranches:
		return "BRANCHES"
	case HubTags:
		return "TAGS"
	default:
		return "BACKLOG"
	}
}

const blanks = "                                                                                                                                "

func Block(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if fitted, ok := fitBlock(s, width, height); ok {
		return fitted
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(s)
}

func fitBlock(s string, width, height int) (string, bool) {
	if strings.ContainsAny(s, "\t\r") {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(s) + width*height)
	n := 0
	for line := range strings.SplitSeq(s, "\n") {
		if n == height {
			break
		}
		w := ansi.StringWidth(line)
		if w > width {
			return "", false
		}
		if n > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		writeBlanks(&b, width-w)
		n++
	}
	for ; n < height; n++ {
		if n > 0 {
			b.WriteByte('\n')
		}
		writeBlanks(&b, width)
	}
	return b.String(), true
}

func writeBlanks(b *strings.Builder, n int) {
	for n > len(blanks) {
		b.WriteString(blanks)
		n -= len(blanks)
	}
	b.WriteString(blanks[:n])
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
	frame := lipgloss.RoundedBorder()
	edge := lipgloss.NewStyle().Foreground(paneBorderColor(focused))
	side := edge.Render(frame.Left)
	row := func(line string) string { return side + " " + line + " " + side }
	span := strings.Repeat(frame.Top, width-2)
	lines := make([]string, 0, height)
	lines = append(lines,
		edge.Render(frame.TopLeft+span+frame.TopRight),
		row(Line(title, innerWidth)),
		row(lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(lipgloss.NormalBorder().Bottom, innerWidth))),
	)
	if innerHeight > 0 {
		for line := range strings.SplitSeq(Block(body, innerWidth, innerHeight), "\n") {
			lines = append(lines, row(line))
		}
	}
	lines = append(lines, edge.Render(frame.BottomLeft+span+frame.BottomRight))
	return strings.Join(lines, "\n")
}

func PaneContentSize(width, height int) (int, int) {
	return max(width-styles.BoxPaddedSize, 0), max(height-styles.PaneChromeHeight, 0)
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

func Muted(s string) string {
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(s)
}

func Loading(spinner, label string) string {
	if spinner == "" {
		return Muted(label)
	}
	return spinner + " " + Muted(label)
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

func FitItems(items []string, sep string, width int) string {
	out := ""
	for i, it := range items {
		candidate := it
		if i > 0 {
			candidate = out + sep + it
		}
		if lipgloss.Width(candidate) > width {
			break
		}
		out = candidate
	}
	return out
}

func HelpItems(h help.Model, bindings []key.Binding) []string {
	out := make([]string, 0, len(bindings))
	for _, b := range bindings {
		if v := h.ShortHelpView([]key.Binding{b}); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func ShortKey(b key.Binding) string {
	ks := b.Keys()
	if len(ks) == 0 {
		return ""
	}
	return ks[len(ks)-1]
}

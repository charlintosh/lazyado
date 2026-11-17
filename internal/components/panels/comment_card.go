package panels

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	commentGutter = 2
	replyIndent   = 2
	likeGlyph     = "♥"
	editedNote    = "edited"
	minBoxedWidth = 36
	boxChrome     = 4
)

var (
	topCorners    = [2]string{"╭", "╮"}
	bottomCorners = [2]string{"╰", "╯"}
)

type CommentCard struct {
	Author    string
	Mine      bool
	Date      time.Time
	Edited    bool
	Likes     int
	LikedByMe bool
	Body      string
	Reply     bool
	Focused   bool
	Hints     string
}

func (c CommentCard) Lines(width int) []string {
	if width < minBoxedWidth {
		return c.railLines(width)
	}
	return c.boxLines(width)
}

func (c CommentCard) header() string {
	avatar := lipgloss.NewStyle().Foreground(styles.AvatarColor(c.Author)).Bold(true).Render(widgets.Initials(c.Author))
	name := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	if c.Focused {
		name = name.Foreground(styles.ColorPrimary)
	}
	header := avatar + " " + name.Render(c.Author)
	if c.Mine {
		header += Muted(" (" + youNote + ")")
	}
	header += Muted(" · " + formatCommentDate(c.Date))
	if c.Edited {
		header += Muted(" · " + editedNote)
	}
	return header
}

func (c CommentCard) likesLabel() string {
	if c.Likes == 0 {
		return ""
	}
	label := fmt.Sprintf("%s %d", likeGlyph, c.Likes)
	if !c.LikedByMe {
		return Muted(label)
	}
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(label + " · " + youNote)
}

func (c CommentCard) hints() string {
	if !c.Focused {
		return ""
	}
	return c.Hints
}

func (c CommentCard) pad() string {
	if !c.Reply {
		return ""
	}
	return strings.Repeat(" ", replyIndent)
}

func (c CommentCard) body(width int) []string {
	body := components.RenderMarkdown(c.Body, max(width, commentMinBodyLen))
	return strings.Split(strings.Trim(body, "\n"), "\n")
}

func (c CommentCard) boxLines(width int) []string {
	pad := c.pad()
	width -= len(pad)
	lines := []string{pad + boxEdge(topCorners, c.header(), c.likesLabel(), width, c.Focused)}
	for _, l := range c.body(width - boxChrome) {
		lines = append(lines, pad+boxRow(l, width, c.Focused))
	}
	return append(lines, pad+boxEdge(bottomCorners, c.hints(), "", width, c.Focused))
}

func (c CommentCard) railLines(width int) []string {
	mark, pad := commentGutterMark(c.Focused), c.pad()
	room := max(width-commentGutter-len(pad), 1)
	lines := []string{mark + pad + justify(c.header(), c.likesLabel(), room)}
	indent := pad + strings.Repeat(" ", styles.CommentIndent)
	for _, l := range c.body(room - styles.CommentIndent) {
		lines = append(lines, mark+indent+l)
	}
	if hints := c.hints(); hints != "" {
		lines = append(lines, mark+indent+hints)
	}
	return lines
}

func commentSeparator() string {
	return strings.TrimRight(commentGutterMark(false), " ")
}

func justify(left, right string, width int) string {
	if right == "" {
		return ansi.Truncate(left, max(width, 1), "…")
	}
	left = ansi.Truncate(left, max(width-lipgloss.Width(right)-1, 1), "…")
	return left + strings.Repeat(" ", max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
}

func commentGutterMark(focused bool) string {
	if focused {
		return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(railBar) + " "
	}
	return lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(rangeBar) + " "
}

func boxBar(focused bool) lipgloss.Style {
	if focused {
		return lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	}
	return lipgloss.NewStyle().Foreground(styles.ColorBorder)
}

func boxEdge(corners [2]string, left, right string, width int, focused bool) string {
	bar := boxBar(focused)
	room := max(width-boxChrome, 0)
	if right != "" {
		right = " " + right + " "
	}
	if lipgloss.Width(right) > room {
		right = ""
	}
	if left != "" {
		left = " " + ansi.Truncate(left, max(room-lipgloss.Width(right)-2, 1), "…") + " "
	}
	fill := max(room-lipgloss.Width(left)-lipgloss.Width(right), 0)
	return bar.Render(corners[0]+"─") + left + bar.Render(strings.Repeat("─", fill)) + right + bar.Render("─"+corners[1])
}

func boxRow(line string, width int, focused bool) string {
	side := boxBar(focused).Render(rangeBar)
	inner := max(width-boxChrome, 1)
	line = ansi.Truncate(line, inner, "…")
	return side + " " + line + strings.Repeat(" ", max(inner-lipgloss.Width(line), 0)) + " " + side
}

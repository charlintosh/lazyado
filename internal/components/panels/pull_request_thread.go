package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type threadStyle struct {
	width    int
	selected bool
	anchored bool
	comment  int
	names    map[string]string
	me       models.Identity
	hints    string
	expanded map[int]bool
}

type threadRender struct {
	text   string
	top    int
	bottom int
}

func (r threadRender) focused() bool {
	return r.top >= 0
}

func threadStatusBadge(t models.PullRequestThread) string {
	if t.Active() {
		return Badge(strings.ToUpper(t.Status), styles.ColorOrange, styles.ColorOnAccent)
	}
	label := strings.ToUpper(t.Status)
	if t.Status == models.ThreadFixed {
		label = "RESOLVED"
	}
	return Badge(label, styles.ColorChip, styles.ColorLight)
}

func threadAnchor(t models.PullRequestThread) string {
	if t.FilePath == "" {
		return Muted(generalComment)
	}
	anchor := t.FilePath
	if t.Line > 0 {
		anchor = t.FilePath + ":" + models.LineRange(t.Line, t.EndLine)
	}
	return lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(anchor)
}

func threadLabel(t models.PullRequestThread, anchored bool) string {
	switch {
	case anchored:
		return threadAnchor(t)
	case t.MultiLine():
		return lipgloss.NewStyle().Foreground(styles.ColorAccent).Render("Lines " + models.LineRange(t.Line, t.EndLine))
	case t.Line > 0:
		return lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(fmt.Sprintf("Line %d", t.Line))
	}
	return Muted(generalComment)
}

func resolveMentions(content string, names map[string]string) string {
	return mentionPattern.ReplaceAllStringFunc(content, func(match string) string {
		if name, ok := names[strings.ToLower(match[2:len(match)-1])]; ok {
			return "**@" + name + "**"
		}
		return "@user"
	})
}

func pullRequestCard(c models.PullRequestComment, st threadStyle, focused bool) CommentCard {
	card := CommentCard{
		Author:    c.AuthorName,
		Mine:      c.WrittenBy(st.me),
		Date:      c.Date,
		Likes:     len(c.Likes),
		LikedByMe: c.LikedBy(st.me),
		Body:      resolveMentions(c.Content, st.names),
		Reply:     c.ParentID != 0,
		Focused:   focused,
	}
	if focused {
		card.Hints = st.hints
	}
	return card
}

func threadSummary(t models.PullRequestThread, anchored bool) string {
	last := t.Comments[len(t.Comments)-1]
	return threadStatusBadge(t) + " " + threadLabel(t, anchored) +
		Muted(fmt.Sprintf(" · %s · last by %s", countLabel(len(t.Comments), "comment"), last.AuthorName))
}

func boxedCollapsedThread(t models.PullRequestThread, st threadStyle, width int) threadRender {
	hints := ""
	if st.selected {
		hints = st.hints
	}
	lines := []string{
		boxEdge(topCorners, "", "", width, st.selected),
		boxRow(threadSummary(t, st.anchored), width, st.selected),
		boxEdge(bottomCorners, hints, "", width, st.selected),
	}
	r := threadRender{text: strings.Join(lines, "\n"), top: -1, bottom: -1}
	if st.selected {
		r.top, r.bottom = 0, len(lines)-1
	}
	return r
}

func collapsedThread(t models.PullRequestThread, st threadStyle, width int) threadRender {
	summary := threadSummary(t, st.anchored)
	lines := []string{commentGutterMark(st.selected) + ansi.Truncate(summary, max(width-commentGutter, 1), "…")}
	if !st.selected {
		return threadRender{text: lines[0], top: -1, bottom: -1}
	}
	if st.hints != "" {
		lines = append(lines, commentGutterMark(true)+strings.Repeat(" ", styles.CommentIndent)+st.hints)
	}
	return threadRender{text: strings.Join(lines, "\n"), top: 0, bottom: len(lines) - 1}
}

func renderThread(t models.PullRequestThread, st threadStyle) threadRender {
	width := max(st.width, styles.BoxPaddedSize)
	if t.System() {
		text := strings.Join(strings.Fields(resolveMentions(t.Comments[0].Content, st.names)), " ") + " · " + formatCommentDate(t.Date)
		return threadRender{text: "  " + Muted(ansi.Truncate(text, max(width-2, 1), "…")), top: -1, bottom: -1}
	}
	boxed := width >= minBoxedWidth
	switch {
	case t.Collapsed(st.expanded) && boxed:
		return boxedCollapsedThread(t, st, width)
	case t.Collapsed(st.expanded):
		return collapsedThread(t, st, width)
	}
	status := threadStatusBadge(t) + " " + Muted(countLabel(len(t.Comments), "comment"))
	lines := []string{justify(threadLabel(t, st.anchored), status, width)}
	r := threadRender{top: -1, bottom: -1}
	for i, c := range t.Comments {
		if i > 0 && !boxed {
			lines = append(lines, commentSeparator())
		}
		focused := st.selected && c.ID == st.comment
		start := len(lines)
		lines = append(lines, pullRequestCard(c, st, focused).Lines(width)...)
		if focused {
			r.top, r.bottom = start, len(lines)-1
		}
	}
	r.text = strings.Join(lines, "\n")
	return r
}

func threadStops(t models.PullRequestThread, expanded map[int]bool) []commentFocus {
	if t.System() || len(t.Comments) == 0 {
		return nil
	}
	if t.Collapsed(expanded) {
		return []commentFocus{{thread: t.ID, comment: t.Comments[0].ID}}
	}
	stops := make([]commentFocus, 0, len(t.Comments))
	for _, c := range t.Comments {
		stops = append(stops, commentFocus{thread: t.ID, comment: c.ID})
	}
	return stops
}

type commentFocus struct {
	thread  int
	comment int
}

func (f commentFocus) none() bool {
	return f.thread == 0
}

func commentActions(k keys.KeyMap, me models.Identity, t models.PullRequestThread, c models.PullRequestComment, expanded map[int]bool) []key.Binding {
	collapsed := t.Collapsed(expanded)
	var actions []key.Binding
	if collapsed {
		actions = append(actions, widgets.Relabel(k.Select, "expand"))
	}
	actions = append(actions, k.Reply)
	switch {
	case collapsed:
	case c.WrittenBy(me):
		actions = append(actions, widgets.Relabel(k.EditComment, "edit"), widgets.Relabel(k.DeleteComment, "delete"))
	case me.Known():
		label := "like"
		if c.LikedBy(me) {
			label = "unlike"
		}
		actions = append(actions, widgets.Relabel(k.LikeComment, label))
	}
	if t.Resolvable() {
		label := "resolve"
		if !t.Active() {
			label = "reopen"
		}
		actions = append(actions, widgets.Relabel(k.ResolveThread, label))
	}
	return actions
}

func commentHints(k keys.KeyMap, me models.Identity, t models.PullRequestThread, c models.PullRequestComment, expanded map[int]bool, width int) string {
	return widgets.Hints(max(width-commentGutter-styles.CommentIndent-replyIndent, 1), commentActions(k, me, t, c, expanded)...)
}

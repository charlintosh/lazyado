package panels

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	commentBoxInset   = 2
	commentBodyInset  = 6
	commentMinBodyLen = 20
)

const youNote = "you"

type CommentListConfig struct {
	Width    int
	Selected int
	Me       models.Identity
}

func RenderCommentList(comments []models.Comment, s *styles.Styles, cfg CommentListConfig) (string, []int) {
	if len(comments) == 0 {
		return "", nil
	}
	offsets := make([]int, len(comments))
	boxes := make([]string, len(comments))
	line := 0
	for i, c := range comments {
		box := s.CommentBox
		if i == cfg.Selected {
			box = s.CommentBoxSelected
		}
		boxes[i] = box.Width(cfg.Width - commentBoxInset).Render(renderComment(c, s, cfg.Width, cfg.Me.Owns(c)))
		offsets[i] = line
		line += lipgloss.Height(boxes[i])
	}
	return strings.Join(boxes, "\n"), offsets
}

func renderComment(c models.Comment, s *styles.Styles, width int, mine bool) string {
	author := s.CommentAuthor.Render(c.CreatedBy)
	if mine {
		author += s.TextMuted.Render(" " + youNote)
	}
	header := s.CommentHeader.Render(lipgloss.JoinHorizontal(lipgloss.Left,
		author,
		s.TextMuted.Render("  "),
		s.CommentDate.Render(formatCommentDate(c.CreatedDate)),
	))
	body := components.RenderMarkdown(components.CommentMarkdown(c), max(width-commentBodyInset, commentMinBodyLen))
	parts := []string{header, body}
	if !c.ModifiedDate.IsZero() && !c.ModifiedDate.Equal(c.CreatedDate) {
		parts = append(parts, s.CommentEdited.Render(fmt.Sprintf("Edited by %s on %s", c.ModifiedBy, formatCommentDate(c.ModifiedDate))))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func formatCommentDate(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return plural(int(diff.Minutes()), "minute")
	case diff < 24*time.Hour:
		return plural(int(diff.Hours()), "hour")
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2, 2006")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

func RenderDiscussion(comments []models.Comment, width, selected int, me models.Identity) (string, []int) {
	if len(comments) == 0 {
		return "", nil
	}
	offsets := make([]int, len(comments))
	blocks := make([]string, len(comments))
	line := 0
	for i, c := range comments {
		bar := lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).BorderLeft(true).PaddingLeft(1)
		if i == selected {
			bar = bar.BorderStyle(lipgloss.ThickBorder()).BorderForeground(styles.ColorPrimary)
		}
		blocks[i] = bar.Render(discussionComment(c, max(width-styles.CardChromeWidth, 1), i == selected, me.Owns(c)))
		offsets[i] = line
		line += lipgloss.Height(blocks[i]) + 1
	}
	return strings.Join(blocks, "\n\n"), offsets
}

func discussionComment(c models.Comment, width int, selected, mine bool) string {
	name := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	if selected {
		name = name.Foreground(styles.ColorPrimary)
	}
	avatar := lipgloss.NewStyle().Foreground(styles.AvatarColor(c.CreatedBy)).Bold(true).Render(widgets.Initials(c.CreatedBy))
	header := avatar + " " + name.Render(c.CreatedBy)
	if mine {
		header += Muted(" " + youNote)
	}
	header += Muted(" · " + formatCommentDate(c.CreatedDate))
	indent := lipgloss.NewStyle().PaddingLeft(styles.CommentIndent)
	body := components.RenderMarkdown(components.CommentMarkdown(c), max(width-styles.CommentIndent, commentMinBodyLen))
	parts := []string{Line(header, width), indent.Render(body)}
	if !c.ModifiedDate.IsZero() && !c.ModifiedDate.Equal(c.CreatedDate) {
		parts = append(parts, indent.Render(Muted(fmt.Sprintf("Edited by %s on %s", c.ModifiedBy, formatCommentDate(c.ModifiedDate)))))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

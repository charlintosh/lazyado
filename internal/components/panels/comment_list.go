package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/models"
)

const commentMinBodyLen = 20

const youNote = "you"

type CommentListConfig struct {
	Width    int
	Selected int
	Me       models.Identity
	Hints    string
}

func workItemCard(c models.Comment, me models.Identity, focused bool, hints string) CommentCard {
	card := CommentCard{
		Author:    c.CreatedBy,
		Mine:      me.Owns(c),
		Date:      c.CreatedDate,
		Edited:    c.Edited(),
		Likes:     c.Likes,
		LikedByMe: c.LikedByMe,
		Body:      components.CommentMarkdown(c),
		Focused:   focused,
	}
	if focused {
		card.Hints = hints
	}
	return card
}

func RenderComments(comments []models.Comment, cfg CommentListConfig) (string, []int) {
	if len(comments) == 0 {
		return "", nil
	}
	offsets := make([]int, len(comments))
	var lines []string
	for i, c := range comments {
		if i > 0 && cfg.Width < minBoxedWidth {
			lines = append(lines, commentSeparator())
		}
		offsets[i] = len(lines)
		lines = append(lines, workItemCard(c, cfg.Me, i == cfg.Selected, cfg.Hints).Lines(cfg.Width)...)
	}
	return strings.Join(lines, "\n"), offsets
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

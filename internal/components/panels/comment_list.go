package panels

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/lipgloss"
)

type CommentListConfig struct {
	Width      int
	Selected   int
	ShowEdited bool
	ShowBoxes  bool
}

func RenderCommentList(comments []models.Comment, s styles.Styles, cfg CommentListConfig) (content string, offsets []int) {
	if len(comments) == 0 {
		return "", nil
	}

	var b strings.Builder
	offsets = make([]int, len(comments))
	currentLine := 0

	for i, comment := range comments {
		offsets[i] = currentLine

		var commentContent strings.Builder

		author := s.CommentAuthor.Render(comment.CreatedBy)
		dateStr := formatCommentDate(comment.CreatedDate)
		date := s.CommentDate.Render(dateStr)
		separator := s.TextMuted.Render(" • ")

		header := lipgloss.JoinHorizontal(lipgloss.Left, author, separator, date)
		commentContent.WriteString(s.CommentHeader.Render(header))
		commentContent.WriteString("\n")

		boxWidth := cfg.Width - 6
		if boxWidth < 20 {
			boxWidth = 20
		}

		sourceText := comment.Text
		if comment.RenderedText != "" {
			sourceText = comment.RenderedText
		}
		text := parseAndStyleHTML(sourceText, boxWidth, s)
		commentContent.WriteString(text)

		if cfg.ShowEdited && !comment.ModifiedDate.IsZero() && !comment.ModifiedDate.Equal(comment.CreatedDate) {
			commentContent.WriteString("\n")
			modStr := fmt.Sprintf("✏ Edited by %s on %s", comment.ModifiedBy, formatCommentDate(comment.ModifiedDate))
			commentContent.WriteString(s.CommentEdited.Render(modStr))
		}

		if cfg.ShowBoxes {
			boxStyle := s.CommentBox
			if i == cfg.Selected {
				boxStyle = s.CommentBoxSelected
			}
			boxStyle = boxStyle.Width(cfg.Width - 2)

			rendered := boxStyle.Render(commentContent.String())
			b.WriteString(rendered)
			b.WriteString("\n")
			currentLine += strings.Count(rendered, "\n") + 1
		} else {
			rendered := commentContent.String()
			b.WriteString(rendered)
			b.WriteString("\n\n")
			currentLine += strings.Count(rendered, "\n") + 2
		}
	}

	return b.String(), offsets
}

func parseAndStyleHTML(htmlText string, width int, s styles.Styles) string {
	if width <= 0 || htmlText == "" {
		return htmlText
	}

	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		return s.CommentText.Render(htmlText)
	}

	var segments []string
	extractStyledSegments(doc, &segments, s)

	return strings.Join(segments, "")
}

func extractStyledSegments(n *html.Node, segments *[]string, s styles.Styles) {
	if n.Type == html.TextNode {
		text := n.Data
		if text != "" {
			*segments = append(*segments, s.CommentText.Render(text))
		}
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "a":
			isMention := false
			for _, attr := range n.Attr {
				if attr.Key == "data-vss-mention" {
					isMention = true
					break
				}
			}

			linkText := getNodeText(n)
			if linkText != "" {
				if isMention {
					*segments = append(*segments, s.CommentMention.Render(linkText))
				} else {
					*segments = append(*segments, s.CommentLink.Render(linkText))
				}
			}
			return

		case "br":
			*segments = append(*segments, "\n")
			return

		case "b", "strong":
			boldText := getNodeText(n)
			if boldText != "" {
				boldStyle := s.CommentText.Copy().Bold(true)
				*segments = append(*segments, boldStyle.Render(boldText))
			}
			return

		case "li":
			*segments = append(*segments, "\n  • ")

		case "ul", "ol":
			*segments = append(*segments, "\n")

		case "div", "p":
			if len(*segments) > 0 {
				*segments = append(*segments, "\n")
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		extractStyledSegments(child, segments, s)
	}
}

func getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(getNodeText(child))
	}
	return text.String()
}

func formatCommentDate(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

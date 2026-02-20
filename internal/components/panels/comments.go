package panels

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var commentsLogger = debug.Scope("comments")

// CommentsPanel shows comments for a selected work item
type CommentsPanel struct {
	comments       []models.Comment
	viewport       viewport.Model
	styles         styles.Styles
	keys           keys.KeyMap
	width          int
	height         int
	ready          bool
	focused        bool
	cursor         int
	commentOffsets []int
}

// NewCommentsPanel creates a new comments panel
func NewCommentsPanel(styles styles.Styles, keys keys.KeyMap) CommentsPanel {
	vp := viewport.New(0, 0)
	return CommentsPanel{
		viewport: vp,
		styles:   styles,
		keys:     keys,
	}
}

// Init initializes the comments panel
func (c CommentsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c CommentsPanel) Update(msg tea.Msg) (CommentsPanel, tea.Cmd) {
	var cmd tea.Cmd

	if c.focused && len(c.comments) > 0 {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			oldCursor := c.cursor
			switch {
			case key.Matches(msg, c.keys.Up):
				if c.cursor > 0 {
					c.cursor--
				}
			case key.Matches(msg, c.keys.Down):
				if c.cursor < len(c.comments)-1 {
					c.cursor++
				}
			case key.Matches(msg, c.keys.Top):
				c.cursor = 0
			case key.Matches(msg, c.keys.Bottom):
				c.cursor = len(c.comments) - 1
			case key.Matches(msg, c.keys.EditComment):
				if c.cursor < len(c.comments) {
					return c, func() tea.Msg {
						return EditCommentMsg{Comment: c.comments[c.cursor]}
					}
				}
			case key.Matches(msg, c.keys.DeleteComment):
				if c.cursor < len(c.comments) {
					return c, func() tea.Msg {
						return DeleteCommentMsg{Comment: c.comments[c.cursor]}
					}
				}
			}
			if c.cursor != oldCursor {
				c.updateViewportContent()
				c.scrollToSelected()
			}
			return c, nil
		}
	}

	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

// View renders the comments panel
func (c CommentsPanel) View() string {
	panelStyle := c.styles.PanelInactive
	if c.focused {
		panelStyle = c.styles.PanelActive
	}

	var b strings.Builder

	numberStyle := c.styles.TextMuted
	if c.focused {
		numberStyle = c.styles.AccentBold
	}
	titleStyle := c.styles.FilterGroupTitle
	if c.focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}

	if len(c.comments) > 0 {
		titleText := fmt.Sprintf("Discussion (%d)", len(c.comments))
		b.WriteString("  " + numberStyle.Render("7. ") + titleStyle.Render(titleText))
	} else {
		b.WriteString("  " + numberStyle.Render("7. ") + titleStyle.Render("Discussion"))
	}
	b.WriteString("\n\n")

	if len(c.comments) == 0 {
		b.WriteString(c.styles.Subtitle.Render("  No comments"))
		return panelStyle.
			Width(c.width).
			Height(c.height).
			Render(b.String())
	}

	b.WriteString(c.viewport.View())
	canScroll := c.viewport.TotalLineCount() > c.viewport.VisibleLineCount()
	if canScroll {
		pct := int(c.viewport.ScrollPercent() * 100)
		scrollHint := lipgloss.NewStyle().Foreground(styles.ColorMuted).
			Render(fmt.Sprintf(" %d%% ", pct))
		b.WriteString("\n" + scrollHint)
	}

	return panelStyle.
		Width(c.width).
		Height(c.height).
		Render(b.String())
}

// SetSize updates the panel dimensions
func (c *CommentsPanel) SetSize(width, height int) {
	c.width = width
	c.height = height

	c.viewport.Width = width - styles.PanelBorderOffset
	c.viewport.Height = height - 3
	if c.viewport.Height < 1 {
		c.viewport.Height = 1
	}

	if c.ready {
		c.updateViewportContent()
	}
}

// SetComments updates the comments list
func (c *CommentsPanel) SetComments(comments []models.Comment) {
	commentsLogger.Debugf("set_comments_count=%d", len(comments))
	commentsLogger.Debugf("viewport_dimensions width=%d height=%d", c.viewport.Width, c.viewport.Height)
	c.comments = comments
	c.ready = true
	c.cursor = 0
	c.updateViewportContent()
	commentsLogger.Debugf("viewport_content_length=%d", len(c.viewport.View()))
}

// ClearComments clears the comments list
func (c *CommentsPanel) ClearComments() {
	c.comments = nil
	c.ready = false
	c.cursor = 0
	c.viewport.SetContent("")
}

// updateViewportContent renders the comments into the viewport
func (c *CommentsPanel) updateViewportContent() {
	commentsLogger.Debugf("update_viewport comments=%d width=%d height=%d",
		len(c.comments), c.viewport.Width, c.viewport.Height)

	if len(c.comments) == 0 {
		commentsLogger.Debugf("no_comments_to_display")
		c.viewport.SetContent("")
		return
	}

	// Make sure viewport has dimensions set
	if c.viewport.Width <= 0 || c.viewport.Height <= 0 {
		commentsLogger.Debugf("viewport_dimensions_not_set")
		return
	}

	var b strings.Builder
	c.commentOffsets = make([]int, len(c.comments))
	currentLine := 0

	for i, comment := range c.comments {
		c.commentOffsets[i] = currentLine
		isSelected := c.focused && i == c.cursor

		// Build comment content
		var commentContent strings.Builder

		// Header: Author and date with visual hierarchy
		author := c.styles.CommentAuthor.Render(comment.CreatedBy)
		dateStr := formatCommentDate(comment.CreatedDate)
		date := c.styles.CommentDate.Render(dateStr)
		separator := c.styles.TextMuted.Render(" • ")

		header := lipgloss.JoinHorizontal(lipgloss.Left, author, separator, date)
		commentContent.WriteString(c.styles.CommentHeader.Render(header))
		commentContent.WriteString("\n")

		// Comment text with proper wrapping and styling for mentions/links
		boxWidth := c.viewport.Width - 6 // Account for box borders and padding
		if boxWidth < 20 {
			boxWidth = 20
		}
		// Use renderedText if available (HTML format with mentions), otherwise plain text
		sourceText := comment.Text
		if comment.RenderedText != "" {
			sourceText = comment.RenderedText
		}
		text := c.parseAndStyleHTML(sourceText, boxWidth)
		commentContent.WriteString(text)

		// Modified indicator if applicable
		if !comment.ModifiedDate.IsZero() && !comment.ModifiedDate.Equal(comment.CreatedDate) {
			commentContent.WriteString("\n")
			modStr := fmt.Sprintf("✏ Edited by %s on %s", comment.ModifiedBy, formatCommentDate(comment.ModifiedDate))
			commentContent.WriteString(c.styles.CommentEdited.Render(modStr))
		}

		// Wrap comment in styled box that takes full width
		boxStyle := c.styles.CommentBox
		if isSelected {
			boxStyle = c.styles.CommentBoxSelected
		}
		// Account for borders (2 chars) when setting width
		boxStyle = boxStyle.Width(c.viewport.Width - 2)

		rendered := boxStyle.Render(commentContent.String())
		b.WriteString(rendered)
		b.WriteString("\n")
		currentLine += strings.Count(rendered, "\n") + 1
	}

	c.viewport.SetContent(b.String())
}

func (c *CommentsPanel) scrollToSelected() {
	if c.cursor < 0 || c.cursor >= len(c.commentOffsets) {
		return
	}
	topLine := c.commentOffsets[c.cursor]

	bottomLine := c.viewport.TotalLineCount() - 1
	if c.cursor+1 < len(c.commentOffsets) {
		bottomLine = c.commentOffsets[c.cursor+1] - 1
	}

	vpTop := c.viewport.YOffset
	vpBottom := vpTop + c.viewport.Height - 1

	if topLine < vpTop {
		c.viewport.SetYOffset(topLine)
	} else if bottomLine > vpBottom {
		c.viewport.SetYOffset(bottomLine - c.viewport.Height + 1)
	}
}

// wrapText wraps text to fit within the specified width
func (c *CommentsPanel) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		if lineIdx > 0 {
			result.WriteString("\n")
		}

		// If line is empty, keep it
		if len(line) == 0 {
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			}
		}
		result.WriteString(currentLine)
	}

	return result.String()
}

// parseAndStyleHTML parses HTML comment text and applies styles for mentions and links
func (c *CommentsPanel) parseAndStyleHTML(htmlText string, width int) string {
	if width <= 0 || htmlText == "" {
		return htmlText
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(htmlText))
	if err != nil {
		// Fallback to plain text if parsing fails
		return c.styles.CommentText.Render(htmlText)
	}

	// Extract styled segments from HTML
	var segments []string
	c.extractStyledSegments(doc, &segments)

	// Join segments and wrap
	result := strings.Join(segments, "")
	return result
}

// extractStyledSegments recursively extracts text from HTML nodes and applies styling
func (c *CommentsPanel) extractStyledSegments(n *html.Node, segments *[]string) {
	if n.Type == html.TextNode {
		text := n.Data
		// Only skip completely empty text nodes
		if text != "" && strings.TrimSpace(text) != "" {
			// Preserve original spacing - don't collapse spaces here
			*segments = append(*segments, c.styles.CommentText.Render(text))
		}
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "a":
			// Check if it's a mention or regular link
			isMention := false
			for _, attr := range n.Attr {
				if attr.Key == "data-vss-mention" {
					isMention = true
					break
				}
			}

			// Extract link text (preserve the @ symbol!)
			linkText := c.getNodeText(n)
			if linkText != "" {
				if isMention {
					// Style as mention (cyan/bold)
					*segments = append(*segments, c.styles.CommentMention.Render(linkText))
				} else {
					// Style as link (blue/underline)
					*segments = append(*segments, c.styles.CommentLink.Render(linkText))
				}
			}
			return

		case "br":
			*segments = append(*segments, "\n")
			return

		case "b", "strong":
			// Bold text
			boldText := c.getNodeText(n)
			if boldText != "" {
				boldStyle := c.styles.CommentText.Copy().Bold(true)
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

	// Process children
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.extractStyledSegments(child, segments)
	}
}

// getNodeText extracts all text content from a node and its children
// Preserves all characters including @ symbols and spaces
func (c *CommentsPanel) getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		// Preserve all text content exactly as-is
		return n.Data
	}

	var text strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(c.getNodeText(child))
	}
	return text.String()
}

// formatCommentDate formats a comment date for display
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

// Message types

// CommentsLoadedMsg is sent when comments are loaded
type CommentsLoadedMsg struct {
	Comments []models.Comment
}

// EditCommentMsg is sent when user wants to edit a comment
type EditCommentMsg struct {
	Comment models.Comment
}

// DeleteCommentMsg is sent when user wants to delete a comment
type DeleteCommentMsg struct {
	Comment models.Comment
}

// SetFocused updates the focused state
func (c *CommentsPanel) SetFocused(focused bool) {
	c.focused = focused
	if c.ready && len(c.comments) > 0 {
		c.updateViewportContent()
	}
}

// SelectedComment returns the currently selected comment
func (c *CommentsPanel) SelectedComment() *models.Comment {
	if !c.focused || c.cursor >= len(c.comments) {
		return nil
	}
	return &c.comments[c.cursor]
}

// GetAvailableActions returns a list of available actions for the comments panel
func (c *CommentsPanel) GetAvailableActions() []string {
	if len(c.comments) == 0 {
		return []string{}
	}

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.ColorPrimary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.ColorTextMuted)

	// Use KeyMap help strings to keep labels centralized and consistent
	addHelp := c.keys.AddComment.Help()
	editHelp := c.keys.EditComment.Help()
	deleteHelp := c.keys.DeleteComment.Help()

	actions := []string{
		keyStyle.Render(addHelp.Key) + descStyle.Render(" - "+addHelp.Desc),
		keyStyle.Render(editHelp.Key) + descStyle.Render(" - "+editHelp.Desc),
		keyStyle.Render(deleteHelp.Key) + descStyle.Render(" - "+deleteHelp.Desc),
	}

	return actions
}

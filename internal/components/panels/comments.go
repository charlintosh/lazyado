package panels

import (
	"fmt"
	"strings"
	"time"

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
	comments []models.Comment
	viewport viewport.Model
	styles   styles.Styles
	keys     keys.KeyMap
	width    int
	height   int
	ready    bool
	focused  bool
	cursor   int
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
			return c, nil
		}
	}

	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

// View renders the comments panel
func (c CommentsPanel) View() string {
	if !c.ready {
		content := c.styles.Subtitle.Render("Loading comments...")
		panelStyle := c.styles.PanelInactive
		if c.focused {
			panelStyle = c.styles.PanelActive
		}
		return panelStyle.
			Width(c.width).
			Height(c.height).
			Render(content)
	}

	if len(c.comments) == 0 {
		content := c.styles.Subtitle.Render("No comments")
		panelStyle := c.styles.PanelInactive
		if c.focused {
			panelStyle = c.styles.PanelActive
		}
		return panelStyle.
			Width(c.width).
			Height(c.height).
			Render(content)
	}

	panelStyle := c.styles.PanelInactive
	if c.focused {
		panelStyle = c.styles.PanelActive
	}
	return panelStyle.
		Width(c.width).
		Height(c.height).
		Render(c.viewport.View())
}

// SetSize updates the panel dimensions
func (c *CommentsPanel) SetSize(width, height int) {
	c.width = width
	c.height = height

	// Account for panel borders and padding
	contentWidth := width - 2
	contentHeight := height - 2

	c.viewport.Width = contentWidth
	c.viewport.Height = contentHeight

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

	// Title with number (consistent with filter panel style)
	numberStyle := c.styles.TextMuted
	if c.focused {
		numberStyle = c.styles.AccentBold
	}
	titleStyle := c.styles.FilterGroupTitle
	if c.focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	titlePrefix := numberStyle.Render("6. ")
	titleText := fmt.Sprintf("Comments (%d)", len(c.comments))
	b.WriteString(titlePrefix + titleStyle.Render(titleText))
	b.WriteString("\n\n")

	// Render each comment
	for i, comment := range c.comments {
		isSelected := c.focused && i == c.cursor

		// Header: Author and date
		author := c.styles.DetailLabel.Render(comment.CreatedBy)
		dateStr := formatCommentDate(comment.CreatedDate)
		date := c.styles.Subtitle.Render(dateStr)

		header := lipgloss.JoinHorizontal(lipgloss.Left, author, " • ", date)

		if isSelected {
			header = c.styles.ListItemSelected.Render(header)
		}

		b.WriteString(header)
		b.WriteString("\n")

		// Comment text
		text := c.wrapText(comment.Text, c.viewport.Width-2)
		commentStyle := lipgloss.NewStyle().
			Foreground(styles.ColorOffWhite).
			PaddingLeft(1)

		if isSelected {
			commentStyle = commentStyle.Background(styles.ColorHighlight)
		}

		b.WriteString(commentStyle.Render(text))
		b.WriteString("\n")

		// Modified indicator if applicable
		if !comment.ModifiedDate.IsZero() && !comment.ModifiedDate.Equal(comment.CreatedDate) {
			modStr := fmt.Sprintf("Edited by %s on %s", comment.ModifiedBy, formatCommentDate(comment.ModifiedDate))
			modStyle := c.styles.Subtitle.Italic(true)
			b.WriteString(modStyle.Render(modStr))
			b.WriteString("\n")
		}

		// Separator between comments
		if i < len(c.comments)-1 {
			separator := strings.Repeat("─", c.viewport.Width)
			sepStyle := lipgloss.NewStyle().Foreground(styles.ColorBorder)
			b.WriteString(sepStyle.Render(separator))
			b.WriteString("\n\n")
		}
	}

	c.viewport.SetContent(b.String())
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

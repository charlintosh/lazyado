package panels

import (
	"fmt"
	"strings"

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

	if c.viewport.Width <= 0 || c.viewport.Height <= 0 {
		commentsLogger.Debugf("viewport_dimensions_not_set")
		return
	}

	selected := -1
	if c.focused {
		selected = c.cursor
	}

	content, offsets := RenderCommentList(c.comments, c.styles, CommentListConfig{
		Width:      c.viewport.Width,
		Selected:   selected,
		ShowEdited: true,
		ShowBoxes:  true,
	})
	c.commentOffsets = offsets
	c.viewport.SetContent(content)
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

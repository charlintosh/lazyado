package panels

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
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
	vp := viewport.New()
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
		case tea.KeyPressMsg:
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

func (c CommentsPanel) View() string {
	if len(c.comments) == 0 {
		hint := c.styles.TextMuted.Render("No comments yet. Press " + c.keys.AddComment.Help().Key + " to start the discussion.")
		return FitBlock(hint, c.width, c.height)
	}
	body := c.viewport.View()
	if c.viewport.TotalLineCount() > c.viewport.VisibleLineCount() {
		pct := int(c.viewport.ScrollPercent() * 100)
		body += "\n" + lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("%d%%", pct))
	}
	return FitBlock(body, c.width, c.height)
}

func (c CommentsPanel) Count() int {
	return len(c.comments)
}

func (c *CommentsPanel) SetSize(width, height int) {
	if c.width == width && c.height == height {
		return
	}
	c.width = width
	c.height = height
	c.viewport.SetWidth(max(width, 1))
	c.viewport.SetHeight(max(height-1, 1))
	if c.ready {
		c.updateViewportContent()
	}
}

// SetComments updates the comments list
func (c *CommentsPanel) SetComments(comments []models.Comment) {
	commentsLogger.Debugf("set_comments_count=%d", len(comments))
	commentsLogger.Debugf("viewport_dimensions width=%d height=%d", c.viewport.Width(), c.viewport.Height())
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
		len(c.comments), c.viewport.Width(), c.viewport.Height())

	if len(c.comments) == 0 {
		commentsLogger.Debugf("no_comments_to_display")
		c.viewport.SetContent("")
		return
	}

	if c.viewport.Width() <= 0 || c.viewport.Height() <= 0 {
		commentsLogger.Debugf("viewport_dimensions_not_set")
		return
	}

	selected := -1
	if c.focused {
		selected = c.cursor
	}

	content, offsets := RenderCommentList(c.comments, c.styles, CommentListConfig{
		Width:      c.viewport.Width(),
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

	vpTop := c.viewport.YOffset()
	vpBottom := vpTop + c.viewport.Height() - 1

	if topLine < vpTop {
		c.viewport.SetYOffset(topLine)
	} else if bottomLine > vpBottom {
		c.viewport.SetYOffset(bottomLine - c.viewport.Height() + 1)
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
		return Hints(styles.ColorSurface, c.keys.AddComment)
	}
	return Hints(styles.ColorSurface, c.keys.Up, c.keys.AddComment, c.keys.EditComment, c.keys.DeleteComment)
}

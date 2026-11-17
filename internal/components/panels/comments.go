package panels

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type CommentsPanel struct {
	thread   CommentThread
	viewport viewport.Model
	offsets  []int
	styles   *styles.Styles
	keys     keys.KeyMap
	width    int
	height   int
	focused  bool
	loading  bool
}

func NewCommentsPanel(st *styles.Styles, k keys.KeyMap) CommentsPanel {
	return CommentsPanel{
		thread:   NewCommentThread(k),
		viewport: widgets.NewViewport(k),
		styles:   st,
		keys:     k,
	}
}

func (c CommentsPanel) Update(msg tea.Msg) (CommentsPanel, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if !c.focused || c.loading {
			return c, nil
		}
		before := c.thread.Cursor()
		var cmd tea.Cmd
		c.thread, cmd = c.thread.Update(k)
		if c.thread.Cursor() != before {
			c.render()
			top, bottom := c.thread.Span(c.offsets, c.viewport.TotalLineCount())
			widgets.Reveal(&c.viewport, top, bottom)
		}
		return c, cmd
	}
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

func (c CommentsPanel) View() string {
	if c.thread.Len() == 0 {
		hint := c.styles.TextMuted.Render("No comments yet. Press " + c.keys.AddComment.Help().Key + " to start the discussion.")
		return Block(hint, c.width, c.height)
	}
	body := c.viewport.View()
	if c.viewport.TotalLineCount() > c.viewport.VisibleLineCount() {
		body += "\n" + Muted(fmt.Sprintf("%d%%", int(c.viewport.ScrollPercent()*100)))
	}
	return Block(body, c.width, c.height)
}

func (c CommentsPanel) Count() int { return c.thread.Len() }

func (c CommentsPanel) Loading() bool { return c.loading }

func (c CommentsPanel) WorkItemID() int { return c.thread.WorkItemID() }

func (c *CommentsPanel) SetSize(width, height int) {
	if c.width == width && c.height == height {
		return
	}
	c.width = width
	c.height = height
	c.viewport.SetWidth(max(width, 1))
	c.viewport.SetHeight(max(height-1, 1))
	c.render()
}

func (c *CommentsPanel) StartLoading(workItemID int) {
	c.thread = c.thread.WithComments(workItemID, nil)
	c.loading = true
	c.viewport.GotoTop()
	c.render()
}

func (c *CommentsPanel) SetComments(workItemID int, comments []models.Comment) {
	if workItemID != c.thread.WorkItemID() {
		return
	}
	c.thread = c.thread.WithComments(workItemID, comments)
	c.loading = false
	c.render()
}

func (c *CommentsPanel) SetIdentity(me models.Identity) {
	c.thread = c.thread.WithIdentity(me)
	c.render()
}

func (c *CommentsPanel) Clear() {
	c.thread = c.thread.WithComments(0, nil)
	c.loading = false
	c.render()
}

func (c *CommentsPanel) SetFocused(focused bool) {
	if c.focused == focused {
		return
	}
	c.focused = focused
	c.render()
}

func (c *CommentsPanel) render() {
	if c.viewport.Width() <= 0 {
		return
	}
	content, offsets := c.thread.Render(c.viewport.Width(), c.focused)
	c.offsets = offsets
	c.viewport.SetContent(content)
}

func (c *CommentsPanel) KeyBindings() []key.Binding {
	return c.thread.KeyBindings()
}

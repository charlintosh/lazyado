package panels

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type EditCommentMsg struct {
	WorkItemID int
	Comment    models.Comment
}

type DeleteCommentMsg struct {
	WorkItemID int
	Comment    models.Comment
}

type CommentReadOnlyMsg struct {
	IdentityKnown bool
}

type CommentThread struct {
	workItemID int
	comments   []models.Comment
	cursor     int
	keys       keys.KeyMap
	me         models.Identity
}

func NewCommentThread(k keys.KeyMap) CommentThread {
	return CommentThread{keys: k}
}

func (t CommentThread) WithComments(workItemID int, comments []models.Comment) CommentThread {
	if workItemID != t.workItemID {
		t.cursor = 0
	}
	t.workItemID = workItemID
	t.comments = comments
	t.cursor = max(min(t.cursor, len(comments)-1), 0)
	return t
}

func (t CommentThread) WithIdentity(me models.Identity) CommentThread {
	t.me = me
	return t
}

func (t CommentThread) CanModify() bool {
	return t.cursor >= 0 && t.cursor < len(t.comments) && t.me.Owns(t.comments[t.cursor])
}

func (t CommentThread) WorkItemID() int { return t.workItemID }

func (t CommentThread) Len() int { return len(t.comments) }

func (t CommentThread) Cursor() int { return t.cursor }

func (t CommentThread) Update(msg tea.KeyPressMsg) (CommentThread, tea.Cmd) {
	if len(t.comments) == 0 {
		return t, nil
	}
	switch {
	case key.Matches(msg, t.keys.Up):
		t.cursor = max(t.cursor-1, 0)
	case key.Matches(msg, t.keys.Down):
		t.cursor = min(t.cursor+1, len(t.comments)-1)
	case key.Matches(msg, t.keys.Top):
		t.cursor = 0
	case key.Matches(msg, t.keys.Bottom):
		t.cursor = len(t.comments) - 1
	case key.Matches(msg, t.keys.EditComment), key.Matches(msg, t.keys.DeleteComment):
		return t, t.modify(key.Matches(msg, t.keys.EditComment))
	}
	return t, nil
}

func (t CommentThread) modify(edit bool) tea.Cmd {
	if !t.CanModify() {
		readOnly := CommentReadOnlyMsg{IdentityKnown: t.me.Known()}
		return func() tea.Msg { return readOnly }
	}
	comment := t.comments[t.cursor]
	if edit {
		msg := EditCommentMsg{WorkItemID: t.workItemID, Comment: comment}
		return func() tea.Msg { return msg }
	}
	msg := DeleteCommentMsg{WorkItemID: t.workItemID, Comment: comment}
	return func() tea.Msg { return msg }
}

func (t CommentThread) Render(s *styles.Styles, width int, focused bool) (string, []int) {
	selected := -1
	if focused {
		selected = t.cursor
	}
	return RenderCommentList(t.comments, s, CommentListConfig{Width: width, Selected: selected, Me: t.me})
}

func (t CommentThread) RenderFlat(width int, focused bool) (string, []int) {
	selected := -1
	if focused {
		selected = t.cursor
	}
	return RenderDiscussion(t.comments, width, selected, t.me)
}

func (t CommentThread) Span(offsets []int, totalLines int) (int, int) {
	if t.cursor < 0 || t.cursor >= len(offsets) {
		return 0, 0
	}
	bottom := totalLines - 1
	if t.cursor+1 < len(offsets) {
		bottom = offsets[t.cursor+1] - 1
	}
	return offsets[t.cursor], bottom
}

func (t CommentThread) KeyBindings() []key.Binding {
	if len(t.comments) == 0 {
		return []key.Binding{t.keys.AddComment}
	}
	if !t.CanModify() {
		return []key.Binding{t.keys.Up, t.keys.AddComment}
	}
	return []key.Binding{t.keys.Up, t.keys.AddComment, t.keys.EditComment, t.keys.DeleteComment}
}

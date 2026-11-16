package modals

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const commentPreviewLen = 60

type WorkItemDeleteRequestMsg struct {
	ID int
}

type CommentDeleteRequestMsg struct {
	WorkItemID int
	CommentID  int
}

type ConfirmModal struct {
	env     Env
	title   string
	message string
	confirm tea.Msg
}

func NewDeleteWorkItemModal(env Env, item models.WorkItem) ConfirmModal {
	return ConfirmModal{
		env:     env,
		title:   "Delete work item",
		message: "Are you sure you want to delete:\n" + itemLabel(item),
		confirm: WorkItemDeleteRequestMsg{ID: item.ID},
	}
}

func NewDeleteCommentModal(env Env, workItemID int, comment models.Comment) ConfirmModal {
	preview := ansi.Truncate(components.PlainText(components.CommentMarkdown(comment)), commentPreviewLen, "...")
	return ConfirmModal{
		env:     env,
		title:   "Delete comment",
		message: fmt.Sprintf("Are you sure you want to delete this comment?\n%q", preview),
		confirm: CommentDeleteRequestMsg{WorkItemID: workItemID, CommentID: comment.ID},
	}
}

func (m ConfirmModal) Init() tea.Cmd { return nil }

func (m ConfirmModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.env.Keys.Confirm):
			return m, send(m.confirm)
		case key.Matches(msg, m.env.Keys.Back):
			return m, closeModal
		}
	}
	return m, nil
}

func (m ConfirmModal) View() string {
	width := widgets.DialogWidth(styles.ModalWidthMD, m.env.Width)
	inner := widgets.DialogInnerWidth(width)
	body := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(styles.ColorText).Width(inner).Render(m.message),
		"",
		lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("This action cannot be undone!"),
	)
	k := m.env.Keys
	return widgets.Dialog{
		Title:  m.title,
		Body:   body,
		Footer: widgets.Hints(inner, widgets.Relabel(k.Confirm, "delete"), widgets.Relabel(k.Back, "cancel")),
		Width:  width,
		Accent: styles.ColorError,
	}.View()
}

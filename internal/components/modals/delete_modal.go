package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DeleteModal is a modal for confirming work item deletion
type DeleteModal struct {
	visible    bool
	item       *models.WorkItem
	comment    *models.Comment
	workItemID int
	styles     styles.Styles
	keys       keys.KeyMap
	width      int
	height     int
}

// NewDeleteModal creates a new delete confirmation modal
func NewDeleteModal(styles styles.Styles, keys keys.KeyMap) DeleteModal {
	return DeleteModal{
		styles: styles,
		keys:   keys,
	}
}

// Init initializes the modal
func (m DeleteModal) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m DeleteModal) Update(msg tea.Msg) (DeleteModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Confirm) {
			if m.item != nil {
				return m, func() tea.Msg {
					return TaskDeleteRequestMsg{TaskID: m.item.ID}
				}
			}
			if m.comment != nil {
				return m, func() tea.Msg {
					return CommentDeleteRequestMsg{WorkItemID: m.workItemID, CommentID: m.comment.ID}
				}
			}
		}

		if key.Matches(msg, m.keys.Back) {
			m.visible = false
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}

	return m, nil
}

// View renders the modal
func (m DeleteModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := 50
	modalHeight := 8

	// Build content
	var b strings.Builder

	// Title
	titleText := "⚠ Delete Work Item"
	if m.comment != nil {
		titleText = "⚠ Delete Comment"
	}
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorError).
		Render(titleText)
	b.WriteString(title)
	b.WriteString("\n\n")

	// Confirmation message
	var message string
	if m.item != nil {
		message = fmt.Sprintf("Are you sure you want to delete:\n#%d - %s", m.item.ID, m.item.Title)
	} else if m.comment != nil {
		commentPreview := m.comment.Text
		if len(commentPreview) > 50 {
			commentPreview = commentPreview[:47] + "..."
		}
		message = fmt.Sprintf("Are you sure you want to delete this comment?\n\"%s\"", commentPreview)
	}
	messageStyle := lipgloss.NewStyle().
		Foreground(styles.ColorText)
	b.WriteString(messageStyle.Render(message))
	b.WriteString("\n\n")

	// Warning
	warning := lipgloss.NewStyle().
		Foreground(styles.ColorWarning).
		Render("This action cannot be undone!")
	b.WriteString(warning)
	b.WriteString("\n\n")

	// Instructions (derived from KeyMap)
	confirm := m.keys.Confirm.Help()
	back := m.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s • %s: %s", confirm.Key, confirm.Desc, back.Key, back.Desc)
	b.WriteString(m.styles.ModalInstructions.Render(instr))

	// Style the modal
	content := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorError).
		Render(b.String())

	// Center the modal
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// SetSize sets the modal size
func (m *DeleteModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetVisible sets the modal visibility
func (m *DeleteModal) SetVisible(visible bool) {
	m.visible = visible
}

// IsVisible returns whether the modal is visible
func (m *DeleteModal) IsVisible() bool {
	return m.visible
}

// SetItem sets the item to delete
func (m *DeleteModal) SetItem(item *models.WorkItem) {
	m.item = item
	m.comment = nil
	m.workItemID = 0
}

// SetCommentItem sets the comment to delete
func (m *DeleteModal) SetCommentItem(comment *models.Comment, workItemID int) {
	m.comment = comment
	m.item = nil
	m.workItemID = workItemID
}

// CommentDeleteRequestMsg is sent when deleting a comment
type CommentDeleteRequestMsg struct {
	WorkItemID int
	CommentID  int
}

// TaskDeleteRequestMsg is sent when deleting a task
type TaskDeleteRequestMsg struct {
	TaskID int
}

// TaskDeletedMsg is sent after a task is deleted
type TaskDeletedMsg struct {
	TaskID int
}

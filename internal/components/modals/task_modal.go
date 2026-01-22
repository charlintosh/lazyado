package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TaskModal is a modal for creating/editing tasks
type TaskModal struct {
	visible         bool
	editMode        bool
	parent          *models.WorkItem
	task            *models.WorkItem
	titleInput      textinput.Model
	descriptionArea textarea.Model
	focusedField    int // 0 = title, 1 = description
	styles          styles.Styles
	keys            keys.KeyMap
	width           int
	height          int
	err             string
}

// NewTaskModal creates a new task modal
func NewTaskModal(styles styles.Styles, keys keys.KeyMap) TaskModal {
	ti := textinput.New()
	ti.Placeholder = "Enter task title..."
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 50

	ta := textarea.New()
	ta.Placeholder = "Enter description (optional)..."
	ta.CharLimit = 1000
	ta.SetWidth(50)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	return TaskModal{
		titleInput:      ti,
		descriptionArea: ta,
		focusedField:    0,
		styles:          styles,
		keys:            keys,
	}
}

// Init initializes the modal
func (m TaskModal) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m TaskModal) Update(msg tea.Msg) (TaskModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.visible = false
			m.err = ""
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case key.Matches(msg, m.keys.NextPanel):
			// Switch focus between title and description
			if m.focusedField == 0 {
				m.focusedField = 1
				m.titleInput.Blur()
				m.descriptionArea.Focus()
			} else {
				m.focusedField = 0
				m.descriptionArea.Blur()
				m.titleInput.Focus()
			}
			return m, nil
		case key.Matches(msg, m.keys.Save):
			// Only handle Enter key for submission, not space
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				m.err = "Title cannot be empty"
				return m, nil
			}

			description := strings.TrimSpace(m.descriptionArea.Value())

			if m.editMode && m.task != nil {
				// Edit mode
				return m, func() tea.Msg {
					return TaskUpdateRequestMsg{
						TaskID:      m.task.ID,
						NewTitle:    title,
						Description: description,
					}
				}
			} else if m.parent != nil {
				// Create mode
				return m, func() tea.Msg {
					return TaskCreateRequestMsg{
						ParentID:    m.parent.ID,
						Title:       title,
						Description: description,
					}
				}
			}
		}
	}

	// Update the focused field
	if m.focusedField == 0 {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.descriptionArea, cmd = m.descriptionArea.Update(msg)
	}
	return m, cmd
}

// View renders the modal
func (m TaskModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := 60
	modalHeight := 18

	// Build content
	var b strings.Builder

	// Title
	var title string
	if m.editMode {
		title = m.styles.ModalTitle.Render("Edit Task")
	} else {
		title = m.styles.ModalTitle.Render("Create New Task")
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	// Parent info
	if m.parent != nil && !m.editMode {
		parentInfo := lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Render(fmt.Sprintf("Parent: #%d %s", m.parent.ID, m.parent.Title))
		b.WriteString(parentInfo)
		b.WriteString("\n\n")
	}

	// Title input
	titleLabel := "Title:"
	if m.focusedField == 0 {
		titleLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("Title:")
	}
	b.WriteString(titleLabel + "\n")
	b.WriteString(m.titleInput.View())
	b.WriteString("\n\n")

	// Description input
	descLabel := "Description (optional):"
	if m.focusedField == 1 {
		descLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("Description (optional):")
	}
	b.WriteString(descLabel + "\n")
	b.WriteString(m.descriptionArea.View())
	b.WriteString("\n\n")

	// Error message
	if m.err != "" {
		errStyle := m.styles.ErrorText
		b.WriteString(errStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	// Instructions - derive from KeyMap to keep help consistent
	next := m.keys.NextPanel.Help()
	save := m.keys.Save.Help()
	back := m.keys.Back.Help()

	// Build a plain instruction string and render it with ModalInstructions
	instr := fmt.Sprintf("%s: %s • %s: %s • %s: %s", next.Key, next.Desc, save.Key, save.Desc, back.Key, back.Desc)
	b.WriteString(m.styles.ModalInstructions.Render(instr))

	// Style the modal
	contentStr := b.String()
	modalHeight = lipgloss.Height(contentStr) + 4
	content := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Render(contentStr)

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
func (m *TaskModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetVisible sets the modal visibility
func (m *TaskModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.focusedField = 0
		m.titleInput.Focus()
		m.descriptionArea.Blur()
	} else {
		m.titleInput.Blur()
		m.descriptionArea.Blur()
	}
}

// IsVisible returns whether the modal is visible
func (m *TaskModal) IsVisible() bool {
	return m.visible
}

// SetParent sets the parent work item for creating a child task
func (m *TaskModal) SetParent(parent *models.WorkItem) {
	m.parent = parent
	m.task = nil
	m.editMode = false
	m.titleInput.SetValue("")
	m.descriptionArea.SetValue("")
	m.focusedField = 0
	m.err = ""
}

// SetTask sets the task to edit
func (m *TaskModal) SetTask(task *models.WorkItem) {
	m.task = task
	m.parent = nil
	m.editMode = true
	m.titleInput.SetValue(task.Title)
	m.descriptionArea.SetValue(task.Description)
	m.focusedField = 0
	m.err = ""
}

// TaskCreateRequestMsg is sent when creating a new task
type TaskCreateRequestMsg struct {
	ParentID    int
	Title       string
	Description string
}

// TaskUpdateRequestMsg is sent when updating a task
type TaskUpdateRequestMsg struct {
	TaskID      int
	NewTitle    string
	Description string
}

// TaskCreatedMsg is sent after a task is created
type TaskCreatedMsg struct {
	TaskID int
}

// TaskUpdatedMsg is sent after a task is updated
type TaskUpdatedMsg struct {
	TaskID int
}

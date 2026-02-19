package modals

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BranchModal is a modal for creating a branch linked to a work item
type BranchModal struct {
	visible   bool
	item      *models.WorkItem
	textInput textinput.Model
	styles    styles.Styles
	keys      keys.KeyMap
	width     int
	height    int
	err       error
}

// NewBranchModal creates a new branch modal
func NewBranchModal(st styles.Styles, keys keys.KeyMap) BranchModal {
	ti := textinput.New()
	ti.Placeholder = "feature/123-task-name"
	ti.CharLimit = styles.InputCharLimitLG
	ti.Width = styles.TextInputWidthLG

	return BranchModal{
		textInput: ti,
		styles:    st,
		keys:      keys,
	}
}

// Init initializes the modal
func (m BranchModal) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m BranchModal) Update(msg tea.Msg) (BranchModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.visible = false
			m.err = nil
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case msg.Type == tea.KeyEnter:
			branchName := strings.TrimSpace(m.textInput.Value())
			if branchName == "" {
				m.err = fmt.Errorf("branch name cannot be empty")
				return m, nil
			}
			if !isValidBranchName(branchName) {
				m.err = fmt.Errorf("invalid branch name")
				return m, nil
			}
			m.err = nil
			return m, func() tea.Msg {
				return BranchCreateRequestMsg{
					Item:       *m.item,
					BranchName: branchName,
				}
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the modal
func (m BranchModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := styles.ModalWidthMD
	modalHeight := styles.ModalHeightMD

	// Build content
	var b strings.Builder

	// Title
	title := m.styles.ModalTitle.Render("Create Branch")
	b.WriteString(title + "\n")

	// Item info
	if m.item != nil {
		itemInfo := lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Render("#" + components.Itoa(m.item.ID) + " " + components.TruncateStr(m.item.Title, 35))
		b.WriteString(itemInfo + "\n\n")
	}

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorLight)
	b.WriteString(labelStyle.Render("Branch name:") + "\n")

	// Text input
	b.WriteString(m.textInput.View() + "\n")

	// Error message
	if m.err != nil {
		errStyle := m.styles.ErrorText
		b.WriteString(errStyle.Render(m.err.Error()) + "\n")
	} else {
		b.WriteString("\n")
	}

	// Help text (derived from KeyMap)
	sel := m.keys.Select.Help()
	back := m.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s  %s: %s", sel.Key, sel.Desc, back.Key, back.Desc)
	b.WriteString(m.styles.ModalInstructions.Render(instr))

	// Modal style
	modal := m.styles.ModalBox.Width(modalWidth).Height(modalHeight).Render(b.String())

	// Center the modal
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// SetVisible sets the visibility
func (m *BranchModal) SetVisible(visible bool) {
	m.visible = visible
	m.err = nil
	if visible {
		m.textInput.Focus()
		// Generate suggested branch name from work item
		if m.item != nil {
			suggested := generateBranchName(m.item)
			m.textInput.SetValue(suggested)
			m.textInput.CursorEnd()
		}
	} else {
		m.textInput.Blur()
	}
}

// IsVisible returns whether the modal is visible
func (m *BranchModal) IsVisible() bool {
	return m.visible
}

// SetItem sets the work item to link
func (m *BranchModal) SetItem(item *models.WorkItem) {
	m.item = item
}

// SetSize sets the modal container size
func (m *BranchModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// generateBranchName generates a suggested branch name from work item
func generateBranchName(item *models.WorkItem) string {
	// Get work item type prefix
	prefix := "feature"
	switch item.Type {
	case models.WorkItemTypeBug:
		prefix = "bugfix"
	case models.WorkItemTypeTask:
		prefix = "task"
	case models.WorkItemTypeStory:
		prefix = "feature"
	case models.WorkItemTypeEpic:
		prefix = "epic"
	}

	// Sanitize title for branch name
	title := strings.ToLower(item.Title)
	// Replace spaces and special chars with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	title = re.ReplaceAllString(title, "-")
	// Remove leading/trailing hyphens
	title = strings.Trim(title, "-")
	// Truncate to reasonable length
	if len(title) > styles.BranchNameMaxLen {
		title = title[:styles.BranchNameMaxLen]
		// Don't cut off in middle of a word if possible
		if lastHyphen := strings.LastIndex(title, "-"); lastHyphen > styles.BranchNameWordBoundaryMin {
			title = title[:lastHyphen]
		}
	}

	return fmt.Sprintf("%s/%d-%s", prefix, item.ID, title)
}

// isValidBranchName validates branch name
func isValidBranchName(name string) bool {
	// Basic validation for git branch names
	if name == "" {
		return false
	}
	// Check for invalid characters
	invalid := regexp.MustCompile(`[\s~^:?*\[\]\\]`)
	if invalid.MatchString(name) {
		return false
	}
	// Can't start or end with /
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}
	// Can't have consecutive dots
	if strings.Contains(name, "..") {
		return false
	}
	return true
}

// BranchCreateRequestMsg is sent when user confirms branch creation
type BranchCreateRequestMsg struct {
	Item       models.WorkItem
	BranchName string
}

// BranchCreatedMsg is sent when branch creation is complete
type BranchCreatedMsg struct {
	BranchName string
}

// BranchCreateErrorMsg is sent when branch creation fails
type BranchCreateErrorMsg struct {
	Err error
}

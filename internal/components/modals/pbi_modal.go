package modals

import (
	"fmt"
	"strconv"
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

// PBIModal is a unified modal for creating and editing PBIs
type PBIModal struct {
	visible                bool
	editMode               bool
	item                   *models.WorkItem
	titleInput             textinput.Model
	descriptionArea        textarea.Model
	acceptanceCriteriaArea textarea.Model
	priorityInput          textinput.Model
	effortInput            textinput.Model
	tagsInput              textinput.Model
	assigneeInput          textinput.Model
	areaInput              textinput.Model
	sprintCursor           int
	members                []models.TeamMember
	filteredMembers        []models.TeamMember
	assigneeCursor         int
	showAssigneeList       bool
	selectedAssignee       string
	focusedField           int
	styles                 styles.Styles
	keys                   keys.KeyMap
	width                  int
	height                 int
	err                    string
	iterations             []models.Iteration
	areas                  []models.Area
}

// NewPBIModal creates a new PBI modal
func NewPBIModal(styles styles.Styles, keys keys.KeyMap) PBIModal {
	ti := textinput.New()
	ti.Placeholder = "Enter title..."
	ti.Focus()
	ti.CharLimit = 255
	ti.Width = 50

	da := textarea.New()
	da.Placeholder = "Enter description (optional)..."
	da.CharLimit = 1000
	da.SetWidth(50)
	da.SetHeight(3)
	da.ShowLineNumbers = false

	aca := textarea.New()
	aca.Placeholder = "Enter acceptance criteria (optional)..."
	aca.CharLimit = 1000
	aca.SetWidth(50)
	aca.SetHeight(3)
	aca.ShowLineNumbers = false

	pi := textinput.New()
	pi.Placeholder = "1-4"
	pi.CharLimit = 1
	pi.Width = 10

	ei := textinput.New()
	ei.Placeholder = "0-999"
	ei.CharLimit = 5
	ei.Width = 10

	tagi := textinput.New()
	tagi.Placeholder = "tag1; tag2; tag3"
	tagi.CharLimit = 500
	tagi.Width = 50

	assi := textinput.New()
	assi.Placeholder = "Type to search assignee..."
	assi.CharLimit = 100
	assi.Width = 50

	areai := textinput.New()
	areai.Placeholder = "Area path"
	areai.CharLimit = 255
	areai.Width = 50

	return PBIModal{
		titleInput:             ti,
		descriptionArea:        da,
		acceptanceCriteriaArea: aca,
		priorityInput:          pi,
		effortInput:            ei,
		tagsInput:              tagi,
		assigneeInput:          assi,
		areaInput:              areai,
		focusedField:           0,
		styles:                 styles,
		keys:                   keys,
	}
}

// Init initializes the modal
func (m PBIModal) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m PBIModal) Update(msg tea.Msg) (PBIModal, tea.Cmd) {
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
			m.blurAll()
			m.showAssigneeList = false
			m.focusedField++
			maxField := 6
			if !m.editMode {
				maxField = 7 // Include sprint and area in create mode
			}
			if m.focusedField > maxField {
				m.focusedField = 0
			}
			m.focusField(m.focusedField)
			return m, nil
		case key.Matches(msg, m.keys.PrevPanel):
			m.blurAll()
			m.showAssigneeList = false
			m.focusedField--
			if m.focusedField < 0 {
				maxField := 6
				if !m.editMode {
					maxField = 7
				}
				m.focusedField = maxField
			}
			m.focusField(m.focusedField)
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.focusedField == 6 && m.showAssigneeList {
				if m.assigneeCursor > 0 {
					m.assigneeCursor--
				}
				return m, nil
			} else if m.focusedField == 7 && !m.editMode {
				// Sprint selection in create mode
				if m.sprintCursor > 0 {
					m.sprintCursor--
				}
				return m, nil
			}
		case key.Matches(msg, m.keys.Down):
			if m.focusedField == 6 && m.showAssigneeList {
				if m.assigneeCursor < len(m.filteredMembers)-1 {
					m.assigneeCursor++
				}
				return m, nil
			} else if m.focusedField == 7 && !m.editMode {
				// Sprint selection in create mode
				if m.sprintCursor < len(m.iterations)-1 {
					m.sprintCursor++
				}
				return m, nil
			}
		case key.Matches(msg, m.keys.Select):
			if m.focusedField == 6 && m.showAssigneeList && len(m.filteredMembers) > 0 {
				selected := m.filteredMembers[m.assigneeCursor]
				m.selectedAssignee = selected.UniqueName
				m.assigneeInput.SetValue(selected.DisplayName)
				m.showAssigneeList = false
				return m, nil
			}
		case key.Matches(msg, m.keys.Save):
			title := strings.TrimSpace(m.titleInput.Value())
			if title == "" {
				m.err = "Title cannot be empty"
				return m, nil
			}

			description := strings.TrimSpace(m.descriptionArea.Value())
			acceptanceCriteria := strings.TrimSpace(m.acceptanceCriteriaArea.Value())

			priority := 0
			priorityStr := strings.TrimSpace(m.priorityInput.Value())
			if priorityStr != "" {
				p, err := strconv.Atoi(priorityStr)
				if err != nil || p < 1 || p > 4 {
					m.err = "Priority must be 1-4"
					return m, nil
				}
				priority = p
			}

			effort := 0.0
			effortStr := strings.TrimSpace(m.effortInput.Value())
			if effortStr != "" {
				e, err := strconv.ParseFloat(effortStr, 64)
				if err != nil || e < 0 {
					m.err = "Effort must be a positive number"
					return m, nil
				}
				effort = e
			}

			tagsStr := strings.TrimSpace(m.tagsInput.Value())
			var tags []string
			if tagsStr != "" {
				parts := strings.Split(tagsStr, ";")
				for _, part := range parts {
					tag := strings.TrimSpace(part)
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}

			if m.editMode {
				return m, func() tea.Msg {
					return PBIUpdateRequestMsg{
						ItemID:      m.item.ID,
						Title:       title,
						Description: description,
						Priority:    priority,
						Effort:      effort,
						Tags:        tags,
						AssignedTo:  m.selectedAssignee,
					}
				}
			} else {
				// Create mode - get area and iteration
				areaPath := strings.TrimSpace(m.areaInput.Value())
				iterationPath := ""
				if m.sprintCursor >= 0 && m.sprintCursor < len(m.iterations) {
					iterationPath = m.iterations[m.sprintCursor].Path
				}

				return m, func() tea.Msg {
					return ParentCreateRequestMsg{
						WorkItemType:       "Product Backlog Item",
						Title:              title,
						Description:        description,
						AcceptanceCriteria: acceptanceCriteria,
						IterationPath:      iterationPath,
						AreaPath:           areaPath,
						AssignedTo:         m.selectedAssignee,
						Effort:             effort,
						Priority:           priority,
						Tags:               tags,
						DefectCause:        "",
					}
				}
			}
		}
	}

	// Update the focused field
	switch m.focusedField {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 1:
		m.descriptionArea, cmd = m.descriptionArea.Update(msg)
	case 2:
		m.acceptanceCriteriaArea, cmd = m.acceptanceCriteriaArea.Update(msg)
	case 3:
		m.priorityInput, cmd = m.priorityInput.Update(msg)
	case 4:
		m.effortInput, cmd = m.effortInput.Update(msg)
	case 5:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	case 6:
		m.assigneeInput, cmd = m.assigneeInput.Update(msg)
		m.filterAssignees()
	case 7:
		// Sprint (create mode) or area (both modes)
		if !m.editMode {
			m.areaInput, cmd = m.areaInput.Update(msg)
		}
	}
	return m, cmd
}

func (m *PBIModal) blurAll() {
	m.titleInput.Blur()
	m.descriptionArea.Blur()
	m.acceptanceCriteriaArea.Blur()
	m.priorityInput.Blur()
	m.effortInput.Blur()
	m.tagsInput.Blur()
	m.assigneeInput.Blur()
	m.areaInput.Blur()
}

func (m *PBIModal) focusField(field int) {
	switch field {
	case 0:
		m.titleInput.Focus()
	case 1:
		m.descriptionArea.Focus()
	case 2:
		m.acceptanceCriteriaArea.Focus()
	case 3:
		m.priorityInput.Focus()
	case 4:
		m.effortInput.Focus()
	case 5:
		m.tagsInput.Focus()
	case 6:
		m.assigneeInput.Focus()
		m.showAssigneeList = true
		m.filterAssignees()
	case 7:
		if !m.editMode {
			m.areaInput.Focus()
		}
	}
}

func (m *PBIModal) filterAssignees() {
	filter := strings.ToLower(m.assigneeInput.Value())
	if filter == "" {
		m.filteredMembers = m.members
	} else {
		m.filteredMembers = make([]models.TeamMember, 0)
		for _, member := range m.members {
			if strings.Contains(strings.ToLower(member.DisplayName), filter) ||
				strings.Contains(strings.ToLower(member.UniqueName), filter) {
				m.filteredMembers = append(m.filteredMembers, member)
			}
		}
	}
	if m.assigneeCursor >= len(m.filteredMembers) {
		m.assigneeCursor = 0
	}
	m.showAssigneeList = len(m.filteredMembers) > 0
}

// View renders the modal
func (m PBIModal) View() string {
	if !m.visible {
		return ""
	}

	modalWidth := 70
	var b strings.Builder

	var modalTitle string
	if m.editMode {
		modalTitle = "Edit Product Backlog Item"
	} else {
		modalTitle = "Create Product Backlog Item"
	}
	title := m.styles.ModalTitle.Render(modalTitle)
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.editMode && m.item != nil {
		itemInfo := m.styles.TextMuted.Render(fmt.Sprintf("ID: #%d", m.item.ID))
		b.WriteString(itemInfo)
		b.WriteString("\n\n")
	}

	focusColor := styles.ColorPrimary

	// Title
	b.WriteString(m.renderFieldLabel("Title:", m.focusedField == 0, focusColor))
	b.WriteString(m.titleInput.View())
	b.WriteString("\n\n")

	// Description
	b.WriteString(m.renderFieldLabel("Description:", m.focusedField == 1, focusColor))
	b.WriteString(m.descriptionArea.View())
	b.WriteString("\n\n")

	// Acceptance Criteria (only in create mode)
	if !m.editMode {
		b.WriteString(m.renderFieldLabel("Acceptance Criteria (optional):", m.focusedField == 2, focusColor))
		b.WriteString(m.acceptanceCriteriaArea.View())
		b.WriteString("\n\n")
	}

	// Priority and Effort on same line
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderField("Priority (1-4):", m.priorityInput.View(), m.focusedField == 3, focusColor),
		"    ",
		m.renderField("Effort:", m.effortInput.View(), m.focusedField == 4, focusColor),
	)
	b.WriteString(row)
	b.WriteString("\n\n")

	// Tags
	b.WriteString(m.renderFieldLabel("Tags (separate with ;):", m.focusedField == 5, focusColor))
	b.WriteString(m.tagsInput.View())
	b.WriteString("\n\n")

	// Assignee
	b.WriteString(m.renderFieldLabel("Assign To:", m.focusedField == 6, focusColor))
	b.WriteString(m.assigneeInput.View())

	if m.focusedField == 6 && m.showAssigneeList && len(m.filteredMembers) > 0 {
		b.WriteString("\n")
		maxVisible := 4
		end := len(m.filteredMembers)
		if end > maxVisible {
			end = maxVisible
		}

		for i := 0; i < end; i++ {
			member := m.filteredMembers[i]
			cursor := "  "
			if i == m.assigneeCursor {
				cursor = "▸ "
			}

			style := m.styles.TextMuted
			if i == m.assigneeCursor {
				style = m.styles.AccentBold
			}

			b.WriteString(cursor + style.Render(member.DisplayName) + "\n")
		}

		if len(m.filteredMembers) > maxVisible {
			moreStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
			b.WriteString(moreStyle.Render(fmt.Sprintf("  ... %d more", len(m.filteredMembers)-maxVisible)))
		}
	}
	b.WriteString("\n\n")

	// Sprint selection (only in create mode)
	if !m.editMode {
		b.WriteString(m.renderFieldLabel("Sprint:", m.focusedField == 7, focusColor))
		if len(m.iterations) > 0 && m.sprintCursor >= 0 && m.sprintCursor < len(m.iterations) {
			b.WriteString(m.styles.TextMuted.Render(m.iterations[m.sprintCursor].Name))
		} else {
			b.WriteString(m.styles.TextMuted.Render("None"))
		}
		b.WriteString("\n\n")

		// Area
		b.WriteString(m.renderFieldLabel("Area:", false, focusColor))
		b.WriteString(m.areaInput.View())
		b.WriteString("\n\n")
	}

	// Error message
	if m.err != "" {
		errStyle := m.styles.ErrorText
		b.WriteString(errStyle.Render(m.err))
		b.WriteString("\n\n")
	}

	// Instructions
	next := m.keys.NextPanel.Help()
	prev := m.keys.PrevPanel.Help()
	save := m.keys.Save.Help()
	back := m.keys.Back.Help()
	var instr string
	if !m.editMode && m.focusedField == 7 {
		up := m.keys.Up.Help()
		down := m.keys.Down.Help()
		instr = fmt.Sprintf("%s/%s: Change sprint  •  %s: %s  •  %s: %s", up.Key, down.Key, save.Key, save.Desc, back.Key, back.Desc)
	} else {
		instr = fmt.Sprintf("%s/%s: Switch field  •  %s: %s  •  %s: %s", next.Key, prev.Key, save.Key, save.Desc, back.Key, back.Desc)
	}
	b.WriteString(m.styles.ModalInstructions.Render(instr))

	content := lipgloss.NewStyle().
		Width(modalWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Render(b.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m PBIModal) renderFieldLabel(label string, focused bool, focusColor lipgloss.Color) string {
	if focused {
		return lipgloss.NewStyle().Foreground(focusColor).Render(label) + "\n"
	}
	return label + "\n"
}

func (m PBIModal) renderField(label, value string, focused bool, focusColor lipgloss.Color) string {
	var b strings.Builder
	if focused {
		label = lipgloss.NewStyle().Foreground(focusColor).Render(label)
	}
	b.WriteString(label + "\n")
	b.WriteString(value)
	return b.String()
}

// SetSize sets the modal size
func (m *PBIModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetVisible sets the modal visibility
func (m *PBIModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.focusedField = 0
		m.blurAll()
		m.titleInput.Focus()
	} else {
		m.blurAll()
	}
}

// IsVisible returns whether the modal is visible
func (m *PBIModal) IsVisible() bool {
	return m.visible
}

// SetItem sets the PBI for editing
func (m *PBIModal) SetItem(item *models.WorkItem) {
	m.item = item
	m.editMode = true
	m.titleInput.SetValue(item.Title)
	m.descriptionArea.SetValue(item.Description)

	if item.Priority > 0 {
		m.priorityInput.SetValue(strconv.Itoa(item.Priority))
	} else {
		m.priorityInput.SetValue("")
	}

	if item.Effort > 0 {
		m.effortInput.SetValue(fmt.Sprintf("%.1f", item.Effort))
	} else {
		m.effortInput.SetValue("")
	}

	if len(item.Tags) > 0 {
		m.tagsInput.SetValue(strings.Join(item.Tags, "; "))
	} else {
		m.tagsInput.SetValue("")
	}

	if item.AssignedTo != "" {
		m.assigneeInput.SetValue(item.AssignedTo)
		m.selectedAssignee = item.AssignedTo
		for _, member := range m.members {
			if member.DisplayName == item.AssignedTo {
				m.selectedAssignee = member.UniqueName
				break
			}
		}
	} else {
		m.assigneeInput.SetValue("")
		m.selectedAssignee = ""
	}

	m.focusedField = 0
	m.err = ""
	m.showAssigneeList = false
}

// SetData sets the iterations and areas for creating new items
func (m *PBIModal) SetData(iterations []models.Iteration, areas []models.Area) {
	m.iterations = iterations
	m.areas = areas
	m.editMode = false
	m.titleInput.SetValue("")
	m.descriptionArea.SetValue("")
	m.acceptanceCriteriaArea.SetValue("")
	m.priorityInput.SetValue("")
	m.effortInput.SetValue("")
	m.tagsInput.SetValue("")
	m.assigneeInput.SetValue("")
	m.areaInput.SetValue("")
	m.selectedAssignee = ""
	m.sprintCursor = 0
	m.focusedField = 0
	m.err = ""
	m.showAssigneeList = false
}

// SetDefaultArea sets the default area path
func (m *PBIModal) SetDefaultArea(areaPath string) {
	m.areaInput.SetValue(areaPath)
}

// SetMembers sets the available team members for assignment
func (m *PBIModal) SetMembers(members []models.TeamMember) {
	m.members = members
	m.filteredMembers = members
}

// PBIUpdateRequestMsg is sent when updating a PBI
type PBIUpdateRequestMsg struct {
	ItemID      int
	Title       string
	Description string
	Priority    int
	Effort      float64
	Tags        []string
	AssignedTo  string
}

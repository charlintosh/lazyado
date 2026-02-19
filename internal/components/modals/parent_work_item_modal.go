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

type workItemType int

const (
	workItemTypePBI workItemType = iota
	workItemTypeBug
)

var workItemTypeNames = map[workItemType]string{
	workItemTypePBI: "Product Backlog Item",
	workItemTypeBug: "Bug",
}

var workItemTypeVisual = map[workItemType]string{
	workItemTypePBI: "Product Backlog Item",
	workItemTypeBug: "Bug",
}

var labelText = map[string]string{
	"modalTitle":         "Create Parent Work Item",
	"type":               "Type:",
	"title":              "Title:",
	"description":        "Description (optional):",
	"acceptanceCriteria": "Acceptance Criteria (optional):",
	"area":               "Area:",
	"sprint":             "Sprint:",
	"effort":             "Effort:",
	"priority":           "Priority:",
	"tags":               "Tags (separate with ;):",
	"assignTo":           "Assign To:",
	"defectCause":        "Defect Cause (optional):",
	"none":               "None",
}

var placeholderText = map[string]string{
	"title":              "Enter work item title...",
	"description":        "Enter description (optional)...",
	"acceptanceCriteria": "Enter acceptance criteria (optional)...",
	"effort":             "0",
	"priority":           "0",
	"tags":               "tag1; tag2; tag3",
	"area":               "Area path (will use first visible item's area)",
	"assignee":           "Type to search assignee...",
	"defectCause":        "Enter defect cause (optional)...",
}

var errorMessages = map[string]string{
	"titleRequired":  "Title is required",
	"effortNumber":   "Effort must be a number",
	"priorityNumber": "Priority must be a number",
	"effortRange":    "Effort must be between 0 and 1000",
	"priorityRange":  "Priority must be between 0 and 100",
	"tagsEmpty":      "Tags must not contain empty entries",
	"tagsLength":     "Each tag must be 50 characters or less",
}

// ParentModal is a modal for creating parent work items (PBI/Bug) - simplified like TaskModal
type ParentModal struct {
	visible                bool
	titleInput             textinput.Model
	descriptionArea        textarea.Model
	acceptanceCriteriaArea textarea.Model
	effortInput            textinput.Model
	priorityInput          textinput.Model
	tagsInput              textinput.Model
	areaInput              textinput.Model
	assigneeInput          textinput.Model
	defectCauseInput       textinput.Model
	members                []models.TeamMember
	filteredMembers        []models.TeamMember
	assigneeCursor         int
	showAssigneeList       bool
	selectedAssignee       string
	focusedField           int // 0 = type, 1 = title, 2 = description, 3 = A&C, 4 = area, 5 = sprint, 6 = effort/tags, 7 = priority/defectCause, 8 = tags/assignee, 9 = assignee (PBI)
	styles                 styles.Styles
	keys                   keys.KeyMap
	width                  int
	height                 int
	err                    string

	workItemType   workItemType
	selectedSprint int

	iterations []models.Iteration
}

// NewParentModal creates a new parent modal
func NewParentModal(st styles.Styles, keys keys.KeyMap) ParentModal {
	ti := textinput.New()
	ti.Placeholder = placeholderText["title"]
	ti.CharLimit = styles.TitleCharLimit
	ti.Width = styles.TextareaWidthLG

	ta := textarea.New()
	ta.Placeholder = placeholderText["description"]
	ta.CharLimit = styles.ContentCharLimitLG
	ta.SetWidth(styles.TextareaWidthLG)
	ta.SetHeight(styles.TextareaHeightMD)
	ta.ShowLineNumbers = false

	aca := textarea.New()
	aca.Placeholder = placeholderText["acceptanceCriteria"]
	aca.CharLimit = styles.ContentCharLimitLG
	aca.SetWidth(styles.TextareaWidthLG)
	aca.SetHeight(styles.TextareaHeightSM)
	aca.ShowLineNumbers = false

	ei := textinput.New()
	ei.Placeholder = placeholderText["effort"]
	ei.CharLimit = styles.InputCharLimitSM
	ei.Width = styles.TextInputWidthSM

	pi := textinput.New()
	pi.Placeholder = placeholderText["priority"]
	pi.CharLimit = styles.InputCharLimitSM
	pi.Width = styles.TextInputWidthSM

	tagi := textinput.New()
	tagi.Placeholder = placeholderText["tags"]
	tagi.CharLimit = styles.TagsCharLimit
	tagi.Width = styles.TextareaWidthLG

	areai := textinput.New()
	areai.Placeholder = placeholderText["area"]
	areai.CharLimit = styles.TagsCharLimit
	areai.Width = styles.TextareaWidthLG

	assi := textinput.New()
	assi.Placeholder = placeholderText["assignee"]
	assi.CharLimit = styles.InputCharLimitLG
	assi.Width = styles.TextareaWidthLG

	dci := textinput.New()
	dci.Placeholder = placeholderText["defectCause"]
	dci.CharLimit = styles.TagsCharLimit
	dci.Width = styles.TextareaWidthLG

	return ParentModal{
		titleInput:             ti,
		descriptionArea:        ta,
		acceptanceCriteriaArea: aca,
		effortInput:            ei,
		priorityInput:          pi,
		tagsInput:              tagi,
		areaInput:              areai,
		assigneeInput:          assi,
		defectCauseInput:       dci,
		members:                []models.TeamMember{},
		filteredMembers:        []models.TeamMember{},
		assigneeCursor:         0,
		showAssigneeList:       false,
		selectedAssignee:       "",
		focusedField:           1,
		workItemType:           workItemTypePBI,
		selectedSprint:         0,
		styles:                 st,
		keys:                   keys,
	}
}

// Init initializes the modal
func (p ParentModal) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (p ParentModal) Update(msg tea.Msg) (ParentModal, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.keys.Back):
			p.visible = false
			p.err = ""
			return p, func() tea.Msg { return ModalClosedMsg{} }

		case key.Matches(msg, p.keys.NextPanel):
			p.blurAll()
			p.showAssigneeList = false
			p.focusedField++
			maxField := 8
			if p.workItemType == workItemTypePBI {
				maxField = 9
			}
			if p.focusedField > maxField {
				p.focusedField = 0
			}
			p.focusField()
			return p, nil

		case key.Matches(msg, p.keys.PrevPanel):
			p.blurAll()
			p.showAssigneeList = false
			p.focusedField--
			if p.focusedField < 0 {
				maxField := 8
				if p.workItemType == workItemTypePBI {
					maxField = 9
				}
				p.focusedField = maxField
			}
			p.focusField()
			return p, nil

		case key.Matches(msg, p.keys.Up):
			assigneeFieldForPBI := 9
			assigneeFieldForBug := 8
			isAssigneeField := (p.workItemType == workItemTypePBI && p.focusedField == assigneeFieldForPBI) ||
				(p.workItemType == workItemTypeBug && p.focusedField == assigneeFieldForBug)
			if isAssigneeField && p.showAssigneeList {
				if p.assigneeCursor > 0 {
					p.assigneeCursor--
				}
				return p, nil
			}

		case key.Matches(msg, p.keys.Down):
			assigneeFieldForPBI := 9
			assigneeFieldForBug := 8
			isAssigneeField := (p.workItemType == workItemTypePBI && p.focusedField == assigneeFieldForPBI) ||
				(p.workItemType == workItemTypeBug && p.focusedField == assigneeFieldForBug)
			if isAssigneeField && p.showAssigneeList {
				if p.assigneeCursor < len(p.filteredMembers)-1 {
					p.assigneeCursor++
				}
				return p, nil
			}

		case key.Matches(msg, p.keys.Select):
			assigneeFieldForPBI := 9
			assigneeFieldForBug := 8
			isAssigneeField := (p.workItemType == workItemTypePBI && p.focusedField == assigneeFieldForPBI) ||
				(p.workItemType == workItemTypeBug && p.focusedField == assigneeFieldForBug)
			if isAssigneeField && p.showAssigneeList && len(p.filteredMembers) > 0 {
				selected := p.filteredMembers[p.assigneeCursor]
				p.selectedAssignee = selected.UniqueName
				p.assigneeInput.SetValue(selected.DisplayName)
				p.showAssigneeList = false
				return p, nil
			}

			switch p.focusedField {
			case 0:
				if p.workItemType == workItemTypePBI {
					p.workItemType = workItemTypeBug
				} else {
					p.workItemType = workItemTypePBI
				}
				return p, nil
			case 5:
				if len(p.iterations) > 0 {
					p.selectedSprint = (p.selectedSprint + 1) % len(p.iterations)
				}
				return p, nil
			default:
				return p, nil
			}

		case key.Matches(msg, p.keys.Save):
			// Submit the form on Ctrl+S (Save) — does NOT select assignee
			return p.submit()

		case key.Matches(msg, p.keys.ModalLeft):
			if p.focusedField == 0 {
				if p.workItemType == workItemTypePBI {
					p.workItemType = workItemTypeBug
				} else {
					p.workItemType = workItemTypePBI
				}
			} else if p.focusedField == 5 {
				if len(p.iterations) > 0 {
					p.selectedSprint--
					if p.selectedSprint < 0 {
						p.selectedSprint = len(p.iterations) - 1
					}
				}
			}
			return p, nil

		case key.Matches(msg, p.keys.ModalRight):
			if p.focusedField == 0 {
				if p.workItemType == workItemTypePBI {
					p.workItemType = workItemTypeBug
				} else {
					p.workItemType = workItemTypePBI
				}
			} else if p.focusedField == 5 {
				if len(p.iterations) > 0 {
					p.selectedSprint = (p.selectedSprint + 1) % len(p.iterations)
				}
			}
			return p, nil
		}
	}

	// Update the focused input field
	switch p.focusedField {
	case 1:
		p.titleInput, cmd = p.titleInput.Update(msg)
	case 2:
		p.descriptionArea, cmd = p.descriptionArea.Update(msg)
	case 3:
		p.acceptanceCriteriaArea, cmd = p.acceptanceCriteriaArea.Update(msg)
	case 4:
		p.areaInput, cmd = p.areaInput.Update(msg)
	case 5:
	case 6:
		if p.workItemType == workItemTypePBI {
			p.effortInput, cmd = p.effortInput.Update(msg)
		} else {
			p.tagsInput, cmd = p.tagsInput.Update(msg)
		}
	case 7:
		if p.workItemType == workItemTypePBI {
			p.priorityInput, cmd = p.priorityInput.Update(msg)
		} else {
			p.defectCauseInput, cmd = p.defectCauseInput.Update(msg)
		}
	case 8:
		if p.workItemType == workItemTypePBI {
			p.tagsInput, cmd = p.tagsInput.Update(msg)
		} else {
			p.assigneeInput, cmd = p.assigneeInput.Update(msg)
			p.filterAssignees()
		}
	case 9:
		p.assigneeInput, cmd = p.assigneeInput.Update(msg)
		p.filterAssignees()
	}
	return p, cmd
}

func (p *ParentModal) blurAll() {
	p.titleInput.Blur()
	p.descriptionArea.Blur()
	p.acceptanceCriteriaArea.Blur()
	p.areaInput.Blur()
	p.effortInput.Blur()
	p.priorityInput.Blur()
	p.tagsInput.Blur()
	p.assigneeInput.Blur()
	p.defectCauseInput.Blur()
}

func (p *ParentModal) focusField() {
	switch p.focusedField {
	case 1:
		p.titleInput.Focus()
	case 2:
		p.descriptionArea.Focus()
	case 3:
		p.acceptanceCriteriaArea.Focus()
	case 4:
		p.areaInput.Focus()
	case 5:
	case 6:
		if p.workItemType == workItemTypePBI {
			p.effortInput.Focus()
		} else {
			p.tagsInput.Focus()
		}
	case 7:
		if p.workItemType == workItemTypePBI {
			p.priorityInput.Focus()
		} else {
			p.defectCauseInput.Focus()
		}
	case 8:
		if p.workItemType == workItemTypePBI {
			p.tagsInput.Focus()
		} else {
			p.assigneeInput.Focus()
			p.showAssigneeList = true
			p.filterAssignees()
		}
	case 9:
		p.assigneeInput.Focus()
		p.showAssigneeList = true
		p.filterAssignees()
	}
}

func (p *ParentModal) filterAssignees() {
	filter := strings.ToLower(p.assigneeInput.Value())
	if filter == "" {
		p.filteredMembers = p.members
	} else {
		p.filteredMembers = make([]models.TeamMember, 0)
		for _, member := range p.members {
			if strings.Contains(strings.ToLower(member.DisplayName), filter) ||
				strings.Contains(strings.ToLower(member.UniqueName), filter) {
				p.filteredMembers = append(p.filteredMembers, member)
			}
		}
	}
	// Reset cursor if out of bounds
	if p.assigneeCursor >= len(p.filteredMembers) {
		p.assigneeCursor = 0
	}
	p.showAssigneeList = len(p.filteredMembers) > 0
}

func (p ParentModal) submit() (ParentModal, tea.Cmd) {
	title := strings.TrimSpace(p.titleInput.Value())
	if title == "" {
		p.err = errorMessages["titleRequired"]
		return p, nil
	}

	description := strings.TrimSpace(p.descriptionArea.Value())
	acceptanceCriteria := strings.TrimSpace(p.acceptanceCriteriaArea.Value())

	areaPath := strings.TrimSpace(p.areaInput.Value())

	var iterationPath string
	if p.selectedSprint >= 0 && p.selectedSprint < len(p.iterations) {
		iterationPath = p.iterations[p.selectedSprint].Path
	}

	var effort float64
	effortStr := strings.TrimSpace(p.effortInput.Value())
	if effortStr != "" {
		parsed, err := strconv.ParseFloat(effortStr, 64)
		if err != nil {
			p.err = errorMessages["effortNumber"]
			return p, nil
		}
		effort = parsed
	}

	var priority int
	priorityStr := strings.TrimSpace(p.priorityInput.Value())
	if priorityStr != "" {
		parsed, err := strconv.Atoi(priorityStr)
		if err != nil {
			p.err = errorMessages["priorityNumber"]
			return p, nil
		}
		priority = parsed
	}

	tagsStr := strings.TrimSpace(p.tagsInput.Value())
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

	if effort < 0 || effort > 1000 {
		p.err = errorMessages["effortRange"]
		return p, nil
	}
	if priority < 0 || priority > 100 {
		p.err = errorMessages["priorityRange"]
		return p, nil
	}

	for _, t := range tags {
		if len(t) == 0 {
			p.err = errorMessages["tagsEmpty"]
			return p, nil
		}
		if len(t) > 50 {
			p.err = errorMessages["tagsLength"]
			return p, nil
		}
	}

	var defectCause string
	if p.workItemType == workItemTypeBug {
		defectCause = strings.TrimSpace(p.defectCauseInput.Value())
	}

	return p, func() tea.Msg {
		return ParentCreateRequestMsg{
			WorkItemType:       workItemTypeNames[p.workItemType],
			Title:              title,
			Description:        description,
			AcceptanceCriteria: acceptanceCriteria,
			IterationPath:      iterationPath,
			AreaPath:           areaPath,
			Effort:             effort,
			Priority:           priority,
			Tags:               tags,
			AssignedTo:         p.selectedAssignee,
			DefectCause:        defectCause,
		}
	}
}

// View renders the modal
func (p ParentModal) View() string {
	if !p.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := styles.ModalWidthXL
	modalHeight := styles.ModalHeightXXL

	// Build content
	var b strings.Builder

	title := p.styles.ModalTitle.Render(labelText["modalTitle"])
	b.WriteString(title)
	b.WriteString("\n\n")

	typeLabel := labelText["type"]
	if p.focusedField == 0 {
		typeLabel = p.styles.AccentBold.Render(labelText["type"])
	}
	typeValue := workItemTypeVisual[p.workItemType]
	if p.focusedField == 0 {
		typeValue = "◀ " + p.styles.AccentBold.Render(workItemTypeVisual[p.workItemType]) + " ▶"
	}
	b.WriteString(typeLabel + " " + typeValue)
	b.WriteString("\n\n")

	titleLabel := labelText["title"]
	if p.focusedField == 1 {
		titleLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["title"])
	}
	b.WriteString(titleLabel + "\n")
	b.WriteString(p.titleInput.View())
	b.WriteString("\n\n")

	descLabel := labelText["description"]
	if p.focusedField == 2 {
		descLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["description"])
	}
	b.WriteString(descLabel + "\n")
	b.WriteString(p.descriptionArea.View())
	b.WriteString("\n\n")

	acLabel := labelText["acceptanceCriteria"]
	if p.focusedField == 3 {
		acLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["acceptanceCriteria"])
	}
	b.WriteString(acLabel + "\n")
	b.WriteString(p.acceptanceCriteriaArea.View())
	b.WriteString("\n\n")

	areaLabel := labelText["area"]
	if p.focusedField == 4 {
		areaLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["area"])
	}
	b.WriteString(areaLabel + "\n")
	b.WriteString(p.areaInput.View())
	b.WriteString("\n\n")

	sprintLabel := labelText["sprint"]
	if p.focusedField == 5 {
		sprintLabel = p.styles.AccentBold.Render(labelText["sprint"])
	}
	sprintValue := labelText["none"]
	if p.selectedSprint >= 0 && p.selectedSprint < len(p.iterations) {
		sprintValue = p.iterations[p.selectedSprint].DisplayName()
	}
	if p.focusedField == 5 {
		sprintValue = "◀ " + p.styles.AccentBold.Render(sprintValue) + " ▶"
	}
	b.WriteString(sprintLabel + " " + sprintValue)
	b.WriteString("\n\n")

	if p.workItemType == workItemTypePBI {
		effortLabel := labelText["effort"]
		if p.focusedField == 6 {
			effortLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["effort"])
		}
		b.WriteString(effortLabel + "\n")
		b.WriteString(p.effortInput.View())
		b.WriteString("\n\n")

		priorityLabel := labelText["priority"]
		if p.focusedField == 7 {
			priorityLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["priority"])
		}
		b.WriteString(priorityLabel + "\n")
		b.WriteString(p.priorityInput.View())
		b.WriteString("\n\n")

		tagsLabel := labelText["tags"]
		if p.focusedField == 8 {
			tagsLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["tags"])
		}
		b.WriteString(tagsLabel + "\n")
		b.WriteString(p.tagsInput.View())
		b.WriteString("\n\n")

		assigneeLabel := labelText["assignTo"]
		if p.focusedField == 9 {
			assigneeLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["assignTo"])
		}
		b.WriteString(assigneeLabel + "\n")
		b.WriteString(p.assigneeInput.View())

		// Show assignee list if focused and has results
		if p.focusedField == 9 && p.showAssigneeList && len(p.filteredMembers) > 0 {
			b.WriteString("\n")
			maxVisible := styles.AssigneeListVisibleItems
			end := len(p.filteredMembers)
			if end > maxVisible {
				end = maxVisible
			}

			focusColor := styles.ColorPrimary
			for i := 0; i < end; i++ {
				member := p.filteredMembers[i]
				cursor := "  "
				if i == p.assigneeCursor {
					cursor = "▸ "
				}

				style := p.styles.TextMuted
				if i == p.assigneeCursor {
					style = style.Bold(true).Foreground(focusColor)
				}

				b.WriteString(cursor + style.Render(member.DisplayName) + "\n")
			}

			if len(p.filteredMembers) > maxVisible {
				moreStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
				b.WriteString(moreStyle.Render(fmt.Sprintf("  ... %d more", len(p.filteredMembers)-maxVisible)))
			}
		}
		b.WriteString("\n\n")
	} else {
		tagsLabel := labelText["tags"]
		if p.focusedField == 6 {
			tagsLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["tags"])
		}
		b.WriteString(tagsLabel + "\n")
		b.WriteString(p.tagsInput.View())
		b.WriteString("\n\n")

		defectCauseLabel := labelText["defectCause"]
		if p.focusedField == 7 {
			defectCauseLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["defectCause"])
		}
		b.WriteString(defectCauseLabel + "\n")
		b.WriteString(p.defectCauseInput.View())
		b.WriteString("\n\n")

		assigneeLabel := labelText["assignTo"]
		if p.focusedField == 8 {
			assigneeLabel = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(labelText["assignTo"])
		}
		b.WriteString(assigneeLabel + "\n")
		b.WriteString(p.assigneeInput.View())

		// Show assignee list if focused and has results
		if p.focusedField == 8 && p.showAssigneeList && len(p.filteredMembers) > 0 {
			b.WriteString("\n")
			maxVisible := styles.AssigneeListVisibleItems
			end := len(p.filteredMembers)
			if end > maxVisible {
				end = maxVisible
			}

			focusColor := styles.ColorPrimary
			for i := 0; i < end; i++ {
				member := p.filteredMembers[i]
				cursor := "  "
				if i == p.assigneeCursor {
					cursor = "▸ "
				}

				style := p.styles.TextMuted
				if i == p.assigneeCursor {
					style = style.Bold(true).Foreground(focusColor)
				}

				b.WriteString(cursor + style.Render(member.DisplayName) + "\n")
			}

			if len(p.filteredMembers) > maxVisible {
				moreStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
				b.WriteString(moreStyle.Render(fmt.Sprintf("  ... %d more", len(p.filteredMembers)-maxVisible)))
			}
		}
		b.WriteString("\n\n")
	}

	// Error message
	if p.err != "" {
		errStyle := p.styles.ErrorText
		b.WriteString(errStyle.Render(p.err))
		b.WriteString("\n\n")
	}

	// Instructions (derived from KeyMap) — use modal arrow bindings
	next := p.keys.NextPanel.Help()
	modalLeft := p.keys.ModalLeft.Help()
	modalRight := p.keys.ModalRight.Help()
	save := p.keys.Save.Help()
	back := p.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s • %s/%s: %s • %s: %s • %s: %s", next.Key, next.Desc, modalLeft.Key, modalRight.Key, "change type", save.Key, save.Desc, back.Key, back.Desc)
	b.WriteString(p.styles.ModalInstructions.Render(instr))

	// Style the modal
	content := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Padding(styles.ModalPaddingV, styles.ModalPaddingH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Render(b.String())

	// Center the modal
	return lipgloss.Place(
		p.width,
		p.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// SetSize sets the modal size
func (p *ParentModal) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetVisible sets the modal visibility
func (p *ParentModal) SetVisible(visible bool) {
	p.visible = visible
	if visible {
		p.focusedField = 1
		p.blurAll()
		p.titleInput.Focus()
		p.titleInput.SetValue("")
		p.descriptionArea.SetValue("")
		p.acceptanceCriteriaArea.SetValue("")
		p.effortInput.SetValue("")
		p.priorityInput.SetValue("")
		p.tagsInput.SetValue("")
		p.assigneeInput.SetValue("")
		p.defectCauseInput.SetValue("")
		p.selectedAssignee = ""
		p.showAssigneeList = false
		p.err = ""
	} else {
		p.blurAll()
	}
}

// IsVisible returns whether the modal is visible
func (p *ParentModal) IsVisible() bool {
	return p.visible
}

// SetData sets the context data for parent creation
func (p *ParentModal) SetData(iterations []models.Iteration, areas []models.Area) {
	p.iterations = iterations
	// Set default sprint to current iteration
	p.selectedSprint = 0
	for i, iter := range iterations {
		if iter.IsCurrent() {
			p.selectedSprint = i
			break
		}
	}
}

// SetDefaultArea sets the default area path (from first visible work item)
func (p *ParentModal) SetDefaultArea(areaPath string) {
	p.areaInput.SetValue(areaPath)
}

// SetMembers sets the available team members for assignment
func (p *ParentModal) SetMembers(members []models.TeamMember) {
	p.members = members
	p.filteredMembers = members
}

// ParentCreateRequestMsg is sent when creating a new parent work item
type ParentCreateRequestMsg struct {
	WorkItemType       string
	Title              string
	Description        string
	AcceptanceCriteria string
	IterationPath      string
	AreaPath           string
	Effort             float64
	Priority           int
	Tags               []string
	AssignedTo         string
	DefectCause        string
}

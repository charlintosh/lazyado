package modals

import (
	"fmt"
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

// AssignModal is a modal for assigning work items to users
type AssignModal struct {
	visible       bool
	item          *models.WorkItem
	members       []models.TeamMember
	filtered      []models.TeamMember
	cursor        int
	styles        styles.Styles
	keys          keys.KeyMap
	width         int
	height        int
	filterInput   textinput.Model
	filterEnabled bool
}

// NewAssignModal creates a new assign modal
func NewAssignModal(st styles.Styles, keys keys.KeyMap) AssignModal {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = styles.InputCharLimitMD
	ti.Width = styles.TextInputWidthMD

	return AssignModal{
		styles:      st,
		keys:        keys,
		filterInput: ti,
	}
}

// Init initializes the modal
func (m AssignModal) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m AssignModal) Update(msg tea.Msg) (AssignModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter input
		if m.filterEnabled {
			switch {
			case key.Matches(msg, m.keys.Back):
				if m.filterInput.Value() != "" {
					m.filterInput.SetValue("")
					m.applyFilter()
					return m, nil
				}
				m.visible = false
				return m, func() tea.Msg { return ModalClosedMsg{} }
			case key.Matches(msg, m.keys.Select):
				if len(m.filtered) > 0 && m.item != nil {
					selected := m.filtered[m.cursor]
					return m, func() tea.Msg {
						return AssignRequestMsg{
							Item:      *m.item,
							UserEmail: selected.UniqueName,
							UserName:  selected.DisplayName,
						}
					}
				}
			case key.Matches(msg, m.keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Select):
			if m.item != nil && m.cursor < len(m.filtered) {
				selected := m.filtered[m.cursor]
				return m, func() tea.Msg {
					return AssignRequestMsg{
						Item:      *m.item,
						UserEmail: selected.UniqueName,
						UserName:  selected.DisplayName,
					}
				}
			}
		case key.Matches(msg, m.keys.Back):
			m.visible = false
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case key.Matches(msg, m.keys.Search):
			m.filterEnabled = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}
	}

	return m, nil
}

func (m *AssignModal) applyFilter() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filtered = m.members
	} else {
		m.filtered = make([]models.TeamMember, 0)
		for _, member := range m.members {
			if strings.Contains(strings.ToLower(member.DisplayName), filter) ||
				strings.Contains(strings.ToLower(member.UniqueName), filter) {
				m.filtered = append(m.filtered, member)
			}
		}
	}
	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

// View renders the modal
func (m AssignModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := styles.ModalWidthMD
	visibleItems := styles.ModalListVisibleItems
	modalHeight := visibleItems + styles.AssignModalBaseHeight

	// Build content
	var b strings.Builder

	// Title
	title := m.styles.ModalTitle.Render("Assign To")
	if m.item != nil {
		itemInfo := m.styles.TextMuted.Render("#" + components.Itoa(m.item.ID) + " " + components.TruncateStr(m.item.Title, 35))
		b.WriteString(title + "\n")
		b.WriteString(itemInfo + "\n\n")
	}

	// Current assignee
	if m.item != nil {
		currentStyle := m.styles.Accent
		assigned := m.item.AssignedTo
		if assigned == "" {
			assigned = "Unassigned"
		}
		b.WriteString(currentStyle.Render("Current: "+assigned) + "\n\n")
	}

	// Filter input
	if m.filterEnabled {
		b.WriteString(m.filterInput.View() + "\n\n")
	} else {
		filterHint := m.styles.TextMuted.Render("Press / to filter")
		b.WriteString(filterHint + "\n\n")
	}

	// Unassign option first
	unassignCursor := "  "
	if m.cursor == 0 && len(m.filtered) == 0 {
		unassignCursor = "▸ "
	}

	// Member options
	if len(m.filtered) == 0 {
		b.WriteString(m.styles.TextMuted.Render("  No members found") + "\n")
	} else {
		// Calculate scroll offset
		offset := 0
		if m.cursor >= visibleItems {
			offset = m.cursor - visibleItems + 1
		}

		end := offset + visibleItems
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := offset; i < end; i++ {
			member := m.filtered[i]
			cursor := "  "
			if i == m.cursor {
				cursor = "▸ "
			}

			style := m.styles.TextMuted
			if i == m.cursor {
				style = m.styles.AccentBold
			}

			// Highlight if this is the current assignee
			if m.item != nil && member.DisplayName == m.item.AssignedTo {
				style = style.Foreground(styles.ColorSuccess)
			}

			// Show name, truncate if needed
			name := components.TruncateStr(member.DisplayName, modalWidth-styles.ModalListItemOffset)
			b.WriteString(cursor + style.Render(name) + "\n")
		}

		// Show scroll indicator
		if len(m.filtered) > visibleItems {
			scrollInfo := m.styles.TextMuted.Render("  (" + components.Itoa(m.cursor+1) + "/" + components.Itoa(len(m.filtered)) + ")")
			b.WriteString(scrollInfo + "\n")
		}
	}

	_ = unassignCursor // Reserved for future unassign option

	// Help text (derived from KeyMap)
	b.WriteString("\n")
	sel := m.keys.Select.Help()
	back := m.keys.Back.Help()
	search := m.keys.Search.Help()
	if m.filterEnabled {
		instr := fmt.Sprintf("%s: %s  %s: %s", sel.Key, sel.Desc, back.Key, back.Desc)
		b.WriteString(m.styles.ModalInstructions.Render(instr))
	} else {
		instr := fmt.Sprintf("%s: %s  %s: %s  %s: %s", sel.Key, sel.Desc, search.Key, search.Desc, back.Key, back.Desc)
		b.WriteString(m.styles.ModalInstructions.Render(instr))
	}

	// Modal style
	modal := m.styles.ModalBox.Width(modalWidth).Height(modalHeight).Render(b.String())

	// Center the modal
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// SetVisible sets the visibility
func (m *AssignModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.cursor = 0
		m.filterInput.SetValue("")
		m.filterEnabled = false
		m.filterInput.Blur()
		m.applyFilter()
		// Try to set cursor to current assignee
		if m.item != nil && m.item.AssignedTo != "" {
			for i, member := range m.filtered {
				if member.DisplayName == m.item.AssignedTo {
					m.cursor = i
					break
				}
			}
		}
	}
}

// IsVisible returns whether the modal is visible
func (m *AssignModal) IsVisible() bool {
	return m.visible
}

// SetItem sets the work item to modify
func (m *AssignModal) SetItem(item *models.WorkItem) {
	m.item = item
}

// SetMembers sets the available team members
func (m *AssignModal) SetMembers(members []models.TeamMember) {
	m.members = members
	m.applyFilter()
}

// SetSize sets the modal container size
func (m *AssignModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// AssignRequestMsg is sent when user confirms assignment
type AssignRequestMsg struct {
	Item      models.WorkItem
	UserEmail string
	UserName  string
}

// AssignedMsg is sent when assignment is complete
type AssignedMsg struct {
	Item models.WorkItem
}

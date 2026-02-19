package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var commentModalLogger = debug.Scope("comment_modal")

// CommentModal is a modal for adding and editing comments to work items
type CommentModal struct {
	visible            bool
	editMode           bool
	item               *models.WorkItem
	comment            *models.Comment
	commentArea        textarea.Model
	teamMembers        []models.TeamMember
	styles             styles.Styles
	keys               keys.KeyMap
	width              int
	height             int
	err                string
	showingSuggestions bool
	suggestions        []models.TeamMember
	selectedSuggestion int
	mentionStart       int // Position where @ was typed
}

// NewCommentModal creates a new comment modal
func NewCommentModal(st styles.Styles, keys keys.KeyMap) CommentModal {
	ta := textarea.New()
	ta.Placeholder = "Write a comment... (Use @name to mention someone)"
	ta.CharLimit = styles.ContentCharLimitLG
	ta.SetWidth(styles.TextareaWidthLG)
	ta.SetHeight(styles.TextareaHeightLG)
	ta.ShowLineNumbers = false
	ta.Focus()

	return CommentModal{
		commentArea:        ta,
		styles:             st,
		keys:               keys,
		selectedSuggestion: 0,
		mentionStart:       -1,
	}
}

// Init initializes the modal
func (m CommentModal) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages
func (m CommentModal) Update(msg tea.Msg) (CommentModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle suggestions navigation
		if m.showingSuggestions {
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.selectedSuggestion > 0 {
					m.selectedSuggestion--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				if m.selectedSuggestion < len(m.suggestions)-1 {
					m.selectedSuggestion++
				}
				return m, nil
			case key.Matches(msg, m.keys.Select) || key.Matches(msg, m.keys.NextPanel):
				// Insert selected suggestion (Enter or Tab)
				if m.selectedSuggestion < len(m.suggestions) {
					m.insertMention(m.suggestions[m.selectedSuggestion])
				}
				return m, nil
			case key.Matches(msg, m.keys.Back):
				// Close suggestions (Esc / Back)
				m.showingSuggestions = false
				m.suggestions = nil
				m.selectedSuggestion = 0
				m.mentionStart = -1
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Back) && !m.showingSuggestions:
			m.visible = false
			m.err = ""
			m.commentArea.Reset()
			m.showingSuggestions = false
			m.suggestions = nil
			return m, func() tea.Msg { return ModalClosedMsg{} }
		case key.Matches(msg, m.keys.Save):
			// Submit comment with Save binding (Ctrl+S)
			text := strings.TrimSpace(m.commentArea.Value())
			if text == "" {
				m.err = "Comment cannot be empty"
				return m, nil
			}

			if m.item == nil {
				m.err = "No work item selected"
				return m, nil
			}

			// DEBUG: Log original text
			commentModalLogger.Debugf("original text=%q", text)
			commentModalLogger.Debugf("team_members_count=%d", len(m.teamMembers))
			if len(m.teamMembers) > 0 {
				commentModalLogger.Debugf("first_member=%q id=%s", m.teamMembers[0].DisplayName, m.teamMembers[0].ID)
			}

			// Convert @username mentions to @<ID> format
			processedText := m.processMentions(text)

			// DEBUG: Log processed text
			commentModalLogger.Debugf("processed_text=%q", processedText)

			m.visible = false
			m.commentArea.Reset()
			m.err = ""
			m.showingSuggestions = false
			m.suggestions = nil

			if m.editMode && m.comment != nil {
				// Edit mode
				commentModalLogger.Debugf("saving comment_id=%d text_length=%d", m.comment.ID, len(processedText))
				return m, func() tea.Msg {
					return CommentUpdatedMsg{
						WorkItemID: m.item.ID,
						CommentID:  m.comment.ID,
						Text:       processedText,
					}
				}
			} else {
				// Create mode
				return m, func() tea.Msg {
					return CommentCreateRequestMsg{
						WorkItemID: m.item.ID,
						Text:       processedText,
					}
				}
			}
		}
	}

	// Update textarea
	m.commentArea, cmd = m.commentArea.Update(msg)

	// Check for @ trigger after textarea update
	m.updateSuggestions()

	return m, cmd
}

// View renders the modal
func (m CommentModal) View() string {
	if !m.visible {
		return ""
	}

	var title string
	if m.editMode {
		title = "Edit Comment"
	} else {
		title = "Add Comment"
		if m.item != nil {
			title = fmt.Sprintf("Add Comment to #%d - %s", m.item.ID, m.item.Title)
		}
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorTesting).
		Padding(0, 1)

	// Comment area with border
	commentView := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(styles.ModalPaddingV).
		Width(m.commentArea.Width() + styles.PanelPaddedOffset).
		Render(m.commentArea.View())

	// Show suggestions if active
	suggestionsView := ""
	if m.showingSuggestions && len(m.suggestions) > 0 {
		suggestionsView = m.renderSuggestions()
	}

	// Available mentions (show first 5 team members as hint)
	mentionsHint := ""
	if len(m.teamMembers) > 0 && !m.showingSuggestions {
		mentionsHint = "\n" + lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Render("💡 Tip: Type @ to see available users")
	}

	// Error message
	errorView := ""
	if m.err != "" {
		errorView = "\n" + lipgloss.NewStyle().
			Foreground(styles.ColorError).
			Render("⚠ "+m.err)
	}

	// Instructions (derived from KeyMap)
	save := m.keys.Save.Help()
	back := m.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s  %s: %s", save.Key, save.Desc, back.Key, back.Desc)
	instructions := m.styles.ModalInstructions.Render(instr)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		"",
		commentView,
		suggestionsView,
		mentionsHint,
		errorView,
		"",
		instructions,
	)

	// Center modal
	modalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(styles.ModalPaddingV, styles.ModalPaddingH).
		Width(styles.ModalWidthXL)

	modal := modalStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)
}

// SetItem sets the work item for creating a new comment
func (m *CommentModal) SetItem(item *models.WorkItem) {
	m.item = item
	m.comment = nil
	m.editMode = false
	m.commentArea.Reset()
	m.err = ""
	m.showingSuggestions = false
	m.suggestions = nil
	m.selectedSuggestion = 0
	m.mentionStart = -1
}

// SetComment sets the comment for editing
func (m *CommentModal) SetComment(comment *models.Comment, workItemID int) {
	m.comment = comment
	m.item = &models.WorkItem{ID: workItemID}
	m.editMode = true
	m.commentArea.SetValue(comment.Text)
	m.err = ""
	m.showingSuggestions = false
	m.suggestions = nil
	m.selectedSuggestion = 0
	m.mentionStart = -1
	commentModalLogger.Debugf("set_comment comment_id=%d work_item_id=%d", comment.ID, workItemID)
}

// SetMembers sets the team members for @mentions
func (m *CommentModal) SetMembers(members []models.TeamMember) {
	m.teamMembers = members
}

// SetSize sets the modal dimensions
func (m *CommentModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetVisible sets the modal visibility
func (m *CommentModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.commentArea.Focus()
	} else {
		m.commentArea.Blur()
		m.commentArea.Reset()
		m.err = ""
		m.showingSuggestions = false
		m.suggestions = nil
		m.selectedSuggestion = 0
		m.mentionStart = -1
	}
}

// IsVisible returns whether the modal is visible
func (m *CommentModal) IsVisible() bool {
	return m.visible
}

// processMentions converts @username mentions to Azure DevOps HTML format
func (m *CommentModal) processMentions(text string) string {
	commentModalLogger.Debugf("process_mentions input=%q", text)

	if len(m.teamMembers) == 0 {
		commentModalLogger.Debugf("no_team_members")
		return text
	}

	// Sort members by name length (longest first) to avoid partial replacements
	members := make([]models.TeamMember, len(m.teamMembers))
	copy(members, m.teamMembers)

	// Sort by display name length (descending) to replace longer names first
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			if len(members[j].DisplayName) > len(members[i].DisplayName) {
				members[i], members[j] = members[j], members[i]
			}
		}
	}

	commentModalLogger.Debugf("processing_members=%d", len(members))

	// Process each member's mentions
	for _, member := range members {
		displayName := member.DisplayName
		mention := "@" + displayName
		// Azure DevOps format: HTML anchor with data-vss-mention attribute
		// Don't include @ inside the tag to avoid re-matching
		azureMention := fmt.Sprintf(`<a href="#" data-vss-mention="version:2.0,%s">%s</a>`, member.ID, displayName)
		// Simple replacement: look for exact match
		for {
			idx := strings.Index(text, mention)
			if idx == -1 {
				break
			}

			commentModalLogger.Debugf("found_mention_index=%d", idx)

			// Check if it's a complete mention (followed by space, newline, punctuation, or end)
			endIdx := idx + len(mention)
			isComplete := false
			if endIdx >= len(text) {
				isComplete = true
				commentModalLogger.Debugf("mention_at_end_complete")
			} else {
				nextChar := text[endIdx]
				if nextChar == ' ' || nextChar == '\n' || nextChar == ',' || nextChar == '.' || nextChar == '!' || nextChar == '?' || nextChar == ':' || nextChar == ';' {
					isComplete = true
					commentModalLogger.Debugf("mention_followed_by_delimiter=%q - complete", nextChar)
				} else {
					commentModalLogger.Debugf("mention_followed_by=%q - not_complete", nextChar)
				}
			}

			if isComplete {
				// Replace with Azure DevOps HTML format
				commentModalLogger.Debugf("replacing=%q with=%q", mention, azureMention)
				text = text[:idx] + azureMention + text[endIdx:]
			} else {
				// Not complete, mark this @ to skip it in next iteration
				// by temporarily replacing it
				text = text[:idx] + "@@SKIP@@" + text[idx+1:]
			}
		}
	}

	// Restore skipped @
	text = strings.ReplaceAll(text, "@@SKIP@@", "@")

	commentModalLogger.Debugf("output_text=%q", text)
	return text
}

// updateSuggestions checks if @ was typed and updates suggestions
func (m *CommentModal) updateSuggestions() {
	text := m.commentArea.Value()

	// Since we can't access cursor position directly from textarea,
	// we'll look for @ at the end of the text or before cursor
	// This is a simpler approach - check if text ends with @ or @partial_name

	// Find the last @ in the text
	lastAtIndex := strings.LastIndex(text, "@")
	if lastAtIndex == -1 {
		m.showingSuggestions = false
		m.suggestions = nil
		m.mentionStart = -1
		return
	}

	// Check if @ is at start or after whitespace
	if lastAtIndex > 0 {
		prevChar := text[lastAtIndex-1]
		if prevChar != ' ' && prevChar != '\n' {
			m.showingSuggestions = false
			m.suggestions = nil
			m.mentionStart = -1
			return
		}
	}

	// Extract text after the @
	searchTerm := ""
	afterAt := text[lastAtIndex+1:]
	// Find first whitespace or newline after @
	spaceIdx := strings.IndexAny(afterAt, " \n")
	if spaceIdx >= 0 {
		searchTerm = afterAt[:spaceIdx]
		// If there's whitespace after @, we're not in mention mode anymore
		m.showingSuggestions = false
		m.suggestions = nil
		m.mentionStart = -1
		return
	} else {
		searchTerm = afterAt
	}

	// Filter team members by search term
	m.suggestions = []models.TeamMember{}
	searchLower := strings.ToLower(searchTerm)
	for _, member := range m.teamMembers {
		nameLower := strings.ToLower(member.DisplayName)
		if strings.Contains(nameLower, searchLower) {
			m.suggestions = append(m.suggestions, member)
			if len(m.suggestions) >= 8 { // Limit to 8 suggestions
				break
			}
		}
	}

	if len(m.suggestions) > 0 {
		m.showingSuggestions = true
		m.mentionStart = lastAtIndex
		// Reset selection if out of bounds
		if m.selectedSuggestion >= len(m.suggestions) {
			m.selectedSuggestion = 0
		}
	} else {
		m.showingSuggestions = false
		m.mentionStart = -1
	}
}

// insertMention inserts the selected team member into the text
func (m *CommentModal) insertMention(member models.TeamMember) {
	text := m.commentArea.Value()

	// Find the last @ in the text (should be our mention start)
	lastAtIndex := strings.LastIndex(text, "@")
	if lastAtIndex == -1 {
		return
	}

	// Find where the partial name ends (end of text or first whitespace after @)
	afterAt := text[lastAtIndex+1:]
	endIdx := len(text) // Default to end of text
	spaceIdx := strings.IndexAny(afterAt, " \n")
	if spaceIdx >= 0 {
		endIdx = lastAtIndex + 1 + spaceIdx
	}

	// Replace from @ to end with the mention
	newText := text[:lastAtIndex] + "@" + member.DisplayName + " " + text[endIdx:]
	m.commentArea.SetValue(newText)

	// Hide suggestions
	m.showingSuggestions = false
	m.suggestions = nil
	m.selectedSuggestion = 0
	m.mentionStart = -1
}

// renderSuggestions renders the user suggestions list
func (m *CommentModal) renderSuggestions() string {
	if len(m.suggestions) == 0 {
		return ""
	}

	var items []string
	for i, member := range m.suggestions {
		style := lipgloss.NewStyle().
			Padding(0, 1)

		if i == m.selectedSuggestion {
			style = style.
				Background(styles.ColorPrimary).
				Foreground(styles.ColorText).
				Bold(true)
		} else {
			style = style.
				Foreground(styles.ColorTextMuted)
		}

		items = append(items, style.Render(member.DisplayName))
	}

	suggestionsBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(0, 1).
		Render(strings.Join(items, "\n"))

	up := m.keys.Up.Help()
	down := m.keys.Down.Help()
	sel := m.keys.Select.Help()
	next := m.keys.NextPanel.Help()
	back := m.keys.Back.Help()
	header := m.styles.ModalInstructions.Render(fmt.Sprintf("%s/%s: Navigate  •  %s/%s: Select  •  %s: Cancel", up.Key, down.Key, sel.Key, next.Key, back.Key))

	return "\n" + lipgloss.JoinVertical(lipgloss.Left, suggestionsBox, header)
}

// Message types

// CommentCreateRequestMsg is sent when the user wants to create a comment
type CommentCreateRequestMsg struct {
	WorkItemID int
	Text       string
}

// CommentUpdatedMsg is sent when the user wants to update a comment
type CommentUpdatedMsg struct {
	WorkItemID int
	CommentID  int
	Text       string
}

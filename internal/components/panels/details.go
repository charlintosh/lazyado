package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/glamour"
)

// DetailsPanel shows details for a selected work item
type DetailsPanel struct {
	item              *models.WorkItem
	styles            styles.Styles
	keys              keys.KeyMap
	width             int
	height            int
	renderedDesc      string
	renderedDescWidth int
	renderedAC        string
	renderedACWidth   int
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(styles styles.Styles, keys keys.KeyMap) DetailsPanel {
	return DetailsPanel{
		styles: styles,
		keys:   keys,
	}
}

// View renders the details panel
func (d DetailsPanel) View() string {
	if d.item == nil {
		content := d.styles.Subtitle.Render("Select a work item to view details")
		return d.styles.PanelInactive.
			Width(d.width).
			Height(d.height).
			Render(content)
	}

	var b strings.Builder

	// Title
	title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
	if len(title) > d.width-6 {
		title = title[:d.width-9] + "..."
	}
	b.WriteString(d.styles.DetailTitle.Render(title))
	b.WriteString("\n\n")

	// Metadata section with aligned columns
	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	labelWidth := 10
	valueWidth := 12

	// Type and State row
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Type:"))
	b.WriteString(typeStyle.Width(valueWidth).Render(d.item.ShortType()))
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("State:"))
	b.WriteString(stateStyle.Render(string(d.item.State)))
	b.WriteString("\n")

	// Assigned and Sprint row
	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Assigned:"))
	b.WriteString(d.styles.DetailValue.Width(valueWidth).Render(truncate(assignedTo, valueWidth)))
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Sprint:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.SprintName()))
	b.WriteString("\n")

	// Area row
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Area:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.AreaName()))
	b.WriteString("\n")

	// Priority and Effort row (for parent types)
	if d.item.IsParentType() {
		if d.item.Priority > 0 || d.item.Effort > 0 {
			if d.item.Priority > 0 {
				b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Priority:"))
				b.WriteString(d.styles.DetailValue.Width(valueWidth).Render(fmt.Sprintf("%d", d.item.Priority)))
			}
			if d.item.Effort > 0 {
				b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Effort:"))
				b.WriteString(d.styles.DetailValue.Render(fmt.Sprintf("%.1f", d.item.Effort)))
			}
			b.WriteString("\n")
		}
	}

	// Parent (if exists)
	if d.item.ParentID > 0 {
		b.WriteString("\n")
		parentLabel := fmt.Sprintf("Parent: #%d", d.item.ParentID)
		if d.item.ParentTitle != "" {
			parentLabel += " " + d.item.ParentTitle
		}
		b.WriteString(d.styles.Subtitle.Render(parentLabel))
		b.WriteString("\n")
	}

	// Description section
	if d.item.Description != "" {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Description ───"))
		b.WriteString("\n")

		maxWidth := d.width - 8
		if maxWidth < 20 {
			maxWidth = 20
		}

		// Use cached rendered description if available and width matches
		desc := d.renderedDesc
		if desc == "" || d.renderedDescWidth != maxWidth {
			desc = d.renderMarkdown(d.item.Description, maxWidth)
		}

		// Limit description height - leave room for AC and tags if present
		lines := strings.Split(desc, "\n")
		availableLines := d.height - 15 // Account for header metadata
		if d.item.AcceptanceCriteria != "" {
			availableLines = availableLines / 2 // Split space between desc and AC
		}
		if availableLines < 5 {
			availableLines = 5
		}
		if len(lines) > availableLines {
			lines = lines[:availableLines]
			lines = append(lines, "...")
		}

		b.WriteString(strings.Join(lines, "\n"))
		b.WriteString("\n")
	}

	// Acceptance Criteria section
	if d.item.AcceptanceCriteria != "" {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Acceptance Criteria ───"))
		b.WriteString("\n")

		maxWidth := d.width - 8
		if maxWidth < 20 {
			maxWidth = 20
		}

		// Use cached rendered acceptance criteria if available and width matches
		ac := d.renderedAC
		if ac == "" || d.renderedACWidth != maxWidth {
			ac = d.renderAcceptanceCriteria(d.item.AcceptanceCriteria, maxWidth)
		}

		// Limit acceptance criteria height
		lines := strings.Split(ac, "\n")
		availableLines := d.height - 15
		if d.item.Description != "" {
			availableLines = availableLines / 2
		}
		if availableLines < 5 {
			availableLines = 5
		}
		if len(lines) > availableLines {
			lines = lines[:availableLines]
			lines = append(lines, "...")
		}

		b.WriteString(strings.Join(lines, "\n"))
		b.WriteString("\n")
	}

	// Tags section
	if len(d.item.Tags) > 0 {
		b.WriteString("\n")
		b.WriteString(d.styles.DetailSectionTitle.Render("─── Tags ───"))
		b.WriteString("\n")

		var tagStrings []string
		for _, tag := range d.item.Tags {
			tagStrings = append(tagStrings, d.styles.DetailTag.Render(tag))
		}
		b.WriteString(strings.Join(tagStrings, " "))
	}

	content := b.String()
	return d.styles.PanelInactive.
		Width(d.width).
		Height(d.height).
		BorderForeground(styles.ColorBorder).
		Render(content)
}

// GetAvailableActions returns a list of available actions for the current item
func (d *DetailsPanel) GetAvailableActions() []string {
	if d.item == nil {
		return []string{}
	}

	keyStyle := d.styles.AccentBold
	descStyle := d.styles.TextMuted

	actions := []string{
		keyStyle.Render("v") + descStyle.Render(" - View fullscreen details"),
		keyStyle.Render("s") + descStyle.Render(" - Change state"),
		keyStyle.Render("a") + descStyle.Render(" - Assign to user"),
		keyStyle.Render("n") + descStyle.Render(" - Create parent item"),
		keyStyle.Render("e") + descStyle.Render(" - Edit item"),
		keyStyle.Render("d") + descStyle.Render(" - Delete item"),
		keyStyle.Render("c") + descStyle.Render(" - Add comment"),
	}

	// Add create task action only for parent types
	if d.item.IsParentType() {
		actions = append(actions[:5], append([]string{
			keyStyle.Render("t") + descStyle.Render(" - Create child task"),
		}, actions[5:]...)...)
	}

	// Add expand/collapse actions
	if d.item.HasChildren() {
		if d.item.IsExpanded {
			actions = append(actions, keyStyle.Render("←/h")+descStyle.Render(" - Collapse children"))
		} else {
			actions = append(actions, keyStyle.Render("→/l")+descStyle.Render(" - Expand children"))
		}
	}

	return actions
}

// SetItem sets the work item to display
func (d *DetailsPanel) SetItem(item *models.WorkItem) {
	d.item = item
	d.renderedDesc = ""
	d.renderedDescWidth = 0
	d.renderedAC = ""
	d.renderedACWidth = 0
}

// renderMarkdown renders markdown content and caches the result
func (d *DetailsPanel) renderMarkdown(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)

	var result string
	if err == nil {
		rendered, renderErr := renderer.Render(content)
		if renderErr == nil {
			result = strings.TrimSpace(rendered)
		} else {
			result = wordWrap(content, width)
		}
	} else {
		result = wordWrap(content, width)
	}

	d.renderedDesc = result
	d.renderedDescWidth = width
	return result
}

// renderAcceptanceCriteria renders acceptance criteria markdown content and caches the result
func (d *DetailsPanel) renderAcceptanceCriteria(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)

	var result string
	if err == nil {
		rendered, renderErr := renderer.Render(content)
		if renderErr == nil {
			result = strings.TrimSpace(rendered)
		} else {
			result = wordWrap(content, width)
		}
	} else {
		result = wordWrap(content, width)
	}

	d.renderedAC = result
	d.renderedACWidth = width
	return result
}

// SetSize sets the size of the details panel
func (d *DetailsPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	currentLineLength := 0

	for i, word := range words {
		if currentLineLength+len(word)+1 > width {
			result.WriteString("\n")
			currentLineLength = 0
		} else if i > 0 {
			result.WriteString(" ")
			currentLineLength++
		}
		result.WriteString(word)
		currentLineLength += len(word)
	}

	return result.String()
}

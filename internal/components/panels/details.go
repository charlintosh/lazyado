package panels

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// DetailsPanel shows details for a selected work item
type DetailsPanel struct {
	item     *models.WorkItem
	viewport viewport.Model
	styles   styles.Styles
	keys     keys.KeyMap
	width    int
	height   int
	ready    bool
	focused  bool
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(st styles.Styles, k keys.KeyMap) DetailsPanel {
	return DetailsPanel{
		viewport: viewport.New(0, 0),
		styles:   st,
		keys:     k,
	}
}

// View renders the details panel
func (d DetailsPanel) View() string {
	panelStyle := d.styles.PanelInactive
	if d.focused {
		panelStyle = d.styles.PanelActive
	}

	var b strings.Builder

	numberStyle := d.styles.TextMuted
	if d.focused {
		numberStyle = d.styles.AccentBold
	}
	titleStyle := d.styles.FilterGroupTitle
	if d.focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	b.WriteString("  " + numberStyle.Render("6. ") + titleStyle.Render("Details"))
	b.WriteString("\n\n")

	if d.item == nil {
		b.WriteString(d.styles.Subtitle.Render("Select a work item to view details"))
		return panelStyle.
			Width(d.width).
			Height(d.height).
			Render(b.String())
	}

	if d.ready {
		b.WriteString(d.viewport.View())
		canScroll := d.viewport.TotalLineCount() > d.viewport.VisibleLineCount()
		if canScroll {
			pct := int(d.viewport.ScrollPercent() * 100)
			scrollHint := lipgloss.NewStyle().Foreground(styles.ColorMuted).
				Render(fmt.Sprintf(" %d%% ", pct))
			b.WriteString("\n" + scrollHint)
		}
	}

	return panelStyle.
		Width(d.width).
		Height(d.height).
		Render(b.String())
}

// SetItem sets the work item to display
func (d *DetailsPanel) SetItem(item *models.WorkItem) {
	d.item = item
	d.viewport.GotoTop()
	d.updateViewportContent()
}

func (d DetailsPanel) Update(msg tea.Msg) (DetailsPanel, tea.Cmd) {
	if !d.focused || !d.ready {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keys.Top):
			d.viewport.GotoTop()
			return d, nil
		case key.Matches(msg, d.keys.Bottom):
			d.viewport.GotoBottom()
			return d, nil
		}
	}

	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *DetailsPanel) SetFocused(focused bool) {
	d.focused = focused
}

// SetSize sets the size of the details panel and rebuilds viewport content.
func (d *DetailsPanel) SetSize(width, height int) {
	if d.width == width && d.height == height && d.ready {
		return
	}
	d.width = width
	d.height = height
	vpWidth := d.width - styles.PanelBorderOffset
	if vpWidth < styles.FilterMinViewportWidth {
		vpWidth = styles.FilterMinViewportWidth
	}
	d.viewport.Width = vpWidth
	d.viewport.Height = d.height - 3
	if d.viewport.Height < 1 {
		d.viewport.Height = 1
	}
	if d.item != nil {
		d.updateViewportContent()
	}
}

// updateViewportContent builds the full content string and loads it into the viewport.
func (d *DetailsPanel) updateViewportContent() {
	if d.item == nil || d.viewport.Width <= 0 {
		return
	}
	wrapWidth := d.viewport.Width - 2
	if wrapWidth < 10 {
		wrapWidth = 10
	}
	d.viewport.SetContent(d.buildContent(wrapWidth))
	d.ready = true
}

// buildContent assembles all sections into a single string for the viewport.
func (d *DetailsPanel) buildContent(wrapWidth int) string {
	var b strings.Builder
	d.writeHeader(&b, wrapWidth)
	d.writeMetadata(&b)
	d.writeDescriptionSection(&b, wrapWidth)
	d.writeACSection(&b, wrapWidth)
	d.writeTagsSection(&b)
	return b.String()
}

// writeHeader writes the item title line.
func (d *DetailsPanel) writeHeader(b *strings.Builder, wrapWidth int) {
	title := fmt.Sprintf("#%d %s", d.item.ID, d.item.Title)
	if wrapWidth > 3 && len(title) > wrapWidth {
		title = title[:wrapWidth-3] + "..."
	}
	b.WriteString(d.styles.DetailTitle.Render(title))
	b.WriteString("\n\n")
}

// writeMetadata writes type/state/assigned/sprint/area/priority/effort/parent rows.
func (d *DetailsPanel) writeMetadata(b *strings.Builder) {
	labelWidth := styles.DetailMetaLabelWidth
	valueWidth := styles.DetailMetaValueWidth

	typeStyle := d.styles.TypeBadge(string(d.item.Type))
	stateStyle := d.styles.StateBadge(string(d.item.State))

	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Type:"))
	b.WriteString(typeStyle.Width(valueWidth).Render(d.item.ShortType()))
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("State:"))
	b.WriteString(stateStyle.Render(string(d.item.State)))
	b.WriteString("\n")

	assignedTo := d.item.AssignedTo
	if assignedTo == "" {
		assignedTo = "Unassigned"
	}
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Assigned:"))
	b.WriteString(d.styles.DetailValue.Width(valueWidth).Render(truncateStr(assignedTo, valueWidth)))
	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Sprint:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.SprintName()))
	b.WriteString("\n")

	b.WriteString(d.styles.DetailLabel.Width(labelWidth).Render("Area:"))
	b.WriteString(d.styles.DetailValue.Render(d.item.AreaName()))
	b.WriteString("\n")

	d.writePriorityEffort(b, labelWidth, valueWidth)
	d.writeParent(b)
}

// writePriorityEffort writes priority/effort rows only for parent work item types.
func (d *DetailsPanel) writePriorityEffort(b *strings.Builder, labelWidth, valueWidth int) {
	if !d.item.IsParentType() {
		return
	}
	if d.item.Priority <= 0 && d.item.Effort <= 0 {
		return
	}
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

// writeParent writes the parent reference line when one exists.
func (d *DetailsPanel) writeParent(b *strings.Builder) {
	if d.item.ParentID <= 0 {
		return
	}
	parentLabel := fmt.Sprintf("Parent: #%d", d.item.ParentID)
	if d.item.ParentTitle != "" {
		parentLabel += " " + d.item.ParentTitle
	}
	b.WriteString("\n")
	b.WriteString(d.styles.Subtitle.Render(parentLabel))
	b.WriteString("\n")
}

// writeDescriptionSection writes the description block when present.
func (d *DetailsPanel) writeDescriptionSection(b *strings.Builder, wrapWidth int) {
	if d.item.Description == "" {
		return
	}
	b.WriteString("\n")
	b.WriteString(d.styles.DetailSectionTitle.Render("─── Description ───"))
	b.WriteString("\n")
	b.WriteString(renderDetailMarkdown(d.item.Description, wrapWidth))
	b.WriteString("\n")
}

// writeACSection writes the acceptance criteria block when present.
func (d *DetailsPanel) writeACSection(b *strings.Builder, wrapWidth int) {
	if d.item.AcceptanceCriteria == "" {
		return
	}
	b.WriteString("\n")
	b.WriteString(d.styles.DetailSectionTitle.Render("─── Acceptance Criteria ───"))
	b.WriteString("\n")
	b.WriteString(renderDetailMarkdown(d.item.AcceptanceCriteria, wrapWidth))
	b.WriteString("\n")
}

// writeTagsSection writes the tags block when tags exist.
func (d *DetailsPanel) writeTagsSection(b *strings.Builder) {
	if len(d.item.Tags) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(d.styles.DetailSectionTitle.Render("─── Tags ───"))
	b.WriteString("\n")
	var tagStrings []string
	for _, tag := range d.item.Tags {
		tagStrings = append(tagStrings, d.styles.DetailTag.Render(tag))
	}
	b.WriteString(strings.Join(tagStrings, " "))
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

	if d.item.IsParentType() {
		actions = append(actions[:5], append([]string{
			keyStyle.Render("t") + descStyle.Render(" - Create child task"),
		}, actions[5:]...)...)
	}

	if d.item.HasChildren() {
		if d.item.IsExpanded {
			actions = append(actions, keyStyle.Render("←/h")+descStyle.Render(" - Collapse children"))
		} else {
			actions = append(actions, keyStyle.Render("→/l")+descStyle.Render(" - Expand children"))
		}
	}

	return actions
}

// renderDetailMarkdown renders markdown using glamour, falling back to wordWrap.
func renderDetailMarkdown(content string, width int) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return wordWrap(content, width)
	}
	rendered, renderErr := renderer.Render(content)
	if renderErr != nil {
		return wordWrap(content, width)
	}
	return strings.TrimSpace(rendered)
}

// truncateStr truncates a string to maxLen runes, appending "..." if needed.
func truncateStr(s string, maxLen int) string {
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

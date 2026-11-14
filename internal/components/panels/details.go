package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

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
		viewport: viewport.New(),
		styles:   st,
		keys:     k,
	}
}

func (d DetailsPanel) View() string {
	if d.item == nil {
		return FitBlock(d.styles.TextMuted.Render("Select a work item to preview it here."), d.width, d.height)
	}
	if !d.ready {
		return FitBlock("", d.width, d.height)
	}
	body := d.viewport.View()
	if d.viewport.TotalLineCount() > d.viewport.VisibleLineCount() {
		pct := int(d.viewport.ScrollPercent() * 100)
		body += "\n" + lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("%d%%", pct))
	}
	return FitBlock(body, d.width, d.height)
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
	if msg, ok := msg.(tea.KeyPressMsg); ok {
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

func (d *DetailsPanel) SetSize(width, height int) {
	if d.width == width && d.height == height && d.ready {
		return
	}
	d.width = width
	d.height = height
	d.viewport.SetWidth(max(width, 1))
	d.viewport.SetHeight(max(height-1, 1))
	if d.item != nil {
		d.updateViewportContent()
	}
}

func (d *DetailsPanel) updateViewportContent() {
	if d.item == nil || d.viewport.Width() <= 0 {
		return
	}
	d.viewport.SetContent(d.buildContent(max(d.viewport.Width(), styles.DetailWrapMinWidth)))
	d.ready = true
}

func (d *DetailsPanel) buildContent(width int) string {
	var sections []string
	sections = append(sections, renderFormHeader(d.item, width))
	sections = append(sections, renderFieldGrid(d.item, width))
	if d.item.Description != "" {
		sections = append(sections, SectionLabel("Description")+"\n"+
			renderHTMLContent(d.item.DescriptionHTML, d.item.Description, width))
	}
	if d.item.AcceptanceCriteria != "" {
		sections = append(sections, SectionLabel("Acceptance Criteria")+"\n"+
			renderHTMLContent(d.item.AcceptanceCriteriaHTML, d.item.AcceptanceCriteria, width))
	}
	if children := renderChildList(d.item, width); children != "" {
		sections = append(sections, children)
	}
	return strings.Join(sections, "\n\n")
}

func renderFormHeader(item *models.WorkItem, width int) string {
	typeColor := styles.TypeColor(string(item.Type))
	stripe := lipgloss.NewStyle().Foreground(typeColor).Render("▌")
	label := lipgloss.NewStyle().Foreground(typeColor).Bold(true).
		Render(fmt.Sprintf("%s %d", styles.TypeLabel(string(item.Type)), item.ID))
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Width(max(width-2, 1)).Render(item.Title)
	lines := []string{stripe + " " + label}
	for _, l := range strings.Split(title, "\n") {
		lines = append(lines, stripe+" "+l)
	}
	return strings.Join(lines, "\n")
}

func renderFieldGrid(item *models.WorkItem, width int) string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailFieldLabelW)
	value := lipgloss.NewStyle().Foreground(styles.ColorText)
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	valueWidth := max(width-styles.DetailFieldLabelW, 1)

	assigned := value.Render(item.AssignedTo)
	if item.AssignedTo == "" {
		assigned = muted.Render("Unassigned")
	}
	rows := [][2]string{
		{"State", StateDot(string(item.State), nil)},
		{"Assigned", assigned},
		{"Iteration", value.Render(item.IterationPath)},
		{"Area", value.Render(item.AreaPath)},
	}
	if item.IsParentType() {
		rows = append(rows,
			[2]string{"Priority", value.Render(orDash(item.Priority > 0, fmt.Sprintf("%d", item.Priority)))},
			[2]string{"Effort", value.Render(orDash(item.Effort > 0, formatEffort(item.Effort)))},
		)
	}
	if item.ParentID > 0 {
		parent := fmt.Sprintf("%d", item.ParentID)
		if item.ParentTitle != "" {
			parent += " " + item.ParentTitle
		}
		rows = append(rows, [2]string{"Parent", lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(parent)})
	}
	if len(item.Tags) > 0 {
		var chips []string
		for _, t := range item.Tags {
			chips = append(chips, Chip(t))
		}
		rows = append(rows, [2]string{"Tags", strings.Join(chips, " ")})
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, label.Render(r[0])+FitWidth(r[1], valueWidth))
	}
	return strings.Join(lines, "\n")
}

func renderChildList(item *models.WorkItem, width int) string {
	if len(item.Children) == 0 {
		return ""
	}
	done, total := DoneCount(item.Children)
	lines := []string{
		SectionLabel(fmt.Sprintf("Child items  %d / %d done", done, total)),
		ProgressBar(done, total, min(width, styles.DetailProgressBarW)),
	}
	for _, c := range item.Children {
		row := TypeGlyph(string(c.Type), nil) + " " +
			lipgloss.NewStyle().Foreground(styles.ColorLight).Render(fmt.Sprintf("%d ", c.ID)) +
			c.Title
		dot := lipgloss.NewStyle().Foreground(styles.StateColor(string(c.State))).Render(" ●")
		lines = append(lines, FitWidth(row, max(width-2, 1))+dot)
	}
	return strings.Join(lines, "\n")
}

func orDash(ok bool, s string) string {
	if ok {
		return s
	}
	return "—"
}

func (d *DetailsPanel) GetAvailableActions() []string {
	if d.item == nil {
		return nil
	}
	return Hints(styles.ColorSurface, d.keys.View, d.keys.ChangeState, d.keys.Assign, d.keys.EditTask, d.keys.AddComment)
}

// renderDetailMarkdown renders markdown using glamour, falling back to wordWrap.
func renderHTMLContent(htmlContent, fallback string, width int) string {
	md := htmlContent
	if md != "" {
		md = components.HTMLToMarkdown(md)
	} else {
		md = fallback
	}
	return renderDetailMarkdown(md, width)
}

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
	return dedent(rendered)
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

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	indent := -1
	for _, l := range lines {
		plain := ansi.Strip(l)
		if strings.TrimSpace(plain) == "" {
			continue
		}
		n := len(plain) - len(strings.TrimLeft(plain, " "))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i, l := range lines {
		if indent > 0 {
			l = ansi.Cut(l, indent, ansi.StringWidth(l))
		}
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

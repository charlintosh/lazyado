package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	llist "charm.land/lipgloss/v2/list"
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
		return Block(Muted("Select a work item to preview it here."), d.width, d.height)
	}
	if !d.ready {
		return Block("", d.width, d.height)
	}
	body := d.viewport.View()
	if d.viewport.TotalLineCount() > d.viewport.VisibleLineCount() {
		body += "\n" + Muted(fmt.Sprintf("%d%%", int(d.viewport.ScrollPercent()*100)))
	}
	return Block(body, d.width, d.height)
}

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
	sections := []string{
		renderFormHeader(d.item, width),
		renderFieldGrid(d.item, width),
	}
	if d.item.Description != "" {
		sections = append(sections, Heading("Description")+"\n"+
			renderHTMLContent(d.item.DescriptionHTML, d.item.Description, width))
	}
	if d.item.AcceptanceCriteria != "" {
		sections = append(sections, Heading("Acceptance Criteria")+"\n"+
			renderHTMLContent(d.item.AcceptanceCriteriaHTML, d.item.AcceptanceCriteria, width))
	}
	if children := renderChildList(d.item, width); children != "" {
		sections = append(sections, children)
	}
	return strings.Join(sections, "\n\n")
}

func renderFormHeader(item *models.WorkItem, width int) string {
	badge := TypeBadge(string(item.Type), fmt.Sprintf("%s %d", styles.TypeLabel(string(item.Type)), item.ID))
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Width(max(width, 1)).Render(item.Title)
	return lipgloss.JoinVertical(lipgloss.Left, badge, title)
}

func renderFieldGrid(item *models.WorkItem, width int) string {
	value := lipgloss.NewStyle().Foreground(styles.ColorText)
	assigned := value.Render(item.AssignedTo)
	if item.AssignedTo == "" {
		assigned = Muted("Unassigned")
	}
	rows := [][2]string{
		{"State", StateText(string(item.State))},
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
		rows = append(rows, [2]string{"Tags", Chips(item.Tags)})
	}
	return FieldRows(rows, styles.DetailFieldLabelW, width)
}

func renderChildList(item *models.WorkItem, width int) string {
	if len(item.Children) == 0 {
		return ""
	}
	done, total := DoneCount(item.Children)
	bar := newProgress(min(width, styles.DetailProgressBarW))
	children := llist.New().
		Enumerator(llist.Dash).
		EnumeratorStyle(lipgloss.NewStyle().Foreground(styles.ColorTextMuted).MarginRight(1))
	for _, c := range item.Children {
		row := lipgloss.NewStyle().Foreground(styles.ColorLight).Render(fmt.Sprintf("%d ", c.ID)) + c.Title
		state := StateText(string(c.State))
		children.Item(Line(row, max(width-lipgloss.Width(state)-styles.ChildListChrome, 1)) + " " + state)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		Heading(fmt.Sprintf("Child items  %d of %d done", done, total)),
		bar.ViewAs(float64(done)/float64(total)),
		children.String(),
	)
}

func orDash(ok bool, s string) string {
	if ok {
		return s
	}
	return "-"
}

func (d *DetailsPanel) KeyBindings() []key.Binding {
	if d.item == nil {
		return nil
	}
	return []key.Binding{d.keys.View, d.keys.ChangeState, d.keys.Assign, d.keys.EditTask, d.keys.AddComment}
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

package panels

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	dateFormat     = "Jan 2"
	dateFormatYear = "Jan 2, 2006"
	typeGlyph      = "■"
	stateGlyph     = "●"
	closeGlyph     = "✕"
	formGap        = "   "
	peopleGap      = "    "
)

type CloseDetailViewMsg struct{}

type OpenCommentModalMsg struct {
	Item models.WorkItem
}

type DetailView struct {
	item           *models.WorkItem
	thread         CommentThread
	loading        bool
	viewport       viewport.Model
	styles         styles.Styles
	keys           keys.KeyMap
	width          int
	height         int
	commentFocused bool
	offsets        []int
	discussionTop  int
}

type headerAction struct {
	label   string
	binding key.Binding
}

func NewDetailView(st styles.Styles, k keys.KeyMap) DetailView {
	return DetailView{
		thread:   NewCommentThread(k),
		viewport: widgets.NewViewport(k),
		styles:   st,
		keys:     k,
	}
}

func (d DetailView) Update(msg tea.Msg) (DetailView, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		d.viewport, cmd = d.viewport.Update(msg)
		return d, cmd
	}
	switch {
	case key.Matches(k, d.keys.Back) && d.commentFocused:
		d.commentFocused = false
		d.render()
		return d, nil
	case key.Matches(k, d.keys.Back), key.Matches(k, d.keys.Quit):
		return d, func() tea.Msg { return CloseDetailViewMsg{} }
	case key.Matches(k, d.keys.AddComment):
		if d.item == nil {
			return d, nil
		}
		item := *d.item
		return d, func() tea.Msg { return OpenCommentModalMsg{Item: item} }
	case key.Matches(k, d.keys.NextPanel):
		if d.thread.Len() > 0 {
			d.commentFocused = !d.commentFocused
			d.render()
			d.revealComment()
		}
		return d, nil
	case d.commentFocused:
		before := d.thread.Cursor()
		var cmd tea.Cmd
		d.thread, cmd = d.thread.Update(k)
		if d.thread.Cursor() != before {
			d.render()
			d.revealComment()
		}
		return d, cmd
	}
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d DetailView) View() string {
	if d.item == nil {
		return Block("", d.width, d.height)
	}
	return Block(lipgloss.JoinVertical(lipgloss.Left, d.renderHeader(), d.viewport.View()), d.width, d.height)
}

func (d DetailView) Item() *models.WorkItem { return d.item }

func (d DetailView) CommentFocused() bool { return d.commentFocused }

func (d DetailView) Loading() bool { return d.loading }

func (d DetailView) stacked() bool {
	return d.width < styles.DetailBreakpointSM
}

func (d DetailView) headerHeight() int {
	if d.stacked() {
		return styles.DetailHeaderLinesStacked
	}
	return styles.DetailHeaderLines
}

func (d *DetailView) SetSize(width, height int) {
	if d.width == width && d.height == height {
		return
	}
	d.width = width
	d.height = height
	d.viewport.SetWidth(max(width-1, styles.DetailMinViewportWidth))
	d.viewport.SetHeight(max(height-d.headerHeight(), styles.DetailMinViewportHeight))
	d.render()
}

func (d *DetailView) SetItem(item *models.WorkItem) {
	d.item = item
	d.commentFocused = false
	d.loading = item != nil
	id := 0
	if item != nil {
		id = item.ID
	}
	d.thread = d.thread.WithComments(id, nil)
	d.viewport.GotoTop()
	d.render()
}

func (d *DetailView) Refresh(item models.WorkItem) {
	if d.item == nil || d.item.ID != item.ID {
		return
	}
	d.item = &item
	d.render()
}

func (d *DetailView) SetComments(workItemID int, comments []models.Comment) {
	if d.item == nil || d.item.ID != workItemID {
		return
	}
	d.thread = d.thread.WithComments(workItemID, comments)
	d.loading = false
	if d.thread.Len() == 0 {
		d.commentFocused = false
	}
	d.render()
}

func (d *DetailView) SetIdentity(me models.Identity) {
	d.thread = d.thread.WithIdentity(me)
	d.render()
}

func (d *DetailView) KeyBindings() []key.Binding {
	if d.commentFocused {
		bindings := []key.Binding{d.keys.Up}
		if d.thread.CanModify() {
			bindings = append(bindings, d.keys.EditComment, d.keys.DeleteComment)
		}
		return append(bindings, d.keys.AddComment, d.keys.NextPanel, d.keys.Back)
	}
	bindings := []key.Binding{d.keys.ChangeState, d.keys.Assign, d.keys.EditTask}
	if d.item != nil && d.item.IsParentType() {
		bindings = append(bindings, d.keys.CreateTask)
	}
	return append(bindings, d.keys.AddComment, d.keys.Info, d.keys.DeleteTask, d.keys.NextPanel, d.keys.Up, d.keys.Back)
}

func (d *DetailView) revealComment() {
	if !d.commentFocused {
		return
	}
	top, bottom := d.thread.Span(d.offsets, d.viewport.TotalLineCount()-d.discussionTop)
	widgets.Reveal(&d.viewport, d.discussionTop+top, d.discussionTop+bottom)
}

func (d *DetailView) render() {
	if d.item == nil || d.viewport.Width() <= 0 {
		return
	}
	width := d.viewport.Width() - 1
	pad := lipgloss.NewStyle().PaddingLeft(1)
	if d.stacked() {
		d.viewport.SetContent(pad.Render(d.mainColumn(width, d.sidebarSections(width))))
		return
	}
	sidebarWidth := styles.DetailSidebarWidthDefault
	if width > styles.DetailBreakpointLG {
		sidebarWidth = styles.DetailSidebarWidthLG
	}
	mainWidth := max(width-sidebarWidth-styles.DetailColumnGap, styles.DetailWrapMinWidth)
	left := d.mainColumn(mainWidth, nil)
	right := "\n" + strings.Join(d.sidebarSections(sidebarWidth), "\n\n")
	d.viewport.SetContent(pad.Render(lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", styles.DetailColumnGap), right)))
}

func formHeading(title string, focused bool) string {
	st := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	if focused {
		st = st.Foreground(styles.ColorPrimary)
	}
	return st.Render(title)
}

func formSection(heading, accessory, body string, width int, focused bool) string {
	rule := styles.ColorBorder
	if focused {
		rule = styles.ColorPrimary
	}
	gap := max(width-lipgloss.Width(heading)-lipgloss.Width(accessory), 1)
	title := Line(heading+strings.Repeat(" ", gap)+accessory, width)
	underline := lipgloss.NewStyle().Foreground(rule).Render(strings.Repeat("─", max(width, 0)))
	return lipgloss.JoinVertical(lipgloss.Left, title, underline, body)
}

func keyLabel(b key.Binding) string {
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render(b.Help().Key)
}

func (d *DetailView) mainColumn(width int, extra []string) string {
	reading := min(max(width, styles.DetailWrapMinWidth), styles.DetailReadingWidth)
	sections := []string{formSection(formHeading("Description", false), "", d.descriptionBody(reading), width, false)}
	if d.item.AcceptanceCriteria != "" {
		criteria := components.RenderHTML(d.item.AcceptanceCriteriaHTML, d.item.AcceptanceCriteria, reading)
		sections = append(sections, formSection(formHeading("Acceptance Criteria", false), "", criteria, width, false))
	}
	sections = append(sections, extra...)
	pre := "\n" + strings.Join(sections, "\n\n")
	discussion, listTop := d.discussionSection(width)
	d.discussionTop = lipgloss.Height(pre) + 1 + listTop
	return pre + "\n\n" + discussion
}

func (d *DetailView) descriptionBody(width int) string {
	if d.item.Description == "" {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true).Render("No description provided")
	}
	return components.RenderHTML(d.item.DescriptionHTML, d.item.Description, width)
}

func (d *DetailView) discussionSection(width int) (string, int) {
	heading := formHeading("Discussion", d.commentFocused) + " " + Muted(strconv.Itoa(d.thread.Len()))
	accessory := ""
	if d.thread.Len() > 0 {
		target := "discussion"
		if d.commentFocused {
			target = "details"
		}
		accessory = keyLabel(d.keys.NextPanel) + " " + Muted(target)
	}
	box := d.commentBox(width)
	body := box + "\n\n" + d.discussionBody(width)
	return formSection(heading, accessory, body, width, d.commentFocused), 2 + lipgloss.Height(box) + 1
}

func (d *DetailView) commentBox(width int) string {
	inner := max(width-styles.BoxPaddedSize, 1)
	hint := Muted("Add a comment…")
	shortcut := keyLabel(d.keys.AddComment)
	gap := max(inner-lipgloss.Width(hint)-lipgloss.Width(shortcut), 1)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1).
		Width(width).
		Render(Line(hint+strings.Repeat(" ", gap)+shortcut, inner))
}

func (d *DetailView) discussionBody(width int) string {
	placeholder := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Italic(true)
	switch {
	case d.loading:
		d.offsets = nil
		return placeholder.Render("Loading comments...")
	case d.thread.Len() == 0:
		d.offsets = nil
		return placeholder.Render("No comments yet")
	}
	content, offsets := d.thread.RenderFlat(width, d.commentFocused)
	d.offsets = offsets
	return content
}

func (d *DetailView) sidebarSections(width int) []string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailFormLabelW)
	row := func(l, v string) string { return Line(label.Render(l)+v, width) }
	planning := strings.Join([]string{
		row("Priority", orDash(d.item.Priority > 0, strconv.Itoa(d.item.Priority))),
		row("Effort", orDash(d.item.Effort > 0, formatEffort(d.item.Effort))),
	}, "\n")
	return []string{
		formSection(formHeading("Planning", false), "", planning, width, false),
		formSection(formHeading("Related Work", false), "", d.relatedBody(width), width, false),
	}
}

func (d *DetailView) relatedBody(width int) string {
	var parts []string
	if d.item.ParentID > 0 {
		parent := strconv.Itoa(d.item.ParentID)
		if d.item.ParentTitle != "" {
			parent += " " + d.item.ParentTitle
		}
		parts = append(parts, Heading("Parent"), Line(lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(parent), width))
	}
	if children := d.item.Children; len(children) > 0 {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		done, total := DoneCount(children)
		parts = append(parts,
			Heading("Child items")+"  "+Muted(fmt.Sprintf("%d of %d done", done, total)),
			newProgress(min(width, styles.DetailProgressBarW)).ViewAs(float64(done)/float64(total)),
		)
		state := lipgloss.NewStyle().Width(styles.DetailFormLabelW)
		for _, c := range children {
			id := lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strconv.Itoa(c.ID))
			parts = append(parts, Line(state.Render(StateText(string(c.State)))+id+" "+c.Title, width))
		}
	}
	if len(parts) == 0 {
		return Muted("No related work")
	}
	return strings.Join(parts, "\n")
}

func (d DetailView) renderHeader() string {
	item := d.item
	inner := max(d.width-styles.CardChromeWidth, 1)
	lines := []string{
		d.typeRow(inner),
		d.titleRow(inner),
		d.peopleRow(inner),
		lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", max(inner-1, 0))),
	}
	lines = append(lines, d.bandRows(inner)...)
	block := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(styles.TypeColor(string(item.Type))).
		PaddingLeft(1).
		Render(strings.Join(lines, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, "", block, d.tabRule(inner))
}

func edgeLine(left, right string, width int) string {
	gap := max(width-1-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return Line(left+strings.Repeat(" ", gap)+right+" ", width)
}

func formatFormDate(t time.Time) string {
	if t.Year() == time.Now().Year() {
		return t.Format(dateFormat)
	}
	return t.Format(dateFormatYear)
}

func (d DetailView) typeRow(width int) string {
	item := d.item
	left := lipgloss.NewStyle().Foreground(styles.TypeColor(string(item.Type))).Render(typeGlyph) + " " +
		lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(styles.TypeLabel(string(item.Type))) + " " +
		lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strconv.Itoa(item.ID))
	closeHint := Muted(closeGlyph) + " " + keyLabel(d.keys.Back)
	var dates []string
	if !item.ChangedDate.IsZero() {
		dates = append(dates, Muted("Updated "+formatFormDate(item.ChangedDate)))
	}
	if !item.CreatedDate.IsZero() {
		dates = append(dates, Muted("Created "+formatFormDate(item.CreatedDate)))
	}
	for n := len(dates); n > 0; n-- {
		right := strings.Join(append(append([]string{}, dates[:n]...), closeHint), formGap)
		if lipgloss.Width(left)+lipgloss.Width(right)+2 <= width {
			return edgeLine(left, right, width)
		}
	}
	return edgeLine(left, closeHint, width)
}

func (d DetailView) titleRow(width int) string {
	id := lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strconv.Itoa(d.item.ID))
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(d.item.Title)
	return Line(id+"  "+title, width)
}

func (d DetailView) actions() []headerAction {
	k := d.keys
	out := []headerAction{{"Edit", k.EditTask}, {"State", k.ChangeState}, {"Assign", k.Assign}}
	if d.item.IsParentType() {
		out = append(out, headerAction{"New task", k.CreateTask})
	}
	return append(out, headerAction{"Comment", k.AddComment}, headerAction{"Info", k.Info}, headerAction{"Delete", k.DeleteTask})
}

func (d DetailView) peopleRow(width int) string {
	item := d.item
	assignee := Muted("Unassigned")
	if item.AssignedTo != "" {
		assignee = lipgloss.NewStyle().Foreground(styles.AvatarColor(item.AssignedTo)).Bold(true).Render(widgets.Initials(item.AssignedTo)) +
			" " + lipgloss.NewStyle().Foreground(styles.ColorText).Render(item.AssignedTo)
	}
	comments := Muted("Loading comments")
	if !d.loading {
		comments = Muted(countLabel(d.thread.Len(), "comment"))
	}
	left := assignee + peopleGap + comments
	if len(item.Tags) > 0 {
		tags := make([]string, len(item.Tags))
		for i, t := range item.Tags {
			tags[i] = "[" + t + "]"
		}
		left += peopleGap + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strings.Join(tags, " "))
	}
	actions := d.actions()
	for n := len(actions); n > 0; n-- {
		parts := make([]string, n)
		for i, a := range actions[:n] {
			parts[i] = lipgloss.NewStyle().Foreground(styles.ColorText).Render(a.label) + " " + keyLabel(a.binding)
		}
		right := strings.Join(parts, formGap)
		if lipgloss.Width(left)+lipgloss.Width(right)+3 <= width {
			return edgeLine(left, right, width)
		}
	}
	return Line(left, width)
}

func countLabel(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

func (d DetailView) tabLabels() []string {
	return []string{"Details", "Discussion " + strconv.Itoa(d.thread.Len())}
}

func (d DetailView) activeTab() int {
	if d.commentFocused {
		return 1
	}
	return 0
}

func (d DetailView) tabsView() string {
	labels := d.tabLabels()
	out := make([]string, len(labels))
	for i, l := range labels {
		st := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == d.activeTab() {
			st = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
		}
		out[i] = st.Render(l)
	}
	return strings.Join(out, formGap)
}

func (d DetailView) bandRows(width int) []string {
	item := d.item
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(styles.DetailFieldLabelW)
	light := lipgloss.NewStyle().Foreground(styles.ColorLight)
	state := lipgloss.NewStyle().Foreground(styles.StateColor(string(item.State))).Render(stateGlyph) + " " +
		lipgloss.NewStyle().Foreground(styles.ColorText).Render(string(item.State))
	reason := light.Render(orDash(item.Reason != "", item.Reason))
	field := func(name, value string, w int) string { return label.Render(name) + Line(value, max(w, 1)) }
	tabs := d.tabsView()
	rest := width - styles.DetailFieldLabelW
	if d.stacked() {
		return []string{
			Line(field("State", state, rest), width),
			Line(field("Reason", reason, rest), width),
			Line(field("Area", item.AreaPath, rest), width),
			edgeLine(field("Iteration", item.IterationPath, rest-lipgloss.Width(tabs)-len(formGap)-1), tabs, width),
		}
	}
	valueW := min(styles.DetailBandValueW, (width-2*styles.DetailFieldLabelW)/2)
	right := rest - valueW - styles.DetailFieldLabelW
	return []string{
		Line(field("State", state, valueW)+field("Area", item.AreaPath, right-1), width),
		edgeLine(field("Reason", reason, valueW)+field("Iteration", item.IterationPath, right-lipgloss.Width(tabs)-len(formGap)-1), tabs, width),
	}
}

func (d DetailView) tabRule(inner int) string {
	labels := d.tabLabels()
	tabsWidth := lipgloss.Width(strings.Join(labels, formGap))
	start := styles.CardChromeWidth + inner - 1 - tabsWidth
	for _, l := range labels[:d.activeTab()] {
		start += lipgloss.Width(l) + len(formGap)
	}
	start = max(start, 0)
	length := min(lipgloss.Width(labels[d.activeTab()]), max(d.width-start, 0))
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder)
	return rule.Render(strings.Repeat("─", start)) +
		lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(strings.Repeat("━", length)) +
		rule.Render(strings.Repeat("─", max(d.width-start-length, 0)))
}

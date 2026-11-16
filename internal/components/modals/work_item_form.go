package modals

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	maxEffort    = 1000
	maxTagLength = 50
	fieldGap     = "    "
)

var (
	errTitleRequired = errors.New("Title is required")
	errPriority      = errors.New("Priority must be a number from 1 to 4")
	errEffort        = errors.New("Effort must be a number from 0 to 1000")
	errTagLength     = errors.New("Each tag must be 50 characters or less")
)

type FormMode int

const (
	FormCreatePBI FormMode = iota
	FormCreateBug
	FormEditWorkItem
	FormCreateTask
	FormEditTask
)

type fieldID int

const (
	fieldTitle fieldID = iota
	fieldDescription
	fieldAcceptance
	fieldPriority
	fieldEffort
	fieldTags
	fieldDefectCause
	fieldAssignee
	fieldSprint
	fieldArea
)

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

type WorkItemUpdateRequestMsg struct {
	ItemID      int
	Title       string
	Description string
	Priority    int
	Effort      float64
	Tags        []string
	AssignedTo  string
}

type TaskCreateRequestMsg struct {
	ParentID    int
	Title       string
	Description string
}

type TaskUpdateRequestMsg struct {
	TaskID      int
	Title       string
	Description string
}

type FormData struct {
	Members          []models.TeamMember
	Iterations       []models.Iteration
	DefaultArea      string
	DefaultIteration string
}

type formEntry struct {
	id    fieldID
	field field
}

type fieldSpan struct {
	top    int
	bottom int
}

type WorkItemForm struct {
	env      Env
	mode     FormMode
	title    string
	subtitle string
	targetID int
	entries  []formEntry
	focus    int
	err      string
	body     viewport.Model
}

func NewCreatePBIForm(env Env, data FormData) WorkItemForm {
	return newForm(env, FormCreatePBI, "Create Product Backlog Item", "", 0, []formEntry{
		{fieldTitle, titleField("")},
		{fieldDescription, descriptionField("")},
		{fieldAcceptance, acceptanceField()},
		{fieldPriority, priorityField(0)},
		{fieldEffort, effortField(0)},
		{fieldTags, tagsField(nil)},
		{fieldAssignee, assigneeField(env, data.Members, "", true)},
		{fieldSprint, sprintField(env, data.Iterations, data.DefaultIteration)},
		{fieldArea, areaPathField(data.DefaultArea)},
	})
}

func NewCreateBugForm(env Env, data FormData) WorkItemForm {
	return newForm(env, FormCreateBug, "Create Bug", "", 0, []formEntry{
		{fieldTitle, titleField("")},
		{fieldDescription, descriptionField("")},
		{fieldAcceptance, acceptanceField()},
		{fieldTags, tagsField(nil)},
		{fieldDefectCause, newTextField("Defect cause", "Optional defect cause", "", styles.TagsCharLimit, false)},
		{fieldAssignee, assigneeField(env, data.Members, "", true)},
		{fieldSprint, sprintField(env, data.Iterations, data.DefaultIteration)},
		{fieldArea, areaPathField(data.DefaultArea)},
	})
}

func NewEditWorkItemForm(env Env, item models.WorkItem, members []models.TeamMember) WorkItemForm {
	return newForm(env, FormEditWorkItem, "Edit "+string(item.Type), itemLabel(item), item.ID, []formEntry{
		{fieldTitle, titleField(item.Title)},
		{fieldDescription, descriptionField(item.Description)},
		{fieldPriority, priorityField(item.Priority)},
		{fieldEffort, effortField(item.Effort)},
		{fieldTags, tagsField(item.Tags)},
		{fieldAssignee, assigneeField(env, members, item.AssignedTo, false)},
	})
}

func NewCreateTaskForm(env Env, parent models.WorkItem) WorkItemForm {
	return newForm(env, FormCreateTask, "Create task", "Parent: "+itemLabel(parent), parent.ID, []formEntry{
		{fieldTitle, titleField("")},
		{fieldDescription, descriptionField("")},
	})
}

func NewEditTaskForm(env Env, task models.WorkItem) WorkItemForm {
	return newForm(env, FormEditTask, "Edit task", itemLabel(task), task.ID, []formEntry{
		{fieldTitle, titleField(task.Title)},
		{fieldDescription, descriptionField(task.Description)},
	})
}

func newForm(env Env, mode FormMode, title, subtitle string, targetID int, entries []formEntry) WorkItemForm {
	m := WorkItemForm{
		env:      env,
		mode:     mode,
		title:    title,
		subtitle: subtitle,
		targetID: targetID,
		entries:  entries,
		body:     viewport.New(),
	}
	m.body.KeyMap = viewport.KeyMap{}
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	return m
}

func (m WorkItemForm) Mode() FormMode { return m.mode }

func (m WorkItemForm) Init() tea.Cmd {
	return textinput.Blink
}

func (m WorkItemForm) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		m.body, cmd = m.body.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		k := m.env.Keys
		switch {
		case key.Matches(msg, k.Back):
			return m, closeModal
		case key.Matches(msg, k.Save):
			return m.submit()
		case key.Matches(msg, k.NextPanel):
			return m.moveFocus(1)
		case key.Matches(msg, k.PrevPanel):
			return m.moveFocus(-1)
		case key.Matches(msg, k.Select) && !m.current().multiline():
			return m.moveFocus(1)
		}
	}
	f, cmd := m.current().update(msg)
	m = m.withField(m.focus, f)
	m.refresh()
	return m, cmd
}

func (m WorkItemForm) withField(i int, f field) WorkItemForm {
	m.entries = slices.Clone(m.entries)
	m.entries[i].field = f
	return m
}

func (m WorkItemForm) current() field {
	return m.entries[m.focus].field
}

func (m WorkItemForm) moveFocus(delta int) (Modal, tea.Cmd) {
	m = m.withField(m.focus, m.current().blur())
	m.focus = (m.focus + delta + len(m.entries)) % len(m.entries)
	f, cmd := m.current().focus()
	m = m.withField(m.focus, f)
	m.refresh()
	return m, cmd
}

func (m *WorkItemForm) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.entries = slices.Clone(m.entries)
	for i, e := range m.entries {
		m.entries[i].field = e.field.resize(inner)
	}
	m.body.SetWidth(inner)
	m.refresh()
}

func (m *WorkItemForm) refresh() {
	content, spans := m.renderFields()
	budget := widgets.DialogBodyHeight(m.env.Height, m.subtitle != "")
	if m.err != "" {
		budget -= 2
	}
	if lipgloss.Height(content) > budget {
		budget--
	}
	m.body.SetHeight(max(min(lipgloss.Height(content), budget), 1))
	m.body.SetContent(content)
	span := spans[m.focus]
	widgets.Reveal(&m.body, span.top, span.bottom)
}

func (m WorkItemForm) renderFields() (string, []fieldSpan) {
	spans := make([]fieldSpan, len(m.entries))
	var blocks []string
	line := 0
	for i := 0; i < len(m.entries); i++ {
		block := m.renderEntry(i)
		group := []int{i}
		if m.entries[i].field.compact() && i+1 < len(m.entries) && m.entries[i+1].field.compact() {
			block = lipgloss.JoinHorizontal(lipgloss.Top, block, fieldGap, m.renderEntry(i+1))
			group = append(group, i+1)
			i++
		}
		h := lipgloss.Height(block)
		for _, g := range group {
			spans[g] = fieldSpan{top: line, bottom: line + h - 1}
		}
		blocks = append(blocks, block)
		line += h + 1
	}
	return strings.Join(blocks, "\n\n"), spans
}

func (m WorkItemForm) renderEntry(i int) string {
	e := m.entries[i]
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	if i == m.focus {
		label = label.Foreground(styles.ColorPrimary).Bold(true)
	}
	return lipgloss.JoinVertical(lipgloss.Left, label.Render(e.field.label()), e.field.view())
}

func (m WorkItemForm) width() int {
	preferred := styles.ModalWidthXL
	if m.mode == FormCreateTask || m.mode == FormEditTask {
		preferred = styles.ModalWidthLG
	}
	return widgets.DialogWidth(preferred, m.env.Width)
}

func (m WorkItemForm) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	body := m.body.View()
	if m.body.TotalLineCount() > m.body.VisibleLineCount() {
		position := fmt.Sprintf("Field %d of %d", m.focus+1, len(m.entries))
		body += "\n" + lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).Align(lipgloss.Right).Render(position)
	}
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err)
	}
	return widgets.Dialog{
		Title:    m.title,
		Subtitle: m.subtitle,
		Body:     body,
		Footer:   widgets.Hints(inner, m.hints()...),
		Width:    m.width(),
	}.View()
}

func (m WorkItemForm) hints() []key.Binding {
	k := m.env.Keys
	bindings := []key.Binding{
		widgets.Relabel(k.NextPanel, "next"),
		widgets.Relabel(k.PrevPanel, "prev"),
		k.Save,
		widgets.Relabel(k.Back, "cancel"),
	}
	if m.current().choice() {
		bindings = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown)}, bindings...)
	}
	return bindings
}

func (m WorkItemForm) value(id fieldID) string {
	for _, e := range m.entries {
		if e.id == id {
			return e.field.value()
		}
	}
	return ""
}

func (m WorkItemForm) submit() (Modal, tea.Cmd) {
	msg, err := m.request()
	if err != nil {
		m.err = err.Error()
		m.refresh()
		return m, nil
	}
	return m, send(msg)
}

func (m WorkItemForm) request() (tea.Msg, error) {
	title := m.value(fieldTitle)
	if title == "" {
		return nil, errTitleRequired
	}
	description := m.value(fieldDescription)
	switch m.mode {
	case FormCreateTask:
		return TaskCreateRequestMsg{ParentID: m.targetID, Title: title, Description: description}, nil
	case FormEditTask:
		return TaskUpdateRequestMsg{TaskID: m.targetID, Title: title, Description: description}, nil
	}
	priority, err := parsePriority(m.value(fieldPriority))
	if err != nil {
		return nil, err
	}
	effort, err := parseEffort(m.value(fieldEffort))
	if err != nil {
		return nil, err
	}
	tags, err := parseTags(m.value(fieldTags))
	if err != nil {
		return nil, err
	}
	if m.mode == FormEditWorkItem {
		return WorkItemUpdateRequestMsg{
			ItemID:      m.targetID,
			Title:       title,
			Description: description,
			Priority:    priority,
			Effort:      effort,
			Tags:        tags,
			AssignedTo:  m.value(fieldAssignee),
		}, nil
	}
	workItemType := models.WorkItemTypePBI
	if m.mode == FormCreateBug {
		workItemType = models.WorkItemTypeBug
	}
	return ParentCreateRequestMsg{
		WorkItemType:       string(workItemType),
		Title:              title,
		Description:        description,
		AcceptanceCriteria: m.value(fieldAcceptance),
		IterationPath:      m.value(fieldSprint),
		AreaPath:           m.value(fieldArea),
		Effort:             effort,
		Priority:           priority,
		Tags:               tags,
		AssignedTo:         m.value(fieldAssignee),
		DefectCause:        m.value(fieldDefectCause),
	}, nil
}

func parsePriority(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 4 {
		return 0, errPriority
	}
	return p, nil
}

func parseEffort(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	e, err := strconv.ParseFloat(s, 64)
	if err != nil || e < 0 || e > maxEffort {
		return 0, errEffort
	}
	return e, nil
}

func parseTags(s string) ([]string, error) {
	var tags []string
	for _, part := range strings.Split(s, ";") {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		if len(tag) > maxTagLength {
			return nil, errTagLength
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func titleField(value string) textField {
	return newTextField("Title", "Enter title...", value, styles.TitleCharLimit, false)
}

func descriptionField(value string) areaField {
	return newAreaField("Description", "Optional description", value, styles.ContentCharLimitLG, styles.TextareaHeightMD)
}

func acceptanceField() areaField {
	return newAreaField("Acceptance criteria", "Optional acceptance criteria", "", styles.ContentCharLimitLG, styles.TextareaHeightSM)
}

func priorityField(p int) textField {
	value := ""
	if p > 0 {
		value = strconv.Itoa(p)
	}
	return newTextField("Priority (1-4)", "1-4", value, styles.PriorityCharLimit, true)
}

func effortField(e float64) textField {
	value := ""
	if e > 0 {
		value = strconv.FormatFloat(e, 'f', -1, 64)
	}
	return newTextField("Effort", "0", value, styles.EffortCharLimit, true)
}

func tagsField(tags []string) textField {
	return newTextField("Tags (separate with ;)", "tag1; tag2; tag3", strings.Join(tags, "; "), styles.TagsCharLimit, false)
}

func areaPathField(value string) textField {
	return newTextField("Area", "Area path", value, styles.TitleCharLimit, false)
}

func assigneeField(env Env, members []models.TeamMember, assignedTo string, allowNone bool) choiceField {
	options, initial := memberOptions(members, assignedTo)
	for i := range options {
		options[i].Marked = false
	}
	if initial == "" && assignedTo != "" {
		options = append([]widgets.Option{{Label: assignedTo, Value: assignedTo}}, options...)
		initial = assignedTo
	}
	if allowNone || assignedTo == "" {
		options = append([]widgets.Option{{Label: "Unassigned", Value: ""}}, options...)
	}
	return newChoiceField(env.Keys, "Assign to", "Unassigned", "member", "members", options, initial)
}

func sprintField(env Env, iterations []models.Iteration, initial string) choiceField {
	options := make([]widgets.Option, len(iterations))
	for i, it := range iterations {
		detail := ""
		if it.IsCurrent() {
			detail = "(current)"
		}
		options[i] = widgets.Option{Label: it.Name, Value: it.Path, Detail: detail}
	}
	return newChoiceField(env.Keys, "Sprint", "None", "sprint", "sprints", options, initial)
}

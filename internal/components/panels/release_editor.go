package panels

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
)

type EditorTab int

const (
	EditorPipelineTab EditorTab = iota
	EditorTasksTab
	EditorVariablesTab
	EditorHistoryTab
	editorTabCount
)

const (
	editorPaneTasks = iota
	editorPaneFields
)

const (
	revisionsIdle = iota
	revisionsLoading
	revisionsLoaded
)

const (
	editorHeaderHeight = 4
	loadingEditor      = "Loading release pipeline"
	startEditingHint   = "Press e to start editing."
)

type OpenReleaseEditorMsg struct {
	Kind models.ReleaseDocKind
	ID   int
}

type CloseReleaseEditorMsg struct{}

type EditReleaseFieldMsg struct {
	Field     models.TaskField
	Where     string
	Request   models.AzureListRequest
	Variables []string
}

type ReviewReleaseEditsMsg struct{}

type DiscardReleaseEditsMsg struct {
	Count int
}

type LoadRevisionsMsg struct {
	DefinitionID int
}

type taskRow struct {
	phase int
	task  int
}

type ReleaseEditor struct {
	keys      keys.KeyMap
	kind      models.ReleaseDocKind
	id        int
	doc       *models.ReleaseDoc
	refs      models.ReleaseDocRefs
	edits     models.DocEdits
	editing   bool
	tab       EditorTab
	stage     int
	pane      int
	task      int
	taskTop   int
	field     int
	fieldTop  int
	variable  int
	varTop    int
	scope     int
	stale     []models.StaleField
	revisions []models.DefinitionRevision
	revState  int
	revErr    string
	body      viewport.Model
	spinner   string
	width     int
	height    int
}

func NewReleaseEditor(k keys.KeyMap) ReleaseEditor {
	return ReleaseEditor{keys: k, body: widgets.NewViewport(k)}
}

func (v *ReleaseEditor) Open(kind models.ReleaseDocKind, id int, tab EditorTab) {
	*v = ReleaseEditor{keys: v.keys, kind: kind, id: id, tab: tab, spinner: v.spinner, width: v.width, height: v.height, body: widgets.NewViewport(v.keys)}
	v.layout()
}

func (v *ReleaseEditor) ID() int                     { return v.id }
func (v *ReleaseEditor) Kind() models.ReleaseDocKind { return v.kind }
func (v *ReleaseEditor) Doc() *models.ReleaseDoc     { return v.doc }
func (v *ReleaseEditor) Edits() models.DocEdits      { return v.edits }
func (v *ReleaseEditor) Editing() bool               { return v.editing }
func (v *ReleaseEditor) Dirty() bool                 { return len(v.edits) > 0 }
func (v *ReleaseEditor) Refs() models.ReleaseDocRefs { return v.refs }
func (v *ReleaseEditor) Tab() EditorTab              { return v.tab }
func (v *ReleaseEditor) Loading() bool {
	return v.id != 0 && (v.doc == nil || v.revState == revisionsLoading)
}
func (v *ReleaseEditor) Matches(kind models.ReleaseDocKind, id int) bool {
	return v.id == id && v.kind == kind
}

func (v *ReleaseEditor) SetSpinner(frame string) {
	if v.spinner == frame {
		return
	}
	v.spinner = frame
	if v.Loading() {
		v.refresh()
	}
}

func (v *ReleaseEditor) SetSize(width, height int) {
	if v.width == width && v.height == height {
		return
	}
	v.width, v.height = width, height
	v.layout()
}

func (v *ReleaseEditor) layout() {
	v.body.SetWidth(max(v.width-2, 1))
	v.body.SetHeight(max(v.bodyHeight(), 1))
	v.refresh()
}

func (v *ReleaseEditor) headerHeight() int {
	if v.kind == models.DocRelease {
		return editorHeaderHeight + 1
	}
	return editorHeaderHeight
}

func (v *ReleaseEditor) bodyHeight() int { return max(v.height-v.headerHeight(), 0) }

func (v *ReleaseEditor) SetDoc(doc *models.ReleaseDoc, refs models.ReleaseDocRefs) {
	if doc == nil || doc.ID != v.id || doc.Kind != v.kind {
		return
	}
	v.doc, v.refs = doc, refs
	v.clamp()
	v.refresh()
}

func (v *ReleaseEditor) SetRefs(refs models.ReleaseDocRefs) {
	v.refs = refs
	v.refresh()
}

func (v *ReleaseEditor) Stale() []models.StaleField { return v.stale }

func (v *ReleaseEditor) Saved(doc *models.ReleaseDoc) {
	v.edits, v.editing, v.stale = nil, false, nil
	v.revState, v.revisions = revisionsIdle, nil
	v.SetDoc(doc, v.refs)
	if v.tab == EditorHistoryTab {
		v.revState = revisionsLoading
	}
}

func (v *ReleaseEditor) Reload(doc *models.ReleaseDoc, edits models.DocEdits) {
	v.edits = edits
	v.revState, v.revisions = revisionsIdle, nil
	v.SetDoc(doc, v.refs)
}

func (v *ReleaseEditor) Discard() {
	v.edits, v.editing, v.stale = nil, false, nil
	v.refresh()
}

func (v *ReleaseEditor) Apply(e models.DocEdit) {
	fields := v.fields()
	v.edits = v.edits.With(e)
	v.stale = slices.DeleteFunc(slices.Clone(v.stale), func(st models.StaleField) bool {
		return st.Path.String() == e.Path.String() || st.Parent.String() == e.Path.String()
	})
	if _, changed := v.edits.Find(e.Path); changed {
		for _, f := range fields {
			if f.Path.String() != e.Path.String() {
				continue
			}
			for _, dep := range models.DependentsOf(fields, f.Key) {
				if dep.EditText() != "" && !v.isStale(dep.Path) {
					v.stale = append(v.stale, models.StaleField{Path: dep.Path, Field: dep.Label, Where: v.where(), Parent: e.Path, Cause: f.Label})
				}
			}
		}
	}
	v.reveal()
	v.refresh()
}

func (v *ReleaseEditor) dropStale(match func(models.StaleField) bool) {
	v.stale = slices.DeleteFunc(slices.Clone(v.stale), match)
}

func (v *ReleaseEditor) isStale(path models.DocPath) bool {
	for _, st := range v.stale {
		if st.Path.String() == path.String() {
			return true
		}
	}
	return false
}

func (v *ReleaseEditor) staleUnder(prefix models.DocPath) int {
	n := 0
	for _, st := range v.stale {
		if st.Path.Has(prefix) {
			n++
		}
	}
	return n
}

func (v *ReleaseEditor) JumpToStale() {
	if len(v.stale) == 0 || v.doc == nil {
		return
	}
	path := v.stale[0].Path
	env, _ := path[1].(int)
	for si, s := range v.doc.Stages {
		if s.Index != env {
			continue
		}
		v.tab, v.stage, v.pane = EditorTasksTab, si, editorPaneFields
		for ri, r := range v.taskRows() {
			if path.Has(v.rowPath(s, r)) && (r.task >= 0 || len(path) < 5 || path[4] != "workflowTasks") {
				v.task = ri
			}
		}
		for fi, f := range v.fields() {
			if f.Path.String() == path.String() {
				v.field = fi
			}
		}
	}
	v.reveal()
	v.refresh()
}

func (v *ReleaseEditor) SetRevisions(id int, revs []models.DefinitionRevision, err error) {
	if v.kind != models.DocDefinition || id != v.id {
		return
	}
	v.revisions, v.revState, v.revErr = revs, revisionsLoaded, ""
	if err != nil {
		v.revErr = err.Error()
	}
	v.refresh()
}

func (v *ReleaseEditor) WantsRevisions() bool {
	return v.kind == models.DocDefinition && v.tab == EditorHistoryTab && v.revState == revisionsLoading
}

func (v *ReleaseEditor) clamp() {
	if v.doc == nil {
		return
	}
	v.stage = min(max(v.stage, 0), max(len(v.doc.Stages)-1, 0))
	v.task = min(max(v.task, 0), max(len(v.taskRows())-1, 0))
	v.variable = min(max(v.variable, 0), max(len(v.scopedVariables())-1, 0))
	v.scope = min(max(v.scope, 0), len(v.doc.Stages)+1)
	v.field = min(max(v.field, 0), max(len(v.fields())-1, 0))
}

func (v *ReleaseEditor) currentStage() (models.DocStage, bool) {
	if v.doc == nil || len(v.doc.Stages) == 0 {
		return models.DocStage{}, false
	}
	return v.doc.Stages[v.stage], true
}

func (v *ReleaseEditor) taskRows() []taskRow {
	s, ok := v.currentStage()
	if !ok {
		return nil
	}
	var rows []taskRow
	for pi, p := range s.Phases {
		rows = append(rows, taskRow{phase: pi, task: -1})
		for ti := range p.Tasks {
			rows = append(rows, taskRow{phase: pi, task: ti})
		}
	}
	return rows
}

func (v *ReleaseEditor) currentRow() (models.DocStage, models.DocPhase, models.DocTask, bool, bool) {
	s, ok := v.currentStage()
	rows := v.taskRows()
	if !ok || v.task >= len(rows) {
		return s, models.DocPhase{}, models.DocTask{}, false, false
	}
	r := rows[v.task]
	p := s.Phases[r.phase]
	if r.task < 0 {
		return s, p, models.DocTask{}, false, true
	}
	return s, p, p.Tasks[r.task], true, true
}

func (v *ReleaseEditor) currentTask() (models.DocStage, models.DocPhase, models.DocTask, bool) {
	s, p, t, isTask, ok := v.currentRow()
	return s, p, t, ok && isTask
}

func (v *ReleaseEditor) rowPath(s models.DocStage, r taskRow) models.DocPath {
	p := s.Phases[r.phase]
	if r.task < 0 {
		return v.doc.PhasePath(s, p)
	}
	return v.doc.TaskPath(s, p, p.Tasks[r.task])
}

func (v *ReleaseEditor) rowEdits(s models.DocStage, r taskRow) int {
	if r.task >= 0 {
		return v.edits.CountUnder(v.rowPath(s, r))
	}
	base := v.rowPath(s, r)
	return v.edits.CountUnder(base) - v.edits.CountUnder(base.Child("workflowTasks"))
}

func (v *ReleaseEditor) rowStale(s models.DocStage, r taskRow) int {
	if r.task >= 0 {
		return v.staleUnder(v.rowPath(s, r))
	}
	base := v.rowPath(s, r)
	return v.staleUnder(base) - v.staleUnder(base.Child("workflowTasks"))
}

func (v *ReleaseEditor) fields() []models.TaskField {
	s, p, t, isTask, ok := v.currentRow()
	if !ok {
		return nil
	}
	if !isTask {
		return v.doc.PhaseFields(s, p, v.refs.Queues, v.edits)
	}
	fields, _ := v.doc.TaskFields(s, p, t, v.refs.Task(t), v.edits)
	return fields
}

func (v *ReleaseEditor) variableNames() []string {
	var names []string
	for _, variable := range v.doc.Variables {
		if !slices.Contains(names, variable.Name) {
			names = append(names, variable.Name)
		}
	}
	slices.SortFunc(names, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return names
}

func (v *ReleaseEditor) scopedVariables() []models.DocVariable {
	if v.doc == nil {
		return nil
	}
	switch v.scope {
	case 0:
		return v.doc.Variables
	case 1:
		return v.doc.StageVariables(-1)
	}
	return v.doc.StageVariables(v.doc.Stages[v.scope-2].Index)
}

func (v ReleaseEditor) Update(msg tea.Msg) (ReleaseEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	case tea.MouseWheelMsg:
		delta := 1
		if msg.Button == tea.MouseWheelUp {
			delta = -1
		}
		switch v.tab {
		case EditorTasksTab:
			v.moveCursor(delta)
		case EditorVariablesTab:
			v.moveVariable(delta)
		default:
			var cmd tea.Cmd
			v.body, cmd = v.body.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

func notice(text string) tea.Cmd {
	return func() tea.Msg { return ReleaseNoticeMsg{Text: text} }
}

func (v ReleaseEditor) handleKey(msg tea.KeyPressMsg) (ReleaseEditor, tea.Cmd) {
	k := v.keys
	if key.Matches(msg, k.Back) {
		return v.back()
	}
	if v.doc == nil {
		return v, nil
	}
	tabs := []key.Binding{k.Panel1, k.Panel2, k.Panel3, k.Panel4}
	for i, b := range tabs {
		if key.Matches(msg, b) {
			return v, v.setTab(EditorTab(i))
		}
	}
	switch {
	case key.Matches(msg, k.NextView):
		return v, v.setTab((v.tab + 1) % editorTabCount)
	case key.Matches(msg, k.PrevView):
		return v, v.setTab((v.tab + editorTabCount - 1) % editorTabCount)
	case key.Matches(msg, k.EditPipeline):
		return v.toggleEditing()
	case key.Matches(msg, k.Save):
		if !v.Dirty() {
			return v, notice("No changes to save.")
		}
		return v, func() tea.Msg { return ReviewReleaseEditsMsg{} }
	case key.Matches(msg, k.OpenBrowser):
		url := v.doc.WebURL
		return v, func() tea.Msg { return BrowseReleaseMsg{DefinitionID: v.doc.DefinitionID, URL: url} }
	case key.Matches(msg, k.PrevStage):
		v.moveStage(-1)
		return v, nil
	case key.Matches(msg, k.NextStage):
		v.moveStage(1)
		return v, nil
	}
	switch v.tab {
	case EditorTasksTab:
		return v.handleTasksKey(msg)
	case EditorVariablesTab:
		return v.handleVariablesKey(msg)
	case EditorPipelineTab:
		switch {
		case key.Matches(msg, k.Left):
			v.moveStage(-1)
			return v, nil
		case key.Matches(msg, k.Right):
			v.moveStage(1)
			return v, nil
		case key.Matches(msg, k.Select):
			return v, v.setTab(EditorTasksTab)
		}
	}
	var cmd tea.Cmd
	v.body, cmd = v.body.Update(msg)
	return v, cmd
}

func (v ReleaseEditor) back() (ReleaseEditor, tea.Cmd) {
	switch {
	case v.editing && v.Dirty():
		n := len(v.edits)
		return v, func() tea.Msg { return DiscardReleaseEditsMsg{Count: n} }
	case v.editing:
		v.editing = false
		v.refresh()
		return v, nil
	case v.tab == EditorTasksTab && v.pane == editorPaneFields:
		v.pane = editorPaneTasks
		v.refresh()
		return v, nil
	}
	return v, func() tea.Msg { return CloseReleaseEditorMsg{} }
}

func (v ReleaseEditor) toggleEditing() (ReleaseEditor, tea.Cmd) {
	if v.editing {
		return v.back()
	}
	if !v.doc.Editable() {
		return v, notice("Every stage of " + v.doc.Name + " has deployed, so there's nothing left to edit.")
	}
	v.editing = true
	v.refresh()
	return v, nil
}

func (v *ReleaseEditor) setTab(tab EditorTab) tea.Cmd {
	if v.tab == tab {
		return nil
	}
	v.tab = tab
	v.body.GotoTop()
	var cmd tea.Cmd
	if tab == EditorHistoryTab && v.kind == models.DocDefinition && v.revState == revisionsIdle {
		v.revState = revisionsLoading
		id := v.id
		cmd = func() tea.Msg { return LoadRevisionsMsg{DefinitionID: id} }
	}
	v.refresh()
	return cmd
}

func (v *ReleaseEditor) moveStage(delta int) {
	if v.doc == nil || len(v.doc.Stages) == 0 {
		return
	}
	next := min(max(v.stage+delta, 0), len(v.doc.Stages)-1)
	if next == v.stage {
		return
	}
	v.stage, v.task, v.taskTop, v.field, v.fieldTop = next, 0, 0, 0, 0
	v.refresh()
}

func (v *ReleaseEditor) moveCursor(delta int) {
	if v.pane == editorPaneFields {
		v.field = min(max(v.field+delta, 0), max(len(v.fields())-1, 0))
	} else {
		next := min(max(v.task+delta, 0), max(len(v.taskRows())-1, 0))
		if next != v.task {
			v.task, v.field, v.fieldTop = next, 0, 0
		}
	}
	v.reveal()
}

func (v *ReleaseEditor) moveVariable(delta int) {
	v.variable = min(max(v.variable+delta, 0), max(len(v.scopedVariables())-1, 0))
	v.reveal()
}

func (v ReleaseEditor) handleTasksKey(msg tea.KeyPressMsg) (ReleaseEditor, tea.Cmd) {
	k := v.keys
	switch {
	case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel):
		v.pane = 1 - v.pane
	case key.Matches(msg, k.Left):
		v.pane = editorPaneTasks
	case key.Matches(msg, k.Right):
		v.pane = editorPaneFields
	case key.Matches(msg, k.Up):
		v.moveCursor(-1)
	case key.Matches(msg, k.Down):
		v.moveCursor(1)
	case key.Matches(msg, k.Select) && v.pane == editorPaneTasks:
		if len(v.taskRows()) > 0 {
			v.pane = editorPaneFields
		}
	case key.Matches(msg, k.Select):
		return v.editField()
	case key.Matches(msg, k.ToggleTask):
		return v.toggleTask()
	case key.Matches(msg, k.UndoField):
		return v.undoTask()
	case key.Matches(msg, k.Yank):
		return v, v.copyTask()
	default:
		return v, nil
	}
	v.reveal()
	v.refresh()
	return v, nil
}

func (v *ReleaseEditor) guardEditing() tea.Cmd {
	if !v.editing {
		return notice(startEditingHint)
	}
	if s, ok := v.currentStage(); ok && s.Locked && v.tab == EditorTasksTab {
		return notice(s.Name + " has already deployed in this release, so it's read-only.")
	}
	return nil
}

func (v *ReleaseEditor) where() string {
	s, p, t, isTask, _ := v.currentRow()
	if !isTask {
		return s.Name + " / " + p.Name
	}
	return s.Name + " / " + t.Name
}

func (v ReleaseEditor) editField() (ReleaseEditor, tea.Cmd) {
	if cmd := v.guardEditing(); cmd != nil {
		return v, cmd
	}
	fields := v.fields()
	if v.field >= len(fields) {
		return v, nil
	}
	f := fields[v.field]
	if f.ReadOnly {
		return v, notice(f.Label + " is set in a way lazyado can't edit. Change it in Azure DevOps.")
	}
	if f.Kind == models.InputBool {
		v.Apply(f.Edit(v.where(), f.ToggledValue()))
		return v, nil
	}
	msg := EditReleaseFieldMsg{Field: f, Where: v.where()}
	if f.Kind == models.InputAzure {
		req, err := models.AzureRequestFor(fields, f)
		if err != nil {
			return v, notice(f.Label + " is listed by Azure: " + err.Error() + ".")
		}
		msg.Request, msg.Variables = req, v.variableNames()
	}
	return v, func() tea.Msg { return msg }
}

func (v ReleaseEditor) toggleTask() (ReleaseEditor, tea.Cmd) {
	if cmd := v.guardEditing(); cmd != nil {
		return v, cmd
	}
	if _, _, _, ok := v.currentTask(); !ok {
		return v, notice("Select a task to turn it on or off.")
	}
	for _, f := range v.fields() {
		if f.Key == "$enabled" {
			v.Apply(f.Edit(v.where(), !models.Truthy(f.Value)))
		}
	}
	return v, nil
}

func (v ReleaseEditor) undoTask() (ReleaseEditor, tea.Cmd) {
	if !v.editing {
		return v, notice(startEditingHint)
	}
	s, ok := v.currentStage()
	rows := v.taskRows()
	if !ok || v.task >= len(rows) {
		return v, nil
	}
	row := v.rowPath(s, rows[v.task])
	if v.pane == editorPaneFields {
		if fields := v.fields(); v.field < len(fields) {
			path := fields[v.field].Path
			v.edits = v.edits.Without(path)
			v.dropStale(func(st models.StaleField) bool { return st.Parent.String() == path.String() })
		}
	} else {
		keep := func(e models.DocEdit) bool {
			return !e.Path.Has(row) || rows[v.task].task < 0 && e.Path.Has(row.Child("workflowTasks"))
		}
		v.edits = slices.DeleteFunc(slices.Clone(v.edits), func(e models.DocEdit) bool { return !keep(e) })
		v.dropStale(func(st models.StaleField) bool { return st.Parent.Has(row) && !st.Path.Has(row.Child("workflowTasks")) })
	}
	v.clamp()
	v.refresh()
	return v, nil
}

func (v *ReleaseEditor) copyTask() tea.Cmd {
	_, _, t, ok := v.currentTask()
	if !ok {
		return notice("Select a task to copy it as YAML.")
	}
	text := models.TaskYAML(t, v.refs.Task(t), v.fields())
	return func() tea.Msg { return CopyTextMsg{Text: text, Notice: "Copied " + t.Name + " as YAML"} }
}

func (v ReleaseEditor) handleVariablesKey(msg tea.KeyPressMsg) (ReleaseEditor, tea.Cmd) {
	k := v.keys
	vars := v.scopedVariables()
	switch {
	case key.Matches(msg, k.Left), key.Matches(msg, k.PrevPanel):
		v.scope = max(v.scope-1, 0)
		v.variable, v.varTop = 0, 0
	case key.Matches(msg, k.Right), key.Matches(msg, k.NextPanel):
		v.scope = min(v.scope+1, len(v.doc.Stages)+1)
		v.variable, v.varTop = 0, 0
	case key.Matches(msg, k.Up):
		v.moveVariable(-1)
	case key.Matches(msg, k.Down):
		v.moveVariable(1)
	case len(vars) == 0:
		return v, nil
	case key.Matches(msg, k.Select):
		if !v.editing {
			return v, notice(startEditingHint)
		}
		variable := vars[v.variable]
		if s, ok := v.doc.StageOfIndex(variable.Stage); ok && s.Locked {
			return v, notice(s.Name + " has already deployed in this release, so its variables are read-only.")
		}
		f := v.doc.VariableField(variable, v.edits)
		return v, func() tea.Msg { return EditReleaseFieldMsg{Field: f, Where: "Variables / " + variable.Scope} }
	case key.Matches(msg, k.UndoField):
		if !v.editing {
			return v, notice(startEditingHint)
		}
		v.edits = v.edits.Without(v.doc.VariablePath(vars[v.variable]))
	case key.Matches(msg, k.Yank):
		variable := vars[v.variable]
		if variable.Secret {
			return v, notice("Azure DevOps never sends secret values back, so there's nothing to copy.")
		}
		value := models.FormatDocValue(v.edits.Value(v.doc.VariablePath(variable), variable.Value))
		return v, func() tea.Msg { return CopyTextMsg{Text: value, Notice: "Copied the value of " + variable.Name} }
	default:
		return v, nil
	}
	v.refresh()
	return v, nil
}

func (v *ReleaseEditor) taskEditCount() int {
	n := 0
	for _, e := range v.edits {
		if len(e.Path) > 2 && e.Path[0] == "environments" && e.Path[2] != "variables" {
			n++
		}
	}
	return n
}

func (v *ReleaseEditor) Mode() string {
	switch {
	case v.editing:
		return "EDIT"
	case v.kind == models.DocRelease:
		return "RELEASE"
	}
	return "PIPELINE"
}

func (v *ReleaseEditor) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("tab", k.Panel1, k.Panel4)
	back := widgets.Relabel(k.Back, "back")
	edit := widgets.Relabel(k.EditPipeline, "edit")
	if v.editing {
		back = widgets.Relabel(k.Back, "stop editing")
		edit = widgets.Relabel(k.EditPipeline, "stop editing")
	}
	if v.doc == nil {
		return []key.Binding{back}
	}
	save := func() key.Binding {
		b := widgets.Relabel(k.Save, "review and save")
		b.SetEnabled(v.Dirty())
		return b
	}
	var bindings []key.Binding
	switch v.tab {
	case EditorPipelineTab:
		bindings = []key.Binding{widgets.Combine("stage", k.Left, k.Right), widgets.Relabel(k.Select, "tasks")}
	case EditorTasksTab:
		bindings = []key.Binding{widgets.Combine("stage", k.PrevStage, k.NextStage), widgets.Relabel(k.NextPanel, "pane")}
		toggle := widgets.Relabel(k.ToggleTask, "task on/off")
		switch {
		case v.editing && v.pane == editorPaneFields:
			bindings = append(bindings, widgets.Relabel(k.Select, "change"), save(), toggle, widgets.Relabel(k.UndoField, "undo field"))
		case v.editing:
			bindings = append(bindings, widgets.Relabel(k.Select, "fields"), save(), toggle, widgets.Relabel(k.UndoField, "undo task"))
		default:
			bindings = append(bindings, widgets.Relabel(k.Select, "fields"))
		}
		bindings = append(bindings, widgets.Relabel(k.Yank, "copy YAML"))
	case EditorVariablesTab:
		bindings = []key.Binding{widgets.Combine("scope", k.Left, k.Right)}
		if v.editing {
			bindings = append(bindings, widgets.Relabel(k.Select, "change"), save(), widgets.Relabel(k.UndoField, "undo"))
		}
		bindings = append(bindings, widgets.Relabel(k.Yank, "copy value"))
	}
	if v.tab != EditorTasksTab && v.tab != EditorVariablesTab || !v.editing {
		bindings = append(bindings, save())
	}
	return slices.DeleteFunc(append(bindings, edit, tabs, k.OpenBrowser, back), func(b key.Binding) bool { return !b.Enabled() })
}

func editCountLabel(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d)", n)
}

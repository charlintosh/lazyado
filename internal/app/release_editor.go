package ui

import (
	"image/color"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/debug"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

var logReleaseEditor = debug.Scope("release_editor")

var guidValue = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type releaseDocLoadedMsg struct {
	seq  int
	doc  *models.ReleaseDoc
	refs models.ReleaseDocRefs
	err  error
}

type releaseDocRefsMsg struct {
	seq  int
	refs models.ReleaseDocRefs
}

type releaseDocSavedMsg struct {
	doc *models.ReleaseDoc
}

type releaseDocConflictMsg struct {
	latest *models.ReleaseDoc
}

type releaseDocSaveFailedMsg struct {
	err error
}

type definitionRevisionsMsg struct {
	id   int
	revs []models.DefinitionRevision
	err  error
}

type agentQueuesMsg struct {
	queues []models.AgentQueue
	field  models.TaskField
	where  string
	err    error
}

type azureListMsg struct {
	key    string
	values []models.AzureValue
	err    error
}

func listQueuesCmd(client *api.Client, field models.TaskField, where string) tea.Cmd {
	return func() tea.Msg {
		queues, err := client.ListAgentQueues()
		return agentQueuesMsg{queues: queues, field: field, where: where, err: err}
	}
}

func azureListCmd(client *api.Client, req models.AzureListRequest) tea.Cmd {
	return func() tea.Msg {
		values, err := client.AzureList(req)
		return azureListMsg{key: req.Key(), values: values, err: err}
	}
}

type serviceEndpointsMsg struct {
	kind      string
	endpoints []models.ServiceEndpoint
	field     models.TaskField
	where     string
	err       error
}

func loadReleaseDocCmd(client *api.Client, seq int, kind models.ReleaseDocKind, id int, known map[string]*models.TaskDefinition) tea.Cmd {
	return func() tea.Msg {
		doc, err := client.GetReleaseDoc(kind, id)
		if err != nil {
			return releaseDocLoadedMsg{seq: seq, err: err}
		}
		return releaseDocLoadedMsg{seq: seq, doc: doc, refs: loadDocRefs(client, doc, known)}
	}
}

func loadReleaseDocRefsCmd(client *api.Client, seq int, doc *models.ReleaseDoc, known map[string]*models.TaskDefinition) tea.Cmd {
	return func() tea.Msg {
		return releaseDocRefsMsg{seq: seq, refs: loadDocRefs(client, doc, known)}
	}
}

func loadDocRefs(client *api.Client, doc *models.ReleaseDoc, known map[string]*models.TaskDefinition) models.ReleaseDocRefs {
	refs := models.ReleaseDocRefs{Tasks: maps.Clone(known)}
	if refs.Tasks == nil {
		refs.Tasks = map[string]*models.TaskDefinition{}
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, ref := range doc.TaskRefs() {
		if _, ok := refs.Tasks[ref.Key()]; ok {
			continue
		}
		wg.Go(func() {
			def, err := client.GetTaskDefinition(ref)
			if err != nil {
				logReleaseEditor.Debugf("task %s: %v", ref.Key(), err)
			}
			mu.Lock()
			refs.Tasks[ref.Key()] = def
			mu.Unlock()
		})
	}
	wg.Go(func() {
		queues, err := client.AgentQueues(doc.QueueIDs())
		if err != nil {
			logReleaseEditor.Debugf("queues: %v", err)
		}
		mu.Lock()
		refs.Queues = queues
		mu.Unlock()
	})
	wg.Go(func() {
		groups, err := client.VariableGroupInfo(doc.GroupIDs())
		if err != nil {
			logReleaseEditor.Debugf("variable groups: %v", err)
		}
		mu.Lock()
		refs.Groups = groups
		mu.Unlock()
	})
	wg.Wait()
	names, err := client.ServiceEndpointNames(endpointIDs(doc, refs))
	if err != nil {
		logReleaseEditor.Debugf("service connections: %v", err)
	}
	refs.Endpoints = names
	return refs
}

func endpointIDs(doc *models.ReleaseDoc, refs models.ReleaseDocRefs) []string {
	var ids []string
	add := func(value string) {
		for id := range strings.SplitSeq(value, ",") {
			id = strings.ToLower(strings.TrimSpace(id))
			if guidValue.MatchString(id) && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	for _, s := range doc.Stages {
		for _, p := range s.Phases {
			for _, t := range p.Tasks {
				def := refs.Task(t)
				if def == nil {
					for _, v := range t.Inputs {
						add(v)
					}
					continue
				}
				for _, in := range def.Inputs {
					if in.Kind() == models.InputEndpoint {
						add(t.Inputs[in.Name])
					}
				}
			}
		}
	}
	slices.Sort(ids)
	return ids
}

func saveReleaseDocCmd(client *api.Client, doc *models.ReleaseDoc, edits models.DocEdits, comment string) tea.Cmd {
	return func() tea.Msg {
		saved, err := client.SaveReleaseDoc(doc, edits, comment)
		if err == nil {
			return releaseDocSavedMsg{doc: saved}
		}
		if doc.Kind == models.DocDefinition {
			if latest, lerr := client.GetReleaseDoc(doc.Kind, doc.ID); lerr == nil && latest.Revision != doc.Revision {
				return releaseDocConflictMsg{latest: latest}
			}
		}
		return releaseDocSaveFailedMsg{err: err}
	}
}

func loadRevisionsCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		revs, err := client.ListDefinitionRevisions(id)
		return definitionRevisionsMsg{id: id, revs: revs, err: err}
	}
}

func listEndpointsCmd(client *api.Client, kind string, field models.TaskField, where string) tea.Cmd {
	return func() tea.Msg {
		list, err := client.ListServiceEndpoints(kind)
		return serviceEndpointsMsg{kind: kind, endpoints: list, field: field, where: where, err: err}
	}
}

func (a *App) viewingReleaseEditor() bool { return a.viewMode == ViewReleaseEditor }

func (a *App) openReleaseEditor(kind models.ReleaseDocKind, id int) tea.Cmd {
	if a.viewMode != ViewReleaseEditor {
		a.rl.editorFrom = a.viewMode
	}
	a.viewMode = ViewReleaseEditor
	a.rl.editor.Open(kind, id, panels.EditorPipelineTab)
	a.rl.editorSeq++
	return loadReleaseDocCmd(a.client, a.rl.editorSeq, kind, id, a.rl.taskDefs)
}

func (a *App) refreshReleaseEditor() tea.Cmd {
	e := &a.rl.editor
	if e.Dirty() {
		return a.info("Save or discard your changes before reloading.")
	}
	if e.ID() == 0 {
		return nil
	}
	a.rl.azureLists, a.rl.queues = nil, nil
	a.rl.editorSeq++
	return loadReleaseDocCmd(a.client, a.rl.editorSeq, e.Kind(), e.ID(), a.rl.taskDefs)
}

func (a *App) closeReleaseEditor() {
	a.viewMode = a.rl.editorFrom
	a.rl.editorSeq++
	a.rl.conflict = nil
}

func (a *App) rememberTasks(tasks map[string]*models.TaskDefinition) {
	merged := maps.Clone(a.rl.taskDefs)
	if merged == nil {
		merged = map[string]*models.TaskDefinition{}
	}
	maps.Copy(merged, tasks)
	a.rl.taskDefs = merged
}

func (a *App) editEndpointField(field models.TaskField, where string) tea.Cmd {
	if list, ok := a.rl.endpoints[field.EndpointType]; ok {
		return a.openModal(modals.NewReleaseFieldModal(a.env(), field, where, modals.FieldChoices{Options: endpointOptions(list)}))
	}
	return a.mutate(listEndpointsCmd(a.client, field.EndpointType, field, where))
}

func endpointOptions(list []models.ServiceEndpoint) []models.TaskOption {
	out := make([]models.TaskOption, len(list))
	for i, e := range list {
		out[i] = models.TaskOption{Value: e.ID, Label: e.Name, Detail: e.Type}
	}
	return out
}

func queueOptions(list []models.AgentQueue) []models.TaskOption {
	out := make([]models.TaskOption, len(list))
	for i, q := range list {
		detail := "self-hosted"
		if q.Hosted {
			detail = "hosted by Microsoft"
		}
		out[i] = models.TaskOption{Value: itoa(q.ID), Label: q.Name, Detail: detail}
	}
	return out
}

func (a *App) editQueueField(field models.TaskField, where string) tea.Cmd {
	if a.rl.queues != nil {
		return a.openModal(modals.NewReleaseFieldModal(a.env(), field, where, modals.FieldChoices{Options: queueOptions(a.rl.queues)}))
	}
	return a.mutate(listQueuesCmd(a.client, field, where))
}

func (a *App) editAzureField(msg panels.EditReleaseFieldMsg) tea.Cmd {
	choices := modals.FieldChoices{Azure: true, Request: msg.Request, Variables: msg.Variables}
	values, cached := a.rl.azureLists[msg.Request.Key()]
	choices.Values, choices.Loaded = values, cached
	open := a.openModal(modals.NewReleaseFieldModal(a.env(), msg.Field, msg.Where, choices))
	if cached {
		return open
	}
	return tea.Batch(open, azureListCmd(a.client, msg.Request))
}

func (a *App) rememberAzureList(key string, values []models.AzureValue) {
	lists := maps.Clone(a.rl.azureLists)
	if lists == nil {
		lists = map[string][]models.AzureValue{}
	}
	lists[key] = values
	a.rl.azureLists = lists
	refs := a.rl.editor.Refs()
	names := maps.Clone(refs.AzureNames)
	if names == nil {
		names = map[string]string{}
	}
	for _, v := range values {
		if v.Label != v.Value {
			names[strings.ToLower(v.Value)] = v.Label
		}
	}
	refs.AzureNames = names
	a.rl.editor.SetRefs(refs)
}

func (a *App) labeledEdit(msg modals.ReleaseFieldEditedMsg) models.DocEdit {
	edit := msg.Field.Edit(msg.Where, msg.Value)
	refs := a.rl.editor.Refs()
	names := map[string]string{}
	switch msg.Field.Kind {
	case models.InputEndpoint:
		names = maps.Clone(refs.Endpoints)
		if names == nil {
			names = map[string]string{}
		}
		for _, e := range a.rl.endpoints[msg.Field.EndpointType] {
			names[strings.ToLower(e.ID)] = e.Name
		}
		refs.Endpoints = names
	case models.InputQueue:
		queues := maps.Clone(refs.Queues)
		if queues == nil {
			queues = map[int]models.AgentQueue{}
		}
		for _, q := range a.rl.queues {
			queues[q.ID] = q
			names[itoa(q.ID)] = q.Name
		}
		for id, q := range queues {
			names[itoa(id)] = q.Name
		}
		refs.Queues = queues
	case models.InputAzure:
		maps.Copy(names, refs.AzureNames)
	default:
		return edit
	}
	a.rl.editor.SetRefs(refs)
	if n := names[strings.ToLower(edit.BeforeText)]; n != "" {
		edit.BeforeText = n
	}
	if n := names[strings.ToLower(edit.AfterText)]; n != "" {
		edit.AfterText = n
	}
	return edit
}

func reapplyNotice(revision int, kept, dropped models.DocEdits) string {
	total := len(kept) + len(dropped)
	head := "Reloaded revision " + itoa(revision)
	if len(dropped) == 0 {
		return head + " and reapplied your " + plural(total, "change") + ". Press Ctrl+s to save."
	}
	var names []string
	for _, e := range dropped {
		names = append(names, e.Field+" ("+e.Where+")")
	}
	return head + ": reapplied " + itoa(len(kept)) + " of " + itoa(total) + ". Someone else also changed " + strings.Join(names, ", ") + ", so those were left as they are now."
}

func (a App) handleReleaseEditorResult(msg tea.Msg) (App, tea.Cmd, bool) {
	c := a.client
	e := &a.rl.editor
	switch msg := msg.(type) {
	case panels.OpenReleaseEditorMsg:
		return a, a.openReleaseEditor(msg.Kind, msg.ID), true
	case panels.CloseReleaseEditorMsg:
		a.closeReleaseEditor()
		return a, nil, true
	case releaseDocLoadedMsg:
		if msg.seq != a.rl.editorSeq || !a.viewingReleaseEditor() {
			return a, nil, true
		}
		if msg.err != nil {
			if e.Doc() == nil {
				a.closeReleaseEditor()
			}
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.rememberTasks(msg.refs.Tasks)
		e.SetDoc(msg.doc, msg.refs)
		return a, nil, true
	case releaseDocRefsMsg:
		if msg.seq == a.rl.editorSeq {
			a.rememberTasks(msg.refs.Tasks)
			e.SetRefs(msg.refs)
		}
		return a, nil, true
	case panels.LoadRevisionsMsg:
		return a, loadRevisionsCmd(c, msg.DefinitionID), true
	case definitionRevisionsMsg:
		e.SetRevisions(msg.id, msg.revs, msg.err)
		return a, nil, true
	case panels.EditReleaseFieldMsg:
		switch {
		case msg.Field.Kind == models.InputEndpoint && msg.Field.EndpointType != "":
			return a, a.editEndpointField(msg.Field, msg.Where), true
		case msg.Field.Kind == models.InputQueue:
			return a, a.editQueueField(msg.Field, msg.Where), true
		case msg.Field.Kind == models.InputAzure:
			return a, a.editAzureField(msg), true
		}
		return a, a.openModal(modals.NewReleaseFieldModal(a.env(), msg.Field, msg.Where, modals.FieldChoices{})), true
	case agentQueuesMsg:
		a.loading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.rl.queues = msg.queues
		if !a.viewingReleaseEditor() || a.modals.open() {
			return a, nil, true
		}
		return a, a.openModal(modals.NewReleaseFieldModal(a.env(), msg.field, msg.where, modals.FieldChoices{Options: queueOptions(msg.queues)})), true
	case azureListMsg:
		if msg.err == nil {
			a.rememberAzureList(msg.key, msg.values)
		}
		if !topIs[modals.ReleaseFieldModal](a.modals) {
			return a, nil, true
		}
		next, cmd := a.updateModal(modals.AzureValuesMsg{Key: msg.key, Values: msg.values, Err: msg.err})
		return next, cmd, true
	case modals.RetryAzureListMsg:
		return a, azureListCmd(c, msg.Request), true
	case modals.BrowseReleaseDocMsg:
		if d := e.Doc(); d != nil {
			return a, openBrowserCmd(d.WebURL), true
		}
		return a, nil, true
	case modals.JumpToStaleMsg:
		a.closeModal()
		e.JumpToStale()
		return a, nil, true
	case serviceEndpointsMsg:
		a.loading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		endpoints := maps.Clone(a.rl.endpoints)
		if endpoints == nil {
			endpoints = map[string][]models.ServiceEndpoint{}
		}
		endpoints[msg.kind] = msg.endpoints
		a.rl.endpoints = endpoints
		if !a.viewingReleaseEditor() || a.modals.open() {
			return a, nil, true
		}
		return a, a.openModal(modals.NewReleaseFieldModal(a.env(), msg.field, msg.where, modals.FieldChoices{Options: endpointOptions(msg.endpoints)})), true
	case modals.ReleaseFieldEditedMsg:
		a.closeModal()
		e.Apply(a.labeledEdit(msg))
		return a, nil, true
	case panels.ReviewReleaseEditsMsg:
		if e.Doc() == nil {
			return a, nil, true
		}
		return a, a.openModal(modals.NewReleaseSaveModal(a.env(), e.Doc(), e.Edits(), e.Stale())), true
	case modals.SaveReleaseEditsRequestMsg:
		if e.Doc() == nil || !e.Dirty() {
			return a, nil, true
		}
		return a, a.mutate(saveReleaseDocCmd(c, e.Doc(), e.Edits(), msg.Comment)), true
	case releaseDocSavedMsg:
		a.loading = false
		if topIs[modals.ReleaseSaveModal](a.modals) {
			a.closeModal()
		}
		if !e.Matches(msg.doc.Kind, msg.doc.ID) {
			return a, nil, true
		}
		e.Saved(msg.doc)
		notice := "Saved " + msg.doc.Name
		if msg.doc.Kind == models.DocDefinition {
			notice = "Saved revision " + itoa(msg.doc.Revision) + " of " + msg.doc.Name
		}
		cmds := []tea.Cmd{a.notify(notice)}
		if e.WantsRevisions() {
			cmds = append(cmds, loadRevisionsCmd(c, msg.doc.ID))
		}
		if msg.doc.Kind == models.DocRelease {
			cmds = append(cmds, a.afterReleaseChange(msg.doc.ID))
		}
		return a, tea.Batch(cmds...), true
	case releaseDocConflictMsg:
		a.loading = false
		a.rl.conflict = msg.latest
		conflict := modals.ReleaseSaveConflictMsg{Revision: msg.latest.Revision, By: msg.latest.ModifiedBy, When: msg.latest.Modified}
		if topIs[modals.ReleaseSaveModal](a.modals) {
			next, cmd := a.updateModal(conflict)
			return next, cmd, true
		}
		return a, a.info("Revision " + itoa(msg.latest.Revision) + " was saved while you were editing. Press Ctrl+s to review."), true
	case releaseDocSaveFailedMsg:
		a.loading = false
		if topIs[modals.ReleaseSaveModal](a.modals) {
			next, cmd := a.updateModal(modals.ReleaseSaveFailedMsg{Err: msg.err})
			return next, cmd, true
		}
		return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
	case modals.ReapplyReleaseEditsMsg:
		latest := a.rl.conflict
		if latest == nil || !e.Matches(latest.Kind, latest.ID) {
			return a, nil, true
		}
		a.closeModal()
		kept, dropped := models.ReapplyEdits(latest, e.Edits())
		e.Reload(latest, kept)
		a.rl.conflict = nil
		a.rl.editorSeq++
		return a, tea.Batch(a.info(reapplyNotice(latest.Revision, kept, dropped)), loadReleaseDocRefsCmd(c, a.rl.editorSeq, latest, a.rl.taskDefs)), true
	case panels.DiscardReleaseEditsMsg:
		return a, a.openModal(modals.NewDiscardReleaseEditsModal(a.env(), msg.Count)), true
	case modals.DiscardReleaseEditsConfirmedMsg:
		a.closeModal()
		e.Discard()
		return a, a.notify("Discarded your changes"), true
	}
	return a, nil, false
}

func (a *App) renderReleaseEditor() string {
	l := a.shell
	crumbs := []string{panels.HubReleases.Section(), panels.HubReleases.Name()}
	if d := a.rl.editor.Doc(); d != nil {
		crumbs = append(crumbs, d.Definition)
		if d.Kind == models.DocRelease {
			crumbs = append(crumbs, d.Name)
		}
	}
	body := panels.Block(a.rl.editor.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader(crumbs...), body, a.renderStatusBar())
}

func (a *App) modeColor() color.Color {
	if a.viewingReleaseEditor() && a.rl.editor.Editing() {
		return styles.ColorOrange
	}
	return styles.ColorPrimary
}

func itoa(n int) string { return strconv.Itoa(n) }

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return itoa(n) + " " + noun + "s"
}

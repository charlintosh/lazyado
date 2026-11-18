package ui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	pipelinesRecent = iota
	pipelinesAll
	pipelinesRuns
)

const (
	pipelineRuns = iota
	pipelineBranches
)

const (
	recentPipelines        = 10
	runsPageSize           = 50
	previewRuns            = 10
	pipelinesPageSize      = 200
	listPollInterval       = 15 * time.Second
	pipelineSearchDebounce = 400 * time.Millisecond
	runPollInterval        = 5 * time.Second
)

var (
	pipelinePivots     = []string{"Recent", "All", "Runs"}
	pipelinePagePivots = []string{"Runs", "Branches"}
)

type pipelineState struct {
	panel     panels.PipelinesPanel
	view      panels.PipelineRunView
	pivot     int
	page      *models.Pipeline
	pagePivot int
	loaded    bool
	loading   bool
	seq       int
	token     string
	pipelines []models.Pipeline
	runs      []models.PipelineRun
	pattern   string
	query     string
	pollSeq   int
	searchSeq int

	previewRun      int
	previewPipeline int
	stages          []models.RunStage
	stagesLoaded    bool
	history         []models.PipelineRun
	historyLoaded   bool

	runSeq     int
	fromReview bool

	stageCache map[int]stageEntry
}

type stageEntry struct {
	stages  []models.RunStage
	state   models.RunState
	loading bool
	stale   bool
}

type pipelineListMsg struct {
	seq       int
	mode      panels.PipelineList
	pipelines []models.Pipeline
	runs      []models.PipelineRun
	token     string
	more      bool
	silent    bool
	searchErr string
	err       error
}

type pipelinePollMsg struct {
	seq int
}

type pipelineSearchDueMsg struct {
	seq  int
	text string
}

type pipelinePreviewDueMsg struct {
	run      int
	pipeline int
}

type pipelineStagesMsg struct {
	run    int
	stages []models.RunStage
}

type pipelineHistoryMsg struct {
	pipeline int
	runs     []models.PipelineRun
}

type listRequest struct {
	seq      int
	mode     panels.PipelineList
	pattern  string
	query    string
	pipeline int
	token    string
	more     bool
	silent   bool
	me       models.Identity
}

func loadPipelineListCmd(client *api.Client, req listRequest) tea.Cmd {
	return func() tea.Msg {
		msg := pipelineListMsg{seq: req.seq, mode: req.mode, more: req.more, silent: req.silent}
		switch req.mode {
		case panels.ListRecent:
			msg.pipelines, msg.err = client.RecentPipelines(recentPipelines)
		case panels.ListAll:
			msg.pipelines, msg.token, msg.err = client.ListPipelines(req.pattern, req.token)
		default:
			criteria, searchErr := runCriteria(client, req)
			if searchErr != "" {
				msg.searchErr = searchErr
				return msg
			}
			msg.runs, msg.token, msg.err = client.ListPipelineRuns(criteria, runsPageSize, req.token)
		}
		return msg
	}
}

func runCriteria(client *api.Client, req listRequest) (models.RunCriteria, string) {
	q := models.ParseRunQuery(req.query)
	if req.pipeline != 0 {
		q.Pipeline = ""
	}
	var candidates []models.Pipeline
	if q.Pipeline != "" {
		found, _, err := client.ListPipelines(models.NamePattern(q.Pipeline), "")
		if err != nil {
			return models.RunCriteria{}, err.Error()
		}
		candidates = found
	}
	criteria, err := q.Criteria(candidates, req.me)
	if err != nil {
		return criteria, err.Error()
	}
	if req.pipeline != 0 {
		criteria.PipelineID = req.pipeline
	}
	return criteria, ""
}

func schedulePipelinePollCmd(seq int) tea.Cmd {
	return tea.Tick(listPollInterval, func(time.Time) tea.Msg { return pipelinePollMsg{seq: seq} })
}

func schedulePipelineSearchCmd(seq int, text string) tea.Cmd {
	return tea.Tick(pipelineSearchDebounce, func(time.Time) tea.Msg { return pipelineSearchDueMsg{seq: seq, text: text} })
}

func schedulePipelinePreviewCmd(run, pipeline int) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg { return pipelinePreviewDueMsg{run: run, pipeline: pipeline} })
}

func loadPipelineStagesCmd(client *api.Client, run int) tea.Cmd {
	return func() tea.Msg {
		records, err := client.GetRunTimeline(run)
		if err != nil {
			records = nil
		}
		return pipelineStagesMsg{run: run, stages: models.BuildStages(records)}
	}
}

func loadPipelineHistoryCmd(client *api.Client, pipeline int) tea.Cmd {
	return func() tea.Msg {
		runs, _, err := client.ListPipelineRuns(models.RunCriteria{PipelineID: pipeline}, previewRuns, "")
		if err != nil {
			runs = nil
		}
		return pipelineHistoryMsg{pipeline: pipeline, runs: runs}
	}
}

func (s *pipelineState) cachedStages(id int) (stageEntry, bool) {
	e, ok := s.stageCache[id]
	return e, ok
}

func (s *pipelineState) requestStages(run models.PipelineRun) bool {
	if s.stageCache == nil {
		s.stageCache = map[int]stageEntry{}
	}
	e, ok := s.stageCache[run.ID]
	if ok && (e.loading || (!e.stale && e.state == run.State())) {
		return false
	}
	e.loading, e.state = true, run.State()
	s.stageCache[run.ID] = e
	return true
}

func (s *pipelineState) storeStages(id int, stages []models.RunStage) {
	if s.stageCache == nil {
		s.stageCache = map[int]stageEntry{}
	}
	e := s.stageCache[id]
	e.stages, e.loading, e.stale = stages, false, false
	s.stageCache[id] = e
}

func (s *pipelineState) markActiveStale() {
	mark := func(r *models.PipelineRun) {
		if r == nil || !r.State().Active() {
			return
		}
		if e, ok := s.stageCache[r.ID]; ok && !e.loading {
			e.stale = true
			s.stageCache[r.ID] = e
		}
	}
	for i := range s.runs {
		mark(&s.runs[i])
	}
	for i := range s.pipelines {
		mark(s.pipelines[i].Latest)
	}
}

func (a *App) syncListStages() tea.Cmd {
	if !a.inPipelines() || a.viewMode != ViewMain || !a.pl.panel.StagesShown() {
		return nil
	}
	var cmds []tea.Cmd
	for _, run := range a.pl.panel.VisibleRuns() {
		if a.pl.requestStages(run) {
			cmds = append(cmds, loadPipelineStagesCmd(a.client, run.ID))
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) inPipelines() bool {
	return a.hub == panels.HubBuilds
}

func (a *App) searchingPipelines() bool {
	return a.viewMode == ViewMain && a.inPipelines() && !a.modals.open() && a.pl.panel.Searching()
}

func (a *App) pipelineMode() panels.PipelineList {
	switch {
	case a.pl.page != nil && a.pl.pagePivot == pipelineBranches:
		return panels.ListBranches
	case a.pl.page != nil:
		return panels.ListPipelineRuns
	case a.pl.pivot == pipelinesAll:
		return panels.ListAll
	case a.pl.pivot == pipelinesRuns:
		return panels.ListRuns
	}
	return panels.ListRecent
}

func (a *App) openPipelines() tea.Cmd {
	a.switchHub(panels.HubBuilds)
	switch {
	case a.pl.loading:
		return nil
	case a.pl.loaded && a.activeRuns():
		a.pl.pollSeq++
		return schedulePipelinePollCmd(a.pl.pollSeq)
	case a.pl.loaded:
		return nil
	}
	return a.loadPipelineList()
}

func sameList(a, b panels.PipelineList) bool {
	page := func(m panels.PipelineList) bool { return m == panels.ListPipelineRuns || m == panels.ListBranches }
	return a == b || (page(a) && page(b))
}

func (a *App) listRequest() listRequest {
	req := listRequest{seq: a.pl.seq, mode: a.pipelineMode(), pattern: a.pl.pattern, query: a.pl.query, me: a.me}
	if a.pl.page != nil {
		req.pipeline = a.pl.page.ID
	}
	return req
}

func (a *App) loadPipelineList() tea.Cmd {
	a.pl.seq++
	a.pl.loading, a.pl.loaded = true, false
	a.pl.token, a.pl.pipelines, a.pl.runs = "", nil, nil
	mode := a.pipelineMode()
	a.pl.panel.ShowLoading(mode)
	if text := a.searchText(); text != "" {
		a.pl.panel.SetSearchText(text)
	}
	return loadPipelineListCmd(a.client, a.listRequest())
}

func (a *App) searchText() string {
	if a.pipelineMode() == panels.ListAll {
		return a.pl.patternText()
	}
	if a.pipelineMode() != panels.ListRecent {
		return a.pl.query
	}
	return ""
}

func (s *pipelineState) patternText() string {
	if len(s.pattern) < 2 {
		return ""
	}
	return s.pattern[1 : len(s.pattern)-1]
}

func (a *App) loadMorePipelines() tea.Cmd {
	if a.pl.token == "" || a.pl.loading || a.pl.panel.Loading() {
		return nil
	}
	a.pl.panel.SetLoadingMore(true)
	req := a.listRequest()
	req.token, req.more = a.pl.token, true
	return loadPipelineListCmd(a.client, req)
}

func (a *App) pollPipelines() tea.Cmd {
	if a.pl.loading || a.pl.panel.Loading() || a.viewMode != ViewMain || !a.inPipelines() {
		return nil
	}
	if a.pipelineMode() == panels.ListAll && a.pl.token != "" && len(a.pl.pipelines) > pipelinesPageSize {
		return nil
	}
	req := a.listRequest()
	req.silent = true
	return loadPipelineListCmd(a.client, req)
}

func (a *App) selectPipelinePivot(pivot int) tea.Cmd {
	if a.pl.page == nil && pivot == a.pl.pivot {
		return nil
	}
	a.pl.page, a.pl.pivot = nil, pivot
	a.pl.query = ""
	if pivot != pipelinesAll {
		a.pl.pattern = ""
	}
	return a.loadPipelineList()
}

func (a *App) cyclePipelinePivot(delta int) tea.Cmd {
	if a.pl.page != nil {
		return a.selectPagePivot((a.pl.pagePivot + delta + len(pipelinePagePivots)) % len(pipelinePagePivots))
	}
	return a.selectPipelinePivot((a.pl.pivot + delta + len(pipelinePivots)) % len(pipelinePivots))
}

func (a *App) selectPagePivot(pivot int) tea.Cmd {
	if a.pl.page == nil || pivot == a.pl.pagePivot {
		return nil
	}
	a.pl.pagePivot = pivot
	if !a.pl.loaded {
		return nil
	}
	a.pl.panel.SetRuns(a.pipelineMode(), a.pl.runs, a.pl.token != "")
	a.pl.panel.SetSearchText(a.pl.query)
	return nil
}

func (a *App) openPipelinePage(p models.Pipeline) tea.Cmd {
	a.pl.page, a.pl.pagePivot, a.pl.query = &p, pipelineRuns, ""
	return a.loadPipelineList()
}

func (a *App) closePipelinePage() tea.Cmd {
	a.pl.page, a.pl.query = nil, ""
	return a.loadPipelineList()
}

func (a *App) searchPipelines(text string) tea.Cmd {
	a.pl.searchSeq++
	a.pl.panel.SetRemoteSearch(false)
	a.applySearch(text)
	return a.loadPipelineList()
}

func (a *App) searchPipelinesLive(text string) tea.Cmd {
	if text == a.searchText() {
		return nil
	}
	a.applySearch(text)
	a.pl.seq++
	a.pl.loading = true
	a.pl.token = ""
	a.pl.panel.SetRemoteSearch(true)
	return loadPipelineListCmd(a.client, a.listRequest())
}

func (a *App) applySearch(text string) {
	switch a.pipelineMode() {
	case panels.ListRecent:
		a.pl.pivot = pipelinesAll
		a.pl.pattern = models.NamePattern(text)
	case panels.ListAll:
		a.pl.pattern = models.NamePattern(text)
	default:
		a.pl.query = text
	}
}

func (a *App) refreshPipelines() tea.Cmd {
	cmds := []tea.Cmd{a.loadPipelineList()}
	if !a.me.Known() {
		cmds = append(cmds, loadIdentityCmd(a.client))
	}
	a.pl.previewRun, a.pl.previewPipeline = 0, 0
	a.pl.stageCache = nil
	return tea.Batch(cmds...)
}

func (a *App) selectedPipeline() *models.Pipeline {
	if a.pl.page != nil {
		return a.pl.page
	}
	return a.pl.panel.SelectedPipeline()
}

func (a *App) openRunForm(p models.Pipeline, ref string) tea.Cmd {
	return a.openModal(modals.NewRunPipelineForm(a.env(), p, ref))
}

func (a *App) syncPipelinePreview() tea.Cmd {
	if !a.inPipelines() || a.viewMode != ViewMain || !a.shell.showPane {
		return nil
	}
	run, pipeline := 0, 0
	if r := a.pl.panel.SelectedRun(); r != nil {
		run = r.ID
	}
	if p := a.pl.panel.SelectedPipeline(); p != nil {
		pipeline = p.ID
	}
	if run == a.pl.previewRun && pipeline == a.pl.previewPipeline {
		return nil
	}
	if run != a.pl.previewRun {
		a.pl.stages, a.pl.stagesLoaded = nil, run == 0
		if e, ok := a.pl.cachedStages(run); ok && !e.loading && !e.stale {
			a.pl.stages, a.pl.stagesLoaded = e.stages, true
		}
	}
	if pipeline != a.pl.previewPipeline {
		a.pl.history, a.pl.historyLoaded = nil, pipeline == 0
	}
	a.pl.previewRun, a.pl.previewPipeline = run, pipeline
	if run == 0 && pipeline == 0 {
		return nil
	}
	return schedulePipelinePreviewCmd(run, pipeline)
}

func (a *App) previewLoading() bool {
	return a.inPipelines() && a.shell.showPane && ((a.pl.previewRun != 0 && !a.pl.stagesLoaded) || (a.pl.previewPipeline != 0 && !a.pl.historyLoaded))
}

func (a *App) activeRuns() bool {
	for _, r := range a.pl.runs {
		if r.State().Active() {
			return true
		}
	}
	for _, p := range a.pl.pipelines {
		if p.Latest != nil && p.Latest.State().Active() {
			return true
		}
	}
	return false
}

func mergeRuns(old, fresh []models.PipelineRun) []models.PipelineRun {
	if len(fresh) == 0 {
		return old
	}
	seen := make(map[int]bool, len(fresh))
	for _, r := range fresh {
		seen[r.ID] = true
	}
	oldest := fresh[len(fresh)-1].When()
	out := append([]models.PipelineRun(nil), fresh...)
	for _, r := range old {
		if !seen[r.ID] && r.When().Before(oldest) {
			out = append(out, r)
		}
	}
	return out
}

func (a App) handlePipelineKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	if a.pl.panel.Searching() {
		var cmd tea.Cmd
		a.pl.panel, cmd = a.pl.panel.Update(msg)
		return a, cmd
	}
	pivots := a.filterKeys()[:len(pipelinePivots)]
	if a.pl.page != nil {
		pivots = pivots[:len(pipelinePagePivots)]
	}
	for i, b := range pivots {
		if key.Matches(msg, b) {
			if a.pl.page != nil {
				return a, a.selectPagePivot(i)
			}
			return a, a.selectPipelinePivot(i)
		}
	}
	if hub, ok := a.hubFor(msg); ok {
		return a, a.goToHub(hub)
	}
	switch {
	case key.Matches(msg, k.Back) && a.pl.page != nil:
		return a, a.closePipelinePage()
	case key.Matches(msg, k.PrevView), key.Matches(msg, k.PrevPanel), key.Matches(msg, k.Left):
		return a, a.cyclePipelinePivot(-1)
	case key.Matches(msg, k.NextView), key.Matches(msg, k.NextPanel), key.Matches(msg, k.Right):
		return a, a.cyclePipelinePivot(1)
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Search):
		return a, a.pl.panel.FocusSearch()
	case key.Matches(msg, k.RunPipeline):
		if p := a.selectedPipeline(); p != nil {
			return a, a.openRunForm(*p, "")
		}
	default:
		var cmd tea.Cmd
		a.pl.panel, cmd = a.pl.panel.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) handlePipelineResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case pipelineListMsg:
		return a.setPipelineList(msg)
	case pipelinePollMsg:
		if msg.seq != a.pl.pollSeq {
			return a, nil, true
		}
		return a, a.pollPipelines(), true
	case panels.MorePipelinesMsg:
		return a, a.loadMorePipelines(), true
	case panels.SearchPipelinesMsg:
		if msg.Text == a.searchText() {
			a.pl.searchSeq++
			return a, nil, true
		}
		return a, a.searchPipelines(msg.Text), true
	case panels.PipelineSearchTypedMsg:
		a.pl.searchSeq++
		return a, schedulePipelineSearchCmd(a.pl.searchSeq, msg.Text), true
	case pipelineSearchDueMsg:
		if msg.seq != a.pl.searchSeq || !a.pl.panel.Searching() {
			return a, nil, true
		}
		return a, a.searchPipelinesLive(msg.text), true
	case panels.OpenPipelineMsg:
		return a, a.openPipelinePage(msg.Pipeline), true
	case panels.OpenRunMsg:
		if msg.AtError {
			cmd := a.openRun(msg.Run, panels.RunLogsTab)
			a.pl.view.JumpToFirstError()
			return a, cmd, true
		}
		return a, a.openRun(msg.Run, panels.RunSummaryTab), true
	case panels.BrowsePipelineMsg:
		url := msg.URL
		if url == "" {
			url = a.client.PipelineWebURL(msg.PipelineID)
		}
		return a, openBrowserCmd(url), true
	case pipelinePreviewDueMsg:
		if msg.run != a.pl.previewRun || msg.pipeline != a.pl.previewPipeline {
			return a, nil, true
		}
		var cmds []tea.Cmd
		if r := a.pl.panel.SelectedRun(); msg.run != 0 && !a.pl.stagesLoaded && r != nil && a.pl.requestStages(*r) {
			cmds = append(cmds, loadPipelineStagesCmd(a.client, msg.run))
		}
		if msg.pipeline != 0 {
			cmds = append(cmds, loadPipelineHistoryCmd(a.client, msg.pipeline))
		}
		return a, tea.Batch(cmds...), true
	case pipelineStagesMsg:
		a.pl.storeStages(msg.run, msg.stages)
		a.pl.panel.SetRunStages(msg.run, panels.StageStates(msg.stages))
		if msg.run == a.pl.previewRun {
			a.pl.stages, a.pl.stagesLoaded = msg.stages, true
		}
		return a, nil, true
	case pipelineHistoryMsg:
		if msg.pipeline == a.pl.previewPipeline {
			a.pl.history, a.pl.historyLoaded = msg.runs, true
		}
		return a, nil, true
	}
	return a.handleRunResult(msg)
}

func (a App) setPipelineList(msg pipelineListMsg) (App, tea.Cmd, bool) {
	if msg.seq != a.pl.seq || !sameList(msg.mode, a.pipelineMode()) {
		return a, nil, true
	}
	msg.mode = a.pipelineMode()
	if msg.more {
		a.pl.panel.SetLoadingMore(false)
	} else if !msg.silent {
		a.pl.loading = false
		a.pl.panel.SetRemoteSearch(false)
	}
	if msg.searchErr != "" {
		a.pl.loaded = true
		if !a.pl.panel.Searching() {
			a.pl.panel.SetPipelines(msg.mode, nil, false)
			a.pl.panel.SetSearchText(a.searchText())
		}
		a.pl.panel.SetSearchError(msg.searchErr)
		return a, nil, true
	}
	if msg.err != nil {
		if msg.silent {
			return a, nil, true
		}
		a.pl.loaded = true
		a.pl.panel.SetPipelines(msg.mode, nil, false)
		return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
	}
	a.pl.loaded = true
	switch {
	case msg.more:
		a.pl.pipelines = append(a.pl.pipelines, msg.pipelines...)
		a.pl.runs = append(a.pl.runs, msg.runs...)
		a.pl.token = msg.token
	case msg.silent && msg.mode >= panels.ListRuns:
		a.pl.runs = mergeRuns(a.pl.runs, msg.runs)
	default:
		a.pl.pipelines, a.pl.runs, a.pl.token = msg.pipelines, msg.runs, msg.token
	}
	if msg.silent {
		a.pl.markActiveStale()
	}
	text := a.pl.panel.SearchText()
	if msg.mode >= panels.ListRuns {
		a.pl.panel.SetRuns(msg.mode, a.pl.runs, a.pl.token != "")
	} else {
		a.pl.panel.SetPipelines(msg.mode, a.pl.pipelines, a.pl.token != "")
	}
	a.pl.panel.SetSearchText(text)
	a.pl.panel.SetApplied(a.searchText())
	if !a.activeRuns() {
		return a, nil, true
	}
	a.pl.pollSeq++
	return a, schedulePipelinePollCmd(a.pl.pollSeq), true
}

func (a *App) pipelinePageTitle() string {
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	return bold.Render(a.pl.page.Name) + panels.Muted(titleSeparator+a.pl.page.FolderLabel())
}

func (a *App) pipelinePivotLabels() []string {
	names := pipelinePivots
	if a.pl.page != nil {
		names = pipelinePagePivots
	}
	labels := make([]string, len(names))
	for i, name := range names {
		labels[i] = a.filterKeys()[i].Help().Key + " " + name
	}
	return labels
}

func (a *App) pipelineCrumb() string {
	if a.pl.page != nil {
		return a.pl.page.Name
	}
	return pipelinePivots[a.pl.pivot]
}

func (a *App) renderPipelinePane(width, height int) string {
	cw, ch := panels.PaneContentSize(width, height)
	content := panels.RenderPipelinePreview(panels.PipelinePreviewProps{
		Pipeline:      a.pl.panel.SelectedPipeline(),
		Run:           a.pl.panel.SelectedRun(),
		History:       a.pl.history,
		HistoryLoaded: a.pl.historyLoaded,
		Stages:        a.pl.stages,
		StagesLoaded:  a.pl.stagesLoaded,
		ShowPipeline:  a.pipelineMode() < panels.ListRuns,
		IssueKey:      a.keys.NextIssue.Help().Key,
		Spinner:       a.spinner.View(),
		Width:         cw,
		Height:        ch,
	})
	return panels.Pane(panels.PaneTitle("", "Preview", false), content, width, height, false)
}

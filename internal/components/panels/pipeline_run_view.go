package panels

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
)

type RunTab int

const (
	RunSummaryTab RunTab = iota
	RunLogsTab
	runTabs
)

const (
	summaryPaneStages = iota
	summaryPaneJobs
	summaryPanes
)

const (
	runSidebarShow    = 150
	runStageListShare = 2
	runStageListParts = 5
	runStageListMax   = 48
	stageRowLead      = " "
	runHeaderHeight   = 4
	runIssuesShown    = 20
	summaryJobRows    = 2
	summaryIssueLines = 2
	noJobsInStage     = "No jobs ran in this stage."
)

type CloseRunViewMsg struct{}

type RunAgainMsg struct {
	Run models.PipelineRun
}

type PipelineRunView struct {
	runLogs
	run     *models.PipelineRun
	changes []models.RunChange
	tab     RunTab
	focus   int
	summary viewport.Model
	keys    keys.KeyMap
	spinner string
	width   int
	height  int
}

func NewPipelineRunView(k keys.KeyMap) PipelineRunView {
	return PipelineRunView{runLogs: newRunLogs(k), keys: k, summary: widgets.NewViewport(k)}
}

func (v *PipelineRunView) Open(run models.PipelineRun, tab RunTab) {
	*v = PipelineRunView{
		runLogs: newRunLogs(v.keys),
		run:     &run,
		tab:     tab,
		keys:    v.keys,
		spinner: v.spinner,
		width:   v.width,
		height:  v.height,
		summary: widgets.NewViewport(v.keys),
	}
	v.lspinner = v.spinner
	v.layout()
}

func (v *PipelineRunView) Run() *models.PipelineRun { return v.run }

func (v *PipelineRunView) Tab() RunTab { return v.tab }

func (v *PipelineRunView) Active() bool {
	return v.run != nil && v.run.State().Active()
}

func (v *PipelineRunView) Loading() bool {
	if v.run == nil {
		return false
	}
	return !v.stagesLoaded || (v.tab == RunLogsTab && v.logLoading())
}

func (v *PipelineRunView) SetSpinner(frame string) {
	if v.spinner == frame {
		return
	}
	v.spinner, v.lspinner = frame, frame
	if v.Loading() {
		v.render()
	}
}

func (v *PipelineRunView) Refresh(run models.PipelineRun) {
	if v.run == nil || v.run.ID != run.ID {
		return
	}
	v.run = &run
	v.render()
}

func (v *PipelineRunView) SetTimeline(id int, records []models.TimelineRecord) {
	if v.run == nil || v.run.ID != id {
		return
	}
	v.setStages(models.BuildStages(records))
	v.renderSummary()
}

func (v *PipelineRunView) SetChanges(id int, changes []models.RunChange) {
	if v.run == nil || v.run.ID != id {
		return
	}
	v.changes = changes
	v.render()
}

func (v *PipelineRunView) SetLog(id, logID, from int, lines []string, truncated bool, err error) {
	if v.run == nil || v.run.ID != id {
		return
	}
	v.setLog(logID, from, lines, truncated, err)
}

func (v *PipelineRunView) MarkStale() { v.markStale() }

func (v *PipelineRunView) WantedLog() (logID, from int, ok bool) {
	if v.run == nil || v.tab != RunLogsTab {
		return 0, 0, false
	}
	return v.wanted()
}

func (v *PipelineRunView) MarkLogRequested(logID int) { v.markRequested(logID) }

func (v *PipelineRunView) SetSize(width, height int) {
	if v.width == width && v.height == height {
		return
	}
	v.width, v.height = width, height
	v.layout()
}

func (v *PipelineRunView) bodyHeight() int { return max(v.height-runHeaderHeight, 0) }

func (v *PipelineRunView) sidebarWidth() int {
	if v.width < runSidebarShow {
		return 0
	}
	return styles.PullRequestSidebarWidth
}

func (v *PipelineRunView) stageListWidth() int {
	return min(max(v.width*runStageListShare/runStageListParts, runTreeMinWidth), runStageListMax)
}

func (v *PipelineRunView) jobsPaneWidth() int {
	w := v.width - v.stageListWidth() - runPaneGap
	if side := v.sidebarWidth(); side > 0 {
		w -= side + styles.DetailColumnGap
	}
	return max(w, 0)
}

func (v *PipelineRunView) mainWidth() int {
	w, _ := PaneContentSize(v.jobsPaneWidth(), v.bodyHeight())
	return max(w, 1)
}

func (v *PipelineRunView) layout() {
	_, h := PaneContentSize(v.jobsPaneWidth(), v.bodyHeight())
	v.summary.SetWidth(v.mainWidth())
	v.summary.SetHeight(max(h, 1))
	v.resizeLogs(v.width, v.bodyHeight())
	v.render()
}

func (v *PipelineRunView) render() {
	if v.run == nil || v.width <= 0 {
		return
	}
	v.renderSummary()
	v.renderLogs()
}

func (v *PipelineRunView) renderSummary() {
	if v.run == nil || v.width <= 0 {
		return
	}
	width := v.mainWidth()
	now := time.Now()
	jobs, top := v.jobsSection(width, now)
	parts := []string{jobs}
	if issues := models.RunIssues(v.stages); len(issues) > 0 {
		parts = append(parts, issuesSection(issues, width))
	}
	if v.sidebarWidth() == 0 {
		parts = append(parts, v.sideSections(width)...)
	}
	v.summary.SetContent(strings.Join(parts, "\n\n"))
	if v.tab == RunSummaryTab && top >= 0 {
		widgets.Reveal(&v.summary, top, top+summaryJobRows)
	}
}

func (v *PipelineRunView) jobsSection(width int, now time.Time) (string, int) {
	switch {
	case !v.stagesLoaded:
		return Loading(v.spinner, loadingRun), -1
	case len(v.stages) == 0:
		return Muted(noStages), -1
	}
	start, end := v.stageJobs(v.stage)
	if start == end {
		return Muted(noJobsInStage), -1
	}
	jobs := v.jobs()
	focused := v.focus == summaryPaneJobs
	var lines []string
	top := -1
	for i := start; i < end; i++ {
		if i > start {
			lines = append(lines, "")
		}
		if i == v.job {
			top = len(lines)
		}
		lines = append(lines, summaryJobRow(jobs[i], i == v.job, focused, width, now)...)
	}
	return strings.Join(lines, "\n"), top
}

func (v *PipelineRunView) stageList(width, height int, now time.Time) string {
	switch {
	case !v.stagesLoaded:
		return Block(" "+Loading(v.spinner, loadingRun), width, height)
	case len(v.stages) == 0:
		return Block(" "+Muted(noStages), width, height)
	}
	focused := v.focus == summaryPaneStages
	lines := make([]string, len(v.stages))
	for i, s := range v.stages {
		lines[i] = pickLine(s.State, s.Name, stageDetail(s, now), stageRowLead, "", i == v.stage, focused, stageNotRun(s), width)
	}
	start := min(max(v.stage-height/2, 0), max(len(lines)-height, 0))
	return Block(strings.Join(lines[start:], "\n"), width, height)
}

func (v *PipelineRunView) summaryBody() string {
	now := time.Now()
	h := v.bodyHeight()
	sw, jw := v.stageListWidth(), v.jobsPaneWidth()
	stagesFocused := v.focus == summaryPaneStages
	siw, sih := PaneContentSize(sw, h)
	stagesTitle := PaneTitle("", "Stages", stagesFocused)
	if tally := StageTally(v.stages); lipgloss.Width(stagesTitle)+lipgloss.Width(tally)+1 <= siw {
		stagesTitle = spreadLine(stagesTitle, Muted(tally), siw)
	}
	stages := Pane(stagesTitle, v.stageList(siw, sih, now), sw, h, stagesFocused)
	jiw, jih := PaneContentSize(jw, h)
	jobsTitle := PaneTitle("", "Jobs", !stagesFocused)
	if v.stagesLoaded && len(v.stages) > 0 {
		stage := v.stages[v.stage]
		done, total := stage.JobCounts()
		jobsTitle = spreadLine(PaneTitle("", "Jobs in "+stage.Name, !stagesFocused), Muted(fmt.Sprintf("%d of %d done", done, total)), jiw)
	}
	jobs := Pane(jobsTitle, Block(v.summary.View(), jiw, jih), jw, h, !stagesFocused)
	parts := []string{stages, strings.Repeat(" ", runPaneGap), jobs}
	if side := v.sidebarWidth(); side > 0 {
		parts = append(parts, strings.Repeat(" ", styles.DetailColumnGap), Block(strings.Join(v.sideSections(side), "\n\n"), side, h))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func summaryJobRow(j models.RunJob, selected, focused bool, width int, now time.Time) []string {
	base := lipgloss.NewStyle()
	bar := " "
	nameStyle := base.Foreground(styles.ColorText)
	detailStyle := base.Foreground(styles.ColorTextMuted)
	switch {
	case selected && focused:
		base, bar = base.Background(styles.ColorHighlight), railBar
		nameStyle = base.Foreground(styles.ColorText).Bold(true)
		detailStyle = base.Foreground(styles.ColorLight)
	case selected:
		base = base.Background(styles.ColorHighlightDim)
		nameStyle = base.Foreground(styles.ColorLight).Bold(true)
		detailStyle = base.Foreground(styles.ColorTextMuted)
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar) + base.Render(" ")
	inner := max(width-lipgloss.Width(lead), 1)
	duration := base.Foreground(styles.ColorTextMuted).Render(recordRight(j.TimelineRecord, now))
	first := fill(runGlyphStyle(j.State, base)+base.Render(" ")+nameStyle.Render(j.Name), base.Render(" ")+duration+base.Render(" "), base, inner)
	second := fill(base.Render("  ")+detailStyle.Render(jobDetail(j)), "", base, inner)
	return []string{lead + first, lead + second}
}

func jobDetail(j models.RunJob) string {
	where := countOf(len(j.Steps), "step")
	for i, s := range j.Steps {
		progress := fmt.Sprintf("step %d of %d", i+1, len(j.Steps))
		switch s.State {
		case models.RunFailed:
			return joinMeta("Failed at "+s.Name, progress, j.Worker)
		case models.RunRunning:
			return joinMeta("Running "+s.Name, progress, j.Worker)
		}
	}
	if j.State == models.RunQueued {
		where = "Waiting for an agent"
	}
	return joinMeta(where, j.Worker)
}

func issuesSection(issues []models.RunIssue, width int) string {
	errors, warnings := issueCounts(issues)
	lines := []string{sectionTitle(fmt.Sprintf("Errors %d · warnings %d", errors, warnings), width)}
	for i, is := range issues {
		if i == runIssuesShown {
			lines = append(lines, Muted(fmt.Sprintf("and %d more", len(issues)-i)))
			break
		}
		if i > 0 {
			lines = append(lines, "")
		}
		state := models.RunPartial
		if is.Error {
			state = models.RunFailed
		}
		lines = append(lines, Line(RunGlyph(state)+" "+Muted(is.Where), width))
		lines = append(lines, issueBlock(is, width, summaryIssueLines)...)
	}
	return strings.Join(lines, "\n")
}

func firstLogLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

func keptRows(rows [][2]string) [][2]string {
	var kept [][2]string
	for _, row := range rows {
		if row[1] != "" {
			kept = append(kept, row)
		}
	}
	return kept
}

func (v *PipelineRunView) sideSections(width int) []string {
	r := *v.run
	rows := keptRows([][2]string{
		{"Pipeline", r.PipelineName},
		{"State", RunStateText(r.State(), r.State().Name())},
		{"Trigger", r.ReasonLabel()},
		{"Requested by", r.RequestedFor.DisplayName},
		{"Repository", r.RepositoryName},
		{"Branch", r.Branch()},
		{"Commit", r.ShortCommit()},
		{"Pool", r.Pool},
		{"Queued", formatStamp(r.Queued)},
		{"Started", formatStamp(r.Started)},
		{"Finished", formatStamp(r.Finished)},
		{"Duration", RunDuration(r, time.Now())},
	})
	sections := []string{sectionTitle("Details", width) + "\n" + FieldRows(rows, fieldLabel, width)}
	if len(v.changes) > 0 {
		lines := []string{sectionTitle(fmt.Sprintf("Changes %d", len(v.changes)), width)}
		for _, c := range v.changes {
			id := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(c.ShortID())
			lines = append(lines, Line(id+" "+firstLogLine(c.Message), width))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return sections
}

func formatStamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	local := t.Local()
	if y, m, d := local.Date(); y == time.Now().Year() && m == time.Now().Month() && d == time.Now().Day() {
		return "today " + local.Format("15:04")
	}
	return local.Format("Jan 2 15:04")
}

func (v PipelineRunView) Update(msg tea.Msg) (PipelineRunView, tea.Cmd) {
	if v.run == nil {
		return v, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	case tea.MouseWheelMsg:
		if v.tab == RunLogsTab {
			return v, v.scrollLog(msg)
		}
		var cmd tea.Cmd
		v.summary, cmd = v.summary.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v PipelineRunView) handleKey(msg tea.KeyPressMsg) (PipelineRunView, tea.Cmd) {
	k := v.keys
	run := *v.run
	switch {
	case key.Matches(msg, k.Back) && v.tab == RunLogsTab && v.pane > runPaneTree:
		v.pane--
		return v, nil
	case key.Matches(msg, k.Back):
		return v, func() tea.Msg { return CloseRunViewMsg{} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return v, shareCmd(v.shareTarget(run), now)
	case key.Matches(msg, k.Panel1):
		v.setTab(RunSummaryTab)
		return v, nil
	case key.Matches(msg, k.Panel2):
		v.setTab(RunLogsTab)
		return v, nil
	case key.Matches(msg, k.NextView), key.Matches(msg, k.PrevView):
		v.setTab((v.tab + 1) % runTabs)
		return v, nil
	case key.Matches(msg, k.RunPipeline):
		return v, func() tea.Msg { return RunAgainMsg{Run: run} }
	case key.Matches(msg, k.OpenBrowser):
		url := v.browseURL()
		return v, func() tea.Msg { return BrowsePipelineMsg{PipelineID: run.PipelineID, URL: url} }
	}
	if v.tab == RunLogsTab {
		cmd := v.handleLogsKey(msg)
		v.renderSummary()
		return v, cmd
	}
	return v.handleSummaryKey(msg)
}

func (v *PipelineRunView) setTab(tab RunTab) {
	if v.tab == tab {
		return
	}
	v.tab = tab
	if tab == RunSummaryTab && len(v.jobs()) > 0 {
		v.stage = v.stageOfJob(v.job)
	}
	v.render()
}

func (v *PipelineRunView) JumpToFirstError() {
	v.startAtFirstError()
	v.render()
}

func (v *PipelineRunView) shareTarget(run models.PipelineRun) share.Target {
	label := ""
	if j, ok := v.selectedJob(); ok && v.tab == RunLogsTab {
		label = j.Name
		if s, ok := v.selectedStep(); ok {
			label += " / " + s.Name
		}
	}
	return share.Run(run, v.browseURL(), label)
}

func (v *PipelineRunView) browseURL() string {
	url := v.run.WebURL
	if url == "" || v.tab != RunLogsTab {
		return url
	}
	if j, ok := v.selectedJob(); ok {
		url += "&view=logs&j=" + j.ID
		if s, ok := v.selectedStep(); ok {
			url += "&t=" + s.ID
		}
	}
	return url
}

func (v PipelineRunView) handleSummaryKey(msg tea.KeyPressMsg) (PipelineRunView, tea.Cmd) {
	k := v.keys
	move := v.moveStageJob
	if v.focus == summaryPaneStages {
		move = v.moveStage
	}
	switch {
	case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel):
		v.focus = (v.focus + 1) % summaryPanes
	case key.Matches(msg, k.Left):
		v.focus = summaryPaneStages
	case key.Matches(msg, k.Right):
		v.focus = summaryPaneJobs
	case key.Matches(msg, k.Up):
		move(-1)
	case key.Matches(msg, k.Down):
		move(1)
	case key.Matches(msg, k.Top):
		move(-1 << 20)
	case key.Matches(msg, k.Bottom):
		move(1 << 20)
	case key.Matches(msg, k.NextIssue), key.Matches(msg, k.PrevIssue):
		v.tab, v.pane = RunLogsTab, runPaneTree
		v.startAtFirstError()
	case key.Matches(msg, k.Select):
		v.openStageLogs()
	default:
		var cmd tea.Cmd
		v.summary, cmd = v.summary.Update(msg)
		return v, cmd
	}
	v.render()
	return v, nil
}

func (v *PipelineRunView) openStageLogs() {
	start, end := v.stageJobs(v.stage)
	if start == end {
		return
	}
	job := v.job
	if v.focus == summaryPaneStages || job < start || job >= end {
		job = v.stageDefaultJob(start, end)
	}
	v.selectJob(job)
	v.tab, v.pane = RunLogsTab, runPaneTree
}

func (v *PipelineRunView) stageDefaultJob(start, end int) int {
	jobs := v.jobs()
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		for i := start; i < end; i++ {
			if jobs[i].State == want {
				return i
			}
		}
	}
	return start
}

func (v *PipelineRunView) moveStage(delta int) {
	if len(v.stages) == 0 {
		return
	}
	next := min(max(v.stage+delta, 0), len(v.stages)-1)
	if next == v.stage {
		return
	}
	if start, end := v.stageJobs(next); start < end {
		v.selectJob(v.stageDefaultJob(start, end))
	}
	v.stage = next
}

func (v *PipelineRunView) moveStageJob(delta int) {
	start, end := v.stageJobs(v.stage)
	if start == end {
		return
	}
	next := start
	if v.job >= start && v.job < end {
		next = min(max(v.job+delta, start), end-1)
	}
	if next != v.job {
		v.selectJob(next)
	}
}

func (v *PipelineRunView) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("summary/logs", k.Panel1, k.Panel2)
	back := widgets.Relabel(k.Back, "back")
	again := widgets.Relabel(k.RunPipeline, "run again")
	issue := widgets.Relabel(k.NextIssue, "first error")
	switch {
	case v.tab == RunSummaryTab && v.focus == summaryPaneStages:
		return []key.Binding{widgets.Relabel(k.Select, "stage logs"), widgets.Combine("stage", k.Up, k.Down), widgets.Relabel(k.NextPanel, "jobs"), issue, again, tabs, k.Share, k.OpenBrowser, back}
	case v.tab == RunSummaryTab:
		return []key.Binding{widgets.Relabel(k.Select, "job logs"), widgets.Combine("job", k.Up, k.Down), widgets.Relabel(k.NextPanel, "stages"), issue, again, tabs, k.Share, k.OpenBrowser, back}
	}
	return append(v.logsBindings(), again, tabs, k.Share, k.OpenBrowser, back)
}

func (v *PipelineRunView) Mode() string {
	if v.tab == RunLogsTab {
		return "LOGS · " + v.paneLabel()
	}
	if v.focus == summaryPaneJobs {
		return "RUN · JOBS"
	}
	return "RUN · STAGES"
}

func (v *PipelineRunView) header() string {
	r := *v.run
	now := time.Now()
	inner := max(v.width-2, 1)
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(r.Title())
	first := RunPill(r.State()) + " " + runNumber(r) + "  " + title
	meta := Muted(joinMeta(r.PipelineName, branchLabel(r), r.ShortCommit(), runTrigger(r), runStarted(r), RunDuration(r, now)))
	tabs, rule := PivotTabs([]string{v.keys.Panel1.Help().Key + " Summary", v.keys.Panel2.Help().Key + " Logs"}, int(v.tab), v.width)
	return strings.Join([]string{" " + Line(first, inner), " " + Line(meta, inner), tabs, rule}, "\n")
}

func (v PipelineRunView) View() string {
	if v.run == nil || v.width <= 0 || v.height <= 0 {
		return ""
	}
	var body string
	if v.tab == RunLogsTab {
		body = Block(v.logsBody(), v.width, v.bodyHeight())
	} else {
		body = Block(v.summaryBody(), v.width, v.bodyHeight())
	}
	return Block(v.header()+"\n"+body, v.width, v.height)
}

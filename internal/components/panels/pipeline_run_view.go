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
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type RunTab int

const (
	RunSummaryTab RunTab = iota
	RunLogsTab
	runTabs
)

const (
	runPaneJobs = iota
	runPaneSteps
	runPaneLog
	runPanes
)

const (
	runHeaderHeight   = 4
	runJobsMinWidth   = 22
	runJobsMaxWidth   = 36
	runStepsMinRows   = 3
	runLogGutterMin   = 4
	runIssuesShown    = 20
	waitingForStep    = "Waiting for this step to start."
	noLogForStep      = "This step has no log."
	loadingLog        = "Loading log"
	loadingRun        = "Loading stages"
	logTruncatedNote  = "Log cut at the size limit. Press o to read the rest in the browser."
	logMarkerError    = "##[error]"
	logMarkerWarning  = "##[warning]"
	logMarkerSection  = "##[section]"
	logMarkerCommand  = "##[command]"
	logMarkerGroup    = "##[group]"
	logMarkerEndGroup = "##[endgroup]"
	logMarkerDebug    = "##[debug]"
)

type CloseRunViewMsg struct{}

type RunAgainMsg struct {
	Run models.PipelineRun
}

type runLog struct {
	lines     []string
	done      bool
	loading   bool
	truncated bool
	err       string
}

type PipelineRunView struct {
	run          *models.PipelineRun
	stages       []models.RunStage
	stagesLoaded bool
	changes      []models.RunChange
	logs         map[int]runLog
	stale        bool
	picked       bool
	tab          RunTab
	summary      viewport.Model
	logView      viewport.Model
	stepView     viewport.Model
	job          int
	step         int
	pane         int
	follow       bool
	keys         keys.KeyMap
	spinner      string
	width        int
	height       int
}

func NewPipelineRunView(k keys.KeyMap) PipelineRunView {
	return PipelineRunView{keys: k, summary: widgets.NewViewport(k), logView: widgets.NewViewport(k), stepView: widgets.NewViewport(k)}
}

func (v *PipelineRunView) Open(run models.PipelineRun, tab RunTab) {
	*v = PipelineRunView{
		run:      &run,
		logs:     map[int]runLog{},
		tab:      tab,
		follow:   true,
		keys:     v.keys,
		spinner:  v.spinner,
		width:    v.width,
		height:   v.height,
		summary:  widgets.NewViewport(v.keys),
		logView:  widgets.NewViewport(v.keys),
		stepView: widgets.NewViewport(v.keys),
	}
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
	if !v.stagesLoaded {
		return true
	}
	if entry, ok := v.logs[v.selectedLogID()]; ok && v.tab == RunLogsTab {
		return entry.loading && len(entry.lines) == 0
	}
	return false
}

func (v *PipelineRunView) SetSpinner(frame string) {
	if v.spinner == frame {
		return
	}
	v.spinner = frame
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
	v.stages, v.stagesLoaded = models.BuildStages(records), true
	if !v.picked {
		v.pickDefault()
	}
	v.clamp()
	v.layout()
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
	entry := v.logs[logID]
	entry.loading = false
	if err != nil {
		entry.err = err.Error()
	} else {
		entry.err = ""
		keep := min(max(from-1, 0), len(entry.lines))
		entry.lines = append(entry.lines[:keep:keep], lines...)
		entry.truncated = truncated
		entry.done = truncated || !v.recordActive(logID)
	}
	v.logs[logID] = entry
	if logID == v.selectedLogID() {
		v.renderLog()
	}
}

func (v *PipelineRunView) recordActive(logID int) bool {
	for _, j := range v.jobs() {
		if j.LogID == logID {
			return j.State.Active()
		}
		for _, s := range j.Steps {
			if s.LogID == logID {
				return s.State.Active()
			}
		}
	}
	return false
}

func (v *PipelineRunView) MarkStale() { v.stale = true }

func (v *PipelineRunView) WantedLog() (logID, from int, ok bool) {
	if v.run == nil || v.tab != RunLogsTab {
		return 0, 0, false
	}
	logID = v.selectedLogID()
	if logID == 0 {
		return 0, 0, false
	}
	entry, seen := v.logs[logID]
	switch {
	case !seen:
		return logID, 1, true
	case entry.loading || entry.done || !v.stale:
		return 0, 0, false
	}
	return logID, len(entry.lines) + 1, true
}

func (v *PipelineRunView) MarkLogRequested(logID int) {
	entry := v.logs[logID]
	entry.loading = true
	v.logs[logID] = entry
	v.stale = false
	if len(entry.lines) == 0 {
		v.renderLog()
	}
}

func (v *PipelineRunView) jobs() []models.RunJob {
	var out []models.RunJob
	for _, s := range v.stages {
		out = append(out, s.Jobs...)
	}
	return out
}

func (v *PipelineRunView) selectedJob() (models.RunJob, bool) {
	jobs := v.jobs()
	if v.job < 0 || v.job >= len(jobs) {
		return models.RunJob{}, false
	}
	return jobs[v.job], true
}

func (v *PipelineRunView) selectedStep() (models.TimelineRecord, bool) {
	j, ok := v.selectedJob()
	if !ok || v.step < 0 || v.step >= len(j.Steps) {
		return models.TimelineRecord{}, false
	}
	return j.Steps[v.step], true
}

func (v *PipelineRunView) selectedLogID() int {
	if s, ok := v.selectedStep(); ok {
		return s.LogID
	}
	if j, ok := v.selectedJob(); ok && len(j.Steps) == 0 {
		return j.LogID
	}
	return 0
}

func (v *PipelineRunView) pickDefault() {
	jobs := v.jobs()
	if len(jobs) == 0 {
		return
	}
	v.picked = true
	v.job = len(jobs) - 1
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		for i, j := range jobs {
			if j.State == want {
				v.job = i
				v.step = defaultStep(j)
				return
			}
		}
	}
	v.step = defaultStep(jobs[v.job])
}

func defaultStep(j models.RunJob) int {
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		for i, s := range j.Steps {
			if s.State == want {
				return i
			}
		}
	}
	return max(len(j.Steps)-1, 0)
}

func (v *PipelineRunView) clamp() {
	jobs := v.jobs()
	v.job = min(max(v.job, 0), max(len(jobs)-1, 0))
	if j, ok := v.selectedJob(); ok {
		v.step = min(max(v.step, 0), max(len(j.Steps)-1, 0))
	}
}

func (v *PipelineRunView) SetSize(width, height int) {
	if v.width == width && v.height == height {
		return
	}
	v.width, v.height = width, height
	v.layout()
}

func (v *PipelineRunView) bodyHeight() int { return max(v.height-runHeaderHeight, 0) }

func (v *PipelineRunView) sidebarWidth() int {
	if v.width < styles.PullRequestSidebarShow {
		return 0
	}
	return styles.PullRequestSidebarWidth
}

func (v *PipelineRunView) mainWidth() int {
	if side := v.sidebarWidth(); side > 0 {
		return max(v.width-side-styles.DetailColumnGap-2, 1)
	}
	return max(v.width-2, 1)
}

func (v *PipelineRunView) jobsWidth() int {
	return min(max(v.width/4, runJobsMinWidth), runJobsMaxWidth)
}

func (v *PipelineRunView) logsWidth() int {
	return max(v.width-v.jobsWidth()-3, 1)
}

func (v *PipelineRunView) stepRows() int {
	j, _ := v.selectedJob()
	return min(max(len(j.Steps), 1), max(v.bodyHeight()/3, runStepsMinRows))
}

func (v *PipelineRunView) layout() {
	v.summary.SetWidth(v.mainWidth())
	v.summary.SetHeight(v.bodyHeight())
	v.stepView.SetWidth(v.logsWidth())
	v.stepView.SetHeight(v.stepRows())
	v.logView.SetWidth(v.logsWidth())
	v.logView.SetHeight(max(v.bodyHeight()-v.stepRows()-3, 1))
	v.render()
}

func (v *PipelineRunView) render() {
	if v.run == nil || v.width <= 0 {
		return
	}
	v.renderSummary()
	v.renderSteps()
	v.renderLog()
}

func (v *PipelineRunView) renderSummary() {
	width := v.mainWidth()
	now := time.Now()
	parts := []string{v.stagesSection(width, now)}
	if issues := models.RunIssues(v.stages); len(issues) > 0 {
		parts = append(parts, v.issuesSection(issues, width))
	}
	if v.sidebarWidth() == 0 {
		parts = append(parts, v.sideSections(width)...)
	}
	v.summary.SetContent(strings.Join(parts, "\n\n"))
}

func (v *PipelineRunView) stagesSection(width int, now time.Time) string {
	lines := []string{sectionTitle("Stages and jobs", width)}
	switch {
	case !v.stagesLoaded:
		return strings.Join(append(lines, Loading(v.spinner, loadingRun)), "\n")
	case len(v.stages) == 0:
		return strings.Join(append(lines, Muted(noStages)), "\n")
	}
	index := 0
	top := 0
	for _, s := range v.stages {
		lines = append(lines, RenderStageRows([]models.RunStage{s}, now, width)...)
		for _, j := range s.Jobs {
			selected := index == v.job
			if selected {
				top = len(lines)
			}
			lines = append(lines, v.jobRow(j, selected, v.tab == RunSummaryTab, width, now, "  "))
			index++
		}
	}
	if v.tab == RunSummaryTab {
		widgets.Reveal(&v.summary, top, top)
	}
	return strings.Join(lines, "\n")
}

func (v *PipelineRunView) jobRow(j models.RunJob, selected, focused bool, width int, now time.Time, indent string) string {
	base := lipgloss.NewStyle()
	bar := " "
	if selected {
		base = base.Background(styles.ColorHighlight)
		if focused {
			bar = railBar
		}
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar) + base.Render(indent)
	right := base.Foreground(styles.ColorTextMuted).Render(RecordDuration(j.TimelineRecord, now))
	name := RunBadge(j.State) + base.Render(" ") + base.Foreground(styles.ColorText).Bold(selected).Render(j.Name)
	return lead + fill(name, base.Render(" ")+right, base, max(width-lipgloss.Width(lead), 1))
}

func (v *PipelineRunView) issuesSection(issues []models.RunIssue, width int) string {
	errors := 0
	for _, is := range issues {
		if is.Error {
			errors++
		}
	}
	title := fmt.Sprintf("Errors %d · warnings %d", errors, len(issues)-errors)
	lines := []string{sectionTitle(title, width)}
	for i, is := range issues {
		if i == runIssuesShown {
			lines = append(lines, Muted(fmt.Sprintf("and %d more", len(issues)-i)))
			break
		}
		badge := Badge("WARN", styles.ColorWarning, styles.ColorOnAccent)
		if is.Error {
			badge = Badge("FAIL", styles.ColorError, styles.ColorOnAccent)
		}
		lines = append(lines, Line(badge+" "+Muted(is.Where), width))
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorText).Width(width).Render(firstLogLine(is.Message)))
	}
	return strings.Join(lines, "\n")
}

func firstLogLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

func (v *PipelineRunView) sideSections(width int) []string {
	r := *v.run
	now := time.Now()
	rows := [][2]string{
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
		{"Duration", RunDuration(r, now)},
	}
	var kept [][2]string
	for _, row := range rows {
		if row[1] != "" {
			kept = append(kept, row)
		}
	}
	sections := []string{sectionTitle("Details", width) + "\n" + FieldRows(kept, fieldLabel, width)}
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

func (v *PipelineRunView) renderSteps() {
	j, ok := v.selectedJob()
	if !ok {
		v.stepView.SetContent("")
		return
	}
	width := v.logsWidth()
	now := time.Now()
	lines := make([]string, 0, len(j.Steps))
	for i, s := range j.Steps {
		job := models.RunJob{TimelineRecord: s}
		lines = append(lines, v.jobRow(job, i == v.step, v.pane == runPaneSteps, width, now, ""))
	}
	if len(lines) == 0 {
		lines = append(lines, Muted(" No steps yet."))
	}
	v.stepView.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&v.stepView, v.step, v.step)
}

func (v *PipelineRunView) renderLog() {
	if v.run == nil || v.width <= 0 {
		return
	}
	width := v.logsWidth()
	logID := v.selectedLogID()
	entry, seen := v.logs[logID]
	var content string
	switch {
	case !v.stagesLoaded:
		content = Loading(v.spinner, loadingRun)
	case logID == 0 && v.selectedStepQueued():
		content = Muted(waitingForStep)
	case logID == 0:
		content = Muted(noLogForStep)
	case entry.err != "":
		content = lipgloss.NewStyle().Foreground(styles.ColorError).Width(width).Render(entry.err)
	case !seen || (entry.loading && len(entry.lines) == 0):
		content = Loading(v.spinner, loadingLog)
	default:
		content = renderLogLines(entry.lines, width)
		if entry.truncated {
			content += "\n" + lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(logTruncatedNote)
		}
	}
	atBottom := v.logView.AtBottom()
	v.logView.SetContent(content)
	if v.follow || atBottom {
		v.logView.GotoBottom()
	}
}

func (v *PipelineRunView) selectedStepQueued() bool {
	if s, ok := v.selectedStep(); ok {
		return s.State == models.RunQueued
	}
	j, ok := v.selectedJob()
	return ok && j.State == models.RunQueued
}

func renderLogLines(lines []string, width int) string {
	gutter := max(len(strconv.Itoa(len(lines))), runLogGutterMin)
	text := max(width-gutter-1, 1)
	num := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	out := make([]string, 0, len(lines))
	for i, l := range lines {
		if strings.HasPrefix(l, logMarkerEndGroup) {
			continue
		}
		out = append(out, num.Render(fmt.Sprintf("%*d", gutter, i+1))+" "+styleLogLine(ansi.Truncate(strings.ReplaceAll(l, "\t", "    "), text, "…")))
	}
	return strings.Join(out, "\n")
}

func styleLogLine(l string) string {
	marker := func(prefix string, st lipgloss.Style) (string, bool) {
		if !strings.HasPrefix(l, prefix) {
			return "", false
		}
		return st.Render(strings.TrimPrefix(l, prefix)), true
	}
	for _, m := range []struct {
		prefix string
		style  lipgloss.Style
	}{
		{logMarkerError, lipgloss.NewStyle().Foreground(styles.ColorError)},
		{logMarkerWarning, lipgloss.NewStyle().Foreground(styles.ColorWarning)},
		{logMarkerSection, lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true)},
		{logMarkerCommand, lipgloss.NewStyle().Foreground(styles.ColorPrimary)},
		{logMarkerGroup, lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)},
		{logMarkerDebug, lipgloss.NewStyle().Foreground(styles.ColorMuted)},
	} {
		if out, ok := marker(m.prefix, m.style); ok {
			return out
		}
	}
	return lipgloss.NewStyle().Foreground(styles.ColorLight).Render(l)
}

func (v PipelineRunView) Update(msg tea.Msg) (PipelineRunView, tea.Cmd) {
	if v.run == nil {
		return v, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		if v.tab == RunLogsTab {
			v.logView, cmd = v.logView.Update(msg)
			v.follow = v.logView.AtBottom()
			return v, cmd
		}
		v.summary, cmd = v.summary.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v PipelineRunView) handleKey(msg tea.KeyPressMsg) (PipelineRunView, tea.Cmd) {
	k := v.keys
	run := *v.run
	switch {
	case key.Matches(msg, k.Back) && v.tab == RunLogsTab && v.pane > runPaneJobs:
		v.pane--
		v.render()
		return v, nil
	case key.Matches(msg, k.Back):
		return v, func() tea.Msg { return CloseRunViewMsg{} }
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
		return v.handleLogsKey(msg)
	}
	return v.handleSummaryKey(msg)
}

func (v *PipelineRunView) setTab(tab RunTab) {
	if v.tab == tab {
		return
	}
	v.tab = tab
	v.render()
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
	switch {
	case key.Matches(msg, k.Up):
		v.moveJob(-1)
	case key.Matches(msg, k.Down):
		v.moveJob(1)
	case key.Matches(msg, k.Top):
		v.moveJob(-len(v.jobs()))
	case key.Matches(msg, k.Bottom):
		v.moveJob(len(v.jobs()))
	case key.Matches(msg, k.Select):
		if _, ok := v.selectedJob(); ok {
			v.tab, v.pane = RunLogsTab, runPaneSteps
			v.render()
		}
	default:
		var cmd tea.Cmd
		v.summary, cmd = v.summary.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *PipelineRunView) moveJob(delta int) {
	jobs := v.jobs()
	if len(jobs) == 0 {
		return
	}
	next := min(max(v.job+delta, 0), len(jobs)-1)
	if next == v.job {
		return
	}
	v.job, v.picked = next, true
	v.step = defaultStep(jobs[next])
	v.follow = true
	v.layout()
}

func (v *PipelineRunView) moveStep(delta int) {
	j, ok := v.selectedJob()
	if !ok || len(j.Steps) == 0 {
		return
	}
	next := min(max(v.step+delta, 0), len(j.Steps)-1)
	if next == v.step {
		return
	}
	v.step, v.picked, v.follow = next, true, true
	v.renderSteps()
	v.renderLog()
}

func (v PipelineRunView) handleLogsKey(msg tea.KeyPressMsg) (PipelineRunView, tea.Cmd) {
	k := v.keys
	switch {
	case key.Matches(msg, k.NextPanel):
		v.pane = (v.pane + 1) % runPanes
		v.render()
		return v, nil
	case key.Matches(msg, k.PrevPanel):
		v.pane = (v.pane + runPanes - 1) % runPanes
		v.render()
		return v, nil
	case key.Matches(msg, k.FollowLog):
		v.follow = !v.follow
		if v.follow {
			v.logView.GotoBottom()
		}
		return v, nil
	case key.Matches(msg, k.Select) && v.pane < runPaneLog:
		v.pane++
		v.render()
		return v, nil
	}
	switch v.pane {
	case runPaneJobs:
		v.moveBy(msg, v.moveJob)
	case runPaneSteps:
		v.moveBy(msg, v.moveStep)
	default:
		switch {
		case key.Matches(msg, k.Top):
			v.logView.GotoTop()
		case key.Matches(msg, k.Bottom):
			v.logView.GotoBottom()
		default:
			var cmd tea.Cmd
			v.logView, cmd = v.logView.Update(msg)
			v.follow = v.logView.AtBottom()
			return v, cmd
		}
		v.follow = v.logView.AtBottom()
	}
	return v, nil
}

func (v *PipelineRunView) moveBy(msg tea.KeyPressMsg, move func(int)) {
	k := v.keys
	switch {
	case key.Matches(msg, k.Up):
		move(-1)
	case key.Matches(msg, k.Down):
		move(1)
	case key.Matches(msg, k.Top):
		move(-1 << 20)
	case key.Matches(msg, k.Bottom):
		move(1 << 20)
	}
}

func (v *PipelineRunView) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("summary/logs", k.Panel1, k.Panel2)
	back := widgets.Relabel(k.Back, "back")
	again := widgets.Relabel(k.RunPipeline, "run again")
	if v.tab == RunSummaryTab {
		return []key.Binding{widgets.Relabel(k.Select, "job logs"), widgets.Combine("job", k.Up, k.Down), again, tabs, k.OpenBrowser, back}
	}
	follow := widgets.Relabel(k.FollowLog, "follow")
	if v.follow {
		follow = widgets.Relabel(k.FollowLog, "stop following")
	}
	move := widgets.Combine("job", k.Up, k.Down)
	switch v.pane {
	case runPaneSteps:
		move = widgets.Combine("step", k.Up, k.Down)
	case runPaneLog:
		move = widgets.Combine("scroll", k.Up, k.Down)
	}
	return []key.Binding{widgets.Relabel(k.NextPanel, "pane"), move, follow, again, tabs, k.OpenBrowser, back}
}

func (v *PipelineRunView) Mode() string {
	if v.tab == RunLogsTab {
		return "LOGS"
	}
	return "RUN"
}

func (v *PipelineRunView) header() string {
	r := *v.run
	inner := max(v.width-2, 1)
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(" · " + r.Title())
	first := RunBadge(r.State()) + " " + runNumber(r) + title
	parts := []string{r.PipelineName, r.ReasonLabel()}
	for _, p := range []string{r.Branch(), r.ShortCommit(), r.RequestedFor.DisplayName, RunDuration(r, time.Now())} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	meta := Muted(strings.Join(parts, " · "))
	tabs := Tabs([]string{v.keys.Panel1.Help().Key + " Summary", v.keys.Panel2.Help().Key + " Logs"}, int(v.tab), true)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, v.width))
	return strings.Join([]string{" " + Line(first, inner), " " + Line(meta, inner), Line(tabs, v.width), rule}, "\n")
}

func (v *PipelineRunView) logsBody() string {
	jw, lw, h := v.jobsWidth(), v.logsWidth(), v.bodyHeight()
	now := time.Now()
	var left []string
	index, top := 0, 0
	for _, s := range v.stages {
		left = append(left, " "+Line(lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(s.Name), jw-1))
		for _, j := range s.Jobs {
			if index == v.job {
				top = len(left)
			}
			left = append(left, v.jobRow(j, index == v.job, v.pane == runPaneJobs, jw, now, ""))
			index++
		}
	}
	if !v.stagesLoaded {
		left = []string{" " + Loading(v.spinner, loadingRun)}
	}
	start := min(max(top-h/2, 0), max(len(left)-h, 0))
	jobs := Block(strings.Join(left[start:], "\n"), jw, h)

	var right []string
	if j, ok := v.selectedJob(); ok {
		titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
		if v.pane == runPaneLog {
			titleStyle = titleStyle.Foreground(styles.ColorPrimary)
		}
		detail := Muted(RecordDuration(j.TimelineRecord, now))
		right = append(right, spreadLine(titleStyle.Render(j.Name), detail, lw))
		agent := j.Worker
		if agent == "" {
			agent = "not assigned yet"
		}
		right = append(right, Line(Muted("Agent: "+agent), lw))
	} else {
		right = append(right, "", "")
	}
	right = append(right, Block(v.stepView.View(), lw, v.stepRows()))
	follow := Muted("not following")
	if v.follow {
		follow = Badge("FOLLOWING", styles.ColorPrimary, styles.ColorOnAccent)
	}
	ruleColor := styles.ColorBorder
	if v.pane == runPaneLog {
		ruleColor = styles.ColorPrimary
	}
	label := " Log "
	right = append(right, lipgloss.NewStyle().Foreground(ruleColor).Render(ruleGlyph+label+strings.Repeat(ruleGlyph, max(lw-lipgloss.Width(label)-lipgloss.Width(follow)-2, 0)))+" "+follow)
	right = append(right, Block(v.logView.View(), lw, max(h-len(right), 1)))
	logs := Block(strings.Join(right, "\n"), lw, h)
	divider := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, jobs, " ", divider, " ", logs)
}

func (v PipelineRunView) View() string {
	if v.run == nil || v.width <= 0 || v.height <= 0 {
		return ""
	}
	var body string
	if v.tab == RunLogsTab {
		body = Block(v.logsBody(), v.width, v.bodyHeight())
	} else {
		body = v.summary.View()
		if side := v.sidebarWidth(); side > 0 {
			sidebar := Block(strings.Join(v.sideSections(side), "\n\n"), side, v.bodyHeight())
			body = lipgloss.JoinHorizontal(lipgloss.Top, Block(body, v.mainWidth(), v.bodyHeight()), strings.Repeat(" ", styles.DetailColumnGap), sidebar)
		}
		body = lipgloss.NewStyle().PaddingLeft(1).Render(Block(body, max(v.width-2, 1), v.bodyHeight()))
	}
	return Block(v.header()+"\n"+body, v.width, v.height)
}

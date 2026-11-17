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
	runHeaderHeight = 4
	runIssuesShown  = 20
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

func (v *PipelineRunView) layout() {
	v.summary.SetWidth(v.mainWidth())
	v.summary.SetHeight(v.bodyHeight())
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
	parts := []string{v.stagesSection(width, now)}
	if issues := models.RunIssues(v.stages); len(issues) > 0 {
		parts = append(parts, issuesSection(issues, width))
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
			lines = append(lines, jobRow(j, selected, v.tab == RunSummaryTab, width, now, "  "))
			index++
		}
	}
	if v.tab == RunSummaryTab {
		widgets.Reveal(&v.summary, top, top)
	}
	return strings.Join(lines, "\n")
}

func issuesSection(issues []models.RunIssue, width int) string {
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
	case key.Matches(msg, k.Back) && v.tab == RunLogsTab && v.pane > runPaneJobs:
		v.pane--
		v.renderLogs()
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
		}
	default:
		var cmd tea.Cmd
		v.summary, cmd = v.summary.Update(msg)
		return v, cmd
	}
	v.render()
	return v, nil
}

func (v *PipelineRunView) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("summary/logs", k.Panel1, k.Panel2)
	back := widgets.Relabel(k.Back, "back")
	again := widgets.Relabel(k.RunPipeline, "run again")
	if v.tab == RunSummaryTab {
		return []key.Binding{widgets.Relabel(k.Select, "job logs"), widgets.Combine("job", k.Up, k.Down), again, tabs, k.Share, k.OpenBrowser, back}
	}
	return append(v.logsBindings(), again, tabs, k.Share, k.OpenBrowser, back)
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

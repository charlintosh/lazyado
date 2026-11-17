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

const (
	runPaneJobs = iota
	runPaneSteps
	runPaneLog
	runPanes
)

const (
	runJobsMinWidth   = 22
	runJobsMaxWidth   = 36
	runStepsMinRows   = 3
	runLogGutterMin   = 4
	waitingForStep    = "Waiting for this step to start."
	noLogForStep      = "This step has no log."
	noJobsYet         = "No jobs yet."
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

type runLog struct {
	lines     []string
	done      bool
	loading   bool
	truncated bool
	err       string
}

type runLogs struct {
	stages       []models.RunStage
	stagesLoaded bool
	logs         map[int]runLog
	stale        bool
	picked       bool
	job          int
	step         int
	pane         int
	follow       bool
	logView      viewport.Model
	stepView     viewport.Model
	lkeys        keys.KeyMap
	lspinner     string
	lw           int
	lh           int
}

func newRunLogs(k keys.KeyMap) runLogs {
	return runLogs{logs: map[int]runLog{}, follow: true, lkeys: k, logView: widgets.NewViewport(k), stepView: widgets.NewViewport(k)}
}

func (l *runLogs) setStages(stages []models.RunStage) {
	l.stages, l.stagesLoaded = stages, true
	if !l.picked {
		l.pickDefault()
	}
	l.clamp()
	l.layoutLogs()
}

func (l *runLogs) setLog(logID, from int, lines []string, truncated bool, err error) {
	entry := l.logs[logID]
	entry.loading = false
	if err != nil {
		entry.err = err.Error()
	} else {
		entry.err = ""
		keep := min(max(from-1, 0), len(entry.lines))
		entry.lines = append(entry.lines[:keep:keep], lines...)
		entry.truncated = truncated
		entry.done = truncated || !l.recordActive(logID)
	}
	l.logs[logID] = entry
	if logID == l.selectedLogID() {
		l.renderLog()
	}
}

func (l *runLogs) recordActive(logID int) bool {
	if r, ok := l.record(logID); ok {
		return r.State.Active()
	}
	return false
}

func (l *runLogs) record(logID int) (models.TimelineRecord, bool) {
	for _, j := range l.jobs() {
		if j.LogID == logID {
			return j.TimelineRecord, true
		}
		for _, s := range j.Steps {
			if s.LogID == logID {
				return s, true
			}
		}
	}
	return models.TimelineRecord{}, false
}

func (l *runLogs) markStale() { l.stale = true }

func (l *runLogs) wanted() (logID, from int, ok bool) {
	logID = l.selectedLogID()
	if logID == 0 {
		return 0, 0, false
	}
	entry, seen := l.logs[logID]
	switch {
	case !seen:
		return logID, 1, true
	case entry.loading || entry.done || !l.stale:
		return 0, 0, false
	}
	return logID, len(entry.lines) + 1, true
}

func (l *runLogs) markRequested(logID int) {
	entry := l.logs[logID]
	entry.loading = true
	l.logs[logID] = entry
	l.stale = false
	if len(entry.lines) == 0 {
		l.renderLog()
	}
}

func (l *runLogs) logLoading() bool {
	entry, ok := l.logs[l.selectedLogID()]
	return ok && entry.loading && len(entry.lines) == 0
}

func (l *runLogs) jobs() []models.RunJob {
	var out []models.RunJob
	for _, s := range l.stages {
		out = append(out, s.Jobs...)
	}
	return out
}

func (l *runLogs) selectedJob() (models.RunJob, bool) {
	jobs := l.jobs()
	if l.job < 0 || l.job >= len(jobs) {
		return models.RunJob{}, false
	}
	return jobs[l.job], true
}

func (l *runLogs) selectedStep() (models.TimelineRecord, bool) {
	j, ok := l.selectedJob()
	if !ok || l.step < 0 || l.step >= len(j.Steps) {
		return models.TimelineRecord{}, false
	}
	return j.Steps[l.step], true
}

func (l *runLogs) selectedLogID() int {
	if s, ok := l.selectedStep(); ok {
		return s.LogID
	}
	if j, ok := l.selectedJob(); ok && len(j.Steps) == 0 {
		return j.LogID
	}
	return 0
}

func (l *runLogs) selectJob(index int) {
	jobs := l.jobs()
	if index < 0 || index >= len(jobs) {
		return
	}
	l.job, l.picked, l.follow = index, true, true
	l.step = defaultStep(jobs[index])
	l.layoutLogs()
}

func (l *runLogs) pickDefault() {
	jobs := l.jobs()
	if len(jobs) == 0 {
		return
	}
	l.picked = true
	l.job = len(jobs) - 1
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		for i, j := range jobs {
			if j.State == want {
				l.job = i
				l.step = defaultStep(j)
				return
			}
		}
	}
	l.step = defaultStep(jobs[l.job])
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

func (l *runLogs) clamp() {
	jobs := l.jobs()
	l.job = min(max(l.job, 0), max(len(jobs)-1, 0))
	if j, ok := l.selectedJob(); ok {
		l.step = min(max(l.step, 0), max(len(j.Steps)-1, 0))
	}
}

func (l *runLogs) resizeLogs(width, height int) {
	l.lw, l.lh = width, height
	l.layoutLogs()
}

func (l *runLogs) jobsWidth() int {
	return min(max(l.lw/4, runJobsMinWidth), runJobsMaxWidth)
}

func (l *runLogs) logsWidth() int {
	return max(l.lw-l.jobsWidth()-3, 1)
}

func (l *runLogs) stepRows() int {
	j, _ := l.selectedJob()
	return min(max(len(j.Steps), 1), max(l.lh/3, runStepsMinRows))
}

func (l *runLogs) layoutLogs() {
	l.stepView.SetWidth(l.logsWidth())
	l.stepView.SetHeight(l.stepRows())
	l.logView.SetWidth(l.logsWidth())
	l.logView.SetHeight(max(l.lh-l.stepRows()-3, 1))
	l.renderLogs()
}

func (l *runLogs) renderLogs() {
	if l.lw <= 0 {
		return
	}
	l.renderSteps()
	l.renderLog()
}

func jobRow(j models.RunJob, selected, focused bool, width int, now time.Time, indent string) string {
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

func (l *runLogs) renderSteps() {
	j, ok := l.selectedJob()
	if !ok {
		l.stepView.SetContent("")
		return
	}
	width := l.logsWidth()
	now := time.Now()
	lines := make([]string, 0, len(j.Steps))
	for i, s := range j.Steps {
		lines = append(lines, jobRow(models.RunJob{TimelineRecord: s}, i == l.step, l.pane == runPaneSteps, width, now, ""))
	}
	if len(lines) == 0 {
		lines = append(lines, Muted(" No steps yet."))
	}
	l.stepView.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&l.stepView, l.step, l.step)
}

func (l *runLogs) renderLog() {
	if l.lw <= 0 {
		return
	}
	width := l.logsWidth()
	logID := l.selectedLogID()
	entry, seen := l.logs[logID]
	var content string
	switch {
	case !l.stagesLoaded:
		content = Loading(l.lspinner, loadingRun)
	case logID == 0 && l.selectedStepQueued():
		content = Muted(waitingForStep)
	case logID == 0:
		content = Muted(noLogForStep)
	case entry.err != "":
		content = lipgloss.NewStyle().Foreground(styles.ColorError).Width(width).Render(entry.err)
	case !seen || (entry.loading && len(entry.lines) == 0):
		content = Loading(l.lspinner, loadingLog)
	default:
		content = renderLogLines(entry.lines, width)
		if entry.truncated {
			content += "\n" + lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(logTruncatedNote)
		}
	}
	atBottom := l.logView.AtBottom()
	l.logView.SetContent(content)
	if l.follow || atBottom {
		l.logView.GotoBottom()
	}
}

func (l *runLogs) selectedStepQueued() bool {
	if s, ok := l.selectedStep(); ok {
		return s.State == models.RunQueued
	}
	j, ok := l.selectedJob()
	return ok && j.State == models.RunQueued
}

func renderLogLines(lines []string, width int) string {
	gutter := max(len(strconv.Itoa(len(lines))), runLogGutterMin)
	text := max(width-gutter-1, 1)
	num := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, logMarkerEndGroup) {
			continue
		}
		out = append(out, num.Render(fmt.Sprintf("%*d", gutter, i+1))+" "+styleLogLine(ansi.Truncate(strings.ReplaceAll(line, "\t", "    "), text, "…")))
	}
	return strings.Join(out, "\n")
}

func styleLogLine(line string) string {
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
		if strings.HasPrefix(line, m.prefix) {
			return m.style.Render(strings.TrimPrefix(line, m.prefix))
		}
	}
	return lipgloss.NewStyle().Foreground(styles.ColorLight).Render(line)
}

func (l *runLogs) moveJob(delta int) {
	jobs := l.jobs()
	if len(jobs) == 0 {
		return
	}
	next := min(max(l.job+delta, 0), len(jobs)-1)
	if next != l.job {
		l.selectJob(next)
	}
}

func (l *runLogs) moveStep(delta int) {
	j, ok := l.selectedJob()
	if !ok || len(j.Steps) == 0 {
		return
	}
	next := min(max(l.step+delta, 0), len(j.Steps)-1)
	if next == l.step {
		return
	}
	l.step, l.picked, l.follow = next, true, true
	l.renderLogs()
}

func (l *runLogs) scrollLog(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	l.logView, cmd = l.logView.Update(msg)
	l.follow = l.logView.AtBottom()
	return cmd
}

func (l *runLogs) handleLogsKey(msg tea.KeyPressMsg) tea.Cmd {
	k := l.lkeys
	switch {
	case key.Matches(msg, k.NextPanel):
		l.pane = (l.pane + 1) % runPanes
	case key.Matches(msg, k.PrevPanel):
		l.pane = (l.pane + runPanes - 1) % runPanes
	case key.Matches(msg, k.FollowLog):
		l.follow = !l.follow
		if l.follow {
			l.logView.GotoBottom()
		}
		return nil
	case key.Matches(msg, k.Select) && l.pane < runPaneLog:
		l.pane++
	case l.pane == runPaneJobs:
		l.moveBy(msg, l.moveJob)
	case l.pane == runPaneSteps:
		l.moveBy(msg, l.moveStep)
	case key.Matches(msg, k.Top):
		l.logView.GotoTop()
		l.follow = false
		return nil
	case key.Matches(msg, k.Bottom):
		l.logView.GotoBottom()
		l.follow = true
		return nil
	default:
		return l.scrollLog(msg)
	}
	l.renderLogs()
	return nil
}

func (l *runLogs) moveBy(msg tea.KeyPressMsg, move func(int)) {
	k := l.lkeys
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

func (l *runLogs) logsBindings() []key.Binding {
	k := l.lkeys
	follow := widgets.Relabel(k.FollowLog, "follow")
	if l.follow {
		follow = widgets.Relabel(k.FollowLog, "stop following")
	}
	move := widgets.Combine("job", k.Up, k.Down)
	switch l.pane {
	case runPaneSteps:
		move = widgets.Combine("step", k.Up, k.Down)
	case runPaneLog:
		move = widgets.Combine("scroll", k.Up, k.Down)
	}
	return []key.Binding{widgets.Relabel(k.NextPanel, "pane"), move, follow}
}

func (l *runLogs) logsBody() string {
	jw, lw, h := l.jobsWidth(), l.logsWidth(), l.lh
	now := time.Now()
	var left []string
	index, top := 0, 0
	for _, s := range l.stages {
		left = append(left, " "+Line(lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(s.Name), jw-1))
		for _, j := range s.Jobs {
			if index == l.job {
				top = len(left)
			}
			left = append(left, jobRow(j, index == l.job, l.pane == runPaneJobs, jw, now, ""))
			index++
		}
	}
	switch {
	case !l.stagesLoaded:
		left = []string{" " + Loading(l.lspinner, loadingRun)}
	case index == 0:
		left = append(left, " "+Muted(noJobsYet))
	}
	start := min(max(top-h/2, 0), max(len(left)-h, 0))
	jobs := Block(strings.Join(left[start:], "\n"), jw, h)

	var right []string
	if j, ok := l.selectedJob(); ok {
		titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
		if l.pane == runPaneLog {
			titleStyle = titleStyle.Foreground(styles.ColorPrimary)
		}
		right = append(right, spreadLine(titleStyle.Render(j.Name), Muted(RecordDuration(j.TimelineRecord, now)), lw))
		agent := j.Worker
		if agent == "" {
			agent = "not assigned yet"
		}
		right = append(right, Line(Muted("Agent: "+agent), lw))
	} else {
		right = append(right, "", "")
	}
	right = append(right, Block(l.stepView.View(), lw, l.stepRows()))
	follow := Muted("not following")
	if l.follow {
		follow = Badge("FOLLOWING", styles.ColorPrimary, styles.ColorOnAccent)
	}
	ruleColor := styles.ColorBorder
	if l.pane == runPaneLog {
		ruleColor = styles.ColorPrimary
	}
	label := " Log "
	right = append(right, lipgloss.NewStyle().Foreground(ruleColor).Render(ruleGlyph+label+strings.Repeat(ruleGlyph, max(lw-lipgloss.Width(label)-lipgloss.Width(follow)-2, 0)))+" "+follow)
	right = append(right, Block(l.logView.View(), lw, max(h-len(right), 1)))
	logs := Block(strings.Join(right, "\n"), lw, h)
	divider := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.TrimSuffix(strings.Repeat("│\n", h), "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, jobs, " ", divider, " ", logs)
}

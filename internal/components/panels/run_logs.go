package panels

import (
	"fmt"
	"image/color"
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
	runPaneTree = iota
	runPaneLog
	runPanes
)

const (
	runTreeMinWidth   = 26
	runTreeMaxWidth   = 44
	runTreeShare      = 10
	runTreeParts      = 3
	runPaneGap        = 1
	runLogHeadRows    = 2
	runLogGutterMin   = 4
	runIssueLead      = 3
	treeCollapsed     = "▸ "
	treeExpanded      = "▾ "
	treeLeaf          = "  "
	treeGuide         = "  │ "
	waitingForStep    = "Waiting for this step to start."
	noLogForStep      = "This step has no log."
	noJobsYet         = "No jobs yet."
	noStageJobs       = "No jobs in this stage yet."
	loadingLog        = "Loading log"
	loadingRun        = "Loading stages"
	jobsInRun         = "Jobs in run"
	logTruncatedNote  = "Log cut at the size limit. Press o to read the rest in the browser."
	logMarkerError    = "##[error]"
	logMarkerWarning  = "##[warning]"
	logMarkerSection  = "##[section]"
	logMarkerCommand  = "##[command]"
	logMarkerGroup    = "##[group]"
	logMarkerEndGroup = "##[endgroup]"
	logMarkerDebug    = "##[debug]"
)

type treeKind int

const (
	treeStage treeKind = iota
	treeJob
	treeStep
)

type jobTreeRow struct {
	kind  treeKind
	stage int
	job   int
	step  int
}

type logGroup struct {
	start int
	end   int
	open  bool
}

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
	scoped       bool
	logs         map[int]runLog
	stale        bool
	picked       bool
	stage        int
	job          int
	step         int
	pane         int
	expanded     map[string]bool
	folds        map[int]map[int]bool
	follow       bool
	issueRaw     int
	pendingIssue bool
	rowRaw       []int
	logView      viewport.Model
	lkeys        keys.KeyMap
	lspinner     string
	lw           int
	lh           int
}

func newRunLogs(k keys.KeyMap) runLogs {
	return runLogs{
		logs:     map[int]runLog{},
		expanded: map[string]bool{},
		folds:    map[int]map[int]bool{},
		follow:   true,
		issueRaw: -1,
		lkeys:    k,
		logView:  widgets.NewViewport(k),
	}
}

func (l *runLogs) setStages(stages []models.RunStage) {
	l.stages, l.stagesLoaded = stages, true
	if !l.picked {
		l.pickDefault()
		if l.pendingIssue {
			l.selectIssue(1)
		}
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

func (l *runLogs) stageJobs(stage int) (start, end int) {
	for i, s := range l.stages {
		if i == stage {
			return start, start + len(s.Jobs)
		}
		start += len(s.Jobs)
	}
	return start, start
}

func (l *runLogs) inScope(job int) bool {
	if !l.scoped {
		return true
	}
	start, end := l.stageJobs(l.stage)
	return job >= start && job < end
}

func (l *runLogs) visibleJobs() int {
	if !l.scoped {
		return len(l.jobs())
	}
	start, end := l.stageJobs(l.stage)
	return end - start
}

func (l *runLogs) scopeTo(stage int) {
	l.scoped = true
	stage = min(max(stage, 0), max(len(l.stages)-1, 0))
	start, end := l.stageJobs(stage)
	if stage == l.stage && l.job >= start && l.job < end {
		return
	}
	l.stage, l.job, l.step, l.picked = stage, -1, -1, true
	l.follow, l.issueRaw, l.pendingIssue = true, -1, false
	if start < end {
		jobs := l.jobs()
		pick := start + pickJob(jobs[start:end])
		l.job, l.step = pick, defaultStep(jobs[pick])
		if l.step >= 0 {
			l.expanded[jobs[pick].ID] = true
		}
	}
	l.renderLogs()
}

func (l *runLogs) stageOfJob(job int) int {
	for i := range l.stages {
		if start, end := l.stageJobs(i); job >= start && job < end {
			return i
		}
	}
	return l.stage
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

func (l *runLogs) selectedRecord() (models.TimelineRecord, bool) {
	if s, ok := l.selectedStep(); ok {
		return s, true
	}
	j, ok := l.selectedJob()
	return j.TimelineRecord, ok
}

func (l *runLogs) selectedLogID() int {
	if r, ok := l.selectedRecord(); ok {
		return r.LogID
	}
	return 0
}

func (l *runLogs) selectJob(index int) {
	jobs := l.jobs()
	if index < 0 || index >= len(jobs) {
		return
	}
	l.setSelection(index, defaultStep(jobs[index]))
}

func (l *runLogs) setSelection(job, step int) {
	jobs := l.jobs()
	if job < 0 || job >= len(jobs) {
		return
	}
	if step >= 0 {
		l.expanded[jobs[job].ID] = true
	}
	changed := job != l.job || step != l.step
	l.job, l.step, l.picked = job, step, true
	l.stage = l.stageOfJob(job)
	if changed {
		l.follow, l.issueRaw = true, -1
	}
	l.renderLogs()
}

func (l *runLogs) pickDefault() {
	jobs := l.jobs()
	if len(jobs) == 0 {
		return
	}
	pick := pickJob(jobs)
	l.job, l.step, l.picked = pick, defaultStep(jobs[pick]), true
	l.stage = l.stageOfJob(pick)
	if l.step >= 0 {
		l.expanded[jobs[pick].ID] = true
	}
}

func pickJob(jobs []models.RunJob) int {
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		if i, ok := firstJob(jobs, want); ok {
			return i
		}
	}
	return len(jobs) - 1
}

func firstJob(jobs []models.RunJob, state models.RunState) (int, bool) {
	for i, j := range jobs {
		if j.State == state {
			return i, true
		}
	}
	return 0, false
}

func defaultStep(j models.RunJob) int {
	for _, want := range []models.RunState{models.RunRunning, models.RunFailed} {
		for i, s := range j.Steps {
			if s.State == want {
				return i
			}
		}
	}
	return len(j.Steps) - 1
}

func (l *runLogs) clamp() {
	jobs := l.jobs()
	l.stage = min(max(l.stage, 0), max(len(l.stages)-1, 0))
	switch {
	case !l.scoped:
		l.job = min(max(l.job, 0), max(len(jobs)-1, 0))
	case !l.inScope(l.job):
		l.job, l.step = -1, -1
	}
	if j, ok := l.selectedJob(); ok {
		l.step = min(max(l.step, -1), len(j.Steps)-1)
	}
}

func (l *runLogs) resizeLogs(width, height int) {
	l.lw, l.lh = width, height
	l.layoutLogs()
}

func (l *runLogs) treeWidth() int {
	return min(max(l.lw*runTreeShare/(runTreeShare*runTreeParts), runTreeMinWidth), runTreeMaxWidth)
}

func (l *runLogs) logPaneWidth() int {
	return max(l.lw-l.treeWidth()-runPaneGap, 0)
}

func (l *runLogs) layoutLogs() {
	w, h := PaneContentSize(l.logPaneWidth(), l.lh)
	l.logView.SetWidth(max(w, 1))
	l.logView.SetHeight(max(h-runLogHeadRows, 1))
	l.renderLogs()
}

func (l *runLogs) renderLogs() {
	if l.lw <= 0 {
		return
	}
	l.renderLog()
}

func (l *runLogs) treeRows() []jobTreeRow {
	var rows []jobTreeRow
	job := 0
	for si, s := range l.stages {
		if l.scoped && si != l.stage {
			job += len(s.Jobs)
			continue
		}
		if !l.scoped {
			rows = append(rows, jobTreeRow{kind: treeStage, stage: si, job: -1, step: -1})
		}
		for _, j := range s.Jobs {
			rows = append(rows, jobTreeRow{kind: treeJob, stage: si, job: job, step: -1})
			if l.expanded[j.ID] {
				for st := range j.Steps {
					rows = append(rows, jobTreeRow{kind: treeStep, stage: si, job: job, step: st})
				}
			}
			job++
		}
	}
	return rows
}

func (l *runLogs) selectableRows() []jobTreeRow {
	var out []jobTreeRow
	for _, r := range l.treeRows() {
		if r.kind != treeStage {
			out = append(out, r)
		}
	}
	return out
}

func (l *runLogs) isSelected(r jobTreeRow) bool {
	return r.kind != treeStage && r.job == l.job && r.step == l.step
}

func (l *runLogs) moveTree(delta int) {
	rows := l.selectableRows()
	if len(rows) == 0 {
		return
	}
	cur := 0
	for i, r := range rows {
		if l.isSelected(r) {
			cur = i
		}
	}
	next := rows[min(max(cur+delta, 0), len(rows)-1)]
	l.setSelection(next.job, next.step)
}

func (l *runLogs) expandOrDescend() {
	j, ok := l.selectedJob()
	switch {
	case !ok || l.step >= 0 || len(j.Steps) == 0:
		return
	case !l.expanded[j.ID]:
		l.expanded[j.ID] = true
	default:
		l.setSelection(l.job, 0)
	}
}

func (l *runLogs) collapseOrAscend() {
	j, ok := l.selectedJob()
	switch {
	case !ok:
		return
	case l.step >= 0:
		l.setSelection(l.job, -1)
	default:
		l.expanded[j.ID] = false
	}
}

func (l *runLogs) openSelected() {
	j, ok := l.selectedJob()
	if ok && l.step < 0 && len(j.Steps) > 0 && !l.expanded[j.ID] {
		l.expanded[j.ID] = true
		return
	}
	l.pane = runPaneLog
}

func (l *runLogs) treeView(width, height int, now time.Time) string {
	focused := l.pane == runPaneTree
	jobs := l.jobs()
	var lines []string
	top := 0
	for _, r := range l.treeRows() {
		switch r.kind {
		case treeStage:
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, stageTreeLine(l.stages[r.stage], width, now))
		case treeJob:
			if l.isSelected(r) {
				top = len(lines)
			}
			j := jobs[r.job]
			chevron := treeLeaf
			switch {
			case len(j.Steps) > 0 && l.expanded[j.ID]:
				chevron = treeExpanded
			case len(j.Steps) > 0:
				chevron = treeCollapsed
			}
			lines = append(lines, treeLine(j.TimelineRecord, "", chevron, l.isSelected(r), focused, width, now))
		case treeStep:
			if l.isSelected(r) {
				top = len(lines)
			}
			lines = append(lines, treeLine(jobs[r.job].Steps[r.step], treeGuide, "", l.isSelected(r), focused, width, now))
		}
	}
	switch {
	case !l.stagesLoaded:
		lines = []string{" " + Loading(l.lspinner, loadingRun)}
	case l.scoped && l.visibleJobs() == 0:
		lines = append(lines, " "+Muted(noStageJobs))
	case len(jobs) == 0:
		lines = append(lines, " "+Muted(noJobsYet))
	}
	start := min(max(top-height/2, 0), max(len(lines)-height, 0))
	return Block(strings.Join(lines[start:], "\n"), width, height)
}

func stageTreeLine(s models.RunStage, width int, now time.Time) string {
	right := RunGlyph(s.State)
	if d := RecordDuration(s.TimelineRecord, now); d != "" {
		right += " " + Muted(d)
	}
	return spreadLine(" "+Heading(s.Name), right, width)
}

func recordRight(r models.TimelineRecord, now time.Time) string {
	switch r.State {
	case models.RunSkipped:
		return "skipped"
	case models.RunQueued:
		return ""
	}
	return RecordDuration(r, now)
}

func treeLine(r models.TimelineRecord, guide, chevron string, selected, focused bool, width int, now time.Time) string {
	dim := r.State == models.RunSkipped || (r.State == models.RunQueued && r.Started.IsZero())
	return pickLine(r.State, r.Name, recordRight(r, now), guide, chevron, selected, focused, dim, width)
}

func pickLine(state models.RunState, name, right, guide, chevron string, selected, focused, dim bool, width int) string {
	base := lipgloss.NewStyle()
	bar := " "
	nameStyle := base.Foreground(styles.ColorText)
	if dim {
		nameStyle = base.Foreground(styles.ColorTextMuted)
	}
	switch {
	case selected && focused:
		base, bar = base.Background(styles.ColorHighlight), railBar
		nameStyle = base.Foreground(styles.ColorText).Bold(true)
	case selected:
		base = base.Background(styles.ColorHighlightDim)
		nameStyle = base.Foreground(styles.ColorLight).Bold(true)
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar) +
		base.Foreground(styles.ColorBorder).Render(guide) +
		base.Foreground(styles.ColorTextMuted).Render(chevron)
	left := lead + runGlyphStyle(state, base) + base.Render(" ") + nameStyle.Render(name)
	tail := base.Foreground(styles.ColorTextMuted).Render(right)
	return fill(left, base.Render(" ")+tail+base.Render(" "), base, width)
}

func (l *runLogs) renderLog() {
	if l.lw <= 0 {
		return
	}
	width := l.logView.Width()
	logID := l.selectedLogID()
	entry, seen := l.logs[logID]
	l.rowRaw = nil
	var content string
	switch {
	case !l.stagesLoaded:
		content = Loading(l.lspinner, loadingRun)
	case l.scoped && l.visibleJobs() == 0:
		content = Muted(noStageJobs)
	case logID == 0 && l.selectedQueued():
		content = Muted(waitingForStep)
	case logID == 0:
		content = Muted(noLogForStep)
	case entry.err != "":
		content = lipgloss.NewStyle().Foreground(styles.ColorError).Width(width).Render(entry.err)
	case !seen || (entry.loading && len(entry.lines) == 0):
		content = Loading(l.lspinner, loadingLog)
	default:
		l.resolvePendingIssue(entry)
		content, l.rowRaw = l.renderLogLines(logID, entry.lines, width)
		if entry.truncated {
			content += "\n" + lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(logTruncatedNote)
		}
	}
	atBottom := l.logView.AtBottom()
	l.logView.SetContent(content)
	switch {
	case l.issueRaw >= 0 && !l.follow:
		l.revealRaw(l.issueRaw)
	case l.follow || atBottom:
		l.logView.GotoBottom()
	}
}

func (l *runLogs) resolvePendingIssue(entry runLog) {
	if !l.pendingIssue {
		return
	}
	if rows := markedLines(entry.lines, logMarkerError); len(rows) > 0 {
		l.issueRaw, l.follow, l.pendingIssue = rows[0], false, false
		return
	}
	if entry.done {
		l.pendingIssue = false
	}
}

func (l *runLogs) revealRaw(raw int) {
	for i, r := range l.rowRaw {
		if r == raw {
			l.logView.SetYOffset(max(i-l.logView.Height()/runIssueLead, 0))
			return
		}
	}
}

func (l *runLogs) selectedQueued() bool {
	r, ok := l.selectedRecord()
	return ok && r.State == models.RunQueued
}

func markedLines(lines []string, marker string) []int {
	var out []int
	for i, line := range lines {
		if strings.HasPrefix(line, marker) {
			out = append(out, i)
		}
	}
	return out
}

func logGroups(lines []string) []logGroup {
	var out []logGroup
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], logMarkerGroup) {
			continue
		}
		g := logGroup{start: i, end: len(lines), open: true}
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(lines[j], logMarkerEndGroup) {
				g.end, g.open = j, false
				break
			}
			if strings.HasPrefix(lines[j], logMarkerGroup) {
				g.end, g.open = j-1, false
				break
			}
		}
		out = append(out, g)
		i = g.end
	}
	return out
}

func (g logGroup) hasIssues(lines []string) bool {
	for _, line := range lines[g.start:min(g.end, len(lines))] {
		if strings.HasPrefix(line, logMarkerError) || strings.HasPrefix(line, logMarkerWarning) {
			return true
		}
	}
	return false
}

func (l *runLogs) groupExpanded(logID int, g logGroup, lines []string) bool {
	if set, ok := l.folds[logID][g.start]; ok {
		return set
	}
	return g.open || g.hasIssues(lines)
}

func (l *runLogs) setFold(logID, start int, expanded bool) {
	if l.folds[logID] == nil {
		l.folds[logID] = map[int]bool{}
	}
	l.folds[logID][start] = expanded
}

func (l *runLogs) toggleFolds() {
	logID := l.selectedLogID()
	lines := l.logs[logID].lines
	groups := logGroups(lines)
	expand := false
	for _, g := range groups {
		if !l.groupExpanded(logID, g, lines) {
			expand = true
		}
	}
	for _, g := range groups {
		l.setFold(logID, g.start, expand)
	}
	l.follow = false
	l.renderLog()
}

func (l *runLogs) openGroupAt(raw int) {
	logID := l.selectedLogID()
	for _, g := range logGroups(l.logs[logID].lines) {
		if raw > g.start && raw <= g.end {
			l.setFold(logID, g.start, true)
		}
	}
}

func (l *runLogs) renderLogLines(logID int, lines []string, width int) (string, []int) {
	gutter := max(len(strconv.Itoa(len(lines))), runLogGutterMin)
	text := max(width-gutter-1, 1)
	groups := map[int]logGroup{}
	for _, g := range logGroups(lines) {
		groups[g.start] = g
	}
	out := make([]string, 0, len(lines))
	raws := make([]int, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, logMarkerEndGroup) {
			continue
		}
		base := lipgloss.NewStyle()
		num := base.Foreground(styles.ColorMuted)
		switch {
		case i == l.issueRaw:
			base = base.Background(styles.ColorErrorCurrent)
			num = base.Foreground(styles.ColorText).Bold(true)
		case strings.HasPrefix(line, logMarkerError):
			base = base.Background(styles.ColorErrorTint)
			num = base.Foreground(styles.ColorMuted)
		case strings.HasPrefix(line, logMarkerWarning):
			base = base.Background(styles.ColorWarningTint)
			num = base.Foreground(styles.ColorMuted)
		}
		var body string
		if g, ok := groups[i]; ok {
			open := l.groupExpanded(logID, g, lines)
			body = groupHeader(line, g, open, base, text)
			if !open {
				out = append(out, num.Render(fmt.Sprintf("%*d", gutter, i+1))+base.Render(" ")+body)
				raws = append(raws, i)
				i = g.end
				continue
			}
		} else {
			body = styleLogLine(ansi.Truncate(strings.ReplaceAll(line, "\t", "    "), text, "…"), base)
		}
		pad := base.Render(strings.Repeat(" ", max(text-lipgloss.Width(body), 0)))
		out = append(out, num.Render(fmt.Sprintf("%*d", gutter, i+1))+base.Render(" ")+body+pad)
		raws = append(raws, i)
	}
	return strings.Join(out, "\n"), raws
}

func groupHeader(line string, g logGroup, open bool, base lipgloss.Style, width int) string {
	name := strings.TrimPrefix(line, logMarkerGroup)
	chevron := treeExpanded
	hidden := ""
	if !open {
		chevron = treeCollapsed
		hidden = "  " + countOf(max(g.end-g.start-1, 0), "line")
	}
	head := base.Foreground(styles.ColorTextMuted).Render(chevron) + base.Foreground(styles.ColorText).Bold(true).Render(name)
	return ansi.Truncate(head+base.Foreground(styles.ColorMuted).Render(hidden), width, "…")
}

func styleLogLine(line string, base lipgloss.Style) string {
	for _, m := range []struct {
		prefix string
		lead   string
		style  lipgloss.Style
	}{
		{logMarkerError, "✗ ", base.Foreground(styles.ColorError)},
		{logMarkerWarning, "! ", base.Foreground(styles.ColorWarning)},
		{logMarkerSection, "▶ ", base.Foreground(styles.ColorAccent).Bold(true)},
		{logMarkerCommand, "", base.Foreground(styles.ColorPrimary)},
		{logMarkerDebug, "", base.Foreground(styles.ColorMuted)},
	} {
		if strings.HasPrefix(line, m.prefix) {
			return m.style.Render(m.lead + strings.TrimPrefix(line, m.prefix))
		}
	}
	return base.Foreground(styles.ColorLight).Render(line)
}

func (l *runLogs) issueTargets() [][2]int {
	jobs := l.jobs()
	var out [][2]int
	seen := map[[2]int]bool{}
	add := func(job, step int) {
		pos := [2]int{job, step}
		if !seen[pos] {
			seen[pos] = true
			out = append(out, pos)
		}
	}
	for _, is := range models.RunIssues(l.stages) {
		if !is.Error {
			continue
		}
		if job, step, ok := l.recordPosition(is.RecordID); ok {
			add(job, step)
		}
	}
	for ji, j := range jobs {
		for si, s := range j.Steps {
			if s.State == models.RunFailed {
				add(ji, si)
			}
		}
	}
	if !l.scoped {
		return out
	}
	kept := out[:0]
	for _, t := range out {
		if l.inScope(t[0]) {
			kept = append(kept, t)
		}
	}
	return kept
}

func (l *runLogs) recordPosition(id string) (job, step int, ok bool) {
	jobs := l.jobs()
	for ji, j := range jobs {
		if j.ID == id {
			return ji, defaultStep(j), true
		}
		for si, s := range j.Steps {
			if s.ID == id {
				return ji, si, true
			}
		}
	}
	for si, s := range l.stages {
		if s.ID != id {
			continue
		}
		start, end := l.stageJobs(si)
		for ji := start; ji < end; ji++ {
			if jobs[ji].State == models.RunFailed {
				return ji, defaultStep(jobs[ji]), true
			}
		}
	}
	return 0, 0, false
}

func (l *runLogs) selectIssue(dir int) bool {
	targets := l.issueTargets()
	if len(targets) == 0 {
		return false
	}
	cur := -1
	for i, t := range targets {
		if t == [2]int{l.job, l.step} {
			cur = i
		}
	}
	next := 0
	switch {
	case cur >= 0:
		next = (cur + dir + len(targets)) % len(targets)
	case dir < 0:
		next = len(targets) - 1
	}
	if next == cur {
		return false
	}
	l.pendingIssue = true
	l.setSelection(targets[next][0], targets[next][1])
	return true
}

func (l *runLogs) jumpIssue(dir int) {
	rows := markedLines(l.logs[l.selectedLogID()].lines, logMarkerError)
	if len(rows) == 0 {
		l.selectIssue(dir)
		return
	}
	next := rows[0]
	if dir < 0 {
		next = rows[len(rows)-1]
	}
	for i := range rows {
		r := rows[i]
		if dir < 0 {
			r = rows[len(rows)-1-i]
		}
		if (dir > 0 && r > l.issueRaw) || (dir < 0 && l.issueRaw >= 0 && r < l.issueRaw) {
			next = r
			break
		}
	}
	l.issueRaw, l.follow = next, false
	l.openGroupAt(next)
	l.renderLog()
}

func (l *runLogs) startAtFirstError() {
	l.pendingIssue = true
	if l.stagesLoaded {
		targets := l.issueTargets()
		if len(targets) > 0 {
			l.setSelection(targets[0][0], targets[0][1])
		}
	}
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
	case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel):
		l.pane = (l.pane + 1) % runPanes
	case key.Matches(msg, k.FollowLog):
		l.follow = !l.follow
		if l.follow {
			l.issueRaw = -1
			l.logView.GotoBottom()
		}
	case key.Matches(msg, k.NextIssue):
		l.jumpIssue(1)
	case key.Matches(msg, k.PrevIssue):
		l.jumpIssue(-1)
	case key.Matches(msg, k.ToggleRows):
		l.toggleFolds()
	case l.pane == runPaneTree:
		l.handleTreeKey(msg)
	case key.Matches(msg, k.Left):
		l.pane = runPaneTree
	case key.Matches(msg, k.Top):
		l.logView.GotoTop()
		l.follow = false
	case key.Matches(msg, k.Bottom):
		l.logView.GotoBottom()
		l.follow = true
	default:
		return l.scrollLog(msg)
	}
	return nil
}

func (l *runLogs) handleTreeKey(msg tea.KeyPressMsg) {
	k := l.lkeys
	switch {
	case key.Matches(msg, k.Up):
		l.moveTree(-1)
	case key.Matches(msg, k.Down):
		l.moveTree(1)
	case key.Matches(msg, k.Top):
		l.moveTree(-1 << 20)
	case key.Matches(msg, k.Bottom):
		l.moveTree(1 << 20)
	case key.Matches(msg, k.Right):
		l.expandOrDescend()
	case key.Matches(msg, k.Left):
		l.collapseOrAscend()
	case key.Matches(msg, k.Select):
		l.openSelected()
	}
}

func (l *runLogs) logsBindings() []key.Binding {
	k := l.lkeys
	follow := widgets.Relabel(k.FollowLog, "follow")
	if l.follow {
		follow = widgets.Relabel(k.FollowLog, "stop following")
	}
	issue := widgets.Relabel(k.NextIssue, "next error")
	if l.pane == runPaneLog {
		return []key.Binding{widgets.Combine("scroll", k.Up, k.Down), widgets.Relabel(k.NextPanel, "jobs"), issue, widgets.Relabel(k.ToggleRows, "fold groups"), follow}
	}
	return []key.Binding{widgets.Combine("step", k.Up, k.Down), widgets.Combine("expand", k.Left, k.Right), widgets.Relabel(k.NextPanel, "log"), issue, follow}
}

func (l *runLogs) paneLabel() string {
	if l.pane == runPaneLog {
		return "LOG"
	}
	return "JOBS"
}

func (l *runLogs) logsBody() string {
	now := time.Now()
	tw, lw := l.treeWidth(), l.logPaneWidth()
	treeFocused := l.pane == runPaneTree
	iw, ih := PaneContentSize(tw, l.lh)
	title := spreadLine(PaneTitle("", l.treeTitle(), treeFocused), Muted(countOf(l.visibleJobs(), "job")), iw)
	tree := Pane(title, l.treeView(iw, ih, now), tw, l.lh, treeFocused)
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, strings.Repeat(" ", runPaneGap), l.logPane(lw, now))
}

func (l *runLogs) treeTitle() string {
	if !l.scoped || l.stage >= len(l.stages) {
		return jobsInRun
	}
	return "Jobs · " + l.stages[l.stage].Name
}

func (l *runLogs) logPane(width int, now time.Time) string {
	focused := l.pane == runPaneLog
	iw, ih := PaneContentSize(width, l.lh)
	r, ok := l.selectedRecord()
	if !ok {
		return Pane(PaneTitle("", "Log", focused), Block(l.logView.View(), iw, ih), width, l.lh, focused)
	}
	titleStyle := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	if focused {
		titleStyle = titleStyle.Foreground(styles.ColorPrimary)
	}
	title := spreadLine(RunGlyph(r.State)+" "+titleStyle.Render(r.Name)+"  "+RunStateText(r.State, r.State.Name()), Muted(RecordDuration(r, now)), iw)
	body := []string{l.logCrumb(iw), l.logToolbar(r, iw), l.logView.View()}
	return Pane(title, Block(strings.Join(body, "\n"), iw, ih), width, l.lh, focused)
}

func (l *runLogs) logCrumb(width int) string {
	j, _ := l.selectedJob()
	path := []string{l.stages[l.stageOfJob(l.job)].Name, j.Name}
	if s, ok := l.selectedStep(); ok {
		path = append(path, s.Name)
	}
	agent := j.Worker
	if agent == "" {
		agent = "not assigned yet"
	}
	started := ""
	if r, ok := l.selectedRecord(); ok && !r.Started.IsZero() {
		started = "started " + formatStamp(r.Started)
	}
	crumb := Muted(strings.Join(path, " › "))
	for _, right := range []string{joinMeta("Agent: "+agent, started), "Agent: " + agent} {
		if lipgloss.Width(crumb)+lipgloss.Width(right)+1 <= width {
			return spreadLine(crumb, Muted(right), width)
		}
	}
	return Line(crumb, width)
}

func (l *runLogs) logToolbar(r models.TimelineRecord, width int) string {
	entry := l.logs[r.LogID]
	errors := len(markedLines(entry.lines, logMarkerError))
	warnings := len(markedLines(entry.lines, logMarkerWarning))
	var left []string
	if errors > 0 {
		left = append(left, issueChip(RunGlyph(models.RunFailed), countOf(errors, "error"), styles.ColorErrorTint))
	}
	if warnings > 0 {
		left = append(left, issueChip(RunGlyph(models.RunPartial), countOf(warnings, "warning"), styles.ColorWarningTint))
	}
	if errors > 0 {
		left = append(left, Muted(l.lkeys.NextIssue.Help().Key+"/"+l.lkeys.PrevIssue.Help().Key+" jump"))
	}
	right := l.logPosition(entry)
	if r.State.Active() {
		right = Muted(l.lkeys.FollowLog.Help().Key + " follow")
		if l.follow {
			right = Badge("FOLLOWING", styles.ColorPrimary, styles.ColorOnAccent)
		}
	}
	return spreadLine(strings.Join(left, " "), right, width)
}

func issueChip(glyph, text string, tint color.Color) string {
	return lipgloss.NewStyle().Background(tint).Render(" ") + glyph + lipgloss.NewStyle().Background(tint).Foreground(styles.ColorText).Render(" "+text+" ")
}

func (l *runLogs) logPosition(entry runLog) string {
	rows := len(l.rowRaw)
	if rows == 0 || len(entry.lines) == 0 {
		return ""
	}
	first := l.logView.YOffset()
	last := min(first+l.logView.Height(), rows) - 1
	return Muted(fmt.Sprintf("lines %d–%d of %d", l.rowRaw[first]+1, l.rowRaw[max(last, first)]+1, len(entry.lines)))
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

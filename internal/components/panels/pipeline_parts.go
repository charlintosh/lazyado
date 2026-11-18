package panels

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	runBadgeWidth    = 4
	stripCell        = "  "
	fieldLabel       = 14
	loadingStages    = "Loading stages"
	noStages         = "No stages yet."
	metaSeparator    = " · "
	branchGlyph      = "⎇ "
	stageCellFilled  = "█"
	stageCellPending = "░"
	stageCellSkipped = "·"
	previewStages    = 4
	previewFailed    = 2
	previewMessage   = 3
	previewIssueGap  = 2
)

var runGlyphs = map[models.RunState]string{
	models.RunUnknown:   "○",
	models.RunQueued:    "○",
	models.RunRunning:   "◐",
	models.RunSucceeded: "✓",
	models.RunPartial:   "!",
	models.RunFailed:    "✗",
	models.RunCanceled:  "⊘",
	models.RunSkipped:   "–",
}

func runStateColor(s models.RunState) color.Color {
	switch s {
	case models.RunSucceeded:
		return styles.ColorSuccess
	case models.RunFailed:
		return styles.ColorError
	case models.RunRunning:
		return styles.ColorPrimary
	case models.RunPartial:
		return styles.ColorWarning
	case models.RunCanceled:
		return styles.ColorMuted
	}
	return styles.ColorChip
}

func glyphColor(s models.RunState) color.Color {
	if c := runStateColor(s); c != styles.ColorChip {
		return c
	}
	return styles.ColorMuted
}

func RunBadge(s models.RunState) string {
	label := s.Label()
	label += strings.Repeat(" ", max(runBadgeWidth-len(label), 0))
	fg := styles.ColorOnAccent
	if runStateColor(s) == styles.ColorChip {
		fg = styles.ColorLight
	}
	return Badge(label, runStateColor(s), fg)
}

func RunGlyph(s models.RunState) string {
	return runGlyphStyle(s, lipgloss.NewStyle())
}

func runGlyphStyle(s models.RunState, base lipgloss.Style) string {
	return base.Foreground(glyphColor(s)).Bold(true).Render(runGlyphs[s])
}

func RunPill(s models.RunState) string {
	fg := styles.ColorOnAccent
	if runStateColor(s) == styles.ColorChip {
		fg = styles.ColorLight
	}
	return Badge(runGlyphs[s]+" "+strings.ToUpper(s.Name()), runStateColor(s), fg)
}

func RunStateText(s models.RunState, text string) string {
	c := runStateColor(s)
	if c == styles.ColorChip {
		c = styles.ColorTextMuted
	}
	return lipgloss.NewStyle().Foreground(c).Render(text)
}

func runStrip(runs []models.PipelineRun) string {
	cells := make([]string, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		cells = append(cells, lipgloss.NewStyle().Background(runStateColor(runs[i].State())).Render(stripCell))
	}
	return strings.Join(cells, " ")
}

func stageCell(s models.RunState, base lipgloss.Style) string {
	switch s {
	case models.RunQueued, models.RunUnknown:
		return base.Foreground(styles.ColorBorder).Render(stageCellPending)
	case models.RunSkipped:
		return base.Foreground(styles.ColorBorder).Render(stageCellSkipped)
	}
	return base.Foreground(glyphColor(s)).Render(stageCellFilled)
}

func StageCells(states []models.RunState, limit int, base lipgloss.Style) string {
	shown := states[:min(len(states), limit)]
	var b strings.Builder
	for _, s := range shown {
		b.WriteString(stageCell(s, base))
	}
	if extra := len(states) - len(shown); extra > 0 {
		b.WriteString(base.Foreground(styles.ColorTextMuted).Render(fmt.Sprintf("+%d", extra)))
	}
	return b.String()
}

func StageStrip(states []models.RunState, width int) string {
	n := len(states)
	cells := make([]string, n)
	for i, s := range states {
		cells[i] = stageCell(s, lipgloss.NewStyle())
	}
	switch {
	case n == 0:
		return ""
	case 2*n-1 <= width:
		return strings.Join(cells, " ")
	case n <= width:
		return strings.Join(cells, "")
	}
	return Line(stageCounts(states), width)
}

func stageCounts(states []models.RunState) string {
	counts := map[models.RunState]int{}
	for _, s := range states {
		counts[s]++
	}
	var parts []string
	for _, s := range []models.RunState{models.RunSucceeded, models.RunPartial, models.RunFailed, models.RunRunning, models.RunQueued, models.RunUnknown, models.RunCanceled, models.RunSkipped} {
		if counts[s] > 0 {
			parts = append(parts, RunGlyph(s)+" "+strconv.Itoa(counts[s]))
		}
	}
	return strings.Join(parts, "  ")
}

type stageGroup struct {
	label  string
	states []models.RunState
}

var stageGroups = []stageGroup{
	{"done", []models.RunState{models.RunSucceeded, models.RunPartial}},
	{"failed", []models.RunState{models.RunFailed}},
	{"running", []models.RunState{models.RunRunning}},
	{"queued", []models.RunState{models.RunQueued, models.RunUnknown}},
	{"canceled", []models.RunState{models.RunCanceled}},
	{"skipped", []models.RunState{models.RunSkipped}},
}

func StageTally(stages []models.RunStage) string {
	var parts []string
	for _, g := range stageGroups {
		n := 0
		for _, s := range stages {
			if slices.Contains(g.states, s.State) {
				n++
			}
		}
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, g.label))
		}
	}
	return strings.Join(parts, metaSeparator)
}

func StageStates(stages []models.RunStage) []models.RunState {
	out := make([]models.RunState, len(stages))
	for i, s := range stages {
		out[i] = s.State
	}
	return out
}

func ShortAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("Jan 2")
}

func RunDuration(r models.PipelineRun, now time.Time) string {
	switch r.State() {
	case models.RunQueued:
		return "queued"
	case models.RunUnknown:
		return ""
	}
	return models.FormatDuration(r.Duration(now))
}

func RecordDuration(r models.TimelineRecord, now time.Time) string {
	if r.Started.IsZero() {
		return ""
	}
	return models.FormatDuration(r.Duration(now))
}

func runNumber(r models.PipelineRun) string {
	return lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Render("#" + r.Number)
}

func joinMeta(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, metaSeparator)
}

func branchLabel(r models.PipelineRun) string {
	if b := r.Branch(); b != "" {
		return branchGlyph + b
	}
	return ""
}

func RunMeta(r models.PipelineRun) string {
	return joinMeta(r.ReasonLabel(), branchLabel(r), r.ShortCommit(), r.RequestedFor.DisplayName)
}

func runTrigger(r models.PipelineRun) string {
	reason, who := r.ReasonLabel(), r.RequestedFor.DisplayName
	if reason != "" && who != "" {
		return reason + " by " + who
	}
	return reason + who
}

func runStarted(r models.PipelineRun) string {
	switch {
	case !r.Started.IsZero():
		return "started " + formatStamp(r.Started)
	case !r.Queued.IsZero():
		return "queued " + formatStamp(r.Queued)
	}
	return ""
}

func spreadLine(left, right string, width int) string {
	room := width - lipgloss.Width(right)
	if room < 1 {
		return Line(left, width)
	}
	left = ansi.Truncate(left, max(room-1, 0), "…")
	return left + strings.Repeat(" ", max(room-lipgloss.Width(left), 0)) + right
}

func wrapLines(text string, width, limit int, st lipgloss.Style) []string {
	if width <= 0 || text == "" {
		return nil
	}
	lines := strings.Split(lipgloss.NewStyle().Width(width).Render(text), "\n")
	if len(lines) > limit {
		lines = lines[:limit]
		lines[limit-1] = ansi.Truncate(strings.TrimRight(lines[limit-1], " ")+"…", width, "…")
	}
	for i, l := range lines {
		lines[i] = st.Render(strings.TrimRight(l, " "))
	}
	return lines
}

func stageDetail(s models.RunStage, now time.Time) string {
	switch {
	case s.State == models.RunSkipped:
		return "skipped"
	case s.State == models.RunQueued && s.Started.IsZero():
		return "queued"
	case s.State == models.RunUnknown:
		return "not run"
	}
	done, total := s.JobCounts()
	return joinMeta(fmt.Sprintf("%d/%d", done, total), RecordDuration(s.TimelineRecord, now))
}

func stageDetailText(s models.RunStage, now time.Time) string {
	if s.State == models.RunRunning {
		return RunStateText(s.State, stageDetail(s, now))
	}
	return Muted(stageDetail(s, now))
}

func stageNotRun(s models.RunStage) bool {
	switch s.State {
	case models.RunUnknown, models.RunSkipped:
		return true
	case models.RunQueued:
		return s.Started.IsZero()
	}
	return false
}

func pickStages(stages []models.RunStage) []int {
	picked := map[int]bool{}
	take := func(i int) {
		if i >= 0 && len(picked) < previewStages {
			picked[i] = true
		}
	}
	failed := 0
	for i, s := range stages {
		if s.State == models.RunFailed && failed < previewFailed {
			take(i)
			failed++
		}
	}
	for i, s := range stages {
		if s.State == models.RunRunning {
			take(i)
		}
	}
	last := -1
	for i, s := range stages {
		if !picked[i] && !s.Finished.IsZero() && !stageNotRun(s) && (last < 0 || s.Finished.After(stages[last].Finished)) {
			last = i
		}
	}
	take(last)
	for i, s := range stages {
		if !picked[i] && (s.State == models.RunQueued || s.State == models.RunUnknown) {
			take(i)
			break
		}
	}
	out := make([]int, 0, len(picked))
	for i := range stages {
		if picked[i] {
			out = append(out, i)
		}
	}
	return out
}

func moreStages(stages []models.RunStage, picked []int) string {
	var rest []models.RunStage
	for i, s := range stages {
		if !slices.Contains(picked, i) {
			rest = append(rest, s)
		}
	}
	if len(rest) == 0 {
		return ""
	}
	tally := StageTally(rest)
	if !strings.Contains(tally, metaSeparator) {
		_, label, _ := strings.Cut(tally, " ")
		return fmt.Sprintf("+ %d more %s", len(rest), label)
	}
	return fmt.Sprintf("+ %d more: %s", len(rest), strings.ReplaceAll(tally, metaSeparator, ", "))
}

func stageSummary(stages []models.RunStage) string {
	done, failed, running := 0, 0, 0
	for _, s := range stages {
		switch s.State {
		case models.RunSucceeded, models.RunPartial:
			done++
		case models.RunFailed:
			failed++
		case models.RunRunning:
			running++
		}
	}
	parts := []string{fmt.Sprintf("%d of %d done", done, len(stages))}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", running))
	}
	return joinMeta(parts...)
}

type PipelinePreviewProps struct {
	Pipeline      *models.Pipeline
	Run           *models.PipelineRun
	History       []models.PipelineRun
	HistoryLoaded bool
	Stages        []models.RunStage
	StagesLoaded  bool
	ShowPipeline  bool
	IssueKey      string
	Spinner       string
	Width         int
	Height        int
}

func RenderPipelinePreview(p PipelinePreviewProps) string {
	if p.Pipeline == nil && p.Run == nil {
		return Muted("Select a pipeline or a run.")
	}
	var sections []string
	if p.Pipeline != nil && p.ShowPipeline {
		bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
		sections = append(sections, Line(bold.Render(p.Pipeline.Name), p.Width)+"\n"+Muted(p.Pipeline.FolderLabel()))
	}
	if r := p.Run; r != nil {
		sections = append(sections, previewRunHead(*r, p))
		sections = append(sections, previewStageList(p))
		if where := previewWhere(p); where != "" {
			sections = append(sections, where)
		}
	} else if p.ShowPipeline {
		sections = append(sections, Muted("Never run"))
	}
	if history := previewHistory(p); history != "" {
		sections = append(sections, history)
	}
	return Block(strings.Join(sections, "\n\n"), p.Width, p.Height)
}

func previewRunHead(r models.PipelineRun, p PipelinePreviewProps) string {
	now := time.Now()
	pipeline := r.PipelineName
	if p.ShowPipeline {
		pipeline = ""
	}
	lines := []string{spreadLine(RunPill(r.State())+" "+runNumber(r), Muted(RunDuration(r, now)), p.Width)}
	lines = append(lines, wrapLines(r.Title(), p.Width, 2, lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true))...)
	for _, meta := range []string{joinMeta(pipeline, branchLabel(r), r.ShortCommit()), joinMeta(runTrigger(r), runStarted(r))} {
		if meta != "" {
			lines = append(lines, Line(Muted(meta), p.Width))
		}
	}
	return strings.Join(lines, "\n")
}

func previewStageList(p PipelinePreviewProps) string {
	switch {
	case !p.StagesLoaded:
		return Heading("Stages") + "\n" + Loading(p.Spinner, loadingStages)
	case len(p.Stages) == 0:
		return Heading("Stages") + "\n" + Muted(noStages)
	}
	now := time.Now()
	lines := []string{spreadLine(Heading("Stages"), Muted(stageSummary(p.Stages)), p.Width), StageStrip(StageStates(p.Stages), p.Width)}
	picked := pickStages(p.Stages)
	for _, i := range picked {
		s := p.Stages[i]
		nameStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
		if stageNotRun(s) {
			nameStyle = nameStyle.Foreground(styles.ColorTextMuted)
		}
		lines = append(lines, spreadLine(RunGlyph(s.State)+" "+nameStyle.Render(s.Name), " "+stageDetailText(s, now), p.Width))
	}
	if more := moreStages(p.Stages, picked); more != "" {
		lines = append(lines, Line(Muted("  "+more), p.Width))
	}
	return strings.Join(lines, "\n")
}

func previewWhere(p PipelinePreviewProps) string {
	if !p.StagesLoaded || p.Run == nil {
		return ""
	}
	issues := models.RunIssues(p.Stages)
	errors, warnings := issueCounts(issues)
	if errors > 0 {
		first := issues[0]
		lines := []string{Heading("Where it failed"), Line(RunGlyph(models.RunFailed)+" "+lipgloss.NewStyle().Foreground(styles.ColorText).Render(first.Where), p.Width)}
		lines = append(lines, issueBlock(first, p.Width, previewMessage)...)
		lines = append(lines, Line(Muted(joinMeta(issueCountText(errors, warnings), p.IssueKey+" open this log")), p.Width))
		return strings.Join(lines, "\n")
	}
	if where, ok := runningWhere(p.Stages); ok && p.Run.State().Active() {
		return Heading("Running now") + "\n" + Line(RunGlyph(models.RunRunning)+" "+where, p.Width)
	}
	if warnings > 0 {
		return Heading("Warnings") + "\n" + Line(Muted(issueCountText(errors, warnings)), p.Width)
	}
	return ""
}

func runningWhere(stages []models.RunStage) (string, bool) {
	for _, s := range stages {
		for _, j := range s.Jobs {
			if j.State != models.RunRunning {
				continue
			}
			for _, st := range j.Steps {
				if st.State == models.RunRunning {
					return strings.Join([]string{s.Name, j.Name, st.Name}, " › "), true
				}
			}
			return s.Name + " › " + j.Name, true
		}
	}
	return "", false
}

func issueCounts(issues []models.RunIssue) (errors, warnings int) {
	for _, is := range issues {
		if is.Error {
			errors++
		} else {
			warnings++
		}
	}
	return errors, warnings
}

func countOf(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}

func issueCountText(errors, warnings int) string {
	return countOf(errors, "error") + metaSeparator + countOf(warnings, "warning")
}

func issueBlock(is models.RunIssue, width, limit int) []string {
	tint, bar := styles.ColorWarningTint, styles.ColorWarning
	if is.Error {
		tint, bar = styles.ColorErrorTint, styles.ColorError
	}
	lead := lipgloss.NewStyle().Foreground(bar).Background(tint).Render("▌") + lipgloss.NewStyle().Background(tint).Render(" ")
	text := max(width-previewIssueGap, 1)
	body := lipgloss.NewStyle().Background(tint).Foreground(styles.ColorText)
	var out []string
	for _, l := range wrapLines(strings.TrimSpace(is.Message), text, limit, lipgloss.NewStyle()) {
		out = append(out, lead+body.Render(l+strings.Repeat(" ", max(text-lipgloss.Width(l), 0))))
	}
	return out
}

func previewHistory(p PipelinePreviewProps) string {
	switch {
	case p.Pipeline == nil:
		return ""
	case !p.HistoryLoaded:
		return Loading(p.Spinner, "Loading recent runs")
	case len(p.History) == 0:
		return ""
	}
	stats := models.StatsOf(p.History)
	summary := fmt.Sprintf("%d of %d passed", stats.Passed, stats.Completed)
	if stats.Completed > 0 {
		summary += metaSeparator + "median " + models.FormatDuration(stats.Median)
	}
	title := fmt.Sprintf("Last %d runs", len(p.History))
	if !p.ShowPipeline {
		title = p.Pipeline.Name + metaSeparator + title
	}
	return strings.Join([]string{Line(Heading(title), p.Width), runStrip(p.History), Muted(summary)}, "\n")
}

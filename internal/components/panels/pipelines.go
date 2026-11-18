package panels

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type PipelineList int

const (
	ListRecent PipelineList = iota
	ListAll
	ListRuns
	ListPipelineRuns
	ListBranches
)

const (
	pipelineRowHeight   = 2
	pipelineRowGap      = 1
	pipelineNameColumn  = 34
	pipelineColumnsShow = 96
	pipelineWhenColumn  = 11
	pipelineStageColumn = 15
	pipelineStageCells  = 12
	pipelineStagesShow  = 72
	dayToday            = "Today"
	dayYesterday        = "Yesterday"
	dayWeek             = "Last 7 days"
	dayOlder            = "Older"
	folderOpen          = "▾ "
	pipelineSearchHint  = "pipeline name"
	runSearchHint       = "text, state:failed, branch:main, pipeline:name, by:@me"
	loadingPipelines    = "Loading pipelines"
	loadingRuns         = "Loading runs"
	loadingMoreRuns     = "Loading more"
	searchingRemote     = "searching"
	moreOnScroll        = "More load as you scroll"
)

type OpenPipelineMsg struct {
	Pipeline models.Pipeline
}

type OpenRunMsg struct {
	Run     models.PipelineRun
	AtError bool
}

type BrowsePipelineMsg struct {
	PipelineID int
	URL        string
}

type MorePipelinesMsg struct{}

type SearchPipelinesMsg struct {
	Text string
}

type PipelineSearchTypedMsg struct {
	Text string
}

type pipelineRow struct {
	folder   string
	day      bool
	pipeline *models.Pipeline
	run      *models.PipelineRun
	top      int
}

func (r pipelineRow) header() bool { return r.pipeline == nil && r.run == nil }

func (r pipelineRow) key() string {
	switch {
	case r.run != nil:
		return fmt.Sprintf("r%d", r.run.ID)
	case r.pipeline != nil:
		return fmt.Sprintf("p%d", r.pipeline.ID)
	}
	return ""
}

type PipelinesPanel struct {
	mode        PipelineList
	pipelines   []models.Pipeline
	runs        []models.PipelineRun
	rows        []pipelineRow
	cursor      int
	more        bool
	loading     bool
	loadingMore bool
	search      textinput.Model
	searching   bool
	searchErr   string
	applied     string
	remote      bool
	viewport    viewport.Model
	keys        keys.KeyMap
	spinner     string
	focused     bool
	stages      map[int][]models.RunState
	width       int
	height      int
}

func NewPipelinesPanel(k keys.KeyMap) PipelinesPanel {
	return PipelinesPanel{viewport: widgets.NewViewport(k), keys: k, stages: map[int][]models.RunState{}, search: widgets.SearchInput(searchPrompt, pipelineSearchHint), loading: true}
}

func (p *PipelinesPanel) Mode() PipelineList { return p.mode }

func (p *PipelinesPanel) runMode() bool { return p.mode >= ListRuns }

func (p *PipelinesPanel) SetPipelines(mode PipelineList, pipelines []models.Pipeline, more bool) {
	p.switchMode(mode)
	p.pipelines, p.runs, p.more, p.loading, p.loadingMore = pipelines, nil, more, false, false
	p.rebuild()
}

func (p *PipelinesPanel) SetRuns(mode PipelineList, runs []models.PipelineRun, more bool) {
	p.switchMode(mode)
	p.runs, p.pipelines, p.more, p.loading, p.loadingMore = runs, nil, more, false, false
	p.rebuild()
}

func (p *PipelinesPanel) ShowLoading(mode PipelineList) {
	p.switchMode(mode)
	p.pipelines, p.runs, p.more, p.loading, p.loadingMore = nil, nil, false, true, false
	p.rebuild()
}

func (p *PipelinesPanel) switchMode(mode PipelineList) {
	if p.mode == mode {
		return
	}
	p.mode, p.cursor = mode, 0
	p.search.SetValue("")
	p.applied, p.searchErr = "", ""
	p.search.Placeholder = pipelineSearchHint
	if p.runMode() {
		p.search.Placeholder = runSearchHint
	}
}

func (p *PipelinesPanel) SetLoadingMore(loading bool) {
	p.loadingMore = loading
	p.render()
}

func (p *PipelinesPanel) Loading() bool { return p.loading || p.loadingMore }

func (p *PipelinesPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.loadingMore {
		p.render()
	}
}

func (p *PipelinesPanel) SetFocused(focused bool) {
	if p.focused == focused {
		return
	}
	p.focused = focused
	p.render()
}

func (p *PipelinesPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width-4, 1))
	p.layout()
}

func (p *PipelinesPanel) SetSearchError(err string) {
	p.searchErr = err
	p.layout()
}

func (p *PipelinesPanel) SetApplied(text string) { p.applied = text }

func (p *PipelinesPanel) SetRemoteSearch(on bool) { p.remote = on }

func (p *PipelinesPanel) Applied() string { return p.applied }

func (p *PipelinesPanel) SearchText() string { return p.search.Value() }

func (p *PipelinesPanel) Searching() bool { return p.searching }

func (p *PipelinesPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.layout()
	return p.search.Focus()
}

func (p *PipelinesPanel) blurSearch() {
	p.searching = false
	p.search.Blur()
	p.layout()
}

func (p *PipelinesPanel) searchShown() bool {
	return p.searching || p.search.Value() != "" || p.searchErr != ""
}

func (p *PipelinesPanel) searchChromeHeight() int {
	h := 0
	if p.searchShown() {
		h = 2
	}
	if p.searchErr != "" {
		h++
	}
	return h
}

func (p *PipelinesPanel) layout() {
	p.viewport.SetWidth(max(p.width, 1))
	p.viewport.SetHeight(max(p.height-p.searchChromeHeight(), 1))
	p.render()
}

func (p *PipelinesPanel) terms() []string {
	if p.runMode() {
		return models.ParseRunQuery(p.search.Value()).Terms
	}
	return models.SearchTerms(p.search.Value())
}

func (p *PipelinesPanel) rebuild() {
	selected := ""
	if p.cursor >= 0 && p.cursor < len(p.rows) {
		selected = p.rows[p.cursor].key()
	}
	p.rows = p.rows[:0:0]
	if p.runMode() {
		q := models.ParseRunQuery(p.search.Value())
		runs := p.runs
		if p.mode == ListBranches {
			runs = models.LatestPerBranch(runs)
		}
		day := ""
		now := time.Now()
		for i := range runs {
			if !q.Matches(runs[i]) {
				continue
			}
			if label := dayLabel(runs[i].When(), now); p.mode != ListBranches && label != day {
				day = label
				p.rows = append(p.rows, pipelineRow{folder: label, day: true})
			}
			p.rows = append(p.rows, pipelineRow{run: &runs[i]})
		}
	} else {
		terms := p.terms()
		folder := "\x00"
		for i := range p.pipelines {
			pl := &p.pipelines[i]
			if !models.PipelineMatches(*pl, terms) {
				continue
			}
			if p.mode == ListAll && pl.FolderLabel() != folder {
				folder = pl.FolderLabel()
				p.rows = append(p.rows, pipelineRow{folder: folder})
			}
			p.rows = append(p.rows, pipelineRow{pipeline: pl})
		}
	}
	p.cursor = p.firstSelectable(0, 1)
	for i, r := range p.rows {
		if r.key() == selected && selected != "" {
			p.cursor = i
		}
	}
	p.layout()
}

func dayLabel(t, now time.Time) string {
	if t.IsZero() {
		return dayOlder
	}
	day := func(v time.Time) time.Time {
		y, m, d := v.Local().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}
	days := int(day(now).Sub(day(t)).Hours() / 24)
	switch {
	case days <= 0:
		return dayToday
	case days == 1:
		return dayYesterday
	case days < 7:
		return dayWeek
	}
	return dayOlder
}

func (p *PipelinesPanel) SetRunStages(id int, states []models.RunState) {
	p.stages[id] = states
	p.render()
}

func (p *PipelinesPanel) StagesShown() bool {
	return p.width >= pipelineStagesShow && p.mode != ListAll
}

func (p *PipelinesPanel) VisibleRuns() []models.PipelineRun {
	top := p.viewport.YOffset()
	bottom := top + p.viewport.Height()
	var out []models.PipelineRun
	for _, r := range p.rows {
		if r.header() || r.top+pipelineRowHeight <= top || r.top >= bottom {
			continue
		}
		if run := r.latest(); run != nil {
			out = append(out, *run)
		}
	}
	return out
}

func (r pipelineRow) latest() *models.PipelineRun {
	if r.run != nil {
		return r.run
	}
	if r.pipeline != nil {
		return r.pipeline.Latest
	}
	return nil
}

func (p *PipelinesPanel) firstSelectable(from, dir int) int {
	for i := from; i >= 0 && i < len(p.rows); i += dir {
		if !p.rows[i].header() {
			return i
		}
	}
	return -1
}

func (p *PipelinesPanel) selectedRow() (pipelineRow, bool) {
	if p.cursor < 0 || p.cursor >= len(p.rows) {
		return pipelineRow{}, false
	}
	return p.rows[p.cursor], true
}

func (p *PipelinesPanel) SelectedPipeline() *models.Pipeline {
	r, ok := p.selectedRow()
	switch {
	case !ok:
		return nil
	case r.pipeline != nil:
		return r.pipeline
	}
	return &models.Pipeline{ID: r.run.PipelineID, Name: r.run.PipelineName, Folder: r.run.PipelineFolder, Latest: r.run}
}

func (p *PipelinesPanel) SelectedRun() *models.PipelineRun {
	r, ok := p.selectedRow()
	switch {
	case !ok:
		return nil
	case r.run != nil:
		return r.run
	}
	return r.pipeline.Latest
}

func (p *PipelinesPanel) Runs() []models.PipelineRun { return p.runs }

func (p *PipelinesPanel) Count() int {
	n := 0
	for _, r := range p.rows {
		if !r.header() {
			n++
		}
	}
	return n
}

func (p *PipelinesPanel) Title() string {
	noun := "Pipelines"
	switch p.mode {
	case ListRecent:
		noun = "Recently run pipelines"
	case ListRuns, ListPipelineRuns:
		noun = "Runs"
	case ListBranches:
		noun = "Branches"
	}
	if p.loading {
		return noun
	}
	total := len(p.pipelines)
	if p.runMode() {
		total = len(p.runs)
		if p.mode == ListBranches {
			total = len(models.LatestPerBranch(p.runs))
		}
	}
	if p.Count() != total {
		return fmt.Sprintf("%s %d of %d", noun, p.Count(), total)
	}
	return fmt.Sprintf("%s %d", noun, total)
}

func (p *PipelinesPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.searching {
		back := widgets.Relabel(k.Back, "done")
		if p.search.Value() != "" {
			back = widgets.Relabel(k.Back, "clear")
		}
		return []key.Binding{widgets.Relabel(k.Select, "done"), widgets.Combine("move", k.PickerUp, k.PickerDown), back}
	}
	search := widgets.Relabel(k.Search, "search")
	views := widgets.Combine("view", k.PrevPanel, k.NextPanel)
	if _, ok := p.selectedRow(); !ok {
		return []key.Binding{search, views, k.Refresh}
	}
	open := widgets.Relabel(k.Select, "open run")
	bindings := []key.Binding{open, k.RunPipeline}
	if !p.runMode() {
		bindings = []key.Binding{widgets.Relabel(k.Select, "runs"), widgets.Relabel(k.View, "last run"), k.RunPipeline}
	}
	if run := p.SelectedRun(); run != nil && (run.State() == models.RunFailed || run.State() == models.RunPartial) {
		bindings = append(bindings[:1:1], append([]key.Binding{widgets.Relabel(k.NextIssue, "first error")}, bindings[1:]...)...)
	}
	if p.mode == ListPipelineRuns || p.mode == ListBranches {
		views = widgets.Combine("runs/branches", k.Panel1, k.Panel2)
		bindings = append(bindings, search, k.Share, k.OpenBrowser, views, widgets.Relabel(k.Back, "back"), k.Refresh)
		return bindings
	}
	return append(bindings, search, k.Share, k.OpenBrowser, views, k.Refresh)
}

func (p PipelinesPanel) Update(msg tea.Msg) (PipelinesPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if p.searching {
			return p.handleSearchKey(msg)
		}
		return p.handleKey(msg)
	case tea.PasteMsg:
		if p.searching {
			return p.updateSearch(msg)
		}
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			return p, p.move(-1)
		case tea.MouseWheelDown:
			return p, p.move(1)
		}
	}
	return p, nil
}

func (p PipelinesPanel) handleKey(msg tea.KeyPressMsg) (PipelinesPanel, tea.Cmd) {
	k := p.keys
	if now, ok := ShareRequested(k, msg); ok {
		return p, p.share(now)
	}
	page := max(p.height/(pipelineRowHeight+pipelineRowGap)/2, 1)
	switch {
	case key.Matches(msg, k.Up):
		return p, p.move(-1)
	case key.Matches(msg, k.Down):
		return p, p.move(1)
	case key.Matches(msg, k.HalfPageUp):
		return p, p.move(-page)
	case key.Matches(msg, k.HalfPageDown):
		return p, p.move(page)
	case key.Matches(msg, k.Top):
		return p, p.move(-len(p.rows))
	case key.Matches(msg, k.Bottom):
		return p, p.move(len(p.rows))
	case key.Matches(msg, k.Select):
		return p, p.open()
	case key.Matches(msg, k.View):
		if run := p.SelectedRun(); run != nil {
			chosen := *run
			return p, func() tea.Msg { return OpenRunMsg{Run: chosen} }
		}
	case key.Matches(msg, k.OpenBrowser):
		return p, p.browse()
	case key.Matches(msg, k.NextIssue):
		if run := p.SelectedRun(); run != nil {
			chosen := *run
			return p, func() tea.Msg { return OpenRunMsg{Run: chosen, AtError: true} }
		}
	}
	return p, nil
}

func (p *PipelinesPanel) open() tea.Cmd {
	r, ok := p.selectedRow()
	switch {
	case !ok:
		return nil
	case r.run != nil:
		chosen := *r.run
		return func() tea.Msg { return OpenRunMsg{Run: chosen} }
	}
	chosen := *r.pipeline
	return func() tea.Msg { return OpenPipelineMsg{Pipeline: chosen} }
}

func (p *PipelinesPanel) share(now bool) tea.Cmd {
	r, ok := p.selectedRow()
	switch {
	case !ok:
		return nil
	case r.run != nil:
		return shareCmd(share.Run(*r.run, "", ""), now)
	}
	return shareCmd(share.Pipeline(*r.pipeline), now)
}

func (p *PipelinesPanel) browse() tea.Cmd {
	r, ok := p.selectedRow()
	if !ok {
		return nil
	}
	msg := BrowsePipelineMsg{}
	if r.run != nil {
		msg.PipelineID, msg.URL = r.run.PipelineID, r.run.WebURL
	} else {
		msg.PipelineID, msg.URL = r.pipeline.ID, r.pipeline.WebURL
	}
	return func() tea.Msg { return msg }
}

func (p *PipelinesPanel) move(delta int) tea.Cmd {
	if p.cursor < 0 {
		return nil
	}
	dir := 1
	if delta < 0 {
		dir = -1
	}
	next := p.cursor
	for steps := delta * dir; steps > 0; steps-- {
		candidate := p.firstSelectable(next+dir, dir)
		if candidate < 0 {
			break
		}
		next = candidate
	}
	if next != p.cursor {
		p.cursor = next
		p.render()
	}
	if p.more && !p.loadingMore && p.firstSelectable(p.cursor+1, 1) < 0 {
		return func() tea.Msg { return MorePipelinesMsg{} }
	}
	return nil
}

func (p PipelinesPanel) handleSearchKey(msg tea.KeyPressMsg) (PipelinesPanel, tea.Cmd) {
	k := p.keys
	arrow := msg.Text == ""
	switch {
	case key.Matches(msg, k.Back):
		if p.search.Value() == "" {
			p.blurSearch()
			return p, nil
		}
		p.search.SetValue("")
		p.searchErr = ""
		p.rebuild()
		if p.applied != "" {
			return p, func() tea.Msg { return SearchPipelinesMsg{} }
		}
		return p, nil
	case key.Matches(msg, k.Select):
		text := strings.TrimSpace(p.search.Value())
		p.blurSearch()
		if text == p.applied {
			return p, nil
		}
		return p, func() tea.Msg { return SearchPipelinesMsg{Text: text} }
	case arrow && key.Matches(msg, k.Up):
		return p, p.move(-1)
	case arrow && key.Matches(msg, k.Down):
		return p, p.move(1)
	}
	return p.updateSearch(msg)
}

func (p PipelinesPanel) updateSearch(msg tea.Msg) (PipelinesPanel, tea.Cmd) {
	before := p.search.Value()
	var cmd tea.Cmd
	p.search, cmd = p.search.Update(msg)
	if p.search.Value() == before {
		return p, cmd
	}
	p.searchErr = ""
	p.rebuild()
	text := strings.TrimSpace(p.search.Value())
	return p, tea.Batch(cmd, func() tea.Msg { return PipelineSearchTypedMsg{Text: text} })
}

func (p *PipelinesPanel) render() {
	if p.width <= 0 {
		return
	}
	var lines []string
	cursorTop := 0
	for i := range p.rows {
		r := &p.rows[i]
		if r.header() {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			r.top = len(lines)
			if r.day {
				lines = append(lines, " "+Heading(r.folder))
				continue
			}
			lines = append(lines, " "+lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(folderOpen+r.folder))
			continue
		}
		if len(lines) > 0 && !p.rows[i-1].header() {
			lines = append(lines, "")
		}
		r.top = len(lines)
		if i == p.cursor {
			cursorTop = r.top
		}
		lines = append(lines, p.row(*r, i == p.cursor)...)
	}
	switch {
	case p.loadingMore:
		lines = append(lines, "", " "+Loading(p.spinner, loadingMoreRuns))
	case p.more && len(lines) > 0:
		lines = append(lines, "", " "+Muted(moreOnScroll))
	}
	p.viewport.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&p.viewport, cursorTop, cursorTop+pipelineRowHeight)
}

func (p *PipelinesPanel) row(r pipelineRow, selected bool) []string {
	base := lipgloss.NewStyle()
	bar := " "
	if selected {
		base = base.Background(styles.ColorHighlight)
		if p.focused {
			bar = railBar
		}
	}
	lead := base.Foreground(styles.ColorPrimary).Render(bar) + base.Render(" ")
	inner := max(p.width-2, 1)
	var first, second [3]string
	if r.run != nil {
		first, second = p.runCells(*r.run, base, selected)
	} else {
		first, second = p.pipelineCells(*r.pipeline, base, selected)
	}
	return []string{lead + p.cells(first, base, inner), lead + p.cells(second, base, inner)}
}

func (p *PipelinesPanel) cells(c [3]string, base lipgloss.Style, width int) string {
	right := c[2]
	if right != "" {
		right = base.Render(" ") + right
	}
	if c[1] == "" || p.width < pipelineColumnsShow {
		return fill(c[0], right, base, width)
	}
	name := min(pipelineNameColumn, width/3)
	left := fill(c[0], "", base, name)
	return left + base.Render(" ") + fill(c[1], right, base, max(width-name-1, 1))
}

func fill(left, right string, base lipgloss.Style, width int) string {
	room := max(width-lipgloss.Width(right), 0)
	left = ansi.Truncate(left, room, "…")
	return left + base.Render(strings.Repeat(" ", max(room-lipgloss.Width(left), 0))) + right
}

func (p *PipelinesPanel) rightColumns(run *models.PipelineRun, base lipgloss.Style) (string, string) {
	muted := base.Foreground(styles.ColorTextMuted)
	if run == nil {
		return "", ""
	}
	when := muted.Render(fmt.Sprintf("%*s", pipelineWhenColumn, ShortAgo(run.When())))
	took := base.Foreground(durationColor(*run)).Render(fmt.Sprintf("%*s", pipelineWhenColumn, RunDuration(*run, time.Now())))
	if !p.StagesShown() {
		return when, took
	}
	pips := StageCells(p.stages[run.ID], pipelineStageCells, base)
	pad := func(s string) string {
		return s + base.Render(strings.Repeat(" ", max(pipelineStageColumn-lipgloss.Width(s), 0)))
	}
	return pad(pips) + when, pad("") + took
}

func rowStyles(base lipgloss.Style, selected bool) (lipgloss.Style, lipgloss.Style) {
	if selected {
		return base.Foreground(styles.ColorText).Bold(true), base.Foreground(styles.ColorLight)
	}
	return base.Foreground(styles.ColorText), base.Foreground(styles.ColorTextMuted)
}

func (p *PipelinesPanel) pipelineCells(pl models.Pipeline, base lipgloss.Style, selected bool) ([3]string, [3]string) {
	terms := p.terms()
	strong, muted := rowStyles(base, selected)
	name := highlight(pl.Name, terms, base.Foreground(styles.ColorText).Bold(true))
	state := models.RunUnknown
	if pl.Latest != nil {
		state = pl.Latest.State()
	}
	badge := runGlyphStyle(state, base) + base.Render(" ")
	indent := base.Render(strings.Repeat(" ", lipgloss.Width(badge)))
	run := pl.Latest
	if run == nil {
		if p.width < pipelineColumnsShow {
			return [3]string{badge + name, "", ""}, [3]string{indent + muted.Render("Never run"), "", ""}
		}
		return [3]string{badge + name, muted.Render("Never run"), ""}, [3]string{indent + muted.Render(pl.FolderLabel()), "", ""}
	}
	summary := base.Foreground(styles.ColorAccent).Bold(true).Render("#"+run.Number) + strong.UnsetBold().Render(" "+run.Title())
	first, second := p.rightColumns(run, base)
	if p.width < pipelineColumnsShow {
		return [3]string{badge + name, "", first}, [3]string{indent + summary, "", second}
	}
	return [3]string{badge + name, summary, first}, [3]string{indent + muted.Render(pl.FolderLabel()), muted.Render(run.Trigger()), second}
}

func (p *PipelinesPanel) runCells(run models.PipelineRun, base lipgloss.Style, selected bool) ([3]string, [3]string) {
	terms := p.terms()
	strong, muted := rowStyles(base, selected)
	badge := runGlyphStyle(run.State(), base) + base.Render(" ")
	indent := base.Render(strings.Repeat(" ", lipgloss.Width(badge)))
	number := base.Foreground(styles.ColorAccent).Bold(true).Render("#" + run.Number)
	title := highlight(run.Title(), terms, strong)
	first, second := p.rightColumns(&run, base)
	meta := RunMeta(run)
	switch p.mode {
	case ListRuns:
		meta = joinMeta(run.PipelineName, meta)
	case ListBranches:
		branch := highlight(branchGlyph+run.Branch(), terms, strong)
		return [3]string{badge + branch, "", first}, [3]string{indent + number + base.Render(" ") + highlight(run.Title(), terms, muted), "", second}
	}
	return [3]string{badge + number + base.Render(" ") + title, "", first}, [3]string{indent + highlight(meta, terms, muted), "", second}
}

func durationColor(r models.PipelineRun) color.Color {
	if r.State() == models.RunRunning {
		return styles.ColorPrimary
	}
	return styles.ColorTextMuted
}

func (p *PipelinesPanel) searchView() string {
	if !p.searchShown() {
		return ""
	}
	border := styles.ColorBorder
	if p.searching {
		border = styles.ColorPrimary
	}
	box := lipgloss.NewStyle().Foreground(border).Render(strings.Repeat(ruleGlyph, max(p.width, 0)))
	input := Line(p.search.View(), p.width)
	if p.remote {
		input = spreadLine(p.search.View(), Loading(p.spinner, searchingRemote)+" ", p.width)
	}
	lines := []string{input, box}
	if p.searchErr != "" {
		lines = append(lines, Line(lipgloss.NewStyle().Foreground(styles.ColorError).Render(p.searchErr), p.width))
	}
	return strings.Join(lines, "\n")
}

func (p PipelinesPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}
	top := p.searchView()
	if top != "" {
		top += "\n"
	}
	if p.Count() > 0 {
		return Block(top+p.viewport.View(), p.width, p.height)
	}
	pad := lipgloss.NewStyle().Padding(1, 1)
	if p.loading {
		label := loadingPipelines
		if p.runMode() {
			label = loadingRuns
		}
		return Block(top+pad.Render(Loading(p.spinner, label)), p.width, p.height)
	}
	title, hint := p.emptyText()
	msg := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title),
		lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(max(p.width-2, 1)).Render(hint))
	return Block(top+pad.Render(msg), p.width, p.height)
}

func (p *PipelinesPanel) emptyText() (string, string) {
	k := p.keys
	clear := "Press " + k.Search.Help().Key + " and " + k.Back.Help().Key + " to clear the search."
	switch {
	case p.search.Value() != "" || p.applied != "":
		if p.runMode() {
			return "No runs match this search", clear
		}
		return "No pipelines match this search", clear
	case p.mode == ListRecent:
		return "No pipeline has run yet", "Press " + k.Panel2.Help().Key + " to see every pipeline."
	case p.runMode():
		return "No runs yet", "Press " + k.RunPipeline.Help().Key + " to run a pipeline."
	}
	return "No pipelines in this project", "Create one in Azure DevOps, then press " + k.Refresh.Help().Key + "."
}

func (p *PipelinesPanel) SetSearchText(text string) {
	p.search.SetValue(text)
	p.search.CursorEnd()
	p.applied = text
	p.rebuild()
}

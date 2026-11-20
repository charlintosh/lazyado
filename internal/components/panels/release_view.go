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

type ReleaseTab int

const (
	ReleasePipelineTab ReleaseTab = iota
	ReleaseLogsTab
	releaseTabs
)

const (
	releaseHeaderHeight = 4
	releaseStripHeight  = 1
	loadingRelease      = "Loading release"
	noAttemptYet        = "Not deployed in this release yet."
)

type CloseReleaseViewMsg struct{}

type ReleaseView struct {
	runLogs
	id      int
	release *models.Release
	stage   int
	chosen  bool
	tab     ReleaseTab
	body    viewport.Model
	me      models.Identity
	keys    keys.KeyMap
	spinner string
	width   int
	height  int
}

func NewReleaseView(k keys.KeyMap) ReleaseView {
	return ReleaseView{runLogs: newRunLogs(k), keys: k, body: widgets.NewViewport(k)}
}

func (v *ReleaseView) Open(id int, tab ReleaseTab) {
	*v = ReleaseView{
		runLogs: newRunLogs(v.keys),
		id:      id,
		tab:     tab,
		me:      v.me,
		keys:    v.keys,
		spinner: v.spinner,
		width:   v.width,
		height:  v.height,
		body:    widgets.NewViewport(v.keys),
	}
	v.lspinner = v.spinner
	v.layout()
}

func (v *ReleaseView) ID() int { return v.id }

func (v *ReleaseView) Release() *models.Release { return v.release }

func (v *ReleaseView) Tab() ReleaseTab { return v.tab }

func (v *ReleaseView) SetIdentity(me models.Identity) { v.me = me }

func (v *ReleaseView) Active() bool { return v.release != nil && v.release.Active() }

func (v *ReleaseView) Loading() bool {
	return v.id != 0 && (v.release == nil || (v.tab == ReleaseLogsTab && v.logLoading()))
}

func (v *ReleaseView) SetSpinner(frame string) {
	if v.spinner == frame {
		return
	}
	v.spinner, v.lspinner = frame, frame
	if v.Loading() {
		v.render()
	}
}

func (v *ReleaseView) SetRelease(r models.Release) {
	if v.id != r.ID {
		return
	}
	v.release = &r
	if !v.chosen {
		v.stage = defaultStage(r)
	}
	v.stage = min(max(v.stage, 0), max(len(r.Stages)-1, 0))
	v.setStages(r.RunStages())
	v.scopeTo(v.stage)
	v.markStale()
	v.render()
}

func defaultStage(r models.Release) int {
	for _, want := range []models.StageState{models.StageWaitingApproval, models.StageDeploying, models.StageFailed, models.StageRejected} {
		for i, s := range r.Stages {
			if s.State == want {
				return i
			}
		}
	}
	last := 0
	for i, s := range r.Stages {
		if s.State != models.StageNotDeployed {
			last = i
		}
	}
	return last
}

func (v *ReleaseView) SelectedStage() (models.ReleaseStage, bool) {
	if v.release == nil || v.stage < 0 || v.stage >= len(v.release.Stages) {
		return models.ReleaseStage{}, false
	}
	return v.release.Stages[v.stage], true
}

func (v *ReleaseView) SetLog(id, logID, from int, lines []string, truncated bool, err error) {
	if v.id != id {
		return
	}
	v.setLog(logID, from, lines, truncated, err)
}

func (v *ReleaseView) WantedLog() (logID, from int, url string, ok bool) {
	if v.release == nil || v.tab != ReleaseLogsTab {
		return 0, 0, "", false
	}
	logID, from, ok = v.wanted()
	if !ok {
		return 0, 0, "", false
	}
	r, found := v.record(logID)
	if !found || r.LogURL == "" {
		return 0, 0, "", false
	}
	return logID, from, r.LogURL, true
}

func (v *ReleaseView) MarkLogRequested(logID int) { v.markRequested(logID) }

func (v *ReleaseView) SetSize(width, height int) {
	if v.width == width && v.height == height {
		return
	}
	v.width, v.height = width, height
	v.layout()
}

func (v *ReleaseView) bodyHeight() int { return max(v.height-releaseHeaderHeight, 0) }

func (v *ReleaseView) sidebarWidth() int {
	if v.width < styles.PullRequestSidebarShow {
		return 0
	}
	return styles.PullRequestSidebarWidth
}

func (v *ReleaseView) mainWidth() int {
	if side := v.sidebarWidth(); side > 0 {
		return max(v.width-side-styles.DetailColumnGap-2, 1)
	}
	return max(v.width-2, 1)
}

func (v *ReleaseView) layout() {
	v.body.SetWidth(v.mainWidth())
	v.body.SetHeight(v.bodyHeight())
	v.resizeLogs(v.width, max(v.bodyHeight()-releaseStripHeight, 0))
	v.render()
}

func (v *ReleaseView) render() {
	if v.width <= 0 {
		return
	}
	v.renderPipeline()
	v.renderLogs()
}

func (v *ReleaseView) renderPipeline() {
	width := v.mainWidth()
	if v.release == nil {
		v.body.SetContent(Loading(v.spinner, loadingRelease))
		return
	}
	boxes, _ := stageBoxes(v.release.Stages, v.stage, width)
	parts := []string{sectionTitle("Stages", width) + "\n" + boxes}
	if s, ok := v.SelectedStage(); ok {
		parts = append(parts, v.stageSection(s, width), v.attemptSection(s, width))
		if issues := models.RunIssues([]models.RunStage{s.RunStage()}); len(issues) > 0 {
			parts = append(parts, issuesSection(issues, width))
		}
	}
	if v.sidebarWidth() == 0 {
		parts = append(parts, v.sideSections(width)...)
	}
	v.body.SetContent(strings.Join(parts, "\n\n"))
}

func (v *ReleaseView) stageSection(s models.ReleaseStage, width int) string {
	title := s.Name + Muted(" · ") + StageStateText(s.State)
	rows := [][2]string{{"Trigger", s.Trigger}}
	if a, ok := s.PendingApproval(); ok {
		you := Muted("not assigned to you")
		if a.AssignedTo(v.me) || a.Group {
			you = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("press " + v.keys.Approve.Help().Key + " to approve or reject")
		}
		rows = append(rows,
			[2]string{"Waiting on", a.KindName() + " approval · " + approverName(a)},
			[2]string{"You", you},
			[2]string{"Requested", releaseStamp(a.Created)},
		)
	} else if len(s.Approvers) > 0 {
		rows = append(rows, [2]string{"Approval", "before deploying: " + strings.Join(s.Approvers, ", ")})
	}
	if len(s.PostApproval) > 0 {
		rows = append(rows, [2]string{"After", "post-deployment approval: " + strings.Join(s.PostApproval, ", ")})
	}
	if s.State.Cancelable() {
		rows = append(rows, [2]string{"Cancel", "press " + v.keys.CancelDeploy.Help().Key + " to cancel this deployment"})
	}
	if s.State.Deployable() {
		verb := "deploy"
		if s.Attempt > 0 {
			verb = "redeploy"
		}
		rows = append(rows, [2]string{"Deploy", "press " + v.keys.DeployStage.Help().Key + " to " + verb + " this stage"})
	}
	return sectionTitle(title, width) + "\n" + FieldRows(keptRows(rows), fieldLabel, width)
}

func (v *ReleaseView) attemptSection(s models.ReleaseStage, width int) string {
	if s.Attempt == 0 {
		return sectionTitle("Deployment", width) + "\n" + Muted(noAttemptYet)
	}
	who := s.Requested.DisplayName
	title := fmt.Sprintf("Deployment · attempt %d", s.Attempt)
	if who != "" {
		title += " · " + who
	}
	lines := []string{sectionTitle(title, width)}
	now := time.Now()
	for _, j := range s.Jobs {
		lines = append(lines, jobRow(j, false, false, width, now, ""))
		for _, step := range j.Steps {
			lines = append(lines, jobRow(models.RunJob{TimelineRecord: step}, false, false, width, now, "  "))
		}
	}
	if len(s.Jobs) == 0 {
		lines = append(lines, Muted("No jobs have started yet."))
	} else {
		lines = append(lines, Muted("Enter opens these logs."))
	}
	return strings.Join(lines, "\n")
}

func (v *ReleaseView) sideSections(width int) []string {
	r := *v.release
	var art []string
	for _, a := range r.Artifacts {
		kind := a.Type
		if a.Primary {
			kind += " · primary"
		}
		art = append(art, Line(lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(a.Alias)+Muted(" "+kind), width))
		parts := []string{a.Version}
		if b := a.BranchName(); b != "" {
			parts = append(parts, b)
		}
		art = append(art, Line(Muted(strings.Join(parts, " · ")), width))
	}
	sections := []string{sectionTitle("Artifacts", width) + "\n" + strings.Join(art, "\n")}
	var triggers [][2]string
	for _, s := range r.Stages {
		triggers = append(triggers, [2]string{s.Name, s.Trigger})
	}
	sections = append(sections, sectionTitle("Stage triggers", width)+"\n"+FieldRows(triggers, fieldLabel, width))
	details := keptRows([][2]string{
		{"Created by", r.CreatedBy.DisplayName},
		{"Created", releaseStamp(r.Created)},
		{"Description", r.Description},
	})
	return append(sections, sectionTitle("Details", width)+"\n"+FieldRows(details, fieldLabel, width))
}

func (v ReleaseView) Update(msg tea.Msg) (ReleaseView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	case tea.MouseWheelMsg:
		if v.tab == ReleaseLogsTab {
			return v, v.scrollLog(msg)
		}
		var cmd tea.Cmd
		v.body, cmd = v.body.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v ReleaseView) handleKey(msg tea.KeyPressMsg) (ReleaseView, tea.Cmd) {
	k := v.keys
	switch {
	case key.Matches(msg, k.Back) && v.tab == ReleaseLogsTab && v.pane > runPaneTree:
		v.pane--
		v.renderLogs()
		return v, nil
	case key.Matches(msg, k.Back):
		return v, func() tea.Msg { return CloseReleaseViewMsg{} }
	case key.Matches(msg, k.Panel1):
		v.setTab(ReleasePipelineTab)
		return v, nil
	case key.Matches(msg, k.Panel2):
		v.setTab(ReleaseLogsTab)
		return v, nil
	case key.Matches(msg, k.NextView), key.Matches(msg, k.PrevView):
		v.setTab((v.tab + 1) % releaseTabs)
		return v, nil
	}
	if v.release == nil {
		return v, nil
	}
	r := *v.release
	switch {
	case key.Matches(msg, k.PrevStage):
		v.moveStage(-1)
		return v, nil
	case key.Matches(msg, k.NextStage):
		v.moveStage(1)
		return v, nil
	case key.Matches(msg, k.CreateRelease):
		return v, func() tea.Msg { return CreateReleaseMsg{DefinitionID: r.DefinitionID, Name: r.Definition} }
	case key.Matches(msg, k.OpenBrowser):
		return v, func() tea.Msg { return BrowseReleaseMsg{ReleaseID: r.ID, URL: r.WebURL} }
	case key.Matches(msg, k.Share), key.Matches(msg, k.ShareLink):
		now, _ := ShareRequested(k, msg)
		return v, shareCmd(share.Release(r), now)
	}
	if v.tab == ReleaseLogsTab {
		cmd := v.handleLogsKey(msg)
		return v, cmd
	}
	return v.handlePipelineKey(msg, r)
}

func (v ReleaseView) handlePipelineKey(msg tea.KeyPressMsg, r models.Release) (ReleaseView, tea.Cmd) {
	k := v.keys
	switch {
	case key.Matches(msg, k.Left):
		v.moveStage(-1)
	case key.Matches(msg, k.Right):
		v.moveStage(1)
	case key.Matches(msg, k.Approve):
		s, _ := v.SelectedStage()
		if a, ok := s.PendingApproval(); ok {
			return v, func() tea.Msg { return ApproveReleaseMsg{Approval: a} }
		}
		return v, func() tea.Msg { return ReleaseNoticeMsg{Text: s.Name + " has no approval waiting."} }
	case key.Matches(msg, k.DeployStage):
		s, _ := v.SelectedStage()
		if !s.State.Deployable() {
			return v, func() tea.Msg {
				return ReleaseNoticeMsg{Text: s.Name + " is " + s.State.Name() + ", so it can't be deployed now."}
			}
		}
		return v, func() tea.Msg { return DeployStageMsg{Release: r, Stage: s} }
	case key.Matches(msg, k.CancelDeploy):
		s, _ := v.SelectedStage()
		if !s.State.Cancelable() {
			return v, func() tea.Msg {
				return ReleaseNoticeMsg{Text: s.Name + " is " + s.State.Name() + ", so there's no deployment to cancel."}
			}
		}
		return v, func() tea.Msg { return CancelDeploymentMsg{Release: r, Stage: s} }
	case key.Matches(msg, k.Select):
		v.openStageLogs()
	default:
		var cmd tea.Cmd
		v.body, cmd = v.body.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *ReleaseView) moveStage(delta int) {
	if v.release == nil {
		return
	}
	next := min(max(v.stage+delta, 0), len(v.release.Stages)-1)
	if next == v.stage {
		return
	}
	v.stage, v.chosen = next, true
	v.scopeTo(next)
	v.renderPipeline()
}

func (v *ReleaseView) openStageLogs() {
	v.scopeTo(v.stage)
	if v.visibleJobs() == 0 {
		return
	}
	v.tab, v.pane = ReleaseLogsTab, runPaneTree
	v.renderLogs()
}

func (v *ReleaseView) setTab(tab ReleaseTab) {
	if v.tab == tab {
		return
	}
	v.tab = tab
	v.render()
}

func (v *ReleaseView) KeyBindings() []key.Binding {
	k := v.keys
	tabs := widgets.Combine("pipeline/logs", k.Panel1, k.Panel2)
	back := widgets.Relabel(k.Back, "back")
	stage := widgets.Combine("stage", k.PrevStage, k.NextStage)
	if v.tab == ReleaseLogsTab {
		return append(v.logsBindings(), stage, tabs, k.Share, k.OpenBrowser, back)
	}
	bindings := []key.Binding{widgets.Combine("stage", k.Left, k.Right)}
	if s, ok := v.SelectedStage(); ok {
		if _, waiting := s.PendingApproval(); waiting {
			bindings = append(bindings, k.Approve)
		}
		if s.State.Deployable() {
			bindings = append(bindings, k.DeployStage)
		}
		if s.State.Cancelable() {
			bindings = append(bindings, k.CancelDeploy)
		}
		if len(s.Jobs) > 0 {
			bindings = append(bindings, widgets.Relabel(k.Select, "logs"))
		}
	}
	return append(bindings, k.CreateRelease, tabs, k.Share, k.OpenBrowser, back)
}

func (v *ReleaseView) Mode() string {
	if v.tab == ReleaseLogsTab {
		return "LOGS"
	}
	return "RELEASE"
}

func (v *ReleaseView) header() string {
	inner := max(v.width-2, 1)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat(ruleGlyph, v.width))
	tabs := Tabs([]string{v.keys.Panel1.Help().Key + " Pipeline", v.keys.Panel2.Help().Key + " Logs"}, int(v.tab), true)
	if v.release == nil {
		return strings.Join([]string{" " + Loading(v.spinner, loadingRelease), "", Line(tabs, v.width), rule}, "\n")
	}
	r := *v.release
	state := r.State()
	badge := StageBadge(state)
	if state == models.StageNotDeployed {
		badge = Badge("NEW ", styles.ColorChip, styles.ColorLight)
	}
	title := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Render(r.Name) +
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(" · "+r.Definition)
	meta := []string{}
	if r.CreatedBy.DisplayName != "" {
		meta = append(meta, "Created by "+r.CreatedBy.DisplayName)
	}
	meta = append(meta, releaseStamp(r.Created), r.ArtifactSummary())
	return strings.Join([]string{" " + Line(badge+" "+title, inner), " " + Line(Muted(strings.Join(meta, " · ")), inner), Line(tabs, v.width), rule}, "\n")
}

func (v ReleaseView) View() string {
	if v.id == 0 || v.width <= 0 || v.height <= 0 {
		return ""
	}
	var body string
	if v.tab == ReleaseLogsTab && v.release != nil {
		strip := stageStrip(v.release.Stages, v.stage, v.width, v.keys.PrevStage.Help().Key+" "+v.keys.NextStage.Help().Key+" switch stage")
		body = Block(strip+"\n"+v.logsBody(), v.width, v.bodyHeight())
	} else {
		body = v.body.View()
		if side := v.sidebarWidth(); side > 0 && v.release != nil {
			sidebar := Block(strings.Join(v.sideSections(side), "\n\n"), side, v.bodyHeight())
			body = lipgloss.JoinHorizontal(lipgloss.Top, Block(body, v.mainWidth(), v.bodyHeight()), strings.Repeat(" ", styles.DetailColumnGap), sidebar)
		}
		body = lipgloss.NewStyle().PaddingLeft(1).Render(Block(body, max(v.width-2, 1), v.bodyHeight()))
	}
	return Block(v.header()+"\n"+body, v.width, v.height)
}

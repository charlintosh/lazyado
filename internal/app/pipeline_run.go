package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

type runLoadedMsg struct {
	run models.PipelineRun
}

type runTimelineMsg struct {
	id      int
	records []models.TimelineRecord
	err     error
}

type runChangesMsg struct {
	id      int
	changes []models.RunChange
}

type runLogMsg struct {
	id        int
	logID     int
	from      int
	lines     []string
	truncated bool
	err       error
}

type runPollMsg struct {
	id  int
	seq int
}

type runStartedMsg struct {
	run models.PipelineRun
}

func loadRunCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		run, err := client.GetPipelineRun(id)
		if err != nil {
			return nil
		}
		return runLoadedMsg{run: run}
	}
}

func loadRunTimelineCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		records, err := client.GetRunTimeline(id)
		return runTimelineMsg{id: id, records: records, err: err}
	}
}

func loadRunChangesCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		changes, err := client.GetRunChanges(id)
		if err != nil {
			changes = nil
		}
		return runChangesMsg{id: id, changes: changes}
	}
}

func loadRunLogCmd(client *api.Client, id, logID, from int) tea.Cmd {
	return func() tea.Msg {
		lines, truncated, err := client.GetRunLog(id, logID, from)
		return runLogMsg{id: id, logID: logID, from: from, lines: lines, truncated: truncated, err: err}
	}
}

func scheduleRunPollCmd(id, seq int) tea.Cmd {
	return tea.Tick(runPollInterval, func(time.Time) tea.Msg { return runPollMsg{id: id, seq: seq} })
}

func loadRunDefinitionCmd(client *api.Client, pipelineID int) tea.Cmd {
	return func() tea.Msg {
		msg := modals.RunDefinitionMsg{PipelineID: pipelineID}
		msg.Definition, msg.Err = client.GetPipelineDefinition(pipelineID)
		if msg.Err == nil && msg.Definition.AzureRepo() {
			msg.Refs, msg.Err = client.ListRefs(msg.Definition.RepositoryID)
		}
		return msg
	}
}

func loadRunSpecCmd(client *api.Client, req modals.RunSpecRequestMsg) tea.Cmd {
	return func() tea.Msg {
		msg := modals.RunSpecMsg{PipelineID: req.PipelineID, Ref: req.Ref}
		text, err := client.GetFileAtRef(req.RepositoryID, req.Path, req.Ref)
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Spec, msg.Err = models.ParsePipelineSpec(text)
		return msg
	}
}

func startRunCmd(client *api.Client, req modals.RunPipelineRequestMsg) tea.Cmd {
	return func() tea.Msg {
		id, err := client.RunPipeline(req.Definition, req.Options)
		if err != nil {
			return errMsg{err: err}
		}
		run, err := client.GetPipelineRun(id)
		if err != nil {
			run = models.PipelineRun{ID: id, PipelineID: req.Definition.ID, PipelineName: req.Definition.Name, Status: "notStarted", Ref: req.Options.Ref, WebURL: client.RunWebURL(id)}
		}
		return runStartedMsg{run: run}
	}
}

func (a *App) viewingRun() bool {
	return a.viewMode == ViewRun
}

func (a *App) openPullRequestBuilds(pr models.PullRequest, builds []models.PullRequestCheck) tea.Cmd {
	switch len(builds) {
	case 0:
		return a.info("No policy or status check on this pull request points to a build.")
	case 1:
		return a.openBuildByID(builds[0].BuildID, builds[0].Name)
	}
	return a.openModal(modals.NewPullRequestBuildsModal(a.env(), pr, builds))
}

func (a *App) openBuildByID(id int, name string) tea.Cmd {
	fromReview := a.reviewing()
	cmd := a.openRun(models.PipelineRun{ID: id, PipelineName: name, WebURL: a.client.RunWebURL(id)}, panels.RunSummaryTab)
	a.pl.fromReview = fromReview
	return cmd
}

func (a *App) openRun(run models.PipelineRun, tab panels.RunTab) tea.Cmd {
	a.pl.fromReview = false
	a.viewMode = ViewRun
	a.pl.view.Open(run, tab)
	a.pl.runSeq++
	return tea.Batch(loadRunCmd(a.client, run.ID), loadRunTimelineCmd(a.client, run.ID), loadRunChangesCmd(a.client, run.ID))
}

func (a *App) refreshRun() tea.Cmd {
	run := a.pl.view.Run()
	if run == nil {
		return nil
	}
	a.pl.view.MarkStale()
	return tea.Batch(loadRunCmd(a.client, run.ID), loadRunTimelineCmd(a.client, run.ID))
}

func (a *App) syncRunLog() tea.Cmd {
	if !a.viewingRun() {
		return nil
	}
	run := a.pl.view.Run()
	logID, from, ok := a.pl.view.WantedLog()
	if run == nil || !ok {
		return nil
	}
	a.pl.view.MarkLogRequested(logID)
	return loadRunLogCmd(a.client, run.ID, logID, from)
}

func (a *App) scheduleRunPoll() tea.Cmd {
	run := a.pl.view.Run()
	if run == nil || !a.viewingRun() {
		return nil
	}
	a.pl.runSeq++
	return scheduleRunPollCmd(run.ID, a.pl.runSeq)
}

func (a App) handleRunResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case panels.CloseRunViewMsg:
		a.viewMode = ViewMain
		if a.pl.fromReview {
			a.viewMode = ViewPullRequest
		}
		a.pl.fromReview = false
		a.pl.runSeq++
		return a, nil, true
	case panels.PullRequestBuildsMsg:
		return a, a.openPullRequestBuilds(msg.PullRequest, msg.Builds), true
	case modals.BuildChosenMsg:
		a.closeModal()
		return a, a.openBuildByID(msg.BuildID, msg.Name), true
	case panels.RunAgainMsg:
		r := msg.Run
		return a, a.openRunForm(models.Pipeline{ID: r.PipelineID, Name: r.PipelineName, Folder: r.PipelineFolder}, r.Ref), true
	case runLoadedMsg:
		wasActive := a.pl.view.Active()
		a.pl.view.Refresh(msg.run)
		if current := a.pl.view.Run(); current == nil || current.ID != msg.run.ID {
			return a, nil, true
		}
		if wasActive && !msg.run.State().Active() {
			a.pl.view.MarkStale()
			return a, loadRunTimelineCmd(a.client, msg.run.ID), true
		}
		return a, nil, true
	case runTimelineMsg:
		current := a.pl.view.Run()
		if current == nil || current.ID != msg.id {
			return a, nil, true
		}
		if msg.err != nil {
			a.pl.view.SetTimeline(msg.id, nil)
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.pl.view.SetTimeline(msg.id, msg.records)
		if a.pl.view.Active() {
			return a, a.scheduleRunPoll(), true
		}
		return a, nil, true
	case runChangesMsg:
		a.pl.view.SetChanges(msg.id, msg.changes)
		return a, nil, true
	case runLogMsg:
		a.pl.view.SetLog(msg.id, msg.logID, msg.from, msg.lines, msg.truncated, msg.err)
		return a, nil, true
	case runPollMsg:
		current := a.pl.view.Run()
		if msg.seq != a.pl.runSeq || current == nil || current.ID != msg.id || !a.viewingRun() {
			return a, nil, true
		}
		return a, a.refreshRun(), true
	case modals.RunDefinitionRequestMsg:
		return a, loadRunDefinitionCmd(a.client, msg.PipelineID), true
	case modals.RunSpecRequestMsg:
		return a, loadRunSpecCmd(a.client, msg), true
	case modals.RunDefinitionMsg, modals.RunSpecMsg:
		if !topIs[modals.RunPipelineForm](a.modals) {
			return a, nil, true
		}
		next, cmd := a.updateModal(msg)
		return next, cmd, true
	case modals.RunPipelineRequestMsg:
		return a, a.mutate(startRunCmd(a.client, msg)), true
	case runStartedMsg:
		a.loading = false
		if topIs[modals.RunPipelineForm](a.modals) {
			a.closeModal()
		}
		notice := a.notify("Run #" + msg.run.Number + " queued for " + msg.run.PipelineName)
		if msg.run.Number == "" {
			notice = a.notify("Run queued for " + msg.run.PipelineName)
		}
		return a, tea.Batch(notice, a.openRun(msg.run, panels.RunLogsTab)), true
	}
	return a, nil, false
}

func (a *App) renderRun() string {
	l := a.shell
	crumbs := []string{panels.HubBuilds.Section(), panels.HubBuilds.Name()}
	if run := a.pl.view.Run(); run != nil {
		crumbs = append(crumbs, run.PipelineName, "#"+run.Number)
	}
	body := panels.Block(a.pl.view.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader(crumbs...), body, a.renderStatusBar())
}

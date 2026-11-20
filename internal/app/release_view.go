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

type releaseLoadedMsg struct {
	release models.Release
	err     error
}

type releaseLogMsg struct {
	release   int
	logID     int
	from      int
	lines     []string
	truncated bool
	err       error
}

type releaseViewPollMsg struct {
	id  int
	seq int
}

type releaseChangedMsg struct {
	release int
	notice  string
}

type releaseCreatedMsg struct {
	release models.Release
}

func loadReleaseCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		r, err := client.GetRelease(id)
		return releaseLoadedMsg{release: r, err: err}
	}
}

func loadReleaseLogCmd(client *api.Client, release, logID, from int, url string) tea.Cmd {
	return func() tea.Msg {
		lines, truncated, err := client.GetReleaseLog(url, from)
		return releaseLogMsg{release: release, logID: logID, from: from, lines: lines, truncated: truncated, err: err}
	}
}

func scheduleReleaseViewPollCmd(id, seq int) tea.Cmd {
	return tea.Tick(runPollInterval, func(time.Time) tea.Msg { return releaseViewPollMsg{id: id, seq: seq} })
}

func releaseActionCmd(release int, notice string, run func() error) tea.Cmd {
	return func() tea.Msg {
		if err := run(); err != nil {
			return errMsg{err: err}
		}
		return releaseChangedMsg{release: release, notice: notice}
	}
}

func loadReleaseDefinitionSpecCmd(client *api.Client, id int) tea.Cmd {
	return func() tea.Msg {
		msg := modals.ReleaseDefinitionMsg{DefinitionID: id, Builds: map[string][]models.PipelineRun{}, Refs: map[string][]models.GitRef{}}
		msg.Spec, msg.Err = client.GetReleaseDefinition(id)
		if msg.Err != nil {
			return msg
		}
		for _, art := range msg.Spec.Artifacts {
			switch art.Type {
			case models.ArtifactBuild:
				msg.Builds[art.Alias], _ = client.ListArtifactBuilds(art.DefinitionID, "")
			case models.ArtifactGit:
				msg.Refs[art.Alias], _ = client.ListRefs(art.RepositoryID)
			}
		}
		return msg
	}
}

func loadArtifactVersionsCmd(client *api.Client, req modals.ArtifactVersionsRequestMsg) tea.Cmd {
	return func() tea.Msg {
		builds, err := client.ListArtifactBuilds(req.BuildDefinitionID, req.Ref)
		return modals.ArtifactVersionsMsg{DefinitionID: req.DefinitionID, Alias: req.Alias, Ref: req.Ref, Builds: builds, Err: err}
	}
}

func createReleaseCmd(client *api.Client, rel models.NewRelease) tea.Cmd {
	return func() tea.Msg {
		created, err := client.CreateRelease(rel)
		if err != nil {
			return errMsg{err: err}
		}
		return releaseCreatedMsg{release: created}
	}
}

func (a *App) viewingRelease() bool { return a.viewMode == ViewRelease }

func (a *App) openRelease(id int, tab panels.ReleaseTab) tea.Cmd {
	a.viewMode = ViewRelease
	a.rl.view.Open(id, tab)
	a.rl.viewSeq++
	return loadReleaseCmd(a.client, id)
}

func (a *App) refreshRelease() tea.Cmd {
	id := a.rl.view.ID()
	if id == 0 {
		return nil
	}
	return loadReleaseCmd(a.client, id)
}

func (a *App) syncReleaseLog() tea.Cmd {
	if !a.viewingRelease() {
		return nil
	}
	logID, from, url, ok := a.rl.view.WantedLog()
	if !ok {
		return nil
	}
	a.rl.view.MarkLogRequested(logID)
	return loadReleaseLogCmd(a.client, a.rl.view.ID(), logID, from, url)
}

func (a *App) releaseFor(id int) *models.Release {
	if r := a.rl.view.Release(); r != nil && r.ID == id {
		return r
	}
	if a.rl.preview != nil && a.rl.preview.ID == id {
		return a.rl.preview
	}
	for i := range a.rl.releases {
		if a.rl.releases[i].ID == id {
			return &a.rl.releases[i]
		}
	}
	return nil
}

func (a *App) afterReleaseChange(id int) tea.Cmd {
	var cmds []tea.Cmd
	if a.rl.view.ID() == id {
		cmds = append(cmds, loadReleaseCmd(a.client, id))
	}
	if a.rl.listFor != 0 {
		a.rl.listSeq++
		cmds = append(cmds, loadReleasesCmd(a.client, a.rl.listSeq, a.rl.listFor, "", false, true))
	}
	if a.rl.approvalsLoaded {
		cmds = append(cmds, a.loadApprovals())
	}
	return tea.Batch(cmds...)
}

func (a App) handleReleaseViewResult(msg tea.Msg) (App, tea.Cmd, bool) {
	c := a.client
	switch msg := msg.(type) {
	case panels.OpenReleaseMsg:
		return a, a.openRelease(msg.ReleaseID, panels.ReleasePipelineTab), true
	case panels.CloseReleaseViewMsg:
		a.viewMode = ViewMain
		a.rl.viewSeq++
		return a, nil, true
	case releaseLoadedMsg:
		if msg.err != nil {
			if a.viewingRelease() && a.rl.view.Release() == nil {
				a.viewMode = ViewMain
			}
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.rl.view.SetRelease(msg.release)
		if a.rl.view.ID() != msg.release.ID || !msg.release.Active() {
			return a, nil, true
		}
		a.rl.viewSeq++
		return a, scheduleReleaseViewPollCmd(msg.release.ID, a.rl.viewSeq), true
	case releaseViewPollMsg:
		if msg.seq != a.rl.viewSeq || msg.id != a.rl.view.ID() || !a.viewingRelease() {
			return a, nil, true
		}
		return a, loadReleaseCmd(c, msg.id), true
	case releaseLogMsg:
		a.rl.view.SetLog(msg.release, msg.logID, msg.from, msg.lines, msg.truncated, msg.err)
		return a, nil, true
	case panels.ApproveReleaseMsg:
		return a, a.openModal(modals.NewApprovalModal(a.env(), msg.Approval, a.releaseFor(msg.Approval.ReleaseID))), true
	case panels.DeployStageMsg:
		return a, a.openModal(modals.NewDeployStageModal(a.env(), msg.Release, msg.Stage)), true
	case panels.CancelDeploymentMsg:
		return a, a.openModal(modals.NewCancelDeploymentModal(a.env(), msg.Release, msg.Stage)), true
	case modals.CancelDeploymentRequestMsg:
		a.closeModal()
		req := msg
		return a, a.mutate(releaseActionCmd(req.ReleaseID, "Canceled the "+req.Stage+" deployment", func() error {
			return c.CancelReleaseStage(req.ReleaseID, req.StageID, req.Comment)
		})), true
	case panels.CreateReleaseMsg:
		return a, a.openCreateRelease(msg.DefinitionID, msg.Name), true
	case modals.ReleaseApprovalRequestMsg:
		a.closeModal()
		ap, approve, comment := msg.Approval, msg.Approve, msg.Comment
		notice := ap.StageName + " approved for " + ap.ReleaseName
		if !approve {
			notice = ap.StageName + " rejected for " + ap.ReleaseName
		}
		return a, a.mutate(releaseActionCmd(ap.ReleaseID, notice, func() error { return c.DecideApproval(ap.ID, approve, comment) })), true
	case modals.DeployStageRequestMsg:
		a.closeModal()
		req := msg
		return a, a.mutate(releaseActionCmd(req.ReleaseID, "Deploying "+req.Stage, func() error {
			return c.DeployReleaseStage(req.ReleaseID, req.StageID, req.Comment)
		})), true
	case releaseChangedMsg:
		a.loading = false
		return a, tea.Batch(a.notify(msg.notice), a.afterReleaseChange(msg.release)), true
	case modals.ReleaseDefinitionRequestMsg:
		return a, loadReleaseDefinitionSpecCmd(c, msg.DefinitionID), true
	case modals.ArtifactVersionsRequestMsg:
		return a, loadArtifactVersionsCmd(c, msg), true
	case modals.ReleaseDefinitionMsg, modals.ArtifactVersionsMsg:
		if !topIs[modals.CreateReleaseForm](a.modals) {
			return a, nil, true
		}
		next, cmd := a.updateModal(msg)
		return next, cmd, true
	case modals.CreateReleaseRequestMsg:
		return a, a.mutate(createReleaseCmd(c, msg.Release)), true
	case releaseCreatedMsg:
		a.loading = false
		if topIs[modals.CreateReleaseForm](a.modals) {
			a.closeModal()
		}
		r := msg.release
		cmds := []tea.Cmd{a.notify(r.Name + " created"), a.openRelease(r.ID, panels.ReleasePipelineTab)}
		if r.DefinitionID == a.rl.listFor {
			a.rl.listSeq++
			cmds = append(cmds, loadReleasesCmd(c, a.rl.listSeq, a.rl.listFor, "", false, true))
		}
		return a, tea.Batch(cmds...), true
	}
	return a.handleReleaseEditorResult(msg)
}

func (a *App) renderRelease() string {
	l := a.shell
	crumbs := []string{panels.HubReleases.Section(), panels.HubReleases.Name()}
	if r := a.rl.view.Release(); r != nil {
		crumbs = append(crumbs, r.Definition, r.Name)
	}
	body := panels.Block(a.rl.view.View(), a.width, l.bodyHeight)
	return lipgloss.JoinVertical(lipgloss.Left, a.renderHeader(crumbs...), body, a.renderStatusBar())
}

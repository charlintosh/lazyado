package ui

import (
	"fmt"
	"net/url"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

var branchPivots = []string{"Mine", "All", "Stale"}

type branchesState struct {
	panel     panels.BranchesPanel
	seq       int
	loading   bool
	loadedFor string
	branches  []models.Branch
	selectRef string
}

type branchesLoadedMsg struct {
	seq      int
	branches []models.Branch
	err      error
}

type refChangedMsg struct {
	notice string
	name   string
	tag    bool
	err    error
}

func loadBranchDetailsCmd(client *api.Client, seq int, repo models.Repository) tea.Cmd {
	return func() tea.Msg {
		branches, err := client.ListBranchDetails(repo.ID, repo.DefaultBranch)
		if err != nil {
			return branchesLoadedMsg{seq: seq, err: err}
		}
		if prs, err := client.ListPullRequests(models.PullRequestActive, repo.ID, models.PullRequestCriteria{}); err == nil {
			bySource := make(map[string]int, len(prs))
			for _, pr := range prs {
				bySource[pr.SourceBranch] = pr.ID
			}
			for i := range branches {
				branches[i].PullRequestID = bySource[branches[i].Name]
			}
		}
		return branchesLoadedMsg{seq: seq, branches: branches}
	}
}

func createBranchCmd(client *api.Client, repositoryID string, req modals.CreateBranchRequestMsg) tea.Cmd {
	return func() tea.Msg {
		commit, err := client.ResolveCommit(repositoryID, req.Base)
		if err == nil {
			err = client.CreateBranch(repositoryID, req.Name, commit)
		}
		return refChangedMsg{notice: "Branch " + req.Name + " created", name: req.Name, err: err}
	}
}

func deleteBranchCmd(client *api.Client, repositoryID string, b models.Branch) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteBranch(repositoryID, b.Name, b.ObjectID)
		return refChangedMsg{notice: "Branch " + b.Name + " deleted", err: err}
	}
}

func (a *App) inBranches() bool {
	return a.hub == panels.HubBranches
}

func (a *App) syncBranches() tea.Cmd {
	if !a.inBranches() || a.viewMode != ViewMain || a.code.repo.ID == "" || a.branches.loadedFor == a.code.repo.ID {
		return nil
	}
	s := &a.branches
	s.loadedFor = a.code.repo.ID
	s.seq++
	s.loading = true
	s.panel.ShowLoading()
	return loadBranchDetailsCmd(a.client, s.seq, a.code.repo)
}

func (a *App) refreshBranches() tea.Cmd {
	if a.code.repo.ID == "" {
		return a.restoreCodeRepository()
	}
	a.branches.loadedFor = ""
	return nil
}

func (a *App) branchNames() []string {
	names := make([]string, len(a.branches.branches))
	for i, b := range a.branches.branches {
		names[i] = b.Name
	}
	return names
}

func (a *App) openNewBranch() tea.Cmd {
	if a.branches.loading {
		return a.info("Branches are still loading. Try again in a moment.")
	}
	base := models.BranchVersion(a.code.repo.DefaultBranch)
	if b := a.branches.panel.Selected(); b != nil {
		base = models.BranchVersion(b.Name)
	}
	return a.openModal(modals.NewBranchForm(a.env(), a.code.repo.Name, a.branchNames(), a.code.tags, a.code.repo.DefaultBranch, base))
}

func (a *App) openDeleteBranch(b models.Branch) tea.Cmd {
	switch {
	case b.Default:
		return a.info(b.Name + " is the default branch and can't be deleted.")
	case b.Locked:
		return a.info(b.Name + " is locked. Unlock it in Azure DevOps to delete it.")
	}
	return a.openModal(modals.NewDeleteBranchModal(a.env(), b, a.code.repo.DefaultBranch))
}

func (a *App) branchWebURL(name string) string {
	q := url.Values{}
	q.Set("version", webVersionPrefix[models.VersionBranch]+name)
	return a.code.repo.WebURL + "?" + q.Encode()
}

func (a *App) selectBranchPivot(pivot int) {
	a.branches.panel.SetView((pivot + len(branchPivots)) % len(branchPivots))
}

func (a App) handleBranchesKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	s := &a.branches
	if s.panel.Searching() {
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	for i, b := range a.filterKeys()[:len(branchPivots)] {
		if key.Matches(msg, b) {
			a.selectBranchPivot(i)
			return a, nil
		}
	}
	if hub, ok := a.hubFor(msg); ok && hub != panels.HubCommits {
		return a, a.goToHub(hub)
	}
	sel := s.panel.Selected()
	switch {
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Repository):
		return a, a.openCodeRepositoryPicker()
	case a.code.repo.ID == "":
	case key.Matches(msg, k.PrevView), key.Matches(msg, k.PrevPanel), key.Matches(msg, k.Left):
		a.selectBranchPivot(s.panel.Pivot() - 1)
	case key.Matches(msg, k.NextView), key.Matches(msg, k.NextPanel), key.Matches(msg, k.Right):
		a.selectBranchPivot(s.panel.Pivot() + 1)
	case key.Matches(msg, k.Search):
		return a, s.panel.FocusSearch()
	case key.Matches(msg, k.Back):
		s.panel.ClearFilter()
	case key.Matches(msg, k.NewRef):
		return a, a.openNewBranch()
	case sel == nil:
		if key.Matches(msg, k.HubCommits) {
			return a, a.goToHub(panels.HubCommits)
		}
	case key.Matches(msg, k.DeleteRef):
		return a, a.openDeleteBranch(*sel)
	case key.Matches(msg, k.Select):
		return a, a.browseFilesAt(models.BranchVersion(sel.Name))
	case key.Matches(msg, k.HubCommits):
		a.setCodeVersion(models.BranchVersion(sel.Name))
		return a, a.openCommitsFor(models.CommitCriteria{})
	case key.Matches(msg, k.PullRequestFrom) && !sel.Default:
		return a, a.openPullRequestFormFrom(a.code.repo, sel.Name)
	case key.Matches(msg, k.Copy):
		return a, copyCmd(sel.Name, "Branch name copied to clipboard")
	case key.Matches(msg, k.OpenBrowser):
		return a, openBrowserCmd(a.branchWebURL(sel.Name))
	default:
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) handleBranchesResult(msg tea.Msg) (App, tea.Cmd, bool) {
	s := &a.branches
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		if msg.seq != s.seq {
			return a, nil, true
		}
		s.loading = false
		if msg.err != nil {
			s.panel.SetBranches(nil, time.Now())
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		s.branches = msg.branches
		s.panel.SetBranches(msg.branches, time.Now())
		if s.selectRef != "" {
			s.panel.SelectName(s.selectRef)
			s.selectRef = ""
		}
		return a, nil, true
	case modals.CreateBranchRequestMsg:
		return a, a.mutate(createBranchCmd(a.client, a.code.repo.ID, msg)), true
	case modals.DeleteBranchRequestMsg:
		a.closeModal()
		return a, a.mutate(deleteBranchCmd(a.client, a.code.repo.ID, msg.Branch)), true
	case refChangedMsg:
		a.loading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		if topIs[modals.RefForm](a.modals) {
			a.closeModal()
		}
		if msg.tag {
			a.tags.loadedFor, a.tags.selectRef = "", msg.name
		} else {
			s.loadedFor, s.selectRef = "", msg.name
		}
		a.code.refsSeq++
		a.code.refsLoading = true
		return a, tea.Batch(a.notify(msg.notice), loadRefsCmd(a.client, a.code.refsSeq, a.code.repo.ID)), true
	}
	return a, nil, false
}

func (a *App) renderBranchesBody(width, height int) string {
	if a.code.repo.ID == "" {
		return a.renderNoRepository("Branches", width, height)
	}
	title := a.branches.panel.Title(a.code.repo.DefaultBranch)
	return panels.Pane(panels.PaneTitle("", title, true), a.branches.panel.View(), width, height, true)
}

func (a *App) syncBranchesLayout(width, height int) {
	cw, ch := panels.PaneContentSize(width, height)
	a.branches.panel.SetSize(cw, ch)
}

func (a *App) branchPivotLabels() []string {
	counts := a.branches.panel.Counts()
	labels := make([]string, len(branchPivots))
	for i, name := range branchPivots {
		labels[i] = a.filterKeys()[i].Help().Key + " " + name
		if !a.branches.panel.Loading() {
			labels[i] += fmt.Sprintf(" %d", counts[i])
		}
	}
	return labels
}

func (a *App) branchesBindings() []key.Binding {
	if a.code.repo.ID == "" {
		return []key.Binding{a.keys.Repository}
	}
	return a.branches.panel.KeyBindings()
}

func (a *App) searchingBranches() bool {
	return a.viewMode == ViewMain && a.inBranches() && !a.modals.open() && a.branches.panel.Searching()
}

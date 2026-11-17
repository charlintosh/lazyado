package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

type codeState struct {
	repo      models.Repository
	savedRepo string
	version   models.GitVersion
	seq       int

	branches    []string
	tags        []string
	refsSeq     int
	refsLoaded  bool
	refsLoading bool

	files filesState
}

type refsLoadedMsg struct {
	seq      int
	branches []string
	tags     []string
	err      error
}

func loadRefsCmd(client *api.Client, seq int, repositoryID string) tea.Cmd {
	return func() tea.Msg {
		msg := refsLoadedMsg{seq: seq}
		msg.branches, msg.err = client.ListBranches(repositoryID)
		if msg.err == nil {
			msg.tags, msg.err = client.ListTagNames(repositoryID)
		}
		return msg
	}
}

func (s *codeState) savedName() string {
	if s.repo.ID != "" {
		return s.repo.Name
	}
	return s.savedRepo
}

func (a *App) inCode() bool {
	switch a.hub {
	case panels.HubFiles, panels.HubCommits, panels.HubBranches, panels.HubTags:
		return true
	}
	return false
}

func (a *App) openCodeHub(hub panels.Hub) tea.Cmd {
	a.switchHub(hub)
	if !a.pr.reposLoaded {
		return a.ensureRepositories()
	}
	return a.restoreCodeRepository()
}

func (a *App) ensureRepositories() tea.Cmd {
	if a.pr.reposLoaded || a.pr.reposLoading {
		return nil
	}
	a.pr.reposLoading = true
	return loadRepositoriesCmd(a.client)
}

func (a *App) restoreCodeRepository() tea.Cmd {
	if a.code.repo.ID != "" || !a.pr.reposLoaded {
		return nil
	}
	pick := func(match func(models.Repository) bool) (models.Repository, bool) {
		for _, r := range a.pr.repos {
			if match(r) {
				return r, true
			}
		}
		return models.Repository{}, false
	}
	candidates := []func(models.Repository) bool{
		func(r models.Repository) bool { return a.code.savedRepo != "" && r.Name == a.code.savedRepo },
		func(r models.Repository) bool { return a.pr.repo.ID != "" && r.ID == a.pr.repo.ID },
		func(models.Repository) bool { return len(a.pr.repos) == 1 },
	}
	for _, match := range candidates {
		if r, ok := pick(match); ok {
			return a.setCodeRepository(r)
		}
	}
	if a.inCode() && !a.modals.open() {
		return a.openCodeRepositoryPicker()
	}
	return nil
}

func (a *App) openCodeRepositoryPicker() tea.Cmd {
	if !a.pr.reposLoaded {
		return a.info("Repositories are still loading. Try again in a moment.")
	}
	local := make(map[string]bool, len(a.pr.favorites))
	for _, id := range a.pr.favorites {
		local[id] = true
	}
	favorites := modals.RepositoryFavorites{Starred: a.pr.starred, Local: local}
	return a.openModal(modals.NewSingleRepositoryPickerModal(a.env(), a.pr.repos, favorites, a.code.repo.ID))
}

func (a *App) chooseCodeRepository(chosen models.Repository) tea.Cmd {
	for _, r := range a.pr.repos {
		if r.ID == chosen.ID {
			chosen = r
		}
	}
	return tea.Batch(a.setCodeRepository(chosen), saveFiltersCmd(a.savedState()))
}

func (a *App) setCodeRepository(repo models.Repository) tea.Cmd {
	a.code.repo = repo
	a.code.branches, a.code.tags, a.code.refsLoaded = nil, nil, false
	a.code.refsSeq++
	a.code.refsLoading = true
	a.setCodeVersion(models.BranchVersion(repo.DefaultBranch))
	return loadRefsCmd(a.client, a.code.refsSeq, repo.ID)
}

func (a *App) setCodeVersion(v models.GitVersion) {
	a.code.version = v
	a.code.seq++
	a.resetFiles()
}

func (a *App) openVersionPicker() tea.Cmd {
	if !a.code.refsLoaded {
		return a.info("Branches and tags are still loading. Try again in a moment.")
	}
	return a.openModal(modals.NewVersionPickerModal(a.env(), a.code.branches, a.code.tags, a.code.repo.DefaultBranch, a.code.version))
}

func (a *App) openClone() tea.Cmd {
	if a.code.repo.RemoteURL == "" && a.code.repo.SSHURL == "" {
		return a.info("Choose a repository first.")
	}
	return a.openModal(modals.NewCloneModal(a.env(), a.code.repo))
}

func (a App) handleCodeResult(msg tea.Msg) (App, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case refsLoadedMsg:
		if msg.seq != a.code.refsSeq {
			return a, nil, true
		}
		a.code.refsLoading = false
		if msg.err != nil {
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		a.code.branches, a.code.tags, a.code.refsLoaded = msg.branches, msg.tags, true
		return a, nil, true
	case modals.VersionChosenMsg:
		a.closeModal()
		a.setCodeVersion(msg.Version)
		return a, nil, true
	}
	if next, cmd, ok := a.handleFilesResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handleBranchesResult(msg); ok {
		return next, cmd, true
	}
	if next, cmd, ok := a.handleTagsResult(msg); ok {
		return next, cmd, true
	}
	return a.handleCommitsResult(msg)
}

func (a *App) codeControls(withVersion bool) []panels.HeaderControl {
	_, open := a.repositoryPicker()
	name := "choose"
	if a.code.repo.ID != "" {
		name = a.code.repo.Name
	}
	repo := panels.HeaderControl{
		Prefix: "Repo: ", Label: name + caret, Short: name + caret,
		Key: a.keys.Repository, Set: a.code.repo.ID != "", Open: open, Group: repositoryGroup,
	}
	if a.code.repo.ID == "" || !withVersion {
		return []panels.HeaderControl{repo}
	}
	prefix := map[models.VersionKind]string{models.VersionBranch: "Branch: ", models.VersionTag: "Tag: ", models.VersionCommit: "Commit: "}[a.code.version.Kind]
	label := a.code.version.Label() + caret
	version := panels.HeaderControl{Prefix: prefix, Label: label, Short: label, Key: a.keys.Version, Group: -1}
	return []panels.HeaderControl{repo, version}
}

func (a *App) renderNoRepository(title string, width, height int) string {
	cw, ch := panels.PaneContentSize(width, height)
	hint := panels.Muted("Choose a repository with " + a.keys.Repository.Help().Key + ".")
	if a.pr.reposLoading {
		hint = panels.Loading(a.spinner.View(), "Loading repositories")
	}
	return panels.Pane(panels.PaneTitle("", title, true), panels.Block(hint, cw, ch), width, height, true)
}

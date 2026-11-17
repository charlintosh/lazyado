package ui

import (
	"net/url"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
)

type tagsState struct {
	panel     panels.TagsPanel
	seq       int
	loading   bool
	loadedFor string
	tags      []models.Tag
	selectRef string
	detailFor string
	details   map[string]models.TagDetail
	failed    map[string]error
}

type tagsLoadedMsg struct {
	seq  int
	tags []models.Tag
	err  error
}

type tagDetailDueMsg struct {
	objectID string
}

type tagDetailMsg struct {
	objectID string
	detail   models.TagDetail
	err      error
}

func loadTagsCmd(client *api.Client, seq int, repositoryID string) tea.Cmd {
	return func() tea.Msg {
		tags, err := client.ListTags(repositoryID)
		return tagsLoadedMsg{seq: seq, tags: tags, err: err}
	}
}

func scheduleTagDetailCmd(objectID string) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg { return tagDetailDueMsg{objectID: objectID} })
}

func loadTagDetailCmd(client *api.Client, repositoryID, objectID string) tea.Cmd {
	return func() tea.Msg {
		detail, err := client.GetAnnotatedTag(repositoryID, objectID)
		return tagDetailMsg{objectID: objectID, detail: detail, err: err}
	}
}

func createTagCmd(client *api.Client, repositoryID string, req modals.CreateTagRequestMsg) tea.Cmd {
	return func() tea.Msg {
		commit, err := client.ResolveCommit(repositoryID, req.Base)
		if err == nil {
			_, err = client.CreateTag(repositoryID, req.Name, commit, req.Message)
		}
		return refChangedMsg{notice: "Tag " + req.Name + " created", name: req.Name, tag: true, err: err}
	}
}

func (a *App) inTags() bool {
	return a.hub == panels.HubTags
}

func (a *App) syncTags() tea.Cmd {
	if !a.inTags() || a.viewMode != ViewMain || a.code.repo.ID == "" {
		return nil
	}
	s := &a.tags
	if s.loadedFor != a.code.repo.ID {
		s.loadedFor = a.code.repo.ID
		s.seq++
		s.loading = true
		s.details, s.failed = nil, nil
		s.panel.ShowLoading()
		return loadTagsCmd(a.client, s.seq, a.code.repo.ID)
	}
	id := ""
	if t := s.panel.Selected(); t != nil && t.Annotated() {
		id = t.ObjectID
	}
	if id == s.detailFor {
		return nil
	}
	s.detailFor = id
	if _, ok := s.details[id]; ok || s.failed[id] != nil || id == "" || a.shell.mainWidth < commitPreviewShow {
		return nil
	}
	return scheduleTagDetailCmd(id)
}

func (a *App) refreshTags() tea.Cmd {
	if a.code.repo.ID == "" {
		return a.restoreCodeRepository()
	}
	a.tags.loadedFor = ""
	return nil
}

func (a *App) tagNames() []string {
	names := make([]string, len(a.tags.tags))
	for i, t := range a.tags.tags {
		names[i] = t.Name
	}
	return names
}

func (a *App) openNewTag() tea.Cmd {
	if a.tags.loading || !a.code.refsLoaded {
		return a.info("Tags are still loading. Try again in a moment.")
	}
	base := models.BranchVersion(a.code.repo.DefaultBranch)
	return a.openModal(modals.NewTagForm(a.env(), a.code.repo.Name, a.code.branches, a.tagNames(), a.code.repo.DefaultBranch, base))
}

func (a *App) tagWebURL(name string) string {
	q := url.Values{}
	q.Set("version", webVersionPrefix[models.VersionTag]+name)
	return a.code.repo.WebURL + "?" + q.Encode()
}

func (a App) handleTagsKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	s := &a.tags
	if s.panel.Searching() {
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	sel := s.panel.Selected()
	if key.Matches(msg, k.HubCommits) && sel != nil {
		a.setCodeVersion(models.GitVersion{Kind: models.VersionTag, Name: sel.Name})
		return a, a.openCommitsFor(models.CommitCriteria{})
	}
	if hub, ok := a.hubFor(msg); ok {
		return a, a.goToHub(hub)
	}
	switch {
	case key.Matches(msg, k.Fullscreen):
		a.fullscreen = !a.fullscreen
	case key.Matches(msg, k.QuickSearch):
		return a, a.openModal(modals.NewQuickSearchModal(a.env()))
	case key.Matches(msg, k.Repository):
		return a, a.openCodeRepositoryPicker()
	case a.code.repo.ID == "":
	case key.Matches(msg, k.Search):
		return a, s.panel.FocusSearch()
	case key.Matches(msg, k.Back):
		s.panel.ClearFilter()
	case key.Matches(msg, k.NewRef):
		return a, a.openNewTag()
	case sel == nil:
	case key.Matches(msg, k.Select):
		return a, a.browseFilesAt(models.GitVersion{Kind: models.VersionTag, Name: sel.Name})
	case key.Matches(msg, k.Copy):
		return a, copyCmd(sel.Name, "Tag name copied to clipboard")
	case key.Matches(msg, k.OpenBrowser):
		return a, openBrowserCmd(a.tagWebURL(sel.Name))
	default:
		var cmd tea.Cmd
		s.panel, cmd = s.panel.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a App) handleTagsResult(msg tea.Msg) (App, tea.Cmd, bool) {
	s := &a.tags
	switch msg := msg.(type) {
	case tagsLoadedMsg:
		if msg.seq != s.seq {
			return a, nil, true
		}
		s.loading = false
		if msg.err != nil {
			s.panel.SetTags(nil)
			return a, a.openModal(modals.NewErrorModal(a.env(), msg.err)), true
		}
		s.tags = msg.tags
		s.panel.SetTags(msg.tags)
		if s.selectRef != "" {
			s.panel.SelectName(s.selectRef)
			s.selectRef = ""
		}
		return a, nil, true
	case tagDetailDueMsg:
		if msg.objectID != s.detailFor {
			return a, nil, true
		}
		return a, loadTagDetailCmd(a.client, a.code.repo.ID, msg.objectID), true
	case tagDetailMsg:
		if msg.err != nil {
			if s.failed == nil {
				s.failed = map[string]error{}
			}
			s.failed[msg.objectID] = msg.err
			return a, nil, true
		}
		if s.details == nil {
			s.details = map[string]models.TagDetail{}
		}
		s.details[msg.objectID] = msg.detail
		return a, nil, true
	case modals.CreateTagRequestMsg:
		return a, a.mutate(createTagCmd(a.client, a.code.repo.ID, msg)), true
	}
	return a, nil, false
}

func (a *App) renderTagsBody(width, height int) string {
	s := &a.tags
	if a.code.repo.ID == "" {
		return a.renderNoRepository("Tags", width, height)
	}
	listWidth, previewWidth := width, 0
	if width >= commitPreviewShow {
		listWidth, previewWidth = width-commitPreviewWidth, commitPreviewWidth
	}
	list := panels.Pane(panels.PaneTitle("", s.panel.Title(), true), s.panel.View(), listWidth, height, true)
	if previewWidth == 0 {
		return list
	}
	cw, ch := panels.PaneContentSize(previewWidth, height)
	props := panels.TagPreviewProps{Tag: s.panel.Selected(), Spinner: a.spinner.View(), Width: cw, Height: ch}
	if t := props.Tag; t != nil {
		if d, ok := s.details[t.ObjectID]; ok {
			props.Detail = &d
		}
		props.Err = s.failed[t.ObjectID]
	}
	preview := panels.Pane(panels.PaneTitle("", "Preview", false), panels.RenderTagPreview(props), previewWidth, height, false)
	return lipgloss.JoinHorizontal(lipgloss.Top, list, preview)
}

func (a *App) syncTagsLayout(width, height int) {
	listWidth := width
	if width >= commitPreviewShow {
		listWidth = width - commitPreviewWidth
	}
	cw, ch := panels.PaneContentSize(listWidth, height)
	a.tags.panel.SetSize(cw, ch)
}

func (a *App) tagsBindings() []key.Binding {
	if a.code.repo.ID == "" {
		return []key.Binding{a.keys.Repository}
	}
	return a.tags.panel.KeyBindings()
}

func (a *App) searchingTags() bool {
	return a.viewMode == ViewMain && a.inTags() && !a.modals.open() && a.tags.panel.Searching()
}

func (a *App) tagsLoading() bool {
	s := &a.tags
	_, cached := s.details[s.detailFor]
	return a.pr.reposLoading || s.loading || (s.detailFor != "" && !cached && s.failed[s.detailFor] == nil && a.shell.mainWidth >= commitPreviewShow)
}

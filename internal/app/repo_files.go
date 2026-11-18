package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	paneTree = iota
	paneContent
)

const (
	filesTreeRatio    = 0.32
	filesTreeMinWidth = 26
	filesTreeMaxWidth = 48
	folderKey         = "d:"
	fileKey           = "f:"
)

type fileText struct {
	text string
	err  error
}

type filesState struct {
	tree      panels.RepoTreePanel
	content   panels.RepoContentPanel
	pane      int
	rev       int
	requested map[string]bool
	errs      map[string]error
	texts     map[string]fileText
	dueFor    string
}

func newFilesState(tree panels.RepoTreePanel, content panels.RepoContentPanel) filesState {
	return filesState{tree: tree, content: content, requested: map[string]bool{}, errs: map[string]error{}, texts: map[string]fileText{}}
}

type folderLoadedMsg struct {
	seq   int
	path  string
	items []models.GitItem
	err   error
}

type repoFileLoadedMsg struct {
	seq  int
	path string
	text string
	err  error
}

type repoFileDueMsg struct {
	seq  int
	path string
}

func loadFolderCmd(client *api.Client, seq int, repositoryID, path string, v models.GitVersion) tea.Cmd {
	return func() tea.Msg {
		items, err := client.ListItems(repositoryID, path, v)
		return folderLoadedMsg{seq: seq, path: path, items: items, err: err}
	}
}

func loadRepoFileCmd(client *api.Client, seq int, repositoryID, path string, v models.GitVersion) tea.Cmd {
	return func() tea.Msg {
		text, err := client.GetItemText(repositoryID, path, v)
		return repoFileLoadedMsg{seq: seq, path: path, text: text, err: err}
	}
}

func scheduleRepoFileCmd(seq int, path string) tea.Cmd {
	return tea.Tick(commentsDebounce, func(time.Time) tea.Msg {
		return repoFileDueMsg{seq: seq, path: path}
	})
}

func (a *App) inFiles() bool {
	return a.hub == panels.HubFiles
}

func (a *App) openFiles() tea.Cmd {
	return a.openCodeHub(panels.HubFiles)
}

func (a *App) resetFiles() {
	f := &a.code.files
	f.tree.Reset(a.code.repo.Name)
	f.requested, f.errs, f.texts = map[string]bool{}, map[string]error{}, map[string]fileText{}
	f.dueFor, f.pane = "", paneTree
	f.rev++
}

func (a *App) refreshFiles() tea.Cmd {
	if a.code.repo.ID == "" {
		return a.restoreCodeRepository()
	}
	a.setCodeVersion(a.code.version)
	return nil
}

func (a *App) requestFolder(path string) tea.Cmd {
	f := &a.code.files
	if f.requested[folderKey+path] {
		return nil
	}
	f.requested[folderKey+path] = true
	return loadFolderCmd(a.client, a.code.seq, a.code.repo.ID, path, a.code.version)
}

func (a *App) requestFile(path string) tea.Cmd {
	f := &a.code.files
	if f.requested[fileKey+path] {
		return nil
	}
	f.requested[fileKey+path] = true
	return loadRepoFileCmd(a.client, a.code.seq, a.code.repo.ID, path, a.code.version)
}

func (a *App) syncFiles() tea.Cmd {
	if !a.inFiles() || a.viewMode != ViewMain || a.code.repo.ID == "" {
		return nil
	}
	f := &a.code.files
	var cmds []tea.Cmd
	for _, folder := range f.tree.WantedFolders() {
		cmds = append(cmds, a.requestFolder(folder))
	}
	cmds = append(cmds, a.syncFileSelection())
	f.content.Show(a.fileContent(), f.rev)
	return tea.Batch(cmds...)
}

func (a *App) syncFileSelection() tea.Cmd {
	f := &a.code.files
	sel := f.tree.Selected()
	if sel == nil {
		return nil
	}
	if sel.Folder {
		items, ok := f.tree.Children(sel.Path)
		if !ok {
			return a.scheduleFile(folderKey, sel.Path)
		}
		if readme, ok := models.FolderReadme(items); ok {
			return a.requestFile(readme.Path)
		}
		return nil
	}
	return a.scheduleFile(fileKey, sel.Path)
}

func (a *App) scheduleFile(kind, path string) tea.Cmd {
	f := &a.code.files
	if f.requested[kind+path] || f.dueFor == kind+path {
		return nil
	}
	f.dueFor = kind + path
	return scheduleRepoFileCmd(a.code.seq, path)
}

func (a *App) fileContent() panels.RepoContent {
	f := &a.code.files
	sel := f.tree.Selected()
	if sel == nil {
		return panels.RepoContent{}
	}
	c := panels.RepoContent{Item: *sel}
	if !sel.Folder {
		t, ok := f.texts[sel.Path]
		c.Loaded, c.Text, c.Err = ok, t.text, t.err
		return c
	}
	c.Items, c.Loaded = f.tree.Children(sel.Path)
	c.Err = f.errs[sel.Path]
	if readme, ok := models.FolderReadme(c.Items); ok {
		t, loaded := f.texts[readme.Path]
		c.HasReadme, c.ReadmeLoaded, c.Readme = true, loaded, t.text
		if t.err != nil {
			c.Readme = t.err.Error()
		}
	}
	return c
}

func (a *App) filesLoading() bool {
	f := &a.code.files
	return a.pr.reposLoading || a.code.refsLoading || len(f.tree.WantedFolders()) > 0 || f.content.Loading()
}

func (a *App) fileWebURL(path string, lines share.Lines) string {
	return share.FileURL(a.code.repo.WebURL, path, share.VersionParam(a.code.version), lines)
}

func (a *App) fileShareTarget(item models.GitItem) share.Target {
	f := &a.code.files
	code := share.Code{
		RepoURL:  a.code.repo.WebURL,
		RepoName: a.code.repo.Name,
		Path:     item.Path,
		Version:  a.code.version,
		Commit:   item.Last.ID,
		Folder:   item.Folder,
	}
	if lines, ok := f.content.SelectedLines(); ok && f.pane == paneContent {
		return share.FileLines(code, lines)
	}
	return share.File(code)
}

func (a App) handleFilesKey(msg tea.KeyPressMsg) (App, tea.Cmd) {
	k := a.keys
	f := &a.code.files
	sel := f.tree.Selected()
	if key.Matches(msg, k.HubCommits) && a.code.repo.ID != "" {
		criteria := models.CommitCriteria{}
		if sel != nil && sel.Path != "/" {
			criteria.Path = sel.Path
		}
		return a, a.openCommitsFor(criteria)
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
	case key.Matches(msg, k.Version):
		return a, a.openVersionPicker()
	case key.Matches(msg, k.Clone):
		return a, a.openClone()
	case f.pane == paneContent && f.content.Selecting() && key.Matches(msg, k.Yank):
		return a, f.content.YankSelection()
	case f.pane == paneContent && f.content.Selecting() && key.Matches(msg, k.ShareLines) && sel != nil:
		return a, a.share(a.fileShareTarget(*sel), false)
	case a.isShareKey(msg) && sel != nil:
		return a, a.share(a.fileShareTarget(*sel), a.shareNow(msg))
	case key.Matches(msg, k.OpenBrowser) && sel != nil:
		lines := share.Lines{}
		if f.pane == paneContent {
			lines, _ = f.content.SelectedLines()
		}
		return a, openBrowserCmd(a.fileWebURL(sel.Path, lines))
	case key.Matches(msg, k.Back) && f.pane == paneContent && f.content.Selecting():
		f.content.ClearSelection()
	case f.content.IsMarkdownFile() && key.Matches(msg, k.Panel1):
		f.content.SetMarkdownSource(true)
	case f.content.IsMarkdownFile() && key.Matches(msg, k.Panel2):
		f.content.SetMarkdownSource(false)
	case f.content.IsMarkdownFile() && (key.Matches(msg, k.PrevView) || key.Matches(msg, k.NextView)):
		f.content.SetMarkdownSource(!f.content.MarkdownSource())
	case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel):
		f.pane = 1 - f.pane
	case key.Matches(msg, k.Back) && f.pane == paneContent:
		f.pane = paneTree
	default:
		return a, a.updateFilesPanel(msg)
	}
	return a, nil
}

func (a *App) updateFilesPanel(msg tea.Msg) tea.Cmd {
	f := &a.code.files
	var cmd tea.Cmd
	if f.pane == paneContent {
		f.content, cmd = f.content.Update(msg)
		return cmd
	}
	f.tree, cmd = f.tree.Update(msg)
	return cmd
}

func (a App) handleFilesResult(msg tea.Msg) (App, tea.Cmd, bool) {
	f := &a.code.files
	switch msg := msg.(type) {
	case folderLoadedMsg:
		if msg.seq != a.code.seq {
			return a, nil, true
		}
		if msg.err != nil {
			f.errs[msg.path] = msg.err
		}
		f.tree.SetChildren(msg.path, msg.items)
		f.rev++
		return a, nil, true
	case repoFileLoadedMsg:
		if msg.seq != a.code.seq {
			return a, nil, true
		}
		f.texts[msg.path] = fileText{text: msg.text, err: msg.err}
		f.rev++
		return a, nil, true
	case repoFileDueMsg:
		if msg.seq != a.code.seq {
			return a, nil, true
		}
		sel := f.tree.Selected()
		if sel == nil || sel.Path != msg.path {
			return a, nil, true
		}
		f.dueFor = ""
		if sel.Folder {
			return a, a.requestFolder(sel.Path), true
		}
		return a, a.requestFile(sel.Path), true
	case panels.OpenRepoFileMsg:
		f.pane = paneContent
		return a, nil, true
	}
	return a, nil, false
}

func (a *App) filesLayout(width int) (int, int) {
	if width < releaseTwoPanesWidth {
		return width, 0
	}
	left := min(max(int(float64(width)*filesTreeRatio), filesTreeMinWidth), filesTreeMaxWidth)
	return left, width - left
}

func (a *App) syncFilesLayout(width, height int) {
	f := &a.code.files
	_, ch := panels.PaneContentSize(width, height)
	cw := func(w int) int { pw, _ := panels.PaneContentSize(w, height); return pw }
	left, right := a.filesLayout(width)
	if right == 0 {
		f.tree.SetSize(cw(width), ch)
		f.content.SetSize(cw(width), ch)
		return
	}
	f.tree.SetSize(cw(left), ch)
	f.content.SetSize(cw(right), ch)
}

func (a *App) syncFilesFocus(main bool) {
	hub := main && a.inFiles() && a.viewMode == ViewMain
	a.code.files.tree.SetFocused(hub && a.code.files.pane == paneTree)
	a.code.files.content.SetFocused(hub && a.code.files.pane == paneContent)
}

func (a *App) renderFilesBody(width, height int) string {
	f := &a.code.files
	if a.code.repo.ID == "" {
		return a.renderNoRepository("Files", width, height)
	}
	left, right := a.filesLayout(width)
	treeFocused, contentFocused := f.pane == paneTree, f.pane == paneContent
	tree := panels.Pane(panels.PaneTitle("", "Files", treeFocused), f.tree.View(), left, height, treeFocused)
	contentWidth := right
	if right == 0 {
		contentWidth = width
	}
	title := panels.PaneTitle("", f.content.Title(a.code.repo.Name), contentFocused)
	if f.content.IsMarkdownFile() {
		title += " " + f.content.MarkdownTabs(contentFocused)
	}
	content := panels.Pane(title, f.content.View(), contentWidth, height, contentFocused)
	switch {
	case right > 0:
		return lipgloss.JoinHorizontal(lipgloss.Top, tree, content)
	case contentFocused:
		return content
	}
	return tree
}

func (a *App) filesBindings() []key.Binding {
	if a.code.repo.ID == "" {
		return []key.Binding{a.keys.Repository}
	}
	if a.code.files.pane == paneContent {
		return a.code.files.content.KeyBindings()
	}
	return a.code.files.tree.KeyBindings()
}

func (a *App) filesTitle() string {
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	if a.code.repo.ID == "" {
		return bold.Render("Files")
	}
	parts := []string{a.code.repo.Name}
	if sel := a.code.files.tree.Selected(); sel != nil && sel.Path != "/" {
		parts = append(parts, strings.Split(strings.TrimPrefix(sel.Path, "/"), "/")...)
	}
	last := len(parts) - 1
	return panels.Muted(strings.Join(parts[:last], " / ")+strings.Repeat(" / ", min(last, 1))) + bold.Render(parts[last])
}

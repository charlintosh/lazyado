package ui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/models"
)

const (
	fileIndexesKept = 3
	recentFilesKept = 10
	branchMark      = "⎇ "
	refsHeads       = "refs/heads/"
)

type fileIndexEntry struct {
	index   *models.FileIndex
	gen     int
	loading bool
}

type fileIndexes struct {
	entries map[string]*fileIndexEntry
	order   []string
	gen     int
	recent  map[string][]string
}

type fileIndexLoadedMsg struct {
	key   string
	gen   int
	index *models.FileIndex
	err   error
}

func loadFileIndexCmd(client *api.Client, key string, gen int, repositoryID string, v models.GitVersion) tea.Cmd {
	return func() tea.Msg {
		index, err := client.ListAllItems(repositoryID, v, models.MaxIndexedItems)
		return fileIndexLoadedMsg{key: key, gen: gen, index: index, err: err}
	}
}

func (a *App) fileIndexKey() string {
	return a.code.repo.ID + "@" + a.code.version.Type() + ":" + a.code.version.Name
}

func (a *App) versionLabel() string {
	switch {
	case !a.code.version.IsZero() && a.code.version.Kind == models.VersionBranch:
		return branchMark + a.code.version.Label()
	case !a.code.version.IsZero():
		return a.code.version.Type() + " " + a.code.version.Label()
	}
	return branchMark + strings.TrimPrefix(a.code.repo.DefaultBranch, refsHeads)
}

func (a *App) fileIndex() *models.FileIndex {
	if e := a.code.indexes.entries[a.fileIndexKey()]; e != nil {
		return e.index
	}
	return nil
}

func (a *App) openFileFinder() tea.Cmd {
	key := a.fileIndexKey()
	e := a.code.indexes.entries[key]
	props := modals.FileFinderProps{
		Key:     key,
		Version: a.versionLabel(),
		Known:   a.code.files.tree.KnownFiles(),
		Recent:  a.code.indexes.recent[a.code.repo.ID],
	}
	var load tea.Cmd
	switch {
	case e != nil && e.index != nil:
		props.Index = e.index
	case e == nil || !e.loading:
		load = a.requestFileIndex(key)
	}
	return tea.Batch(a.openModal(modals.NewFileFinder(a.env(), props)), load)
}

func (a *App) requestFileIndex(key string) tea.Cmd {
	s := &a.code.indexes
	if s.entries == nil {
		s.entries = map[string]*fileIndexEntry{}
	}
	s.gen++
	s.entries[key] = &fileIndexEntry{loading: true, gen: s.gen}
	s.order = append(slices.DeleteFunc(s.order, func(k string) bool { return k == key }), key)
	for len(s.order) > fileIndexesKept {
		delete(s.entries, s.order[0])
		s.order = s.order[1:]
	}
	return loadFileIndexCmd(a.client, key, s.gen, a.code.repo.ID, a.code.version)
}

func (a *App) forgetFileIndex() {
	delete(a.code.indexes.entries, a.fileIndexKey())
}

func (a *App) rememberFile(path string) {
	s := &a.code.indexes
	if s.recent == nil {
		s.recent = map[string][]string{}
	}
	repo := a.code.repo.ID
	list := slices.DeleteFunc(slices.Clone(s.recent[repo]), func(p string) bool { return p == path })
	list = append([]string{path}, list...)
	s.recent[repo] = list[:min(len(list), recentFilesKept)]
}

func (a App) handleFileFinderResult(msg tea.Msg) (App, tea.Cmd, bool) {
	f := &a.code.files
	switch msg := msg.(type) {
	case fileIndexLoadedMsg:
		e := a.code.indexes.entries[msg.key]
		if e == nil || e.gen != msg.gen {
			return a, nil, true
		}
		if msg.err != nil {
			delete(a.code.indexes.entries, msg.key)
		} else {
			e.index, e.loading = msg.index, false
		}
		next, cmd := a.toFileFinder(modals.FileFinderIndexMsg{Key: msg.key, Index: msg.index, Err: msg.err})
		return next, cmd, true
	case modals.FileFinderRebuildMsg:
		return a, a.requestFileIndex(a.fileIndexKey()), true
	case modals.FileFinderPreviewRequestMsg:
		if t, ok := f.texts[msg.Path]; ok {
			next, cmd := a.toFileFinder(modals.FileFinderPreviewMsg{Path: msg.Path, Text: t.text, Err: t.err})
			return next, cmd, true
		}
		return a, a.requestFile(msg.Path), true
	case modals.FileChosenMsg:
		a.closeModal()
		var children func(string) []models.GitItem
		if x := a.fileIndex(); x != nil {
			children = x.Children
		}
		f.tree.Reveal(msg.Path, children)
		f.pane = paneContent
		f.rev++
		a.rememberFile(msg.Path)
		return a, nil, true
	}
	return a, nil, false
}

func (a App) toFileFinder(msg tea.Msg) (App, tea.Cmd) {
	if !topIs[modals.FileFinder](a.modals) {
		return a, nil
	}
	return a.updateModal(msg)
}

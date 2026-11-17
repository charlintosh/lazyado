package panels

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	treeRoot      = "/"
	treeBranch    = "├── "
	treeLast      = "└── "
	treeTrunk     = "│   "
	treeSpace     = "    "
	loadingFolder = "Loading"
)

type OpenRepoFileMsg struct {
	Item models.GitItem
}

type treeRow struct {
	item    models.GitItem
	prefix  string
	loading bool
}

type RepoTreePanel struct {
	viewport viewport.Model
	keys     keys.KeyMap
	root     string
	children map[string][]models.GitItem
	expanded map[string]bool
	rows     []treeRow
	cursor   int
	focused  bool
	spinner  string
	width    int
	height   int
}

func NewRepoTreePanel(k keys.KeyMap) RepoTreePanel {
	p := RepoTreePanel{viewport: widgets.NewViewport(k), keys: k}
	p.Reset("")
	return p
}

func (p *RepoTreePanel) Reset(root string) {
	p.root = root
	p.children = map[string][]models.GitItem{}
	p.expanded = map[string]bool{treeRoot: true}
	p.cursor = 0
	p.rebuild()
}

func (p *RepoTreePanel) SetChildren(folder string, items []models.GitItem) {
	if items == nil {
		items = []models.GitItem{}
	}
	p.children[folder] = items
	p.rebuild()
}

func (p *RepoTreePanel) Loaded(folder string) bool {
	_, ok := p.children[folder]
	return ok
}

func (p *RepoTreePanel) Children(folder string) ([]models.GitItem, bool) {
	items, ok := p.children[folder]
	return items, ok
}

func (p *RepoTreePanel) WantedFolders() []string {
	var out []string
	for folder := range p.expanded {
		if !p.Loaded(folder) {
			out = append(out, folder)
		}
	}
	return out
}

func (p *RepoTreePanel) Selected() *models.GitItem {
	if p.cursor < 0 || p.cursor >= len(p.rows) || p.rows[p.cursor].loading {
		return nil
	}
	it := p.rows[p.cursor].item
	return &it
}

func (p *RepoTreePanel) SetFocused(focused bool) {
	if p.focused == focused {
		return
	}
	p.focused = focused
	p.render()
}

func (p *RepoTreePanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if len(p.WantedFolders()) > 0 {
		p.render()
	}
}

func (p *RepoTreePanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.viewport.SetWidth(max(width, 1))
	p.viewport.SetHeight(max(height, 1))
	p.render()
}

func (p *RepoTreePanel) rebuild() {
	selected := ""
	if it := p.Selected(); it != nil {
		selected = it.Path
	}
	p.rows = []treeRow{{item: models.GitItem{Path: treeRoot, Folder: true}}}
	p.appendRows(treeRoot, "")
	p.cursor = 0
	for i, r := range p.rows {
		if !r.loading && r.item.Path == selected {
			p.cursor = i
		}
	}
	p.render()
}

func (p *RepoTreePanel) appendRows(folder, indent string) {
	items, ok := p.children[folder]
	if !ok {
		p.rows = append(p.rows, treeRow{prefix: indent + treeLast, loading: true})
		return
	}
	for i, it := range items {
		last := i == len(items)-1
		branch, trunk := treeBranch, treeTrunk
		if last {
			branch, trunk = treeLast, treeSpace
		}
		p.rows = append(p.rows, treeRow{item: it, prefix: indent + branch})
		if it.Folder && p.expanded[it.Path] {
			p.appendRows(it.Path, indent+trunk)
		}
	}
}

func (p *RepoTreePanel) render() {
	if p.width <= 0 {
		return
	}
	lines := make([]string, len(p.rows))
	for i, r := range p.rows {
		lines[i] = p.renderRow(r, i == p.cursor)
	}
	p.viewport.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&p.viewport, p.cursor, p.cursor)
}

func (p *RepoTreePanel) renderRow(r treeRow, selected bool) string {
	base := lipgloss.NewStyle()
	if selected {
		base = base.Background(styles.ColorHighlight)
	}
	prefix := base.Foreground(styles.ColorBorder).Render(r.prefix)
	var name string
	switch {
	case r.loading:
		name = base.Render(Loading(p.spinner, loadingFolder))
	case r.item.Path == treeRoot:
		name = base.Foreground(styles.ColorText).Bold(true).Render(p.root)
		prefix = ""
	case r.item.Folder:
		name = base.Foreground(styles.ColorPrimary).Bold(selected).Render(r.item.Name() + "/")
	default:
		name = base.Foreground(styles.ColorText).Bold(selected).Render(r.item.Name())
	}
	return fill(prefix+name, "", base, p.width)
}

func (p *RepoTreePanel) move(delta int) {
	dir := 1
	if delta < 0 {
		dir = -1
	}
	target := min(max(p.cursor+delta, 0), len(p.rows)-1)
	for p.rows[target].loading {
		if next := target + dir; next >= 0 && next < len(p.rows) {
			target = next
			continue
		}
		dir = -dir
		target += dir
	}
	if target == p.cursor {
		return
	}
	p.cursor = target
	p.render()
}

func (p *RepoTreePanel) selectPath(path string) {
	for i, r := range p.rows {
		if !r.loading && r.item.Path == path {
			p.cursor = i
			p.render()
			return
		}
	}
}

func (p *RepoTreePanel) expand() tea.Cmd {
	it := p.Selected()
	switch {
	case it == nil:
		return nil
	case !it.Folder:
		item := *it
		return func() tea.Msg { return OpenRepoFileMsg{Item: item} }
	case !p.expanded[it.Path]:
		p.expanded[it.Path] = true
		p.rebuild()
	case p.Loaded(it.Path):
		p.move(1)
	}
	return nil
}

func (p *RepoTreePanel) collapse() {
	it := p.Selected()
	if it == nil {
		return
	}
	if it.Folder && it.Path != treeRoot && p.expanded[it.Path] {
		delete(p.expanded, it.Path)
		p.rebuild()
		return
	}
	if parent := models.ParentPath(it.Path); parent != "" {
		p.selectPath(parent)
	}
}

func (p RepoTreePanel) Update(msg tea.Msg) (RepoTreePanel, tea.Cmd) {
	k := p.keys
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		page := max(p.height/2, 1)
		switch {
		case key.Matches(msg, k.Up):
			p.move(-1)
		case key.Matches(msg, k.Down):
			p.move(1)
		case key.Matches(msg, k.HalfPageUp):
			p.move(-page)
		case key.Matches(msg, k.HalfPageDown):
			p.move(page)
		case key.Matches(msg, k.Top):
			p.move(-len(p.rows))
		case key.Matches(msg, k.Bottom):
			p.move(len(p.rows))
		case key.Matches(msg, k.Right), key.Matches(msg, k.Select):
			return p, p.expand()
		case key.Matches(msg, k.Left):
			p.collapse()
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			p.move(-1)
		} else {
			p.move(1)
		}
	}
	return p, nil
}

func (p *RepoTreePanel) KeyBindings() []key.Binding {
	k := p.keys
	open := widgets.Relabel(k.Select, "open")
	if it := p.Selected(); it != nil && it.Folder {
		open = widgets.Relabel(k.Select, "expand")
	}
	return []key.Binding{open, widgets.Relabel(k.Left, "collapse"), widgets.Relabel(k.HubCommits, "history"), k.Version, k.Repository, k.Clone,
		widgets.Relabel(k.Copy, "copy path"), k.OpenBrowser, widgets.Relabel(k.NextPanel, "pane"), k.Refresh}
}

func (p *RepoTreePanel) View() string {
	return Block(p.viewport.View(), p.width, p.height)
}

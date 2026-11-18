package panels

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	contentNameMin    = 16
	contentAgoWidth   = 13
	contentCommitCol  = 9
	contentMessageMin = 24
)

type RepoContent struct {
	Item         models.GitItem
	Loaded       bool
	Items        []models.GitItem
	Text         string
	Err          error
	Readme       string
	HasReadme    bool
	ReadmeLoaded bool
}

type RepoContentPanel struct {
	viewport viewport.Model
	code     codeView
	codeFor  string
	keys     keys.KeyMap
	content  RepoContent
	rev      int
	focused  bool
	source   bool
	spinner  string
	width    int
	height   int
}

func NewRepoContentPanel(k keys.KeyMap) RepoContentPanel {
	return RepoContentPanel{viewport: widgets.NewViewport(k), code: newCodeView(k), keys: k, rev: -1}
}

func (p *RepoContentPanel) Show(c RepoContent, rev int) {
	if p.rev == rev && p.content.Item.Path == c.Item.Path {
		return
	}
	moved := p.content.Item.Path != c.Item.Path
	p.content, p.rev = c, rev
	p.syncCode()
	p.render()
	if moved {
		p.viewport.GotoTop()
	}
}

func (p *RepoContentPanel) syncCode() {
	c := p.content
	switch {
	case !p.ShowsCode() && !p.IsMarkdownFile():
		p.codeFor = ""
	case p.ShowsCode() && p.codeFor != c.Item.Path:
		p.codeFor = c.Item.Path
		p.code.setLines(components.HighlightCode(c.Item.Path, c.Text))
	}
}

func (p *RepoContentPanel) ShowsCode() bool {
	c := p.content
	return c.Loaded && c.Err == nil && !c.Item.Folder && c.Item.Path != "" && (!c.Item.IsMarkdown() || p.source) && !models.IsBinary(c.Text)
}

func (p *RepoContentPanel) IsMarkdownFile() bool {
	c := p.content
	return c.Loaded && c.Err == nil && !c.Item.Folder && c.Item.IsMarkdown()
}

func (p *RepoContentPanel) MarkdownSource() bool { return p.source }

func (p *RepoContentPanel) SetMarkdownSource(source bool) {
	if p.source == source {
		return
	}
	p.source = source
	if !source {
		p.code.clearSelection()
	}
	p.syncCode()
	p.render()
}

func (p *RepoContentPanel) MarkdownTabs(focused bool) string {
	active := 1
	if p.source {
		active = 0
	}
	return Tabs([]string{"File", "Preview"}, active, focused)
}

func (p *RepoContentPanel) Line() int {
	if !p.ShowsCode() {
		return 0
	}
	return p.code.line()
}

func (p *RepoContentPanel) Content() RepoContent { return p.content }

func (p *RepoContentPanel) Selecting() bool {
	return p.ShowsCode() && p.code.selecting()
}

func (p *RepoContentPanel) ClearSelection() {
	p.code.clearSelection()
}

func (p *RepoContentPanel) SelectedLines() (share.Lines, bool) {
	if !p.ShowsCode() || len(p.code.lines) == 0 {
		return share.Lines{}, false
	}
	lo, hi := p.code.selection()
	text := strings.Split(strings.ReplaceAll(p.content.Text, "\r\n", "\n"), "\n")
	var code []string
	if hi < len(text) {
		code = text[lo : hi+1]
	}
	return share.Lines{From: lo + 1, To: hi + 1, Code: code}, true
}

func (p *RepoContentPanel) YankSelection() tea.Cmd {
	lines, ok := p.SelectedLines()
	if !ok || !p.Selecting() {
		return nil
	}
	p.code.clearSelection()
	return copyLinesCmd(lines.Code)
}

func (p *RepoContentPanel) Loading() bool {
	return p.content.Item.Path != "" && (!p.content.Loaded || (p.content.HasReadme && !p.content.ReadmeLoaded))
}

func (p *RepoContentPanel) SetFocused(focused bool) {
	p.focused = focused
	p.code.setFocused(focused)
}

func (p *RepoContentPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.Loading() {
		p.render()
	}
}

func (p *RepoContentPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.viewport.SetWidth(max(width, 1))
	p.viewport.SetHeight(max(height, 1))
	p.code.setSize(width, height)
	p.render()
}

func (p *RepoContentPanel) Title(root string) string {
	c := p.content
	name := c.Item.Name()
	if c.Item.Path == treeRoot {
		name = root
	}
	if c.Item.Folder && c.Loaded && c.Err == nil {
		return fmt.Sprintf("%s · %d items", name, len(c.Items))
	}
	return name
}

func (p *RepoContentPanel) render() {
	if p.width <= 0 {
		return
	}
	p.viewport.SetContent(strings.Join(p.lines(), "\n"))
}

func (p *RepoContentPanel) lines() []string {
	c := p.content
	switch {
	case c.Item.Path == "":
		return nil
	case !c.Loaded:
		return []string{Loading(p.spinner, "Loading "+c.Item.Name())}
	case c.Err != nil:
		return p.errorLines(c.Err)
	case c.Item.Folder:
		return p.folderLines()
	case models.IsBinary(c.Text):
		return []string{Muted(fmt.Sprintf("Binary file, %d bytes. %s opens it in the browser.", len(c.Text), p.keys.OpenBrowser.Help().Key))}
	case c.Item.IsMarkdown():
		return strings.Split(components.RenderMarkdown(c.Text, p.width), "\n")
	}
	return nil
}

func (p *RepoContentPanel) errorLines(err error) []string {
	if errors.Is(err, models.ErrFileTooLarge) {
		return []string{Muted("This file is too large to show here. " + p.keys.OpenBrowser.Help().Key + " opens it in the browser.")}
	}
	return []string{lipgloss.NewStyle().Foreground(styles.ColorError).Width(max(p.width, 1)).Render(err.Error())}
}

type folderColumns struct {
	name, ago, commit, message int
}

func (p *RepoContentPanel) columns() folderColumns {
	c := folderColumns{ago: contentAgoWidth, commit: contentCommitCol}
	rest := p.width - c.ago - c.commit
	c.name = min(max(rest*2/5, contentNameMin), rest)
	c.message = rest - c.name
	if c.message < contentMessageMin {
		c.name += c.message
		c.message = 0
	}
	if c.name < contentNameMin {
		c.name += c.commit
		c.commit = 0
	}
	return c
}

func (p *RepoContentPanel) folderLines() []string {
	cols := p.columns()
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	cell := func(s string, w int, st lipgloss.Style) string {
		if w <= 0 {
			return ""
		}
		return st.Render(Line(ansi.Truncate(s, w-1, "…"), w))
	}
	head := cell("Name", cols.name, muted) + cell("Last change", cols.ago, muted) + cell("Commit", cols.commit, muted) + cell("Message", cols.message, muted)
	lines := []string{head}
	if len(p.content.Items) == 0 {
		lines = append(lines, Muted("This folder is empty."))
	}
	name := lipgloss.NewStyle().Foreground(styles.ColorText)
	folder := lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	commit := lipgloss.NewStyle().Foreground(styles.ColorViolet)
	for _, it := range p.content.Items {
		label, st := it.Name(), name
		if it.Folder {
			label, st = it.Name()+"/", folder
		}
		lines = append(lines, cell(label, cols.name, st)+cell(ShortAgo(it.Last.Date), cols.ago, muted)+
			cell(models.ShortCommit(it.Last.ID), cols.commit, commit)+cell(it.Last.Subject(), cols.message, name))
	}
	if !p.content.HasReadme {
		return lines
	}
	lines = append(lines, "", sectionTitle("README.md", p.width))
	if !p.content.ReadmeLoaded {
		return append(lines, Loading(p.spinner, "Loading README.md"))
	}
	return append(lines, strings.Split(components.RenderMarkdown(p.content.Readme, p.width), "\n")...)
}

func (p RepoContentPanel) Update(msg tea.Msg) (RepoContentPanel, tea.Cmd) {
	if p.ShowsCode() {
		p.code = p.code.update(msg)
		return p, nil
	}
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *RepoContentPanel) KeyBindings() []key.Binding {
	k := p.keys
	move := widgets.Combine("scroll", k.Up, k.Down)
	if n := p.code.selectedLines(); p.ShowsCode() && n > 0 {
		return []key.Binding{widgets.Combine("extend", k.Up, k.Down), widgets.Relabel(k.Yank, fmt.Sprintf("copy %d lines", n)),
			widgets.Relabel(k.ShareLines, "share"), widgets.Relabel(k.ShareLink, "copy link"), k.OpenBrowser, widgets.Relabel(k.Back, "cancel selection")}
	}
	bindings := []key.Binding{move}
	if p.ShowsCode() {
		bindings = []key.Binding{widgets.Combine("line", k.Up, k.Down), k.SelectLines}
	}
	if p.IsMarkdownFile() {
		bindings = append(bindings, widgets.Combine("file/preview", k.Panel1, k.Panel2))
	}
	return append(bindings, k.Share, widgets.Relabel(k.HubCommits, "history"), k.Version, k.Repository, k.Clone,
		k.OpenBrowser, widgets.Relabel(k.NextPanel, "pane"), widgets.Relabel(k.Back, "tree"))
}

func (p *RepoContentPanel) View() string {
	if p.ShowsCode() {
		return p.code.view()
	}
	return Block(p.viewport.View(), p.width, p.height)
}

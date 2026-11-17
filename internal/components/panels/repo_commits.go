package panels

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	commitSearchHint   = "text, author:name, path:/src"
	commitIDWidth      = 9
	commitAuthorWidth  = 20
	commitDateWidth    = 12
	commitPreviewFiles = 12
	loadingCommits     = "Loading commits"
	mergeLabel         = "Merge"
)

type SearchCommitsMsg struct {
	Query string
}

type MoreCommitsMsg struct{}

type CommitsPanel struct {
	list        denseList
	keys        keys.KeyMap
	commits     []models.Commit
	visible     []int
	criteria    models.CommitCriteria
	search      textinput.Model
	searching   bool
	text        string
	loading     bool
	more        bool
	loadingMore bool
	spinner     string
	width       int
	height      int
}

func NewCommitsPanel(k keys.KeyMap) CommitsPanel {
	return CommitsPanel{list: newDenseList(k), keys: k, loading: true, search: widgets.SearchInput(searchPrompt, commitSearchHint)}
}

func (p *CommitsPanel) ShowLoading(criteria models.CommitCriteria) {
	p.commits, p.criteria, p.loading, p.more, p.loadingMore = nil, criteria, true, false, false
	p.list.cursor = 0
	p.rebuild()
}

func (p *CommitsPanel) SetCommits(commits []models.Commit, more bool) {
	p.commits, p.more, p.loading, p.loadingMore = commits, more, false, false
	p.rebuild()
}

func (p *CommitsPanel) SetLoadingMore(loading bool) {
	p.loadingMore = loading
	p.rebuild()
}

func (p *CommitsPanel) Count() int { return len(p.commits) }

func (p *CommitsPanel) More() bool { return p.more }

func (p *CommitsPanel) Loading() bool { return p.loading || p.loadingMore }

func (p *CommitsPanel) Searching() bool { return p.searching }

func (p *CommitsPanel) Selected() *models.Commit {
	if p.list.cursor < 0 || p.list.cursor >= len(p.visible) {
		return nil
	}
	c := p.commits[p.visible[p.list.cursor]]
	return &c
}

func (p *CommitsPanel) SetFocused(focused bool) {
	if p.list.focused == focused {
		return
	}
	p.list.focused = focused
	p.rebuild()
}

func (p *CommitsPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.Loading() {
		p.rebuild()
	}
}

func (p *CommitsPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width-4, 1))
	p.layout()
}

func (p *CommitsPanel) layout() {
	extra := 1
	if p.searching {
		extra += 2
	}
	p.list.setSize(p.width, max(p.height-extra, 1))
	p.rebuild()
}

func (p *CommitsPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.search.SetValue(strings.TrimSpace(p.criteria.Query() + " " + p.text))
	p.search.CursorEnd()
	p.layout()
	return p.search.Focus()
}

func (p *CommitsPanel) closeSearch() {
	p.searching = false
	p.search.Blur()
	p.layout()
}

func (p *CommitsPanel) Filtered() bool { return p.text != "" || !p.criteria.IsZero() }

func (p *CommitsPanel) Title() string {
	switch {
	case p.loading:
		return "Commits"
	case p.text != "":
		return fmt.Sprintf("Commits %d of %d", len(p.visible), len(p.commits))
	}
	title := fmt.Sprintf("Commits %d", len(p.commits))
	if p.more {
		title += "+"
	}
	return title
}

func (p *CommitsPanel) Filter() string {
	return strings.TrimSpace(p.criteria.Query() + " " + p.text)
}

func (p *CommitsPanel) columns() []column {
	return fitColumns(p.width, 0,
		column{title: "Message"},
		column{title: "Commit", width: commitIDWidth},
		column{title: "Author", width: commitAuthorWidth},
		column{title: "Date", width: commitDateWidth},
	)
}

func (p *CommitsPanel) rebuild() {
	p.visible = p.visible[:0]
	for i, c := range p.commits {
		if models.CommitMatches(c, p.text) {
			p.visible = append(p.visible, i)
		}
	}
	p.list.setCount(len(p.visible))
	cols := p.columns()
	rows := make([]string, len(p.visible))
	for i, idx := range p.visible {
		rows[i] = p.row(p.commits[idx], i, cols)
	}
	footer := ""
	switch {
	case p.loading:
		footer = Loading(p.spinner, loadingCommits)
	case len(p.commits) == 0:
		footer = Muted("No commits match.")
	case len(p.visible) == 0:
		footer = Muted(fmt.Sprintf("None of the %d loaded match.", len(p.commits)))
	case p.loadingMore:
		footer = Loading(p.spinner, "Loading more")
	case p.more:
		footer = Muted(moreOnScroll)
	}
	p.list.render(rows, footer)
}

func (p *CommitsPanel) row(c models.Commit, i int, cols []column) string {
	base, lead := p.list.rowStyle(i)
	text := base.Foreground(styles.ColorText).Bold(i == p.list.cursor)
	cells := []string{
		c.Subject(),
		models.ShortCommit(c.ID),
		c.Author,
		ShortAgo(c.Date),
	}
	tones := []lipgloss.Style{text, base.Foreground(styles.ColorViolet), base.Foreground(styles.ColorText), base.Foreground(styles.ColorTextMuted)}
	out := lead
	for j, col := range cols {
		cell := cells[j]
		if j == 0 && c.IsMerge() {
			cell = mergeLabel + "  " + cell
		}
		out += cellText(cell, col.width, tones[j])
	}
	return fill(out, "", base, p.width)
}

func (p *CommitsPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.searching {
		return []key.Binding{widgets.Relabel(k.Select, "search"), widgets.Relabel(k.Back, "close")}
	}
	bindings := []key.Binding{widgets.Relabel(k.Select, "browse files"), widgets.Relabel(k.Search, "filter")}
	if p.Filtered() {
		bindings = append(bindings, widgets.Relabel(k.Back, "clear filter"))
	}
	return append(bindings, k.Version, k.Repository, widgets.Relabel(k.Copy, "copy ID"), k.OpenBrowser, k.Refresh)
}

func (p CommitsPanel) Update(msg tea.Msg) (CommitsPanel, tea.Cmd) {
	if p.searching {
		return p.updateSearch(msg)
	}
	if moved, handled := p.list.handleMove(msg, p.keys); handled {
		if moved {
			p.rebuild()
		}
		return p, p.moreIfAtEnd()
	}
	return p, nil
}

func (p *CommitsPanel) moreIfAtEnd() tea.Cmd {
	if p.more && !p.loadingMore && !p.loading && p.list.atEnd() {
		return func() tea.Msg { return MoreCommitsMsg{} }
	}
	return nil
}

func (p CommitsPanel) updateSearch(msg tea.Msg) (CommitsPanel, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, p.keys.Select):
			query := p.search.Value()
			p.closeSearch()
			return p, func() tea.Msg { return SearchCommitsMsg{Query: query} }
		case key.Matches(k, p.keys.Back):
			_, p.text = models.ParseCommitQuery(p.Filter())
			p.closeSearch()
			return p, nil
		case key.Matches(k, p.keys.PickerUp), key.Matches(k, p.keys.PickerDown):
			delta := 1
			if key.Matches(k, p.keys.PickerUp) {
				delta = -1
			}
			p.list.move(delta)
			p.rebuild()
			return p, nil
		}
	}
	var cmd tea.Cmd
	p.search, cmd = p.search.Update(msg)
	_, p.text = models.ParseCommitQuery(p.search.Value())
	p.rebuild()
	return p, cmd
}

func (p *CommitsPanel) SetText(text string) {
	p.text = text
	p.rebuild()
}

func (p *CommitsPanel) View() string {
	parts := []string{}
	if p.searching {
		parts = append(parts, Line(p.search.View(), p.width), "")
	}
	parts = append(parts, Line(columnHeader(p.columns()), p.width), p.list.view())
	return Block(strings.Join(parts, "\n"), p.width, p.height)
}

type CommitPreviewProps struct {
	Commit  *models.Commit
	Detail  *models.CommitDetail
	Err     error
	Spinner string
	Width   int
	Height  int
}

func RenderCommitPreview(p CommitPreviewProps) string {
	if p.Commit == nil {
		return Block("", p.Width, p.Height)
	}
	c := *p.Commit
	if p.Detail != nil {
		c = p.Detail.Commit
	}
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	wrap := lipgloss.NewStyle().Foreground(styles.ColorText).Width(max(p.Width, 1))
	lines := []string{bold.Width(max(p.Width, 1)).Render(c.Subject())}
	if body := c.Body(); body != "" {
		lines = append(lines, "", wrap.Render(body))
	}
	commit := lipgloss.NewStyle().Foreground(styles.ColorViolet)
	rows := [][2]string{
		{"Commit", commit.Render(models.ShortCommit(c.ID))},
		{"Author", c.Author},
		{"Date", formatStamp(c.Date)},
	}
	if c.Committer != "" && c.Committer != c.Author {
		rows = append(rows, [2]string{"Committer", c.Committer})
	}
	if p.Detail != nil {
		parents := make([]string, len(c.Parents))
		for i, id := range c.Parents {
			parents[i] = commit.Render(models.ShortCommit(id))
		}
		label := "Parent"
		if len(parents) > 1 {
			label = "Parents"
		}
		if len(parents) > 0 {
			rows = append(rows, [2]string{label, strings.Join(parents, " ")})
		}
	}
	lines = append(lines, "", FieldRows(rows, 11, p.Width))
	lines = append(lines, "", sectionTitle(changesTitle(p.Detail), p.Width))
	if p.Err != nil {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorError).Width(max(p.Width, 1)).Render(p.Err.Error()))
		return Block(strings.Join(lines, "\n"), p.Width, p.Height)
	}
	if p.Detail == nil {
		lines = append(lines, Loading(p.Spinner, "Loading changes"))
		return Block(strings.Join(lines, "\n"), p.Width, p.Height)
	}
	for i, f := range p.Detail.Files {
		if i == commitPreviewFiles {
			lines = append(lines, Muted(fmt.Sprintf("+%d more", len(p.Detail.Files)-i)))
			break
		}
		letter := changeColor(f.Kind).Bold(true).Render(f.Kind.Letter())
		lines = append(lines, letter+" "+Line(strings.TrimPrefix(f.Path, "/"), max(p.Width-2, 1)))
	}
	return Block(strings.Join(lines, "\n"), p.Width, p.Height)
}

func changesTitle(d *models.CommitDetail) string {
	if d == nil {
		return "Changes"
	}
	return fmt.Sprintf("Changes · %d files", len(d.Files))
}

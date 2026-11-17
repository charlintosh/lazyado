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
	tagSearchHint = "tag name"
	tagKindWidth  = 14
	loadingTags   = "Loading tags"
	annotated     = "Annotated"
	lightweight   = "Lightweight"
)

type TagsPanel struct {
	list      denseList
	keys      keys.KeyMap
	tags      []models.Tag
	visible   []int
	search    textinput.Model
	searching bool
	loading   bool
	spinner   string
	width     int
	height    int
}

func NewTagsPanel(k keys.KeyMap) TagsPanel {
	return TagsPanel{list: newDenseList(k), keys: k, loading: true, search: widgets.SearchInput(searchPrompt, tagSearchHint)}
}

func (p *TagsPanel) ShowLoading() {
	p.tags, p.loading = nil, true
	p.rebuild()
}

func (p *TagsPanel) SetTags(tags []models.Tag) {
	p.tags, p.loading = tags, false
	p.rebuild()
}

func (p *TagsPanel) Loading() bool { return p.loading }

func (p *TagsPanel) Searching() bool { return p.searching }

func (p *TagsPanel) Filter() string { return strings.TrimSpace(p.search.Value()) }

func (p *TagsPanel) Selected() *models.Tag {
	if p.list.cursor < 0 || p.list.cursor >= len(p.visible) {
		return nil
	}
	t := p.tags[p.visible[p.list.cursor]]
	return &t
}

func (p *TagsPanel) SelectName(name string) {
	for i, idx := range p.visible {
		if p.tags[idx].Name == name {
			p.list.cursor = i
			p.rebuild()
			return
		}
	}
}

func (p *TagsPanel) SetFocused(focused bool) {
	if p.list.focused == focused {
		return
	}
	p.list.focused = focused
	p.rebuild()
}

func (p *TagsPanel) SetSpinner(frame string) {
	if p.spinner == frame {
		return
	}
	p.spinner = frame
	if p.loading {
		p.rebuild()
	}
}

func (p *TagsPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width, p.height = width, height
	p.search.SetWidth(max(width-4, 1))
	p.layout()
}

func (p *TagsPanel) layout() {
	extra := 1
	if p.searching {
		extra += 2
	}
	p.list.setSize(p.width, max(p.height-extra, 1))
	p.rebuild()
}

func (p *TagsPanel) FocusSearch() tea.Cmd {
	p.searching = true
	p.search.CursorEnd()
	p.layout()
	return p.search.Focus()
}

func (p *TagsPanel) ClearFilter() {
	p.search.SetValue("")
	p.rebuild()
}

func (p *TagsPanel) Title() string {
	if p.loading {
		return "Tags"
	}
	title := fmt.Sprintf("Tags %d", len(p.visible))
	if filter := p.Filter(); filter != "" && !p.searching {
		title += " · /" + filter
	}
	return title
}

func (p *TagsPanel) columns() []column {
	return fitColumns(p.width, 0,
		column{title: "Tag"},
		column{title: "Commit", width: commitIDWidth},
		column{title: "Kind", width: tagKindWidth},
	)
}

func (p *TagsPanel) rebuild() {
	filter := strings.ToLower(p.Filter())
	p.visible = p.visible[:0]
	for i, t := range p.tags {
		if strings.Contains(strings.ToLower(t.Name), filter) {
			p.visible = append(p.visible, i)
		}
	}
	p.list.setCount(len(p.visible))
	cols := p.columns()
	rows := make([]string, len(p.visible))
	for i, idx := range p.visible {
		rows[i] = p.row(p.tags[idx], i, cols)
	}
	footer := ""
	switch {
	case p.loading:
		footer = Loading(p.spinner, loadingTags)
	case len(p.visible) == 0 && filter != "":
		footer = Muted("No tag matches.")
	case len(p.visible) == 0:
		footer = Muted("No tags yet. " + p.keys.NewRef.Help().Key + " creates one.")
	}
	p.list.render(rows, footer)
}

func (p *TagsPanel) row(t models.Tag, i int, cols []column) string {
	base, lead := p.list.rowStyle(i)
	kind := annotated
	if !t.Annotated() {
		kind = lightweight
	}
	cells := []string{t.Name, models.ShortCommit(t.CommitID), kind}
	tones := []lipgloss.Style{base.Foreground(styles.ColorText).Bold(i == p.list.cursor), base.Foreground(styles.ColorViolet), base.Foreground(styles.ColorTextMuted)}
	out := lead
	for j, col := range cols {
		out += cellText(cells[j], col.width, tones[j])
	}
	return fill(out, "", base, p.width)
}

func (p *TagsPanel) KeyBindings() []key.Binding {
	k := p.keys
	if p.searching {
		return []key.Binding{widgets.Relabel(k.Select, "done"), widgets.Relabel(k.Back, "clear")}
	}
	bindings := []key.Binding{widgets.Relabel(k.NewRef, "new tag"), widgets.Relabel(k.Select, "files"), widgets.Relabel(k.HubCommits, "commits")}
	if p.Filter() != "" {
		bindings = append(bindings, widgets.Relabel(k.Back, "clear filter"))
	}
	return append(bindings, widgets.Relabel(k.Search, "search"), k.Share, k.Repository, k.OpenBrowser, k.Refresh)
}

func (p TagsPanel) Update(msg tea.Msg) (TagsPanel, tea.Cmd) {
	if p.searching {
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(k, p.keys.Select):
				p.searching = false
				p.search.Blur()
				p.layout()
				return p, nil
			case key.Matches(k, p.keys.Back):
				p.searching = false
				p.search.Blur()
				p.search.SetValue("")
				p.layout()
				return p, nil
			}
		}
		var cmd tea.Cmd
		p.search, cmd = p.search.Update(msg)
		p.list.cursor = 0
		p.rebuild()
		return p, cmd
	}
	if moved, handled := p.list.handleMove(msg, p.keys); handled && moved {
		p.rebuild()
	}
	return p, nil
}

func (p *TagsPanel) View() string {
	parts := []string{}
	if p.searching {
		parts = append(parts, Line(p.search.View(), p.width), "")
	}
	parts = append(parts, Line(columnHeader(p.columns()), p.width), p.list.view())
	return Block(strings.Join(parts, "\n"), p.width, p.height)
}

type TagPreviewProps struct {
	Tag     *models.Tag
	Detail  *models.TagDetail
	Err     error
	Spinner string
	Width   int
	Height  int
}

func RenderTagPreview(p TagPreviewProps) string {
	if p.Tag == nil {
		return Block("", p.Width, p.Height)
	}
	t := *p.Tag
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	commit := lipgloss.NewStyle().Foreground(styles.ColorViolet).Render(models.ShortCommit(t.CommitID))
	lines := []string{bold.Render(t.Name) + "  " + Muted(strings.ToLower(lightweight))}
	if !t.Annotated() {
		lines = append(lines, "", FieldRows([][2]string{{"Commit", commit}}, 9, p.Width), "",
			lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(max(p.Width, 1)).Render("A lightweight tag has no message or tagger."))
		return Block(strings.Join(lines, "\n"), p.Width, p.Height)
	}
	lines[0] = bold.Render(t.Name) + "  " + Muted(strings.ToLower(annotated))
	if p.Err != nil {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorError).Width(max(p.Width, 1)).Render(p.Err.Error()))
		return Block(strings.Join(lines, "\n"), p.Width, p.Height)
	}
	if p.Detail == nil {
		lines = append(lines, "", Loading(p.Spinner, "Loading tag"))
		return Block(strings.Join(lines, "\n"), p.Width, p.Height)
	}
	d := *p.Detail
	if msg := strings.TrimSpace(d.Message); msg != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(styles.ColorText).Width(max(p.Width, 1)).Render(msg))
	}
	rows := [][2]string{{"Tagger", d.Tagger}, {"Date", formatStamp(d.Date)}, {"Commit", commit}}
	lines = append(lines, "", FieldRows(rows, 9, p.Width))
	return Block(strings.Join(lines, "\n"), p.Width, p.Height)
}

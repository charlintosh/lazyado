package widgets

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const pickerPrompt = "› "

type Option struct {
	Label  string
	Value  string
	Detail string
	Marked bool
}

func (o Option) FilterValue() string { return o.Label }

type PickerConfig struct {
	Rows        int
	Search      bool
	Placeholder string
	Noun        string
	Nouns       string
}

type Picker struct {
	list    list.Model
	input   textinput.Model
	keys    keys.KeyMap
	cfg     PickerConfig
	width   int
	rows    int
	search  bool
	applied string
}

func NewPicker(k keys.KeyMap, options []Option, cfg PickerConfig) Picker {
	items := make([]list.Item, len(options))
	for i, o := range options {
		items[i] = o
	}
	l := NewList(items, optionDelegate{}, k, "")
	l.SetShowTitle(false)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(false)
	if cfg.Noun != "" {
		l.SetStatusBarItemName(cfg.Noun, cfg.Nouns)
	}
	in := SearchInput(pickerPrompt, cfg.Placeholder)
	p := Picker{list: l, input: in, keys: k, cfg: cfg, rows: max(cfg.Rows, 1), search: cfg.Search}
	p.resize()
	return p
}

func SearchInput(prompt, placeholder string) textinput.Model {
	in := textinput.New()
	in.Prompt = prompt
	in.Placeholder = placeholder
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	st.Focused.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	st.Blurred.Prompt = lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	st.Blurred.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	in.SetStyles(st)
	return in
}

func (p Picker) Init() tea.Cmd {
	if p.search {
		return textinput.Blink
	}
	return nil
}

func (p *Picker) Focus() tea.Cmd {
	if p.search {
		return p.input.Focus()
	}
	return nil
}

func (p *Picker) Blur() {
	p.input.Blur()
}

func (p *Picker) SetWidth(width int) {
	p.width = max(width, 1)
	p.input.SetWidth(max(p.width-lipgloss.Width(pickerPrompt)-1, 1))
	p.resize()
}

func (p *Picker) SetRows(rows int) {
	p.rows = max(min(rows, p.cfg.Rows), 1)
	p.resize()
}

func (p *Picker) SetQuery(query string) {
	if p.search {
		p.input.SetValue(query)
		p.input.CursorEnd()
	}
	p.applyQuery(query)
}

func (p Picker) Query() string {
	if p.search {
		return p.input.Value()
	}
	return p.list.FilterInput.Value()
}

func (p *Picker) Select(value string) {
	for i, item := range p.list.VisibleItems() {
		if item.(Option).Value == value {
			p.list.Select(i)
			return
		}
	}
}

func (p Picker) Selected() (Option, bool) {
	o, ok := p.list.SelectedItem().(Option)
	return o, ok
}

func (p Picker) Len() int {
	return len(p.list.VisibleItems())
}

func (p Picker) Update(msg tea.Msg) (Picker, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if p.navigate(k) {
			return p, nil
		}
		if !p.search {
			return p, nil
		}
	}
	if !p.search {
		return p, nil
	}
	before := p.input.Value()
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != before {
		p.applyQuery(p.input.Value())
	}
	return p, cmd
}

func (p *Picker) navigate(k tea.KeyPressMsg) bool {
	switch {
	case key.Matches(k, p.keys.PickerUp):
		p.list.CursorUp()
	case key.Matches(k, p.keys.PickerDown):
		p.list.CursorDown()
	case p.search:
		return false
	case key.Matches(k, p.keys.Up):
		p.list.CursorUp()
	case key.Matches(k, p.keys.Down):
		p.list.CursorDown()
	case key.Matches(k, p.keys.Top):
		p.list.GoToStart()
	case key.Matches(k, p.keys.Bottom):
		p.list.GoToEnd()
	default:
		return false
	}
	return true
}

func (p *Picker) applyQuery(query string) {
	if strings.TrimSpace(query) == strings.TrimSpace(p.applied) {
		return
	}
	p.applied = query
	if strings.TrimSpace(query) == "" {
		p.list.ResetFilter()
	} else {
		p.list.SetFilterText(query)
	}
	p.resize()
}

func (p *Picker) listHeight() int {
	if p.search {
		return p.rows + 1
	}
	visible := max(min(p.rows, p.Len()), 1)
	if p.Len() > p.rows {
		visible++
	}
	return visible
}

func (p *Picker) resize() {
	p.list.SetShowPagination(p.Len() > p.rows)
	p.list.SetSize(max(p.width, 1), p.listHeight())
}

func (p Picker) Height() int {
	h := p.listHeight()
	if p.search {
		h++
	}
	return h
}

func (p Picker) View() string {
	body := p.list.View()
	if !p.search {
		return body
	}
	h := p.listHeight()
	body = lipgloss.NewStyle().Height(h).MaxHeight(h).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, p.input.View(), body)
}

type optionDelegate struct{}

func (optionDelegate) Height() int                             { return 1 }
func (optionDelegate) Spacing() int                            { return 0 }
func (optionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (optionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	o, ok := item.(Option)
	if !ok {
		return
	}
	width := max(m.Width(), 1)
	selected := index == m.Index()
	base := lipgloss.NewStyle().Foreground(styles.ColorLight)
	if o.Marked {
		base = base.Foreground(styles.ColorSuccess)
	}
	cursor := "  "
	if selected {
		base = base.Background(styles.ColorHighlight).Bold(true)
		if !o.Marked {
			base = base.Foreground(styles.ColorText)
		}
		cursor = pickerPrompt
	}
	match := base.Foreground(styles.ColorPrimary).Underline(true)
	label := ansi.Truncate(o.Label, max(width-lipgloss.Width(cursor)-1, 1), "…")
	line := base.Render(cursor) + lipgloss.StyleRunes(label, m.MatchesForItem(index), match, base)
	if o.Detail != "" {
		line += base.Foreground(styles.ColorTextMuted).Bold(false).Render(" " + o.Detail)
	}
	_, _ = io.WriteString(w, base.Inline(true).Width(width).MaxWidth(width).Render(line))
}

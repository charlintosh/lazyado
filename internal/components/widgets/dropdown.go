package widgets

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	dropdownPrompt = "⌕ "
	dropdownMark   = "● "
	dropdownGutter = "  "
)

type DropdownItem struct {
	Label      string
	Value      string
	Group      string
	Badge      string
	BadgeColor color.Color
	Note       string
	Count      string
	Marked     bool
	Muted      bool
}

type dropdownLine struct {
	item  int
	group string
}

type Dropdown struct {
	items   []DropdownItem
	visible []int
	input   textinput.Model
	keys    keys.KeyMap
	cursor  int
	offset  int
	rows    int
	limit   int
	width   int
}

func NewDropdown(k keys.KeyMap, items []DropdownItem, rows int, placeholder string) Dropdown {
	d := Dropdown{items: items, input: SearchInput(dropdownPrompt, placeholder), keys: k, limit: max(rows, 1)}
	d.filter()
	d.rows = min(len(d.lines()), d.limit)
	return d
}

func (d Dropdown) Init() tea.Cmd {
	return textinput.Blink
}

func (d *Dropdown) Focus() tea.Cmd {
	return d.input.Focus()
}

func (d *Dropdown) SetWidth(width int) {
	d.width = max(width, 1)
	d.input.SetWidth(max(d.width-lipgloss.Width(dropdownPrompt)-1, 1))
}

func (d *Dropdown) SetRows(rows int) {
	d.limit = max(rows, 1)
	d.rows = max(min(len(d.allLines()), d.limit), 1)
	d.reveal()
}

func (d Dropdown) Height() int {
	return d.rows + 2
}

func (d Dropdown) Query() string {
	return d.input.Value()
}

func (d *Dropdown) SetQuery(query string) {
	d.input.SetValue(query)
	d.input.CursorEnd()
	d.filter()
}

func (d *Dropdown) Select(value string) {
	for i, idx := range d.visible {
		if d.items[idx].Value == value {
			d.cursor = i
			d.reveal()
			return
		}
	}
}

func (d Dropdown) Selected() (DropdownItem, bool) {
	if d.cursor < 0 || d.cursor >= len(d.visible) {
		return DropdownItem{}, false
	}
	return d.items[d.visible[d.cursor]], true
}

func (d Dropdown) Update(msg tea.Msg) (Dropdown, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, d.keys.PickerUp):
			d.move(-1)
			return d, nil
		case key.Matches(k, d.keys.PickerDown):
			d.move(1)
			return d, nil
		}
	}
	before := d.input.Value()
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	if d.input.Value() != before {
		d.filter()
	}
	return d, cmd
}

func (d *Dropdown) move(delta int) {
	if len(d.visible) == 0 {
		return
	}
	d.cursor = max(min(d.cursor+delta, len(d.visible)-1), 0)
	d.reveal()
}

func (d *Dropdown) filter() {
	q := strings.ToLower(strings.TrimSpace(d.input.Value()))
	visible := make([]int, 0, len(d.items))
	for i, it := range d.items {
		if q == "" || strings.Contains(strings.ToLower(it.Label), q) {
			visible = append(visible, i)
		}
	}
	d.visible = visible
	d.cursor, d.offset = 0, 0
}

func (d Dropdown) allLines() []dropdownLine {
	all := Dropdown{items: d.items}
	for i := range d.items {
		all.visible = append(all.visible, i)
	}
	return all.lines()
}

func (d Dropdown) lines() []dropdownLine {
	var out []dropdownLine
	group := ""
	for i, idx := range d.visible {
		if g := d.items[idx].Group; g != group {
			if g != "" {
				out = append(out, dropdownLine{item: -1, group: g})
			}
			group = g
		}
		out = append(out, dropdownLine{item: i})
	}
	return out
}

func (d *Dropdown) reveal() {
	lines := d.lines()
	at := 0
	for i, l := range lines {
		if l.item == d.cursor {
			at = i
			break
		}
	}
	if at > 0 && lines[at-1].item < 0 {
		at--
	}
	if at < d.offset {
		d.offset = at
	}
	for i, l := range lines {
		if l.item == d.cursor && i >= d.offset+d.rows {
			d.offset = i - d.rows + 1
		}
	}
	d.offset = max(min(d.offset, len(lines)-d.rows), 0)
}

func (d Dropdown) View() string {
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", d.width))
	lines := d.lines()
	out := make([]string, 0, d.rows)
	if len(lines) == 0 {
		out = append(out, lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(dropdownGutter+"No matches"))
	}
	for i := d.offset; i < len(lines) && len(out) < d.rows; i++ {
		l := lines[i]
		if l.item < 0 {
			out = append(out, d.renderGroup(l.group))
			continue
		}
		out = append(out, d.renderItem(d.items[d.visible[l.item]], l.item == d.cursor))
	}
	body := lipgloss.NewStyle().Width(d.width).Height(d.rows).MaxHeight(d.rows).Render(strings.Join(out, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, d.input.View(), rule, body)
}

func (d Dropdown) renderGroup(label string) string {
	label = ansi.Truncate(label, max(d.width-len(dropdownGutter)-2, 1), "…")
	fill := max(d.width-len(dropdownGutter)-lipgloss.Width(label)-1, 0)
	return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(dropdownGutter+label+" ") +
		lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", fill))
}

func (d Dropdown) renderItem(it DropdownItem, selected bool) string {
	base := lipgloss.NewStyle().Foreground(styles.ColorText)
	if it.Muted {
		base = base.Foreground(styles.ColorTextMuted)
	}
	if selected {
		base = base.Background(styles.ColorHighlight).Bold(true)
	}
	mark := base.Render(dropdownGutter)
	if it.Marked {
		mark = base.Foreground(styles.ColorSuccess).Render(dropdownMark)
	}
	badge := ""
	if it.Badge != "" {
		badge = lipgloss.NewStyle().Background(it.BadgeColor).Foreground(styles.ColorOnAccent).Bold(true).Padding(0, 1).Render(it.Badge) + base.Render(" ")
	}
	muted := base.Foreground(styles.ColorTextMuted).Bold(false)
	count := ""
	if it.Count != "" {
		count = muted.Render(" " + it.Count + " ")
	}
	note := ""
	if it.Note != "" {
		note = muted.Render(" " + it.Note)
	}
	room := max(d.width-lipgloss.Width(mark)-lipgloss.Width(badge)-lipgloss.Width(note)-lipgloss.Width(count), 1)
	label := base.Render(ansi.Truncate(it.Label, room, "…"))
	gap := max(d.width-lipgloss.Width(mark)-lipgloss.Width(badge)-lipgloss.Width(label)-lipgloss.Width(note)-lipgloss.Width(count), 0)
	line := mark + badge + label + note + base.Render(strings.Repeat(" ", gap)) + count
	return lipgloss.NewStyle().MaxWidth(d.width).Render(line)
}

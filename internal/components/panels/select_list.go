package panels

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

type listItem struct {
	header   string
	headerBg color.Color
	headerFg color.Color
	note     string
	line     string
	key      string
	render   func(base lipgloss.Style, selected bool, width int) []string
}

type selectList struct {
	viewport viewport.Model
	items    []listItem
	rows     map[int][]string
	cursor   int
	footer   string
	focused  bool
	width    int
	height   int
}

func newSelectList(k keys.KeyMap) selectList {
	return selectList{viewport: widgets.NewViewport(k), cursor: -1}
}

func (l *selectList) setItems(items []listItem) {
	selected := l.selectedKey()
	l.items, l.rows = items, nil
	l.cursor = l.next(-1, 1)
	for i, it := range items {
		if it.selectable() && it.key == selected && selected != "" {
			l.cursor = i
		}
	}
	l.render()
}

func (l *selectList) next(from, dir int) int {
	for i := from + dir; i >= 0 && i < len(l.items); i += dir {
		if l.items[i].selectable() {
			return i
		}
	}
	return -1
}

func (l *selectList) selectedKey() string {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		return ""
	}
	return l.items[l.cursor].key
}

func (l *selectList) count() int {
	n := 0
	for _, it := range l.items {
		if it.selectable() {
			n++
		}
	}
	return n
}

func (l *selectList) move(delta int) bool {
	if l.cursor < 0 || delta == 0 {
		return false
	}
	dir := 1
	if delta < 0 {
		dir = -1
	}
	target := l.cursor
	for steps := delta * dir; steps > 0; steps-- {
		n := l.next(target, dir)
		if n < 0 {
			break
		}
		target = n
	}
	if target == l.cursor {
		return false
	}
	l.cursor = target
	l.render()
	return true
}

func (l *selectList) atEnd() bool {
	return l.cursor >= 0 && l.next(l.cursor, 1) < 0
}

func (l *selectList) handleMove(msg tea.KeyPressMsg, k keys.KeyMap, rowHeight int) (bool, bool) {
	page := max(l.height/(rowHeight+1)/2, 1)
	switch {
	case key.Matches(msg, k.Up):
		return l.move(-1), true
	case key.Matches(msg, k.Down):
		return l.move(1), true
	case key.Matches(msg, k.HalfPageUp):
		return l.move(-page), true
	case key.Matches(msg, k.HalfPageDown):
		return l.move(page), true
	case key.Matches(msg, k.Top):
		return l.move(-len(l.items)), true
	case key.Matches(msg, k.Bottom):
		return l.move(len(l.items)), true
	}
	return false, false
}

func (l *selectList) setFocused(focused bool) {
	if l.focused == focused {
		return
	}
	l.focused = focused
	l.render()
}

func (l *selectList) setSize(width, height int) {
	if l.width != width {
		l.rows = nil
	}
	l.width, l.height = width, height
	l.viewport.SetWidth(max(width, 1))
	l.viewport.SetHeight(max(height, 1))
	l.render()
}

func (l *selectList) render() {
	if l.width <= 0 {
		return
	}
	var lines []string
	top, bottom := 0, 0
	for i, it := range l.items {
		if it.header != "" {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, l.headerLine(it))
			continue
		}
		if it.line != "" {
			lines = append(lines, "  "+Line(it.line, max(l.width-2, 1)))
			continue
		}
		if it.note != "" {
			lines = append(lines, "   "+Line(Muted(it.note), max(l.width-3, 1)))
			continue
		}
		if len(lines) > 0 && !l.items[i-1].isHeader() {
			lines = append(lines, "")
		}
		selected := i == l.cursor
		base := lipgloss.NewStyle()
		bar := " "
		if selected {
			base = base.Background(styles.ColorHighlight)
			if l.focused {
				bar = railBar
			}
			top = len(lines)
		}
		lead := base.Foreground(styles.ColorPrimary).Render(bar) + base.Render(" ")
		row := l.rowLines(i, it, base, selected)
		for _, line := range row {
			lines = append(lines, lead+line)
		}
		if selected {
			bottom = len(lines) - 1
		}
	}
	if l.footer != "" {
		lines = append(lines, "", " "+l.footer)
	}
	l.viewport.SetContent(strings.Join(lines, "\n"))
	widgets.Reveal(&l.viewport, top, bottom)
}

func (l *selectList) rowLines(i int, it listItem, base lipgloss.Style, selected bool) []string {
	if selected {
		return it.render(base, true, max(l.width-2, 1))
	}
	if row, ok := l.rows[i]; ok {
		return row
	}
	row := it.render(base, false, max(l.width-2, 1))
	if l.rows == nil {
		l.rows = make(map[int][]string, len(l.items))
	}
	l.rows[i] = row
	return row
}

func (l *selectList) headerLine(it listItem) string {
	if it.headerBg == nil {
		return " " + lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Bold(true).Render(it.header)
	}
	tag := lipgloss.NewStyle().Background(it.headerBg).Foreground(it.headerFg).Bold(true).Padding(0, 1).Render(it.header)
	rule := lipgloss.NewStyle().Foreground(styles.ColorChip).Render(strings.Repeat(ruleGlyph, max(l.width-lipgloss.Width(tag)-3, 0)))
	return " " + tag + " " + rule
}

func (it listItem) isHeader() bool { return it.header != "" || it.note != "" || it.line != "" }

func (it listItem) selectable() bool { return it.header == "" && it.note == "" && it.line == "" }

func (l *selectList) view() string {
	return Block(l.viewport.View(), l.width, l.height)
}

func emptyState(title, hint string, width, height int) string {
	msg := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title),
		lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(max(width-2, 1)).Render(hint))
	return Block(lipgloss.NewStyle().Padding(1, 1).Render(msg), width, height)
}

func cellsLine(left, right string, base lipgloss.Style, width int) string {
	if right != "" {
		right = base.Render(" ") + right
	}
	return fill(left, right, base, width)
}

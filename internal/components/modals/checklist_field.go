package modals

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

const checklistToggle = " "

type checkItem struct {
	label string
	value string
	on    bool
}

type checklistField struct {
	name    string
	items   []checkItem
	cursor  int
	focused bool
	keys    keys.KeyMap
}

func newChecklistField(k keys.KeyMap, name string, items []checkItem) checklistField {
	return checklistField{name: name, items: items, keys: k}
}

func (f checklistField) label() string   { return f.name }
func (f checklistField) multiline() bool { return false }
func (f checklistField) compact() bool   { return false }
func (f checklistField) choice() bool    { return true }

func (f checklistField) values(on bool) []string {
	var out []string
	for _, it := range f.items {
		if it.on == on {
			out = append(out, it.value)
		}
	}
	return out
}

func (f checklistField) value() string { return strings.Join(f.values(true), listJoin) }

func (f checklistField) focus() (field, tea.Cmd) {
	f.focused = true
	return f, nil
}

func (f checklistField) blur() field {
	f.focused = false
	return f
}

func (f checklistField) toggle() checklistField {
	if f.cursor < 0 || f.cursor >= len(f.items) {
		return f
	}
	items := make([]checkItem, len(f.items))
	copy(items, f.items)
	items[f.cursor].on = !items[f.cursor].on
	f.items = items
	return f
}

func (f checklistField) update(msg tea.Msg) (field, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, nil
	}
	switch {
	case press.Text == checklistToggle:
		return f.toggle(), nil
	case key.Matches(press, f.keys.PickerUp), key.Matches(press, f.keys.Up):
		f.cursor = max(f.cursor-1, 0)
	case key.Matches(press, f.keys.PickerDown), key.Matches(press, f.keys.Down):
		f.cursor = min(f.cursor+1, len(f.items)-1)
	}
	return f, nil
}

func (f checklistField) resize(int) field { return f }

func (f checklistField) view() string {
	lines := make([]string, len(f.items))
	for i, it := range f.items {
		box := uncheckedBox
		style := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if it.on {
			box = checkedBox
			style = style.Foreground(styles.ColorLight)
		}
		if f.focused && i == f.cursor {
			style = style.Foreground(styles.ColorPrimary).Bold(true)
		}
		lines[i] = style.Render(box + it.label)
	}
	return strings.Join(lines, "\n")
}

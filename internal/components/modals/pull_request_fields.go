package modals

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	checkedBox   = "[x] "
	uncheckedBox = "[ ] "
	listJoin     = ", "
)

type multiChoiceField struct {
	name    string
	empty   string
	keys    keys.KeyMap
	cfg     widgets.PickerConfig
	options []widgets.Option
	chosen  map[string]bool
	picker  widgets.Picker
	width   int
	focused bool
}

func newMultiChoiceField(k keys.KeyMap, name, empty, noun, nouns string, options []widgets.Option) multiChoiceField {
	cfg := widgets.PickerConfig{Rows: styles.FormPickerRows, Search: true, Placeholder: "Type to filter, Enter to add or remove", Noun: noun, Nouns: nouns}
	return multiChoiceField{name: name, empty: empty, keys: k, cfg: cfg, options: options, chosen: map[string]bool{}, picker: widgets.NewPicker(k, options, cfg)}
}

func (f multiChoiceField) label() string   { return f.name }
func (f multiChoiceField) multiline() bool { return false }
func (f multiChoiceField) compact() bool   { return false }
func (f multiChoiceField) choice() bool    { return true }

func (f multiChoiceField) values() []string {
	var out []string
	for _, o := range f.options {
		if f.chosen[o.Value] {
			out = append(out, o.Value)
		}
	}
	return out
}

func (f multiChoiceField) value() string {
	return strings.Join(f.values(), listJoin)
}

func (f multiChoiceField) labels() []string {
	var out []string
	for _, o := range f.options {
		if f.chosen[o.Value] {
			out = append(out, o.Label)
		}
	}
	return out
}

func (f multiChoiceField) focus() (field, tea.Cmd) {
	f.focused = true
	cmd := f.picker.Focus()
	return f, cmd
}

func (f multiChoiceField) blur() field {
	f.focused = false
	f.picker.Blur()
	return f
}

func (f multiChoiceField) update(msg tea.Msg) (field, tea.Cmd) {
	var cmd tea.Cmd
	f.picker, cmd = f.picker.Update(msg)
	return f, cmd
}

func (f multiChoiceField) toggle() multiChoiceField {
	selected, ok := f.picker.Selected()
	if !ok {
		return f
	}
	chosen := make(map[string]bool, len(f.chosen)+1)
	for k, v := range f.chosen {
		chosen[k] = v
	}
	chosen[selected.Value] = !chosen[selected.Value]
	f.chosen = chosen
	options := make([]widgets.Option, len(f.options))
	for i, o := range f.options {
		o.Marked = chosen[o.Value]
		options[i] = o
	}
	query := f.picker.Query()
	f.picker = widgets.NewPicker(f.keys, options, f.cfg)
	f.picker.SetWidth(f.width)
	f.picker.SetQuery(query)
	f.picker.Select(selected.Value)
	if f.focused {
		f.picker.Focus()
	}
	return f
}

func (f multiChoiceField) resize(width int) field {
	f.width = width
	f.picker.SetWidth(width)
	return f
}

func (f multiChoiceField) view() string {
	if f.focused {
		return f.picker.View()
	}
	labels := f.labels()
	if len(labels) == 0 {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(inputPrompt + f.empty)
	}
	return lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(inputPrompt) +
		lipgloss.NewStyle().Foreground(styles.ColorLight).Render(strings.Join(labels, listJoin))
}

type toggleField struct {
	name    string
	text    string
	on      bool
	focused bool
}

func (f toggleField) label() string   { return f.name }
func (f toggleField) multiline() bool { return false }
func (f toggleField) compact() bool   { return false }
func (f toggleField) choice() bool    { return false }

func (f toggleField) value() string {
	if f.on {
		return "true"
	}
	return ""
}

func (f toggleField) focus() (field, tea.Cmd) {
	f.focused = true
	return f, nil
}

func (f toggleField) blur() field {
	f.focused = false
	return f
}

func (f toggleField) update(tea.Msg) (field, tea.Cmd) { return f, nil }
func (f toggleField) resize(int) field                { return f }

func (f toggleField) view() string {
	box := uncheckedBox
	if f.on {
		box = checkedBox
	}
	style := lipgloss.NewStyle().Foreground(styles.ColorLight)
	if f.focused {
		style = style.Foreground(styles.ColorPrimary).Bold(true)
	}
	return style.Render(box + f.text)
}

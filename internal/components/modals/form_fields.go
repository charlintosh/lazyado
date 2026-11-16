package modals

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/styles"
)

const inputPrompt = "› "

type field interface {
	label() string
	value() string
	focus() (field, tea.Cmd)
	blur() field
	update(tea.Msg) (field, tea.Cmd)
	resize(width int) field
	view() string
	multiline() bool
	compact() bool
	choice() bool
}

type textField struct {
	name  string
	input textinput.Model
	small bool
}

func newTextField(name, placeholder, value string, limit int, small bool) textField {
	in := textinput.New()
	in.Prompt = inputPrompt
	in.Placeholder = placeholder
	in.CharLimit = limit
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	st.Focused.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	st.Blurred.Prompt = lipgloss.NewStyle().Foreground(styles.ColorBorder)
	st.Blurred.Text = lipgloss.NewStyle().Foreground(styles.ColorLight)
	in.SetStyles(st)
	in.SetValue(value)
	return textField{name: name, input: in, small: small}
}

func (f textField) label() string   { return f.name }
func (f textField) value() string   { return strings.TrimSpace(f.input.Value()) }
func (f textField) view() string    { return f.input.View() }
func (f textField) multiline() bool { return false }
func (f textField) compact() bool   { return f.small }
func (f textField) choice() bool    { return false }

func (f textField) focus() (field, tea.Cmd) {
	cmd := f.input.Focus()
	return f, cmd
}

func (f textField) blur() field {
	f.input.Blur()
	return f
}

func (f textField) update(msg tea.Msg) (field, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

func (f textField) resize(width int) field {
	if f.small {
		width = min(width, styles.FormCompactFieldWidth)
	}
	f.input.SetWidth(max(width-lipgloss.Width(inputPrompt)-1, 1))
	return f
}

type areaField struct {
	name string
	area textarea.Model
}

func newAreaField(name, placeholder, value string, limit, height int) areaField {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.CharLimit = limit
	ta.ShowLineNumbers = false
	ta.SetHeight(height)
	ta.SetValue(value)
	ta.Blur()
	return areaField{name: name, area: ta}
}

func (f areaField) label() string   { return f.name }
func (f areaField) value() string   { return strings.TrimSpace(f.area.Value()) }
func (f areaField) view() string    { return f.area.View() }
func (f areaField) multiline() bool { return true }
func (f areaField) compact() bool   { return false }
func (f areaField) choice() bool    { return false }

func (f areaField) focus() (field, tea.Cmd) {
	cmd := f.area.Focus()
	return f, cmd
}

func (f areaField) blur() field {
	f.area.Blur()
	return f
}

func (f areaField) update(msg tea.Msg) (field, tea.Cmd) {
	var cmd tea.Cmd
	f.area, cmd = f.area.Update(msg)
	return f, cmd
}

func (f areaField) resize(width int) field {
	f.area.SetWidth(width)
	return f
}

type choiceField struct {
	name    string
	picker  widgets.Picker
	chosen  widgets.Option
	empty   string
	focused bool
}

func newChoiceField(k keys.KeyMap, name, empty, noun, nouns string, options []widgets.Option, initial string) choiceField {
	p := widgets.NewPicker(k, options, widgets.PickerConfig{
		Rows:        styles.FormPickerRows,
		Search:      true,
		Placeholder: "Type to filter...",
		Noun:        noun,
		Nouns:       nouns,
	})
	f := choiceField{name: name, empty: empty}
	for _, o := range options {
		if o.Value == initial {
			f.chosen = o
			p.Select(initial)
			break
		}
	}
	f.picker = p
	return f
}

func (f choiceField) label() string   { return f.name }
func (f choiceField) value() string   { return f.chosen.Value }
func (f choiceField) multiline() bool { return false }
func (f choiceField) compact() bool   { return false }
func (f choiceField) choice() bool    { return true }

func (f choiceField) focus() (field, tea.Cmd) {
	f.focused = true
	f.picker.SetQuery("")
	f.picker.Select(f.chosen.Value)
	if o, ok := f.picker.Selected(); ok {
		f.chosen = o
	}
	cmd := f.picker.Focus()
	return f, cmd
}

func (f choiceField) blur() field {
	f.focused = false
	f.picker.Blur()
	return f
}

func (f choiceField) update(msg tea.Msg) (field, tea.Cmd) {
	var cmd tea.Cmd
	f.picker, cmd = f.picker.Update(msg)
	if o, ok := f.picker.Selected(); ok {
		f.chosen = o
	}
	return f, cmd
}

func (f choiceField) resize(width int) field {
	f.picker.SetWidth(width)
	return f
}

func (f choiceField) view() string {
	if f.focused {
		return f.picker.View()
	}
	if f.chosen.Label == "" {
		return lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(inputPrompt + f.empty)
	}
	return lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(inputPrompt) +
		lipgloss.NewStyle().Foreground(styles.ColorLight).Render(f.chosen.Label+" "+f.chosen.Detail)
}

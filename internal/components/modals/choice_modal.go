package modals

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/styles"
)

type choiceSpec struct {
	title       string
	subtitle    string
	note        string
	action      string
	options     []widgets.Option
	initial     string
	search      bool
	placeholder string
	noun        string
	nouns       string
	width       int
	rows        int
	hints       []key.Binding
	choose      func(widgets.Option) tea.Msg
}

type ChoiceModal struct {
	env      Env
	spec     choiceSpec
	picker   widgets.Picker
	showNote bool
}

func newChoiceModal(env Env, spec choiceSpec) ChoiceModal {
	p := widgets.NewPicker(env.Keys, spec.options, widgets.PickerConfig{
		Rows:        spec.rows,
		Search:      spec.search,
		Placeholder: spec.placeholder,
		Noun:        spec.noun,
		Nouns:       spec.nouns,
	})
	m := ChoiceModal{env: env, spec: spec, picker: p}
	m.layout()
	if spec.initial != "" {
		m.picker.Select(spec.initial)
	}
	m.picker.Focus()
	return m
}

func (m ChoiceModal) Init() tea.Cmd {
	return m.picker.Init()
}

func (m ChoiceModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			if m.picker.Query() != "" {
				m.picker.SetQuery("")
				return m, nil
			}
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Select):
			if o, ok := m.picker.Selected(); ok {
				return m, send(m.spec.choose(o))
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	return m, cmd
}

const (
	choiceNoteHeight  = 2
	choiceMinimumRows = 2
)

func (m *ChoiceModal) layout() {
	width := widgets.DialogWidth(m.spec.width, m.env.Width)
	m.picker.SetWidth(widgets.DialogInnerWidth(width))
	rows := widgets.DialogBodyHeight(m.env.Height, m.spec.subtitle != "") - 1
	if m.spec.search {
		rows--
	}
	m.showNote = m.spec.note != "" && rows-choiceNoteHeight >= choiceMinimumRows
	if m.showNote {
		rows -= choiceNoteHeight
	}
	m.picker.SetRows(rows)
}

func (m ChoiceModal) View() string {
	width := widgets.DialogWidth(m.spec.width, m.env.Width)
	body := m.picker.View()
	if m.showNote {
		body = lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(m.spec.note) + "\n\n" + body
	}
	k := m.env.Keys
	action := m.spec.action
	if action == "" {
		action = "select"
	}
	hints := append([]key.Binding{widgets.Combine("move", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, action)}, m.spec.hints...)
	footer := widgets.Hints(widgets.DialogInnerWidth(width), append(hints, widgets.Relabel(k.Back, "cancel"))...)
	return widgets.Dialog{
		Title:    m.spec.title,
		Subtitle: m.spec.subtitle,
		Body:     body,
		Footer:   footer,
		Width:    width,
	}.View()
}

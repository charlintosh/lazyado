package modals

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	fieldValueLimit = 64 * 1024
	fieldAreaHeight = 10
	fieldHelpLines  = 3
	fieldPickerRows = 8
	fieldErrorLines = 4
)

type ReleaseFieldEditedMsg struct {
	Field models.TaskField
	Where string
	Value any
}

type AzureValuesMsg struct {
	Key    string
	Values []models.AzureValue
	Err    error
}

type RetryAzureListMsg struct {
	Request models.AzureListRequest
}

type BrowseReleaseDocMsg struct{}

type FieldChoices struct {
	Options   []models.TaskOption
	Azure     bool
	Request   models.AzureListRequest
	Values    []models.AzureValue
	Loaded    bool
	Variables []string
}

type fieldMode int

const (
	fieldModeText fieldMode = iota
	fieldModeArea
	fieldModePick
)

type azureState int

const (
	azureLoading azureState = iota
	azureLoaded
	azureFailed
)

type ReleaseFieldModal struct {
	env       Env
	field     models.TaskField
	where     string
	mode      fieldMode
	input     textinput.Model
	area      textarea.Model
	picker    widgets.Picker
	current   string
	azure     bool
	state     azureState
	azureErr  string
	request   models.AzureListRequest
	values    []models.AzureValue
	variables []string
	showVars  bool
	err       string
}

func NewReleaseFieldModal(env Env, field models.TaskField, where string, choices FieldChoices) ReleaseFieldModal {
	m := ReleaseFieldModal{env: env, field: field, where: where, current: field.EditText(), variables: choices.Variables}
	switch {
	case choices.Azure:
		m.mode, m.azure, m.request = fieldModePick, true, choices.Request
		if choices.Loaded {
			m.state, m.values = azureLoaded, choices.Values
		}
		m.rebuildPicker()
	case field.Kind == models.InputChoice || field.Kind == models.InputEndpoint || field.Kind == models.InputQueue || len(choices.Options) > 0:
		m.mode = fieldModePick
		options := choices.Options
		if len(options) == 0 {
			options = field.Options
		}
		m.setPicker(options)
	case field.Kind == models.InputMultiLine || field.Kind == models.InputLines:
		m.mode = fieldModeArea
		m.area = newAreaField("", "", m.current, fieldValueLimit, fieldAreaHeight).area
		m.area.Focus()
	default:
		m.input = newTextField("", "", m.current, fieldValueLimit, false).input
		if field.Secret {
			m.input.SetValue("")
			m.input.Placeholder = "Type a new value, or Esc to keep the current one"
			m.input.EchoMode = textinput.EchoPassword
		}
		m.input.CursorEnd()
		m.input.Focus()
	}
	m.layout()
	return m
}

func (m *ReleaseFieldModal) setPicker(options []models.TaskOption) {
	items := make([]widgets.Option, 0, len(options))
	current := m.current
	for _, o := range options {
		detail := o.Detail
		if detail == "" && o.Value != o.Label {
			detail = o.Value
		}
		items = append(items, widgets.Option{Label: o.Label, Value: o.Value, Detail: detail, Marked: strings.EqualFold(o.Value, m.current)})
		if strings.EqualFold(o.Value, m.current) {
			current = o.Value
		}
	}
	m.picker = widgets.NewPicker(m.env.Keys, items, widgets.PickerConfig{Rows: fieldPickerRows, Search: true, Placeholder: "Type to filter...", Noun: "option", Nouns: "options"})
	m.picker.Select(current)
	m.picker.Focus()
	m.layout()
}

func (m *ReleaseFieldModal) rebuildPicker() {
	if m.showVars {
		var options []models.TaskOption
		for _, name := range m.variables {
			options = append(options, models.TaskOption{Value: "$(" + name + ")", Label: "$(" + name + ")", Detail: "pipeline variable"})
		}
		m.setPicker(options)
		return
	}
	var options []models.TaskOption
	for _, v := range m.values {
		options = append(options, models.TaskOption{Value: v.Value, Label: v.Label})
	}
	m.setPicker(options)
}

func (m ReleaseFieldModal) Key() string { return m.request.Key() }

func (m ReleaseFieldModal) Init() tea.Cmd {
	switch m.mode {
	case fieldModePick:
		return m.picker.Init()
	case fieldModeArea:
		return textarea.Blink
	}
	return textinput.Blink
}

func (m ReleaseFieldModal) width() int { return widgets.DialogWidth(styles.ModalWidthPR, m.env.Width) }

func (m *ReleaseFieldModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	body := widgets.DialogBodyHeight(m.env.Height, true)
	switch m.mode {
	case fieldModePick:
		m.picker.SetWidth(inner)
		m.picker.SetRows(max(min(fieldPickerRows, body-fieldHelpLines-3), 2))
	case fieldModeArea:
		m.area.SetWidth(inner)
		m.area.SetHeight(max(min(fieldAreaHeight, body-fieldHelpLines-2), 2))
	default:
		m.input.SetWidth(max(inner-lipgloss.Width(inputPrompt)-1, 1))
	}
}

func (m ReleaseFieldModal) listShown() bool {
	return !m.azure || m.showVars || m.state == azureLoaded
}

func (m ReleaseFieldModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case AzureValuesMsg:
		if !m.azure || msg.Key != m.request.Key() {
			return m, nil
		}
		if msg.Err != nil {
			m.state, m.azureErr = azureFailed, msg.Err.Error()
			return m, nil
		}
		m.state, m.values, m.azureErr = azureLoaded, msg.Values, ""
		if !m.showVars {
			m.rebuildPicker()
		}
		return m, nil
	case tea.KeyPressMsg:
		k := m.env.Keys
		switch {
		case key.Matches(msg, k.Back):
			if m.mode == fieldModePick && m.listShown() && m.picker.Query() != "" {
				m.picker.SetQuery("")
				return m, nil
			}
			if m.showVars && m.state != azureLoaded {
				m.showVars = false
				return m, nil
			}
			return m, closeModal
		case m.azure && key.Matches(msg, k.NextPanel), m.azure && !m.listShown() && key.Matches(msg, k.UseVariable):
			if len(m.variables) == 0 {
				m.err = "This pipeline defines no variables."
				return m, nil
			}
			m.showVars, m.err = !m.showVars, ""
			m.rebuildPicker()
			return m, nil
		case m.azure && !m.listShown() && m.state == azureFailed && key.Matches(msg, k.RetryList):
			m.state, m.azureErr = azureLoading, ""
			return m, send(RetryAzureListMsg{Request: m.request})
		case m.azure && !m.listShown() && key.Matches(msg, k.OpenBrowser):
			return m, send(BrowseReleaseDocMsg{})
		case m.azure && !m.listShown():
			return m, nil
		case key.Matches(msg, k.Save), m.mode != fieldModeArea && key.Matches(msg, k.Select):
			return m.submit()
		}
	}
	var cmd tea.Cmd
	switch m.mode {
	case fieldModePick:
		if m.listShown() {
			m.picker, cmd = m.picker.Update(msg)
		}
	case fieldModeArea:
		m.area, cmd = m.area.Update(msg)
	default:
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m ReleaseFieldModal) submit() (Modal, tea.Cmd) {
	var text string
	switch m.mode {
	case fieldModePick:
		o, ok := m.picker.Selected()
		if !ok {
			m.err = "Choose one of the options."
			return m, nil
		}
		text = o.Value
	case fieldModeArea:
		text = m.area.Value()
	default:
		text = m.input.Value()
		if m.field.Secret && text == "" {
			return m, closeModal
		}
	}
	if m.field.Required && strings.TrimSpace(text) == "" {
		m.err = m.field.Label + " is required."
		return m, nil
	}
	value, err := models.FieldValue(m.field.Kind, m.field.Original, text)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	return m, send(ReleaseFieldEditedMsg{Field: m.field, Where: m.where, Value: value})
}

func (m ReleaseFieldModal) azureLines(inner int) []string {
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner)
	source := muted.Render("Listed by Azure for this task's service connection.")
	if m.showVars {
		source = muted.Render("Pipeline variables. The value is resolved when the stage deploys.")
	}
	lines := []string{source}
	switch {
	case m.showVars:
	case m.state == azureLoading:
		lines = append(lines, "", muted.Render("Asking Azure DevOps for the values this connection can see..."))
	case m.state == azureFailed:
		red := lipgloss.NewStyle().Foreground(styles.ColorError)
		why := strings.Split(lipgloss.NewStyle().Width(inner).Render(m.azureErr), "\n")
		if len(why) > fieldErrorLines {
			why = append(why[:fieldErrorLines-1], ansi.Truncate(why[fieldErrorLines-1], inner-1, "…"))
		}
		lines = append(lines, "", red.Bold(true).Width(inner).Render("Azure DevOps couldn't list "+strings.ToLower(m.field.Label)+"."),
			lipgloss.NewStyle().Foreground(styles.ColorStaleText).Render(strings.Join(why, "\n")), "",
			lipgloss.NewStyle().Foreground(styles.ColorLight).Width(inner).Render("Usually the connection's identity can't read this in Azure, so a deployment would fail the same way. A value Azure didn't list can't be typed here."))
	}
	return lines
}

func (m ReleaseFieldModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	var parts []string
	if help := strings.TrimSpace(m.field.Help); help != "" && !m.azure {
		lines := strings.Split(lipgloss.NewStyle().Width(inner).Render(help), "\n")
		if len(lines) > fieldHelpLines {
			lines = append(lines[:fieldHelpLines-1], ansi.Truncate(lines[fieldHelpLines-1], inner-1, "…"))
		}
		parts = append(parts, muted.Render(strings.Join(lines, "\n")), "")
	}
	if m.azure {
		parts = append(parts, m.azureLines(inner)...)
	}
	switch {
	case m.mode == fieldModePick && m.listShown() && m.picker.Len() == 0 && !m.azure:
		parts = append(parts, muted.Width(inner).Render("Nothing to choose from: Azure DevOps lists none you can use here. Values that aren't listed can't be typed."))
	case m.mode == fieldModePick && m.listShown():
		parts = append(parts, m.picker.View())
	case m.mode == fieldModeArea:
		parts = append(parts, m.area.View())
	case m.mode == fieldModeText:
		parts = append(parts, m.input.View())
	}
	if m.field.Changed() {
		was := m.field
		was.Value = m.field.Original
		parts = append(parts, muted.Render(ansi.Truncate("Saved value: "+was.Text(), inner, "…")))
	}
	if m.err != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err))
	}
	body := lipgloss.NewStyle().MaxHeight(widgets.DialogBodyHeight(m.env.Height, true)).Render(strings.Join(parts, "\n"))
	accent := styles.ColorPrimary
	if m.azure && m.state == azureFailed && !m.showVars {
		accent = styles.ColorError
	}
	return widgets.Dialog{
		Title:    "Change " + m.field.Label,
		Subtitle: m.where,
		Body:     body,
		Footer:   widgets.Hints(inner, m.hints()...),
		Width:    m.width(),
		Accent:   accent,
	}.View()
}

func (m ReleaseFieldModal) hints() []key.Binding {
	k := m.env.Keys
	cancel := widgets.Relabel(k.Back, "cancel")
	if m.azure && !m.listShown() {
		var hints []key.Binding
		if m.state == azureFailed {
			hints = append(hints, widgets.Relabel(k.RetryList, "retry"))
		}
		return append(hints, widgets.Relabel(k.UseVariable, "use a variable"), widgets.Relabel(k.OpenBrowser, "open in browser"), widgets.Relabel(k.Back, "keep saved value"))
	}
	apply := widgets.Relabel(k.Select, "apply")
	if m.mode == fieldModeArea {
		apply = widgets.Relabel(k.Save, "apply")
	}
	hints := []key.Binding{apply, cancel}
	if m.mode == fieldModePick {
		hints = append([]key.Binding{widgets.Combine("move", k.PickerUp, k.PickerDown)}, hints...)
	}
	if m.azure {
		label := "variables"
		if m.showVars {
			label = "Azure list"
		}
		hints = append(hints, widgets.Relabel(k.NextPanel, label))
	}
	return hints
}

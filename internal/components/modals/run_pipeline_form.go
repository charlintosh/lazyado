package modals

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	runFormTitle      = "Run pipeline"
	runValueLimit     = 400
	loadingPipeline   = "Loading branches and tags"
	loadingParameters = "Reading parameters from the YAML"
	yesLabel          = "Yes"
	secretPlaceholder = "secret: leave empty to keep it"
)

type runFieldKind int

const (
	runRef runFieldKind = iota
	runCommit
	runParam
	runStages
	runVariable
	runDiagnostics
)

var runSectionTitles = map[runFieldKind]string{
	runParam:       "Parameters",
	runVariable:    "Variables",
	runDiagnostics: "Advanced",
}

type runEntry struct {
	kind     runFieldKind
	param    models.PipelineParameter
	variable models.PipelineVariable
	field    field
}

type RunDefinitionRequestMsg struct {
	PipelineID int
}

type RunDefinitionMsg struct {
	PipelineID int
	Definition models.PipelineDefinition
	Refs       []models.GitRef
	Err        error
}

type RunSpecRequestMsg struct {
	PipelineID   int
	RepositoryID string
	Path         string
	Ref          string
}

type RunSpecMsg struct {
	PipelineID int
	Ref        string
	Spec       models.PipelineSpec
	Err        error
}

type RunPipelineRequestMsg struct {
	Definition models.PipelineDefinition
	Options    models.RunOptions
}

type RunPipelineForm struct {
	env        Env
	pipeline   models.Pipeline
	preset     string
	def        models.PipelineDefinition
	defLoaded  bool
	refs       []models.GitRef
	spec       models.PipelineSpec
	specRef    string
	specErr    string
	loadingDef bool
	loadingRef string
	entries    []runEntry
	focus      int
	err        string
	body       viewport.Model
	spinner    spinner.Model
}

func NewRunPipelineForm(env Env, pipeline models.Pipeline, ref string) RunPipelineForm {
	m := RunPipelineForm{
		env:        env,
		pipeline:   pipeline,
		preset:     ref,
		loadingDef: true,
		body:       viewport.New(),
		spinner:    spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
	}
	m.body.KeyMap = viewport.KeyMap{}
	m.entries = []runEntry{{kind: runRef, field: newTextField("Branch or tag", "main", ref, runValueLimit, false)}}
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	return m
}

func (m RunPipelineForm) PipelineID() int { return m.pipeline.ID }

func (m RunPipelineForm) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, send(RunDefinitionRequestMsg{PipelineID: m.pipeline.ID}), m.spinner.Tick)
}

func (m RunPipelineForm) busy() bool { return m.loadingDef || m.loadingRef != "" }

func (m RunPipelineForm) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case spinner.TickMsg:
		if !m.busy() {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.refresh()
		return m, cmd
	case RunDefinitionMsg:
		return m.setDefinition(msg)
	case RunSpecMsg:
		return m.setSpec(msg), nil
	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		m.body, cmd = m.body.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		k := m.env.Keys
		switch {
		case key.Matches(msg, k.Back):
			return m, closeModal
		case key.Matches(msg, k.Save):
			return m.submit()
		case key.Matches(msg, k.NextPanel):
			return m.moveFocus(1)
		case key.Matches(msg, k.PrevPanel):
			return m.moveFocus(-1)
		case key.Matches(msg, k.Select):
			return m.enter()
		}
	}
	f, cmd := m.current().update(msg)
	m.entries = slices.Clone(m.entries)
	m.entries[m.focus].field = f
	m.refresh()
	return m, cmd
}

func (m RunPipelineForm) current() field { return m.entries[m.focus].field }

func (m RunPipelineForm) enter() (Modal, tea.Cmd) {
	m.entries = slices.Clone(m.entries)
	switch f := m.current().(type) {
	case checklistField:
		m.entries[m.focus].field = f.toggle()
	case toggleField:
		f.on = !f.on
		m.entries[m.focus].field = f
	default:
		return m.moveFocus(1)
	}
	m.refresh()
	return m, nil
}

func (m RunPipelineForm) moveFocus(delta int) (Modal, tea.Cmd) {
	m.entries = slices.Clone(m.entries)
	m.entries[m.focus].field = m.current().blur()
	m.focus = (m.focus + delta + len(m.entries)) % len(m.entries)
	f, cmd := m.current().focus()
	m.entries[m.focus].field = f
	m, follow := m.syncSpec()
	m.refresh()
	return m, tea.Batch(cmd, follow)
}

func (m RunPipelineForm) ref() string {
	return models.BranchRef(strings.TrimSpace(m.entries[0].field.value()))
}

func (m RunPipelineForm) readsSpec() bool {
	return m.defLoaded && m.def.YAML() && m.def.AzureRepo()
}

func (m RunPipelineForm) syncSpec() (RunPipelineForm, tea.Cmd) {
	ref := m.ref()
	if !m.readsSpec() || ref == "" || ref == m.specRef || ref == m.loadingRef {
		return m, nil
	}
	m.loadingRef, m.specErr = ref, ""
	req := RunSpecRequestMsg{PipelineID: m.pipeline.ID, RepositoryID: m.def.RepositoryID, Path: m.def.YAMLPath, Ref: ref}
	return m, tea.Batch(send(req), m.spinner.Tick)
}

func (m RunPipelineForm) setDefinition(msg RunDefinitionMsg) (Modal, tea.Cmd) {
	if msg.PipelineID != m.pipeline.ID {
		return m, nil
	}
	m.loadingDef = false
	if msg.Err != nil {
		m.specErr = msg.Err.Error()
		m.refresh()
		return m, nil
	}
	m.def, m.defLoaded, m.refs = msg.Definition, true, sortRefs(msg.Refs, msg.Definition.DefaultBranch)
	initial := m.preset
	if initial == "" {
		initial = msg.Definition.DefaultBranch
	}
	m.entries = []runEntry{{kind: runRef, field: m.refField(initial)}, {kind: runCommit, field: newTextField("Commit (optional)", "latest on the branch, or a commit ID", "", runValueLimit, false)}}
	for _, v := range m.def.Variables {
		value, placeholder := v.Value, ""
		if v.Secret {
			value, placeholder = "", secretPlaceholder
		}
		m.entries = append(m.entries, runEntry{kind: runVariable, variable: v, field: newTextField(v.Name, placeholder, value, runValueLimit, false)})
	}
	m.entries = append(m.entries, runEntry{kind: runDiagnostics, field: toggleField{name: "System diagnostics", text: "Enable system diagnostics"}})
	m.focus = 0
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	m, cmd := m.syncSpec()
	m.refresh()
	return m, cmd
}

func sortRefs(refs []models.GitRef, defaultBranch string) []models.GitRef {
	out := slices.Clone(refs)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.Name == defaultBranch || b.Name == defaultBranch:
			return a.Name == defaultBranch
		case a.Tag() != b.Tag():
			return !a.Tag()
		case a.Tag():
			return a.Name > b.Name
		}
		return a.Name < b.Name
	})
	return out
}

func (m RunPipelineForm) refField(initial string) field {
	if len(m.refs) == 0 {
		return newTextField("Branch or tag", "main or refs/tags/v1.0", models.RefLabel(initial), runValueLimit, false)
	}
	options := make([]widgets.Option, len(m.refs))
	for i, r := range m.refs {
		detail := r.ShortCommit()
		if r.Tag() {
			detail = "tag " + detail
		}
		if r.Name == m.def.DefaultBranch {
			detail = "default " + detail
		}
		options[i] = widgets.Option{Label: r.Label(), Value: r.Name, Detail: detail}
	}
	return newChoiceField(m.env.Keys, "Branch or tag", "Choose a branch or tag", "ref", "branches and tags", options, initial).resize(widgets.DialogInnerWidth(m.width()))
}

func (m RunPipelineForm) setSpec(msg RunSpecMsg) RunPipelineForm {
	if msg.PipelineID != m.pipeline.ID || msg.Ref != m.loadingRef {
		return m
	}
	m.loadingRef = ""
	if msg.Err != nil {
		m.specErr = msg.Err.Error()
		m.refresh()
		return m
	}
	typed := map[string]string{}
	for _, e := range m.entries {
		if e.kind == runParam && e.field.value() != paramValue(e.param.Default, e.param) {
			typed[e.param.Name] = e.field.value()
		}
	}
	m.spec, m.specRef, m.specErr, m.err = msg.Spec, msg.Ref, "", ""
	focused := m.entries[m.focus]
	var kept []runEntry
	for _, e := range m.entries {
		if e.kind != runParam && e.kind != runStages {
			kept = append(kept, e)
		}
	}
	var added []runEntry
	for _, p := range m.spec.Parameters {
		value, ok := typed[p.Name]
		if !ok {
			value = paramValue(p.Default, p)
		}
		added = append(added, runEntry{kind: runParam, param: p, field: m.paramField(p, value)})
	}
	if len(m.spec.Stages) > 0 {
		items := make([]checkItem, len(m.spec.Stages))
		for i, s := range m.spec.Stages {
			items[i] = checkItem{label: s.Label(), value: s.Name, on: true}
		}
		added = append(added, runEntry{kind: runStages, field: newChecklistField(m.env.Keys, "Stages to run", items)})
	}
	m.entries = slices.Concat(kept[:2], added, kept[2:])
	m.focus = 0
	for i, e := range m.entries {
		if e.kind == focused.kind && e.kind != runParam && e.kind != runStages && e.variable.Name == focused.variable.Name {
			m.focus = i
		}
	}
	for i := range m.entries {
		m.entries[i].field = m.entries[i].field.blur()
	}
	m.entries[m.focus].field, _ = m.entries[m.focus].field.focus()
	m.layout()
	return m
}

func paramValue(raw string, p models.PipelineParameter) string {
	if p.Type == models.ParamBoolean {
		if strings.EqualFold(raw, "true") {
			return "true"
		}
		return ""
	}
	return raw
}

func (m RunPipelineForm) paramField(p models.PipelineParameter, value string) field {
	switch {
	case p.Type == models.ParamBoolean:
		return toggleField{name: p.Label(), text: yesLabel, on: value == "true"}
	case len(p.Values) > 0:
		options := make([]widgets.Option, len(p.Values))
		for i, v := range p.Values {
			options[i] = widgets.Option{Label: v, Value: v}
		}
		return newChoiceField(m.env.Keys, p.Label(), "Choose a value", "value", "values", options, value).resize(widgets.DialogInnerWidth(m.width()))
	}
	return newTextField(p.Label(), p.Type, value, runValueLimit, p.Type == models.ParamNumber).resize(widgets.DialogInnerWidth(m.width()))
}

func (m RunPipelineForm) width() int {
	return widgets.DialogWidth(styles.ModalWidthXXL, m.env.Width)
}

func (m *RunPipelineForm) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.entries = slices.Clone(m.entries)
	for i, e := range m.entries {
		m.entries[i].field = e.field.resize(inner)
	}
	m.body.SetWidth(inner)
	m.refresh()
}

func (m *RunPipelineForm) refresh() {
	content, spans := m.renderFields()
	budget := widgets.DialogBodyHeight(m.env.Height, true)
	if m.err != "" {
		budget -= 2
	}
	m.body.SetHeight(max(min(lipgloss.Height(content), budget), 1))
	m.body.SetContent(content)
	span := spans[min(m.focus, len(spans)-1)]
	widgets.Reveal(&m.body, span.top, span.bottom)
}

func (m RunPipelineForm) renderFields() (string, []fieldSpan) {
	inner := widgets.DialogInnerWidth(m.width())
	spans := make([]fieldSpan, len(m.entries))
	blocks := make([]string, 0, len(m.entries))
	line := 0
	seen := map[runFieldKind]bool{}
	for i, e := range m.entries {
		if title, ok := runSectionTitles[e.kind]; ok && !seen[e.kind] {
			seen[e.kind] = true
			heading := sectionHeading(title, inner)
			blocks = append(blocks, heading)
			line += lipgloss.Height(heading) + 1
		}
		label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == m.focus {
			label = label.Foreground(styles.ColorPrimary).Bold(true)
		}
		block := lipgloss.JoinVertical(lipgloss.Left, label.Render(e.field.label()), e.field.view())
		if e.kind == runRef {
			if status := m.status(inner); status != "" {
				block += "\n" + status
			}
		}
		h := lipgloss.Height(block)
		spans[i] = fieldSpan{top: line, bottom: line + h - 1}
		blocks = append(blocks, block)
		line += h + 1
	}
	return strings.Join(blocks, "\n\n"), spans
}

func sectionHeading(title string, width int) string {
	text := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title)
	rule := lipgloss.NewStyle().Foreground(styles.ColorChip).Render(strings.Repeat("─", max(width-lipgloss.Width(title)-1, 0)))
	return text + " " + rule
}

func (m RunPipelineForm) status(width int) string {
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(width)
	bad := lipgloss.NewStyle().Foreground(styles.ColorError).Width(width)
	switch {
	case m.loadingDef:
		return m.spinner.View() + muted.Render(" "+loadingPipeline)
	case m.loadingRef != "":
		return m.spinner.View() + muted.Render(" "+loadingParameters)
	case m.specErr != "":
		return bad.Render(m.specErr)
	case !m.defLoaded:
		return ""
	case !m.def.YAML():
		return muted.Render("Classic pipeline: choose the branch and the variables to override.")
	case !m.def.AzureRepo():
		return muted.Render(fmt.Sprintf("The YAML lives in a %s repository, so lazyado can't read its parameters. Defaults are used.", m.def.RepositoryType))
	case m.specRef == "":
		return ""
	}
	note := fmt.Sprintf("Parameters and stages from %s at %s.", strings.TrimPrefix(m.def.YAMLPath, "/"), models.RefLabel(m.specRef))
	if len(m.spec.Fixed) > 0 {
		note += " Using defaults for " + strings.Join(m.spec.Fixed, ", ") + "."
	}
	return muted.Render(note)
}

func (m RunPipelineForm) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	body := m.body.View()
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err)
	}
	return widgets.Dialog{
		Title:    runFormTitle,
		Subtitle: m.pipeline.Name,
		Body:     body,
		Footer:   widgets.Hints(inner, m.hints()...),
		Width:    m.width(),
	}.View()
}

func (m RunPipelineForm) hints() []key.Binding {
	k := m.env.Keys
	bindings := []key.Binding{widgets.Relabel(k.Save, "run"), widgets.Relabel(k.NextPanel, "next"), widgets.Relabel(k.PrevPanel, "prev"), widgets.Relabel(k.Back, "cancel")}
	switch m.current().(type) {
	case checklistField:
		bindings = append([]key.Binding{widgets.Combine("stage", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "run/skip")}, bindings...)
	case toggleField:
		bindings = append([]key.Binding{widgets.Relabel(k.Select, "toggle")}, bindings...)
	case choiceField:
		bindings = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown)}, bindings...)
	}
	return bindings
}

func (m RunPipelineForm) submit() (Modal, tea.Cmd) {
	if next, cmd := m.syncSpec(); cmd != nil {
		next.err = "Reading the parameters for " + models.RefLabel(next.loadingRef) + ". Check them, then run again."
		next.refresh()
		return next, cmd
	}
	msg, err := m.request()
	if err != nil {
		m.err = err.Error()
		m.refresh()
		return m, nil
	}
	m.err = ""
	return m, send(msg)
}

func (m RunPipelineForm) request() (tea.Msg, error) {
	switch {
	case m.loadingDef:
		return nil, errors.New("Still loading the pipeline, try again in a moment")
	case !m.defLoaded:
		return nil, errors.New("Couldn't load this pipeline: " + m.specErr)
	case m.ref() == "":
		return nil, errors.New("Choose a branch or tag")
	case m.loadingRef != "":
		return nil, errors.New("Still reading the parameters for this branch")
	}
	opts := models.RunOptions{Ref: m.ref(), Parameters: map[string]string{}, Variables: map[string]string{}}
	for _, e := range m.entries {
		value := e.field.value()
		switch e.kind {
		case runCommit:
			opts.Commit = value
		case runParam:
			if e.param.Type == models.ParamBoolean {
				value = strconv.FormatBool(value != "")
			}
			if e.param.Type == models.ParamNumber && value != "" {
				if _, err := strconv.ParseFloat(value, 64); err != nil {
					return nil, fmt.Errorf("%s must be a number", e.param.Label())
				}
			}
			if value != "" || e.param.Default != "" {
				opts.Parameters[e.param.Name] = value
			}
		case runStages:
			f := e.field.(checklistField)
			if len(f.values(true)) == 0 {
				return nil, errors.New("Choose at least one stage to run")
			}
			opts.StagesToSkip = f.values(false)
		case runVariable:
			keepsSecret := e.variable.Secret && value == ""
			if value != e.variable.Value && !keepsSecret {
				opts.Variables[e.variable.Name] = value
			}
		case runDiagnostics:
			opts.Diagnostics = value != ""
		}
	}
	return RunPipelineRequestMsg{Definition: m.def, Options: opts}, nil
}

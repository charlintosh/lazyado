package modals

import (
	"errors"
	"fmt"
	"slices"
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
	createReleaseTitle = "Create a new release"
	anyBranch          = "Any branch"
	loadingDefinition  = "Loading the release pipeline"
	loadingVersions    = "Loading builds"
	releaseDescLimit   = 2000
)

type relField int

const (
	relBranch relField = iota
	relVersion
	relStages
	relVariable
	relDescription
)

var releaseSections = map[relField]string{
	relBranch:   "Artifacts",
	relStages:   "Stages",
	relVariable: "Variables",
}

type relEntry struct {
	kind     relField
	alias    string
	variable models.ReleaseVariable
	field    field
}

type ReleaseDefinitionRequestMsg struct {
	DefinitionID int
}

type ReleaseDefinitionMsg struct {
	DefinitionID int
	Spec         models.ReleaseDefinitionSpec
	Builds       map[string][]models.PipelineRun
	Refs         map[string][]models.GitRef
	Err          error
}

type ArtifactVersionsRequestMsg struct {
	DefinitionID      int
	Alias             string
	BuildDefinitionID int
	Ref               string
}

type ArtifactVersionsMsg struct {
	DefinitionID int
	Alias        string
	Ref          string
	Builds       []models.PipelineRun
	Err          error
}

type CreateReleaseRequestMsg struct {
	Release models.NewRelease
}

type artifactState struct {
	spec     models.ReleaseArtifact
	builds   []models.PipelineRun
	branches []string
	refs     []models.GitRef
	ref      string
	loading  string
}

type CreateReleaseForm struct {
	env       Env
	defID     int
	defName   string
	spec      models.ReleaseDefinitionSpec
	loaded    bool
	loading   bool
	loadErr   string
	artifacts map[string]*artifactState
	entries   []relEntry
	focus     int
	err       string
	body      viewport.Model
	spinner   spinner.Model
}

func NewCreateReleaseForm(env Env, definitionID int, name string) CreateReleaseForm {
	m := CreateReleaseForm{
		env: env, defID: definitionID, defName: name, loading: true, artifacts: map[string]*artifactState{},
		body:    viewport.New(),
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
	}
	m.body.KeyMap = viewport.KeyMap{}
	m.entries = []relEntry{{kind: relDescription, field: newAreaField("Description", "What's in this release?", "", releaseDescLimit, styles.TextareaHeightSM)}}
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	return m
}

func (m CreateReleaseForm) DefinitionID() int { return m.defID }

func (m CreateReleaseForm) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, send(ReleaseDefinitionRequestMsg{DefinitionID: m.defID}), m.spinner.Tick)
}

func (m CreateReleaseForm) busy() bool {
	if m.loading {
		return true
	}
	for _, a := range m.artifacts {
		if a.loading != "" {
			return true
		}
	}
	return false
}

func (m CreateReleaseForm) Update(msg tea.Msg) (Modal, tea.Cmd) {
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
	case ReleaseDefinitionMsg:
		return m.setDefinition(msg), nil
	case ArtifactVersionsMsg:
		return m.setVersions(msg), nil
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
			if next, cmd, ok := m.enter(); ok {
				return next, cmd
			}
		}
	}
	f, cmd := m.entries[m.focus].field.update(msg)
	m.entries = slices.Clone(m.entries)
	m.entries[m.focus].field = f
	m.refresh()
	return m, cmd
}

func (m CreateReleaseForm) enter() (Modal, tea.Cmd, bool) {
	m.entries = slices.Clone(m.entries)
	switch f := m.entries[m.focus].field.(type) {
	case checklistField:
		m.entries[m.focus].field = f.toggle()
	case areaField:
		return m, nil, false
	default:
		next, cmd := m.moveFocus(1)
		return next, cmd, true
	}
	m.refresh()
	return m, nil, true
}

func (m CreateReleaseForm) moveFocus(delta int) (Modal, tea.Cmd) {
	m.entries = slices.Clone(m.entries)
	m.entries[m.focus].field = m.entries[m.focus].field.blur()
	m.focus = (m.focus + delta + len(m.entries)) % len(m.entries)
	f, cmd := m.entries[m.focus].field.focus()
	m.entries[m.focus].field = f
	m, follow := m.syncVersions()
	m.refresh()
	return m, tea.Batch(cmd, follow)
}

func (m CreateReleaseForm) syncVersions() (CreateReleaseForm, tea.Cmd) {
	var cmds []tea.Cmd
	for _, e := range m.entries {
		a := m.artifacts[e.alias]
		if e.kind != relBranch || a == nil || a.spec.Type != models.ArtifactBuild {
			continue
		}
		ref := e.field.value()
		if ref == a.ref || ref == a.loading {
			continue
		}
		a.loading = ref
		if ref == "" {
			a.loading = anyBranch
		}
		cmds = append(cmds, send(ArtifactVersionsRequestMsg{DefinitionID: m.defID, Alias: e.alias, BuildDefinitionID: a.spec.DefinitionID, Ref: ref}))
	}
	if len(cmds) > 0 {
		cmds = append(cmds, m.spinner.Tick)
	}
	return m, tea.Batch(cmds...)
}

func (m CreateReleaseForm) setDefinition(msg ReleaseDefinitionMsg) CreateReleaseForm {
	if msg.DefinitionID != m.defID {
		return m
	}
	m.loading = false
	if msg.Err != nil {
		m.loadErr = msg.Err.Error()
		m.refresh()
		return m
	}
	m.spec, m.loaded = msg.Spec, true
	m.artifacts = map[string]*artifactState{}
	var entries []relEntry
	for _, art := range msg.Spec.Artifacts {
		a := &artifactState{spec: art}
		m.artifacts[art.Alias] = a
		switch art.Type {
		case models.ArtifactBuild:
			a.builds = msg.Builds[art.Alias]
			a.branches = branchesOf(a.builds)
			entries = append(entries,
				relEntry{kind: relBranch, alias: art.Alias, field: m.branchField(art.Alias, a.branches, "")},
				relEntry{kind: relVersion, alias: art.Alias, field: m.versionField(art.Alias, a.builds)})
		case models.ArtifactGit:
			a.refs = msg.Refs[art.Alias]
			entries = append(entries, relEntry{kind: relBranch, alias: art.Alias, field: m.gitBranchField(art.Alias, a.refs)})
		}
	}
	var stages []checkItem
	for _, s := range msg.Spec.Stages {
		if s.Automated {
			stages = append(stages, checkItem{label: s.Name + " (" + s.Trigger + ")", value: s.Name, on: true})
		}
	}
	if len(stages) > 0 {
		entries = append(entries, relEntry{kind: relStages, field: newChecklistField(m.env.Keys, "Deploy automatically", stages)})
	}
	for _, v := range msg.Spec.Variables {
		value, placeholder := v.Value, ""
		if v.Secret {
			value, placeholder = "", secretPlaceholder
		}
		entries = append(entries, relEntry{kind: relVariable, variable: v, field: newTextField(v.Name, placeholder, value, runValueLimit, false)})
	}
	m.entries = append(entries, m.entries[len(m.entries)-1])
	for i := range m.entries {
		m.entries[i].field = m.entries[i].field.blur()
	}
	m.focus = 0
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	return m
}

func branchesOf(builds []models.PipelineRun) []string {
	var out []string
	for _, b := range builds {
		if b.Ref != "" && !slices.Contains(out, b.Ref) {
			out = append(out, b.Ref)
		}
	}
	slices.Sort(out)
	return out
}

func (m CreateReleaseForm) branchField(alias string, branches []string, initial string) field {
	options := []widgets.Option{{Label: anyBranch, Value: "", Detail: "latest build"}}
	for _, b := range branches {
		options = append(options, widgets.Option{Label: models.RefLabel(b), Value: b})
	}
	return newChoiceField(m.env.Keys, alias+" · branch", anyBranch, "branch", "branches", options, initial).resize(widgets.DialogInnerWidth(m.width()))
}

func (m CreateReleaseForm) versionField(alias string, builds []models.PipelineRun) field {
	options := make([]widgets.Option, len(builds))
	for i, b := range builds {
		detail := b.State().Label() + " · " + b.Branch()
		if b.ShortCommit() != "" {
			detail += " · " + b.ShortCommit()
		}
		options[i] = widgets.Option{Label: b.Number, Value: strconv.Itoa(b.ID), Detail: detail}
	}
	initial := ""
	if len(builds) > 0 {
		initial = strconv.Itoa(builds[0].ID)
	}
	return newChoiceField(m.env.Keys, alias+" · version", "No releasable builds", "build", "builds", options, initial).resize(widgets.DialogInnerWidth(m.width()))
}

func (m CreateReleaseForm) gitBranchField(alias string, refs []models.GitRef) field {
	var options []widgets.Option
	for _, r := range refs {
		if !r.Tag() {
			options = append(options, widgets.Option{Label: r.Label(), Value: r.Name, Detail: r.ShortCommit()})
		}
	}
	initial := ""
	if len(options) > 0 {
		initial = options[0].Value
		for _, o := range options {
			if o.Label == "main" || o.Label == "master" {
				initial = o.Value
			}
		}
	}
	return newChoiceField(m.env.Keys, alias+" · branch", "Choose a branch", "branch", "branches", options, initial).resize(widgets.DialogInnerWidth(m.width()))
}

func (m CreateReleaseForm) setVersions(msg ArtifactVersionsMsg) CreateReleaseForm {
	a := m.artifacts[msg.Alias]
	if msg.DefinitionID != m.defID || a == nil {
		return m
	}
	want := msg.Ref
	if want == "" {
		want = anyBranch
	}
	if a.loading != want {
		return m
	}
	a.loading, a.ref = "", msg.Ref
	if msg.Err != nil {
		m.err = msg.Err.Error()
		m.refresh()
		return m
	}
	a.builds = msg.Builds
	m.entries = slices.Clone(m.entries)
	for i, e := range m.entries {
		if e.kind == relVersion && e.alias == msg.Alias {
			f := m.versionField(msg.Alias, msg.Builds)
			if i == m.focus {
				f, _ = f.focus()
			}
			m.entries[i].field = f
		}
	}
	m.refresh()
	return m
}

func (m CreateReleaseForm) width() int { return widgets.DialogWidth(styles.ModalWidthXXL, m.env.Width) }

func (m *CreateReleaseForm) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.entries = slices.Clone(m.entries)
	for i, e := range m.entries {
		m.entries[i].field = e.field.resize(inner)
	}
	m.body.SetWidth(inner)
	m.refresh()
}

func (m *CreateReleaseForm) refresh() {
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

func (m CreateReleaseForm) renderFields() (string, []fieldSpan) {
	inner := widgets.DialogInnerWidth(m.width())
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner)
	var blocks []string
	line := 0
	add := func(block string) {
		blocks = append(blocks, block)
		line += lipgloss.Height(block) + 1
	}
	switch {
	case m.loading:
		add(m.spinner.View() + muted.Render(" "+loadingDefinition))
	case m.loadErr != "":
		add(lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.loadErr))
	}
	spans := make([]fieldSpan, len(m.entries))
	seen := map[relField]bool{}
	for i, e := range m.entries {
		if title, ok := releaseSections[e.kind]; ok && !seen[e.kind] {
			seen[e.kind] = true
			add(sectionHeading(title, inner))
			if e.kind == relBranch {
				if extra := m.otherArtifacts(); extra != "" {
					add(muted.Render(extra))
				}
			}
		}
		label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == m.focus {
			label = label.Foreground(styles.ColorPrimary).Bold(true)
		}
		block := lipgloss.JoinVertical(lipgloss.Left, label.Render(e.field.label()), e.field.view())
		if e.kind == relVersion {
			if a := m.artifacts[e.alias]; a != nil && a.loading != "" {
				block += "\n" + m.spinner.View() + muted.Render(" "+loadingVersions)
			}
		}
		if e.kind == relStages {
			if manual := m.manualStages(); manual != "" {
				block += "\n" + muted.Render(manual)
			}
		}
		spans[i] = fieldSpan{top: line, bottom: line + lipgloss.Height(block) - 1}
		add(block)
	}
	return strings.Join(blocks, "\n\n"), spans
}

func (m CreateReleaseForm) otherArtifacts() string {
	var names []string
	for _, a := range m.spec.Artifacts {
		if a.Type != models.ArtifactBuild && a.Type != models.ArtifactGit {
			names = append(names, a.Alias+" ("+a.Type+")")
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Using the default version for " + strings.Join(names, ", ") + "."
}

func (m CreateReleaseForm) manualStages() string {
	var names []string
	for _, s := range m.spec.Stages {
		if !s.Automated {
			names = append(names, s.Name)
		}
	}
	if len(names) == 0 {
		return "Untick a stage to make it wait for a manual deploy."
	}
	return "Untick a stage to make it wait for a manual deploy. Always manual: " + strings.Join(names, ", ") + "."
}

func (m CreateReleaseForm) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	body := m.body.View()
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err)
	}
	k := m.env.Keys
	hints := []key.Binding{widgets.Relabel(k.Save, "create"), widgets.Relabel(k.NextPanel, "next"), widgets.Relabel(k.PrevPanel, "prev"), widgets.Relabel(k.Back, "cancel")}
	switch m.entries[m.focus].field.(type) {
	case checklistField:
		hints = append([]key.Binding{widgets.Combine("stage", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "auto/manual")}, hints...)
	case choiceField:
		hints = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown)}, hints...)
	}
	return widgets.Dialog{
		Title:    createReleaseTitle,
		Subtitle: m.defName,
		Body:     body,
		Footer:   widgets.Hints(inner, hints...),
		Width:    m.width(),
	}.View()
}

func (m CreateReleaseForm) submit() (Modal, tea.Cmd) {
	if next, cmd := m.syncVersions(); cmd != nil {
		next.err = "Loading the builds for the branch you picked. Check the version, then create again."
		next.refresh()
		return next, cmd
	}
	rel, err := m.request()
	if err != nil {
		m.err = err.Error()
		m.refresh()
		return m, nil
	}
	m.err = ""
	return m, send(CreateReleaseRequestMsg{Release: rel})
}

func (m CreateReleaseForm) request() (models.NewRelease, error) {
	switch {
	case m.loading:
		return models.NewRelease{}, errors.New("Still loading the release pipeline")
	case !m.loaded:
		return models.NewRelease{}, fmt.Errorf("Couldn't load the release pipeline: %s", m.loadErr)
	}
	rel := models.NewRelease{DefinitionID: m.defID, Variables: map[string]string{}}
	for _, e := range m.entries {
		a := m.artifacts[e.alias]
		value := e.field.value()
		switch e.kind {
		case relVersion:
			build, ok := findBuild(a.builds, value)
			if !ok {
				return rel, errNoVersion
			}
			rel.Artifacts = append(rel.Artifacts, models.ArtifactChoice{Alias: e.alias, VersionID: strconv.Itoa(build.ID), Version: build.Number, Branch: build.Ref})
		case relBranch:
			if a.spec.Type != models.ArtifactGit {
				continue
			}
			ref, ok := findRef(a.refs, value)
			if !ok {
				return rel, fmt.Errorf("Choose a branch for %s", e.alias)
			}
			rel.Artifacts = append(rel.Artifacts, models.ArtifactChoice{Alias: e.alias, VersionID: ref.Commit, Version: ref.ShortCommit(), Branch: ref.Name})
		case relStages:
			rel.ManualStages = e.field.(checklistField).values(false)
		case relVariable:
			keepsSecret := e.variable.Secret && value == ""
			if value != e.variable.Value && !keepsSecret {
				rel.Variables[e.variable.Name] = value
			}
		case relDescription:
			rel.Description = value
		}
	}
	return rel, nil
}

func findBuild(builds []models.PipelineRun, id string) (models.PipelineRun, bool) {
	for _, b := range builds {
		if strconv.Itoa(b.ID) == id {
			return b, true
		}
	}
	return models.PipelineRun{}, false
}

func findRef(refs []models.GitRef, name string) (models.GitRef, bool) {
	for _, r := range refs {
		if r.Name == name {
			return r, true
		}
	}
	return models.GitRef{}, false
}

package modals

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const refNameLimit = 250

var commitIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{4,40}$`)

type CreateBranchRequestMsg struct {
	Name string
	Base models.GitVersion
}

type CreateTagRequestMsg struct {
	Name    string
	Base    models.GitVersion
	Message string
}

type DeleteBranchRequestMsg struct {
	Branch models.Branch
}

type RefForm struct {
	env      Env
	title    string
	subtitle string
	note     string
	fields   []field
	focus    int
	err      string
	body     viewport.Model
	build    func(values []string) (tea.Msg, error)
}

func newRefForm(env Env, title, subtitle, note string, fields []field, build func([]string) (tea.Msg, error)) RefForm {
	m := RefForm{env: env, title: title, subtitle: subtitle, note: note, fields: fields, build: build, body: viewport.New()}
	m.body.KeyMap = viewport.KeyMap{}
	m.fields[0], _ = m.fields[0].focus()
	m.layout()
	return m
}

func versionOptions(branches, tags []string, defaultBranch string, current models.GitVersion) []widgets.Option {
	options := make([]widgets.Option, 0, len(branches)+len(tags))
	for _, b := range branches {
		o := widgets.Option{Label: b, Value: branchValuePrefix + b, Marked: current.Kind == models.VersionBranch && current.Name == b}
		if b == defaultBranch {
			o.Detail = "default"
		}
		options = append(options, o)
	}
	for _, t := range tags {
		options = append(options, widgets.Option{Label: t, Value: tagValuePrefix + t, Detail: "tag", Marked: current.Kind == models.VersionTag && current.Name == t})
	}
	return options
}

func versionValue(v models.GitVersion) string {
	if v.Kind == models.VersionTag {
		return tagValuePrefix + v.Name
	}
	return branchValuePrefix + v.Name
}

func versionFromValue(value string) models.GitVersion {
	if name, ok := strings.CutPrefix(value, tagValuePrefix); ok {
		return models.GitVersion{Kind: models.VersionTag, Name: name}
	}
	return models.BranchVersion(strings.TrimPrefix(value, branchValuePrefix))
}

func baseFields(env Env, branches, tags []string, defaultBranch string, base models.GitVersion) []field {
	options := versionOptions(branches, tags, defaultBranch, base)
	return []field{
		newChoiceField(env.Keys, "Based on", "Choose a branch or tag", "ref", "refs", options, versionValue(base)),
		newTextField("Or a commit (optional)", "commit ID", "", 40, true),
	}
}

func baseVersion(choice, commit string) (models.GitVersion, error) {
	if commit != "" {
		if !commitIDPattern.MatchString(commit) {
			return models.GitVersion{}, errors.New("a commit ID is 4 to 40 hexadecimal characters")
		}
		return models.GitVersion{Kind: models.VersionCommit, Name: commit}, nil
	}
	if choice == "" {
		return models.GitVersion{}, errors.New("choose what to base it on")
	}
	return versionFromValue(choice), nil
}

func validateNewRef(name, noun string, existing []string) error {
	if err := models.ValidateRefName(name); err != nil {
		return err
	}
	if slices.Contains(existing, name) {
		return fmt.Errorf("a %s named %s already exists", noun, name)
	}
	return nil
}

func NewBranchForm(env Env, repo string, branches, tags []string, defaultBranch string, base models.GitVersion) RefForm {
	fields := append([]field{newTextField("Name", "feature/1102-saved-carts", "", refNameLimit, false)}, baseFields(env, branches, tags, defaultBranch, base)...)
	return newRefForm(env, "New branch", repo, "Then, locally: git fetch, then git switch to it.", fields, func(v []string) (tea.Msg, error) {
		if err := validateNewRef(v[0], "branch", branches); err != nil {
			return nil, err
		}
		b, err := baseVersion(v[1], v[2])
		if err != nil {
			return nil, err
		}
		return CreateBranchRequestMsg{Name: v[0], Base: b}, nil
	})
}

func NewTagForm(env Env, repo string, branches, tags []string, defaultBranch string, base models.GitVersion) RefForm {
	fields := append([]field{newTextField("Name", "v1.0.0", "", refNameLimit, false)}, baseFields(env, branches, tags, defaultBranch, base)...)
	fields = append(fields, newAreaField("Description", "What this tag marks (required)", "", 0, 4))
	note := "Creates an annotated tag, the kind Azure DevOps creates from its web UI."
	return newRefForm(env, "New tag", repo, note, fields, func(v []string) (tea.Msg, error) {
		if err := validateNewRef(v[0], "tag", tags); err != nil {
			return nil, err
		}
		b, err := baseVersion(v[1], v[2])
		if err != nil {
			return nil, err
		}
		if v[3] == "" {
			return nil, errors.New("enter a description")
		}
		return CreateTagRequestMsg{Name: v[0], Base: b, Message: v[3]}, nil
	})
}

func NewDeleteBranchModal(env Env, b models.Branch, defaultBranch string) ConfirmModal {
	lines := []string{"Delete " + b.Name + "?"}
	if b.Last.ID != "" {
		lines = append(lines, fmt.Sprintf("Last commit %s by %s, %d ahead and %d behind %s.", models.ShortCommit(b.Last.ID), b.Last.Author, b.Ahead, b.Behind, defaultBranch))
	}
	if b.PullRequestID != 0 {
		lines = append(lines, "", fmt.Sprintf("! Pull request !%d is active from this branch. It stays open, but can't be completed without its source branch.", b.PullRequestID))
	}
	return ConfirmModal{
		env:     env,
		title:   "Delete branch",
		message: strings.Join(lines, "\n"),
		warning: "Azure DevOps can restore a deleted branch from its Branches page.",
		confirm: DeleteBranchRequestMsg{Branch: b},
	}
}

func (m RefForm) Init() tea.Cmd { return textinput.Blink }

func (m RefForm) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
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
		case key.Matches(msg, k.Select) && !m.fields[m.focus].multiline():
			if m.focus == len(m.fields)-1 {
				return m.submit()
			}
			return m.moveFocus(1)
		}
	}
	f, cmd := m.fields[m.focus].update(msg)
	m.fields = slices.Clone(m.fields)
	m.fields[m.focus] = f
	m.refresh()
	return m, cmd
}

func (m RefForm) moveFocus(delta int) (Modal, tea.Cmd) {
	m.fields = slices.Clone(m.fields)
	m.fields[m.focus] = m.fields[m.focus].blur()
	m.focus = (m.focus + delta + len(m.fields)) % len(m.fields)
	var cmd tea.Cmd
	m.fields[m.focus], cmd = m.fields[m.focus].focus()
	m.refresh()
	return m, cmd
}

func (m RefForm) submit() (Modal, tea.Cmd) {
	values := make([]string, len(m.fields))
	for i, f := range m.fields {
		values[i] = f.value()
	}
	msg, err := m.build(values)
	if err != nil {
		m.err = err.Error()
		m.refresh()
		return m, nil
	}
	return m, send(msg)
}

func (m *RefForm) width() int { return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width) }

func (m *RefForm) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.fields = slices.Clone(m.fields)
	for i, f := range m.fields {
		m.fields[i] = f.resize(inner)
	}
	m.body.SetWidth(inner)
	m.refresh()
}

func (m *RefForm) refresh() {
	var blocks []string
	top, bottom, line := 0, 0, 0
	for i, f := range m.fields {
		label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == m.focus {
			label = label.Foreground(styles.ColorPrimary).Bold(true)
		}
		block := lipgloss.JoinVertical(lipgloss.Left, label.Render(f.label()), f.view())
		h := lipgloss.Height(block)
		if i == m.focus {
			top, bottom = line, line+h-1
		}
		blocks = append(blocks, block)
		line += h + 1
	}
	content := strings.Join(blocks, "\n\n")
	budget := widgets.DialogBodyHeight(m.env.Height, m.subtitle != "") - 2
	if m.err != "" {
		budget -= 2
	}
	m.body.SetHeight(max(min(lipgloss.Height(content), budget), 1))
	m.body.SetContent(content)
	widgets.Reveal(&m.body, top, bottom)
}

func (m RefForm) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	body := m.body.View()
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err)
	}
	if m.note != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).Render(m.note)
	}
	k := m.env.Keys
	hints := []key.Binding{widgets.Relabel(k.NextPanel, "next"), widgets.Relabel(k.Save, "create"), widgets.Relabel(k.Back, "cancel")}
	if m.fields[m.focus].choice() {
		hints = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown)}, hints...)
	}
	return widgets.Dialog{Title: m.title, Subtitle: m.subtitle, Body: body, Footer: widgets.Hints(inner, hints...), Width: m.width()}.View()
}

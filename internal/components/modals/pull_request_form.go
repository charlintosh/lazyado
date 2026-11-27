package modals

import (
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
	pullRequestFormTitle = "New pull request"
	loadingBranchesNote  = "Loading branches"
	comparingNote        = "Comparing branches"
)

type prField int

const (
	prRepository prField = iota
	prSource
	prTarget
	prTitle
	prDescription
	prRequired
	prOptional
	prWorkItems
	prTags
	prDraft
)

type prEntry struct {
	id    prField
	field field
}

type PullRequestFormData struct {
	Repositories []models.Repository
	Repository   models.Repository
	Members      []models.TeamMember
	Open         []models.PullRequest
	Source       string
}

type PullRequestBranchesRequestMsg struct {
	RepositoryID string
}

type PullRequestBranchesMsg struct {
	RepositoryID string
	Branches     []string
	Default      string
	Err          error
}

type PullRequestCompareRequestMsg struct {
	RepositoryID string
	Source       string
	Target       string
}

type PullRequestComparedMsg struct {
	RepositoryID string
	Source       string
	Target       string
	Comparison   models.BranchComparison
	Err          error
}

type PullRequestCreateRequestMsg struct {
	Repository  models.Repository
	PullRequest models.NewPullRequest
}

type OpenPullRequestRequestMsg struct {
	ID int
}

type PullRequestForm struct {
	env             Env
	data            PullRequestFormData
	repo            models.Repository
	branches        []string
	defaultBranch   string
	loadingBranches bool
	compared        [2]string
	comparing       bool
	cmp             models.BranchComparison
	cmpErr          string
	auto            map[prField]string
	entries         []prEntry
	focus           int
	err             string
	body            viewport.Model
	spinner         spinner.Model
}

func NewPullRequestForm(env Env, data PullRequestFormData) PullRequestForm {
	k := env.Keys
	members := reviewerOptions(data.Members)
	var entries []prEntry
	if len(data.Repositories) > 0 {
		options := make([]widgets.Option, len(data.Repositories))
		for i, r := range data.Repositories {
			options[i] = widgets.Option{Label: r.Name, Value: r.ID}
		}
		entries = append(entries, prEntry{prRepository, newChoiceField(k, "Repository", "Choose a repository", "repository", "repositories", options, "")})
	}
	entries = append(entries,
		prEntry{prSource, branchField(env, "Source branch", nil, nil, "")},
		prEntry{prTarget, branchField(env, "Target branch", nil, nil, "")},
		prEntry{prTitle, newTextField("Title", "What does this pull request do?", "", styles.TitleCharLimit, false)},
		prEntry{prDescription, newAreaField("Description", "Markdown supported", "", styles.PullRequestDescriptionLimit, styles.TextareaHeightXL)},
		prEntry{prRequired, newMultiChoiceField(k, "Required reviewers", "None", "member", "members", members)},
		prEntry{prOptional, newMultiChoiceField(k, "Optional reviewers", "None", "member", "members", members)},
		prEntry{prWorkItems, newTextField("Work items (IDs, separate with ,)", "1043, 1050", "", styles.TagsCharLimit, false)},
		prEntry{prTags, tagsField(nil)},
		prEntry{prDraft, toggleField{name: "Options", text: "Create as draft"}},
	)
	m := PullRequestForm{
		env:             env,
		data:            data,
		repo:            data.Repository,
		loadingBranches: data.Repository.ID != "",
		auto:            map[prField]string{},
		entries:         entries,
		body:            viewport.New(),
		spinner:         spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
	}
	m.body.KeyMap = viewport.KeyMap{}
	m.entries[0].field, _ = m.entries[0].field.focus()
	m.layout()
	return m
}

func reviewerOptions(members []models.TeamMember) []widgets.Option {
	var options []widgets.Option
	for _, mb := range members {
		if mb.ID != "" {
			options = append(options, widgets.Option{Label: mb.DisplayName, Value: mb.ID, Detail: mb.UniqueName})
		}
	}
	return options
}

func branchField(env Env, name string, branches []string, open map[string]int, initial string) choiceField {
	options := make([]widgets.Option, len(branches))
	for i, b := range branches {
		detail := ""
		if id, ok := open[b]; ok {
			detail = fmt.Sprintf("!%d open", id)
		}
		options[i] = widgets.Option{Label: b, Value: b, Detail: detail}
	}
	return newChoiceField(env.Keys, name, "Choose a branch", "branch", "branches", options, initial)
}

func (m PullRequestForm) Init() tea.Cmd {
	if m.repo.ID == "" {
		return textinput.Blink
	}
	return tea.Batch(textinput.Blink, send(PullRequestBranchesRequestMsg{RepositoryID: m.repo.ID}), m.spinner.Tick)
}

func (m PullRequestForm) busy() bool {
	return m.loadingBranches || m.comparing
}

func (m PullRequestForm) Update(msg tea.Msg) (Modal, tea.Cmd) {
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
	case PullRequestBranchesMsg:
		return m.setBranches(msg)
	case PullRequestComparedMsg:
		return m.setComparison(msg), nil
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

func (m PullRequestForm) enter() (Modal, tea.Cmd) {
	m.entries = slices.Clone(m.entries)
	switch f := m.current().(type) {
	case multiChoiceField:
		m.entries[m.focus].field = f.toggle()
	case toggleField:
		f.on = !f.on
		m.entries[m.focus].field = f
	case areaField:
		next, cmd := f.update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m.entries[m.focus].field = next
		m.refresh()
		return m, cmd
	default:
		return m.moveFocus(1)
	}
	m.refresh()
	return m, nil
}

func (m PullRequestForm) current() field {
	return m.entries[m.focus].field
}

func (m PullRequestForm) index(id prField) int {
	for i, e := range m.entries {
		if e.id == id {
			return i
		}
	}
	return -1
}

func (m PullRequestForm) value(id prField) string {
	if i := m.index(id); i >= 0 {
		return m.entries[i].field.value()
	}
	return ""
}

func (m PullRequestForm) moveFocus(delta int) (Modal, tea.Cmd) {
	m.entries = slices.Clone(m.entries)
	m.entries[m.focus].field = m.current().blur()
	m.focus = (m.focus + delta + len(m.entries)) % len(m.entries)
	f, cmd := m.current().focus()
	m.entries[m.focus].field = f
	m, follow := m.sync()
	m.refresh()
	return m, tea.Batch(cmd, follow)
}

func (m PullRequestForm) sync() (PullRequestForm, tea.Cmd) {
	if id := m.value(prRepository); id != "" && id != m.repo.ID {
		for _, r := range m.data.Repositories {
			if r.ID == id {
				m.repo = r
			}
		}
		m.branches, m.loadingBranches, m.compared, m.cmp, m.cmpErr = nil, true, [2]string{}, models.BranchComparison{}, ""
		m.replaceBranches("")
		return m, tea.Batch(send(PullRequestBranchesRequestMsg{RepositoryID: id}), m.spinner.Tick)
	}
	source, target := m.value(prSource), m.value(prTarget)
	pair := [2]string{source, target}
	if source == "" || target == "" || source == target || pair == m.compared {
		if pair != m.compared {
			m.compared, m.cmp, m.cmpErr, m.comparing = pair, models.BranchComparison{}, "", false
		}
		return m, nil
	}
	m.compared, m.comparing, m.cmp, m.cmpErr = pair, true, models.BranchComparison{}, ""
	if item := models.WorkItemFromBranch(source); item > 0 {
		m.prefill(prWorkItems, strconv.Itoa(item))
	}
	return m, tea.Batch(send(PullRequestCompareRequestMsg{RepositoryID: m.repo.ID, Source: source, Target: target}), m.spinner.Tick)
}

func (m *PullRequestForm) openBySource() map[string]int {
	open := map[string]int{}
	for _, pr := range m.data.Open {
		if pr.RepositoryID == m.repo.ID && pr.Status == models.PullRequestActive {
			open[pr.SourceBranch] = pr.ID
		}
	}
	return open
}

func (m *PullRequestForm) replaceBranches(target string) {
	m.entries = slices.Clone(m.entries)
	inner := widgets.DialogInnerWidth(m.width())
	open := m.openBySource()
	for _, id := range []prField{prSource, prTarget} {
		i := m.index(id)
		initial := m.data.Source
		if id == prTarget {
			initial = target
		}
		name := m.entries[i].field.label()
		f := branchField(m.env, name, m.branches, open, initial).resize(inner)
		if i == m.focus {
			f, _ = f.focus()
		}
		m.entries[i].field = f
	}
}

func (m PullRequestForm) setBranches(msg PullRequestBranchesMsg) (Modal, tea.Cmd) {
	if msg.RepositoryID != m.repo.ID {
		return m, nil
	}
	m.loadingBranches = false
	if msg.Err != nil {
		m.cmpErr = msg.Err.Error()
		m.refresh()
		return m, nil
	}
	m.branches, m.defaultBranch = msg.Branches, msg.Default
	m.replaceBranches(msg.Default)
	m.refresh()
	return m, nil
}

func (m PullRequestForm) setComparison(msg PullRequestComparedMsg) PullRequestForm {
	if msg.RepositoryID != m.repo.ID || [2]string{msg.Source, msg.Target} != m.compared {
		return m
	}
	m.comparing = false
	if msg.Err != nil {
		m.cmpErr = msg.Err.Error()
		m.refresh()
		return m
	}
	m.cmp = msg.Comparison
	m.prefill(prTitle, m.cmp.SuggestedTitle(msg.Source))
	if m.cmp.Template != "" {
		m.prefill(prDescription, m.cmp.Template)
	}
	m.refresh()
	return m
}

func (m *PullRequestForm) prefill(id prField, text string) {
	i := m.index(id)
	if i < 0 {
		return
	}
	if current := m.entries[i].field.value(); current != "" && current != m.auto[id] {
		return
	}
	m.entries = slices.Clone(m.entries)
	switch f := m.entries[i].field.(type) {
	case textField:
		f.input.SetValue(text)
		m.entries[i].field = f
	case areaField:
		f.area.SetValue(text)
		m.entries[i].field = f
	}
	m.auto[id] = strings.TrimSpace(text)
}

func (m PullRequestForm) width() int {
	return widgets.DialogWidth(styles.ModalWidthPR, m.env.Width)
}

func (m *PullRequestForm) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.entries = slices.Clone(m.entries)
	for i, e := range m.entries {
		m.entries[i].field = e.field.resize(inner)
	}
	m.body.SetWidth(inner)
	m.refresh()
}

func (m *PullRequestForm) refresh() {
	content, spans := m.renderFields()
	budget := widgets.DialogBodyHeight(m.env.Height, true)
	if m.err != "" {
		budget -= 2
	}
	m.body.SetHeight(max(min(lipgloss.Height(content), budget), 1))
	m.body.SetContent(content)
	span := spans[m.focus]
	widgets.Reveal(&m.body, span.top, span.bottom)
}

func (m PullRequestForm) renderFields() (string, []fieldSpan) {
	spans := make([]fieldSpan, len(m.entries))
	blocks := make([]string, 0, len(m.entries))
	line := 0
	for i, e := range m.entries {
		label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
		if i == m.focus {
			label = label.Foreground(styles.ColorPrimary).Bold(true)
		}
		block := lipgloss.JoinVertical(lipgloss.Left, label.Render(e.field.label()), e.field.view())
		if e.id == prTarget {
			if status := m.status(); status != "" {
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

func (m PullRequestForm) status() string {
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	ok := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true)
	warn := lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)
	bad := lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true)
	source, target := m.value(prSource), m.value(prTarget)
	switch {
	case m.loadingBranches:
		return m.spinner.View() + muted.Render(" "+loadingBranchesNote)
	case m.cmpErr != "":
		return bad.Render("✕ ") + muted.Render(m.cmpErr)
	case source == "" || target == "":
		return ""
	case source == target:
		return bad.Render("✕ ") + muted.Render("Source and target are the same branch.")
	case m.comparing || m.compared != [2]string{source, target}:
		return m.spinner.View() + muted.Render(" "+comparingNote)
	case m.cmp.ExistingID != 0:
		return bad.Render("✕ ") + muted.Render(fmt.Sprintf("An active pull request already exists: !%d %s. %s opens it.", m.cmp.ExistingID, m.cmp.ExistingTitle, m.env.Keys.Save.Help().Key))
	case m.cmp.Ahead == 0:
		return bad.Render("✕ ") + muted.Render(fmt.Sprintf("Nothing to merge: %s has no commits that aren't in %s.", source, target))
	}
	summary := fmt.Sprintf("%s ahead, %d behind %s · %s changed", countOf(m.cmp.Ahead, "commit"), m.cmp.Behind, target, countOf(m.cmp.Files, "file"))
	if m.cmp.Behind > 0 {
		return warn.Render("! ") + muted.Render(summary+". You may want to update the branch first.")
	}
	return ok.Render("✓ ") + muted.Render(summary)
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func (m PullRequestForm) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	body := m.body.View()
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err)
	}
	subtitle := "Choose a repository"
	if m.repo.Name != "" {
		subtitle = m.repo.Name
	}
	return widgets.Dialog{
		Title:    pullRequestFormTitle,
		Subtitle: subtitle,
		Body:     body,
		Footer:   widgets.Hints(inner, m.hints()...),
		Width:    m.width(),
	}.View()
}

func (m PullRequestForm) hints() []key.Binding {
	k := m.env.Keys
	bindings := []key.Binding{widgets.Relabel(k.Save, "create"), widgets.Relabel(k.NextPanel, "next"), widgets.Relabel(k.PrevPanel, "prev"), widgets.Relabel(k.Back, "cancel")}
	switch m.current().(type) {
	case multiChoiceField:
		bindings = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "add/remove")}, bindings...)
	case toggleField:
		bindings = append([]key.Binding{widgets.Relabel(k.Select, "toggle")}, bindings...)
	case choiceField:
		bindings = append([]key.Binding{widgets.Combine("choose", k.PickerUp, k.PickerDown)}, bindings...)
	}
	return bindings
}

func (m PullRequestForm) submit() (Modal, tea.Cmd) {
	msg, err := m.request()
	if err != nil {
		m.err = err.Error()
		m.refresh()
		return m, nil
	}
	m.err = ""
	return m, send(msg)
}

func (m PullRequestForm) request() (tea.Msg, error) {
	source, target := m.value(prSource), m.value(prTarget)
	compared := !m.comparing && m.cmpErr == "" && m.compared == [2]string{source, target}
	switch {
	case m.repo.ID == "":
		return nil, fmt.Errorf("Choose a repository")
	case source == "" || target == "":
		return nil, fmt.Errorf("Choose a source and a target branch")
	case source == target:
		return nil, fmt.Errorf("Source and target are the same branch")
	case compared && m.cmp.ExistingID != 0:
		return OpenPullRequestRequestMsg{ID: m.cmp.ExistingID}, nil
	case compared && m.cmp.Ahead == 0:
		return nil, fmt.Errorf("Nothing to merge: %s has no commits that aren't in %s", source, target)
	case m.value(prTitle) == "":
		return nil, errTitleRequired
	}
	items, err := parseWorkItemIDs(m.value(prWorkItems))
	if err != nil {
		return nil, err
	}
	tags, err := parseTags(m.value(prTags))
	if err != nil {
		return nil, err
	}
	return PullRequestCreateRequestMsg{Repository: m.repo, PullRequest: models.NewPullRequest{
		Source:      source,
		Target:      target,
		Title:       m.value(prTitle),
		Description: m.value(prDescription),
		Draft:       m.value(prDraft) != "",
		Reviewers:   m.reviewers(),
		WorkItems:   items,
		Labels:      tags,
	}}, nil
}

func (m PullRequestForm) reviewers() []models.NewReviewer {
	var out []models.NewReviewer
	required := m.entries[m.index(prRequired)].field.(multiChoiceField).values()
	for _, id := range required {
		out = append(out, models.NewReviewer{ID: id, Required: true})
	}
	for _, id := range m.entries[m.index(prOptional)].field.(multiChoiceField).values() {
		if !slices.Contains(required, id) {
			out = append(out, models.NewReviewer{ID: id})
		}
	}
	return out
}

func parseWorkItemIDs(s string) ([]int, error) {
	var ids []int
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		id, err := strconv.Atoi(strings.TrimPrefix(part, "#"))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("Work items must be IDs like 1043, separated by commas")
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

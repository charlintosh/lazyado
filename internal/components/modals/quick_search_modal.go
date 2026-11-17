package modals

import (
	"fmt"
	"path"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	quickSearchTitle       = "Quick search"
	quickSearchPrompt      = "Search: "
	quickSearchPlaceholder = "ID, !PR, #work item or link"
	quickSearchChoiceGap   = 6
	pullRequestKind        = "PR"
)

type QuickSearchRequestMsg struct {
	Query models.SearchQuery
}

type QuickSearchSuccessMsg struct {
	Item        *models.WorkItem
	PullRequest *models.PullRequest
}

type QuickSearchErrorMsg struct {
	Err error
}

type QuickSearchModal struct {
	env     Env
	input   textinput.Model
	spinner spinner.Model
	query   models.SearchQuery
	choices []QuickSearchSuccessMsg
	cursor  int
	loading bool
	err     string
}

func NewQuickSearchModal(env Env) QuickSearchModal {
	in := textinput.New()
	in.Placeholder = quickSearchPlaceholder
	in.CharLimit = styles.SearchCharLimit
	in.Prompt = quickSearchPrompt
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	st.Focused.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	in.SetStyles(st)
	in.Focus()
	m := QuickSearchModal{
		env:     env,
		input:   in,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
	}
	m.layout()
	return m
}

func (m QuickSearchModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m QuickSearchModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case QuickSearchErrorMsg:
		m.loading = false
		m.err = friendlyLookupError(msg.Err)
		return m, m.input.Focus()
	case QuickSearchSuccessMsg:
		m.loading = false
		m.choices, m.cursor = []QuickSearchSuccessMsg{{PullRequest: msg.PullRequest}, {Item: msg.Item}}, 0
		return m, nil
	case tea.PasteMsg:
		msg.Content = strings.TrimSpace(msg.Content)
		m.reset()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *QuickSearchModal) reset() {
	m.err, m.choices, m.cursor = "", nil, 0
}

func (m QuickSearchModal) handleKey(msg tea.KeyPressMsg) (Modal, tea.Cmd) {
	k := m.env.Keys
	if key.Matches(msg, k.Back) {
		return m, closeModal
	}
	if m.loading {
		return m, nil
	}
	navigating := len(m.choices) > 0 && msg.Text == ""
	switch {
	case navigating && key.Matches(msg, k.Up):
		m.cursor = max(m.cursor-1, 0)
		return m, nil
	case navigating && key.Matches(msg, k.Down):
		m.cursor = min(m.cursor+1, len(m.choices)-1)
		return m, nil
	case key.Matches(msg, k.Select) && len(m.choices) > 0:
		return m, send(m.choices[m.cursor])
	case key.Matches(msg, k.Select):
		return m.submit()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.reset()
	return m, cmd
}

func (m QuickSearchModal) submit() (Modal, tea.Cmd) {
	query, err := models.ParseSearchQuery(m.input.Value())
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.reset()
	m.query = query
	m.loading = true
	m.input.Blur()
	return m, tea.Batch(send(QuickSearchRequestMsg{Query: query}), m.spinner.Tick)
}

func (m *QuickSearchModal) layout() {
	inner := widgets.DialogInnerWidth(widgets.DialogWidth(styles.ModalWidthMD, m.env.Width))
	m.input.SetWidth(max(inner-lipgloss.Width(m.input.Prompt)-1, 1))
}

func (m QuickSearchModal) choiceRows(width int) string {
	rows := []string{lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(fmt.Sprintf("%d matches", len(m.choices)))}
	for i, c := range m.choices {
		bar, base := " ", lipgloss.NewStyle()
		if i == m.cursor {
			bar, base = "▌", base.Background(styles.ColorHighlight)
		}
		kind, id, title, detail := choiceText(c)
		head := base.Foreground(styles.ColorPrimary).Bold(true).Render(bar) + kind + base.Render(" ") +
			base.Foreground(styles.ColorAccent).Bold(true).Render(id) + base.Render(" ") + base.Foreground(styles.ColorText).Bold(true).Render(title)
		sub := base.Render(strings.Repeat(" ", quickSearchChoiceGap)) + base.Foreground(styles.ColorTextMuted).Render(detail)
		rows = append(rows,
			base.Width(width).Render(ansi.Truncate(head, width, "…")),
			base.Width(width).Render(ansi.Truncate(sub, width, "…")))
	}
	return strings.Join(rows, "\n")
}

func choiceText(c QuickSearchSuccessMsg) (string, string, string, string) {
	if pr := c.PullRequest; pr != nil {
		kind := lipgloss.NewStyle().Foreground(styles.ColorViolet).Bold(true).Render(pullRequestKind)
		return kind, fmt.Sprintf("!%d", pr.ID), pr.Title, fmt.Sprintf("%s · %s → %s", pr.RepositoryName, pr.SourceBranch, pr.TargetBranch)
	}
	item := c.Item
	kind := lipgloss.NewStyle().Foreground(styles.TypeColor(string(item.Type))).Bold(true).Render(item.ShortType())
	detail := string(item.State)
	if item.IterationPath != "" {
		detail += " · " + path.Base(strings.ReplaceAll(item.IterationPath, `\`, "/"))
	}
	return kind, fmt.Sprintf("#%d", item.ID), item.Title, detail
}

func (m QuickSearchModal) View() string {
	width := widgets.DialogWidth(styles.ModalWidthMD, m.env.Width)
	inner := widgets.DialogInnerWidth(width)
	body := m.input.View()
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Relabel(k.Select, "search"), widgets.Relabel(k.Back, "cancel"))
	switch {
	case m.loading:
		body = m.spinner.View() + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(" Looking up "+m.query.Label())
		footer = widgets.Hints(inner, widgets.Relabel(k.Back, "cancel"))
	case len(m.choices) > 0:
		body += "\n\n" + m.choiceRows(inner)
		footer = widgets.Hints(inner, widgets.Combine("choose", k.Up, k.Down), widgets.Relabel(k.Select, "open"), widgets.Relabel(k.Back, "cancel"))
	}
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render("Error: "+m.err)
	}
	return widgets.Dialog{Title: quickSearchTitle, Body: body, Footer: footer, Width: width}.View()
}

func friendlyLookupError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.Index(msg, "TF"); idx >= 0 {
		if end := strings.Index(msg[idx:], ":"); end > 0 {
			msg = strings.TrimSpace(msg[idx+end+1:])
		}
	}
	if strings.HasPrefix(msg, "API error") {
		return "Work item not found or you don't have permissions"
	}
	return msg
}

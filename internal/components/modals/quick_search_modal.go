package modals

import (
	"strconv"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type QuickSearchRequestMsg struct {
	ID int
}

type QuickSearchSuccessMsg struct {
	Item *models.WorkItem
}

type QuickSearchErrorMsg struct {
	Err error
}

type QuickSearchModal struct {
	env     Env
	input   textinput.Model
	spinner spinner.Model
	loading bool
	err     string
}

func NewQuickSearchModal(env Env) QuickSearchModal {
	in := textinput.New()
	in.Placeholder = "Enter work item ID..."
	in.CharLimit = styles.InputCharLimitSM
	in.Prompt = "ID: "
	st := in.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	st.Focused.Text = lipgloss.NewStyle().Foreground(styles.ColorText)
	in.SetStyles(st)
	in.Focus()
	m := QuickSearchModal{
		env:     env,
		input:   in,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
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
	case tea.PasteMsg:
		msg.Content = digits(msg.Content)
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m QuickSearchModal) handleKey(msg tea.KeyPressMsg) (Modal, tea.Cmd) {
	if key.Matches(msg, m.env.Keys.Back) {
		return m, closeModal
	}
	if m.loading {
		return m, nil
	}
	if key.Matches(msg, m.env.Keys.Select) {
		return m.submit()
	}
	if msg.Text != "" && digits(msg.Text) != msg.Text {
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.err = ""
	return m, cmd
}

func (m QuickSearchModal) submit() (Modal, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		m.err = "Please enter a work item ID"
		return m, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		m.err = "ID must be a positive number"
		return m, nil
	}
	m.err = ""
	m.loading = true
	m.input.Blur()
	return m, tea.Batch(send(QuickSearchRequestMsg{ID: id}), m.spinner.Tick)
}

func (m *QuickSearchModal) layout() {
	inner := widgets.DialogInnerWidth(widgets.DialogWidth(styles.ModalWidthMD, m.env.Width))
	m.input.SetWidth(max(inner-lipgloss.Width(m.input.Prompt)-1, 1))
}

func (m QuickSearchModal) View() string {
	width := widgets.DialogWidth(styles.ModalWidthMD, m.env.Width)
	inner := widgets.DialogInnerWidth(width)
	body := m.input.View()
	if m.loading {
		body = m.spinner.View() + lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(" Fetching work item...")
	}
	if m.err != "" {
		body += "\n\n" + lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render("Error: "+m.err)
	}
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Relabel(k.Select, "open"), widgets.Relabel(k.Back, "cancel"))
	if m.loading {
		footer = widgets.Hints(inner, widgets.Relabel(k.Back, "cancel"))
	}
	return widgets.Dialog{Title: "Quick search work item", Body: body, Footer: footer, Width: width}.View()
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
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

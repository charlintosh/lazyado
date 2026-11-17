package modals

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/share"
	"github.com/charlintosh/lazyado/internal/styles"
)

type CopyRequestMsg struct {
	Text   string
	Notice string
}

type OpenURLRequestMsg struct {
	URL string
}

func ShareCopy(t share.Target, o share.Option) CopyRequestMsg {
	return CopyRequestMsg{Text: o.Text, Notice: t.Notice(o)}
}

func ShareFormatKey(k keys.KeyMap, kind share.Kind) key.Binding {
	switch kind {
	case share.Permalink:
		return k.ShareFormatPermalink
	case share.ID:
		return k.ShareFormatID
	case share.Name:
		return k.ShareFormatName
	case share.Path:
		return k.ShareFormatPath
	case share.Snippet:
		return k.ShareFormatSnippet
	case share.Markdown:
		return k.ShareFormatMarkdown
	case share.Title:
		return k.ShareFormatTitle
	case share.Mention:
		return k.ShareFormatMention
	case share.GitHub:
		return k.ShareFormatGitHub
	case share.RunID:
		return k.ShareFormatRunID
	case share.Logs:
		return k.ShareFormatLogs
	case share.PipelineLink:
		return k.ShareFormatPipeline
	}
	return k.ShareFormatLink
}

type ShareModal struct {
	choice ChoiceModal
	target share.Target
	keys   keys.KeyMap
}

func NewShareModal(env Env, t share.Target) ShareModal {
	labelWidth := 0
	for _, o := range t.Options {
		labelWidth = max(labelWidth, lipgloss.Width(o.Label))
	}
	options := make([]widgets.Option, len(t.Options))
	for i, o := range t.Options {
		letter := ShareFormatKey(env.Keys, o.Kind).Help().Key
		options[i] = widgets.Option{
			Label:  fmt.Sprintf("%s  %-*s", letter, labelWidth, o.Label),
			Value:  strconv.Itoa(i),
			Detail: preview(o.Text),
		}
	}
	k := env.Keys
	choice := newChoiceModal(env, choiceSpec{
		title:    "Share",
		subtitle: t.Subject,
		action:   "copy",
		options:  options,
		width:    styles.ModalWidthPR,
		rows:     len(options),
		hints:    []key.Binding{widgets.Relabel(k.ShareLink, "copy link"), k.OpenBrowser},
		choose: func(o widgets.Option) tea.Msg {
			i, _ := strconv.Atoi(o.Value)
			return ShareCopy(t, t.Options[i])
		},
	})
	return ShareModal{choice: choice, target: t, keys: env.Keys}
}

func preview(text string) string {
	first, rest, multi := strings.Cut(text, "\n")
	if multi {
		return first + fmt.Sprintf(" … (%d lines)", strings.Count(rest, "\n")+2)
	}
	return first
}

func (m ShareModal) Init() tea.Cmd { return m.choice.Init() }

func (m ShareModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if cmd, handled := m.handleKey(k); handled {
			return m, cmd
		}
	}
	next, cmd := m.choice.Update(msg)
	if c, ok := next.(ChoiceModal); ok {
		m.choice = c
		return m, cmd
	}
	return next, cmd
}

func (m ShareModal) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.ShareLink):
		if o, ok := m.target.Default(); ok {
			return send(ShareCopy(m.target, o)), true
		}
		return nil, true
	case key.Matches(msg, m.keys.OpenBrowser):
		if o, ok := m.target.Find(share.Link); ok {
			return send(OpenURLRequestMsg{URL: o.Text}), true
		}
		return nil, true
	}
	for _, o := range m.target.Options {
		if key.Matches(msg, ShareFormatKey(m.keys, o.Kind)) {
			return send(ShareCopy(m.target, o)), true
		}
	}
	return nil, false
}

func (m ShareModal) View() string { return m.choice.View() }

type ClipboardModal struct {
	env      Env
	notice   string
	text     string
	viewport viewport.Model
}

func NewClipboardModal(env Env, text, notice string) ClipboardModal {
	m := ClipboardModal{env: env, notice: notice, text: text, viewport: widgets.NewViewport(env.Keys)}
	m.layout()
	return m
}

func (m ClipboardModal) Init() tea.Cmd { return nil }

func (m ClipboardModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		k := m.env.Keys
		if key.Matches(msg, k.Back) || key.Matches(msg, k.Select) {
			return m, closeModal
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ClipboardModal) width() int {
	return widgets.DialogWidth(styles.ModalWidthPR, m.env.Width)
}

func (m *ClipboardModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	intro := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).
		Render("No system clipboard was found, so lazyado asked your terminal to copy it. If it didn't arrive, select the text below and copy it from your terminal.")
	text := lipgloss.NewStyle().Foreground(styles.ColorText).Width(inner).Render(m.text)
	content := intro + "\n\n" + text
	m.viewport.SetWidth(inner)
	m.viewport.SetHeight(min(lipgloss.Height(content), widgets.DialogBodyHeight(m.env.Height, true)))
	m.viewport.SetContent(content)
}

func (m ClipboardModal) View() string {
	k := m.env.Keys
	bindings := []key.Binding{widgets.Combine("close", k.Select, k.Back)}
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		bindings = append([]key.Binding{widgets.Relabel(k.Down, "scroll")}, bindings...)
	}
	return widgets.Dialog{
		Title:    "Copy from here",
		Subtitle: m.notice,
		Body:     m.viewport.View(),
		Footer:   widgets.Hints(widgets.DialogInnerWidth(m.width()), bindings...),
		Width:    m.width(),
		Accent:   styles.ColorWarning,
	}.View()
}

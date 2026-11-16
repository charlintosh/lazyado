package modals

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const mentionDelimiters = " \n,.!?:;"

type CommentCreateRequestMsg struct {
	WorkItemID int
	Text       string
}

type CommentUpdateRequestMsg struct {
	WorkItemID int
	CommentID  int
	Text       string
}

type CommentModal struct {
	env        Env
	title      string
	subtitle   string
	workItemID int
	commentID  int
	area       textarea.Model
	members    []models.TeamMember
	mentions   widgets.Picker
	mentioning bool
	dismissed  string
	err        string
}

func NewCommentModal(env Env, item models.WorkItem, members []models.TeamMember) CommentModal {
	return newCommentModal(env, "Add comment", itemLabel(item), item.ID, 0, "", members)
}

func NewEditCommentModal(env Env, workItemID int, comment models.Comment, members []models.TeamMember) CommentModal {
	return newCommentModal(env, "Edit comment", fmt.Sprintf("#%d", workItemID), workItemID, comment.ID, comment.Text, members)
}

func newCommentModal(env Env, title, subtitle string, workItemID, commentID int, text string, members []models.TeamMember) CommentModal {
	ta := textarea.New()
	ta.Placeholder = "Write a comment... (use @name to mention someone)"
	ta.CharLimit = styles.ContentCharLimitLG
	ta.ShowLineNumbers = false
	ta.SetValue(text)
	ta.Focus()
	options := make([]widgets.Option, len(members))
	for i, m := range members {
		options[i] = widgets.Option{Label: m.DisplayName, Value: m.ID}
	}
	m := CommentModal{
		env:        env,
		title:      title,
		subtitle:   subtitle,
		workItemID: workItemID,
		commentID:  commentID,
		area:       ta,
		members:    members,
		mentions:   widgets.NewPicker(env.Keys, options, widgets.PickerConfig{Rows: styles.MentionPickerRows}),
	}
	m.layout()
	return m
}

func (m CommentModal) Init() tea.Cmd {
	return textarea.Blink
}

func (m CommentModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		if next, cmd, ok := m.handleMentionKey(msg); ok {
			return next, cmd
		}
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Save):
			return m.submit()
		}
	}
	before := m.area.Value()
	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)
	if m.area.Value() != before {
		m.syncMentions()
	}
	return m, cmd
}

func (m CommentModal) handleMentionKey(msg tea.KeyPressMsg) (CommentModal, tea.Cmd, bool) {
	if !m.mentioning {
		return m, nil, false
	}
	k := m.env.Keys
	switch {
	case key.Matches(msg, k.PickerUp), key.Matches(msg, k.PickerDown):
		m.mentions, _ = m.mentions.Update(msg)
	case key.Matches(msg, k.Select), key.Matches(msg, k.NextPanel):
		if o, ok := m.mentions.Selected(); ok {
			m.area.SetValue(insertMention(m.area.Value(), o.Label))
		}
		m.mentioning = false
	case key.Matches(msg, k.Back):
		m.dismissed = m.area.Value()
		m.mentioning = false
	default:
		return m, nil, false
	}
	return m, nil, true
}

func (m *CommentModal) syncMentions() {
	text := m.area.Value()
	query, ok := mentionQuery(text)
	if !ok || len(m.members) == 0 || text == m.dismissed {
		m.mentioning = false
		return
	}
	m.dismissed = ""
	m.mentions.SetQuery(query)
	m.mentioning = m.mentions.Len() > 0
}

func (m CommentModal) submit() (Modal, tea.Cmd) {
	text := strings.TrimSpace(m.area.Value())
	if text == "" {
		m.err = "Comment cannot be empty"
		return m, nil
	}
	text = processMentions(text, m.members)
	if m.commentID != 0 {
		return m, send(CommentUpdateRequestMsg{WorkItemID: m.workItemID, CommentID: m.commentID, Text: text})
	}
	return m, send(CommentCreateRequestMsg{WorkItemID: m.workItemID, Text: text})
}

func (m *CommentModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.area.SetWidth(inner)
	reserved := styles.MentionPickerRows + 3
	m.area.SetHeight(max(min(styles.TextareaHeightLG, widgets.DialogBodyHeight(m.env.Height, true)-reserved), 1))
	m.mentions.SetWidth(inner)
}

func (m CommentModal) width() int {
	return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width)
}

func (m CommentModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	parts := []string{m.area.View()}
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Relabel(k.Save, "post"), widgets.Relabel(k.Back, "cancel"))
	switch {
	case m.mentioning:
		parts = append(parts, "", lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("Mention"), m.mentions.View())
		footer = widgets.Hints(inner, widgets.Combine("move", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "insert"), widgets.Relabel(k.Back, "dismiss"))
	case len(m.members) > 0:
		parts = append(parts, "", lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("Tip: type @ to mention a team member"))
	}
	if m.err != "" {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(styles.ColorError).Render("Error: "+m.err))
	}
	return widgets.Dialog{
		Title:    m.title,
		Subtitle: m.subtitle,
		Body:     lipgloss.JoinVertical(lipgloss.Left, parts...),
		Footer:   footer,
		Width:    m.width(),
	}.View()
}

func mentionQuery(text string) (string, bool) {
	at := strings.LastIndex(text, "@")
	if at < 0 {
		return "", false
	}
	if at > 0 && !strings.ContainsRune(" \n", rune(text[at-1])) {
		return "", false
	}
	query := text[at+1:]
	if strings.ContainsAny(query, " \n") {
		return "", false
	}
	return query, true
}

func insertMention(text, name string) string {
	at := strings.LastIndex(text, "@")
	if at < 0 {
		return text
	}
	end := len(text)
	if i := strings.IndexAny(text[at+1:], " \n"); i >= 0 {
		end = at + 1 + i
	}
	return text[:at] + "@" + name + " " + text[end:]
}

func processMentions(text string, members []models.TeamMember) string {
	sorted := append([]models.TeamMember(nil), members...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return len(sorted[i].DisplayName) > len(sorted[j].DisplayName)
	})
	for _, member := range sorted {
		text = replaceMention(text, member)
	}
	return text
}

func replaceMention(text string, member models.TeamMember) string {
	mention := "@" + member.DisplayName
	anchor := fmt.Sprintf(`<a href="#" data-vss-mention="version:2.0,%s">%s</a>`, member.ID, member.DisplayName)
	var b strings.Builder
	for {
		idx := strings.Index(text, mention)
		if idx < 0 {
			b.WriteString(text)
			return b.String()
		}
		end := idx + len(mention)
		b.WriteString(text[:idx])
		if end == len(text) || strings.ContainsRune(mentionDelimiters, rune(text[end])) {
			b.WriteString(anchor)
		} else {
			b.WriteString(mention)
		}
		text = text[end:]
	}
}

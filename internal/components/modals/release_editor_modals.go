package modals

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const saveCommentLimit = 2000

type SaveReleaseEditsRequestMsg struct {
	Comment string
}

type ReapplyReleaseEditsMsg struct{}

type JumpToStaleMsg struct{}

type DiscardReleaseEditsConfirmedMsg struct{}

type ReleaseSaveConflictMsg struct {
	Revision int
	By       string
	When     time.Time
}

type ReleaseSaveFailedMsg struct {
	Err error
}

type ReleaseSaveModal struct {
	env      Env
	title    string
	subtitle string
	note     string
	edits    models.DocEdits
	comment  textinput.Model
	offset   int
	saving   bool
	conflict *ReleaseSaveConflictMsg
	stale    []models.StaleField
	err      string
}

func NewReleaseSaveModal(env Env, doc *models.ReleaseDoc, edits models.DocEdits, stale []models.StaleField) ReleaseSaveModal {
	m := ReleaseSaveModal{env: env, edits: edits, stale: stale, title: "Save " + doc.Title()}
	if doc.Kind == models.DocRelease {
		m.subtitle = plural(len(edits), "change") + " · this release only"
		m.note = "Only " + doc.Name + " changes. Its release pipeline stays as it is."
	} else {
		m.subtitle = plural(len(edits), "change") + fmt.Sprintf(" · saved as revision %d", doc.Revision+1)
		m.note = "Releases already created keep their own copy of the pipeline."
	}
	m.comment = newTextField("", "Why? Shown in the History tab", "", saveCommentLimit, false).input
	m.comment.Focus()
	if len(stale) > 0 {
		m.subtitle = plural(len(edits), "change") + " · " + plural(len(stale), "setting") + " need a value first"
		m.comment.Blur()
	}
	m.layout()
	return m
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func (m ReleaseSaveModal) Init() tea.Cmd { return textinput.Blink }

func (m ReleaseSaveModal) width() int { return widgets.DialogWidth(styles.ModalWidthPR, m.env.Width) }

func (m *ReleaseSaveModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.comment.SetWidth(max(inner-lipgloss.Width(inputPrompt)-1, 1))
}

func (m ReleaseSaveModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case ReleaseSaveConflictMsg:
		m.saving, m.conflict, m.err = false, &msg, ""
		m.comment.Blur()
		return m, nil
	case ReleaseSaveFailedMsg:
		m.saving, m.err = false, msg.Err.Error()
		return m, nil
	case tea.KeyPressMsg:
		k := m.env.Keys
		switch {
		case key.Matches(msg, k.Back):
			return m, closeModal
		case len(m.stale) > 0 && key.Matches(msg, k.Select):
			return m, send(JumpToStaleMsg{})
		case len(m.stale) > 0:
			return m, nil
		case m.conflict != nil && key.Matches(msg, k.ReapplyEdits):
			return m, send(ReapplyReleaseEditsMsg{})
		case m.conflict != nil:
			return m, nil
		case key.Matches(msg, k.Save):
			if m.saving {
				return m, nil
			}
			m.saving, m.err = true, ""
			return m, send(SaveReleaseEditsRequestMsg{Comment: strings.TrimSpace(m.comment.Value())})
		case key.Matches(msg, k.PickerUp):
			m.offset = max(m.offset-1, 0)
			return m, nil
		case key.Matches(msg, k.PickerDown):
			m.offset++
			return m, nil
		}
	}
	if m.conflict != nil || len(m.stale) > 0 {
		return m, nil
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m ReleaseSaveModal) diffLines(inner int) []string {
	where := lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true)
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	removed := lipgloss.NewStyle().Background(styles.ColorDiffDelete).Foreground(styles.ColorText)
	added := lipgloss.NewStyle().Background(styles.ColorDiffAdd).Foreground(styles.ColorText)
	line := func(st lipgloss.Style, sign, text string) string {
		text = strings.ReplaceAll(text, "\n", " ")
		return st.Width(inner).MaxWidth(inner).Inline(true).Render(ansi.Truncate(sign+text, inner, "…"))
	}
	var out []string
	group := ""
	for _, e := range m.edits {
		if e.Where != group {
			if group != "" {
				out = append(out, "")
			}
			group = e.Where
			out = append(out, where.Render(ansi.Truncate(group, inner, "…")))
		}
		out = append(out, label.Render(ansi.Truncate(e.Field, inner, "…")), line(removed, "- ", e.BeforeText), line(added, "+ ", e.AfterText))
	}
	return out
}

func (m ReleaseSaveModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner)
	var tail []block
	switch {
	case len(m.stale) > 0:
		red := lipgloss.NewStyle().Foreground(styles.ColorError)
		lines := []string{red.Bold(true).Render("! Can't save yet"),
			lipgloss.NewStyle().Foreground(styles.ColorText).Width(inner).Render("These were listed by Azure for a value you changed, so they may not exist for the new one:")}
		for _, st := range m.stale {
			line := red.Render("! ") + lipgloss.NewStyle().Foreground(styles.ColorText).Render(st.Field) + lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("  "+st.Where+" · after "+st.Cause+" changed")
			lines = append(lines, ansi.Truncate(line, inner, "…"))
		}
		lines = append(lines, muted.Render("Pick each from Azure's new list, or undo the change."))
		tail = append(tail, block{text: strings.Join(lines, "\n"), keep: true})
	case m.conflict != nil:
		c := m.conflict
		who := ""
		if c.By != "" {
			who = " by " + c.By
		}
		tail = append(tail, block{text: note("Not saved:", fmt.Sprintf("revision %d was saved%s while you were editing. Nothing was overwritten and your changes are kept.", c.Revision, who),
			lipgloss.NewStyle().Foreground(styles.ColorError), inner), keep: true})
	case m.err != "":
		tail = append(tail, block{text: lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err), keep: true})
	case m.saving:
		tail = append(tail, block{text: muted.Render("Saving..."), keep: true})
	default:
		tail = append(tail,
			block{text: lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render("Comment") + "\n" + m.comment.View(), keep: true},
			block{text: muted.Render(m.note)})
	}
	budget := widgets.DialogBodyHeight(m.env.Height, true)
	tailText := fitBlocks(tail, max(budget-2, 1))
	diff := m.diffLines(inner)
	room := max(budget-lipgloss.Height(tailText)-1, 1)
	offset := min(m.offset, max(len(diff)-room, 0))
	shown := diff[offset:min(offset+room, len(diff))]
	if len(diff) > room && room > 1 {
		shown = append(shown[:room-1], muted.Render(fmt.Sprintf("%d-%d of %d lines · scroll with %s", offset+1, offset+room-1, len(diff), m.env.Keys.PickerDown.Help().Key)))
	}
	body := lipgloss.NewStyle().MaxHeight(budget).Render(strings.Join(shown, "\n") + "\n\n" + tailText)
	k := m.env.Keys
	hints := []key.Binding{widgets.Relabel(k.Save, "save"), widgets.Relabel(k.Back, "keep editing")}
	accent := styles.ColorPrimary
	if len(m.stale) > 0 {
		hints = []key.Binding{widgets.Relabel(k.Select, "go to the first one"), widgets.Relabel(k.Back, "keep editing")}
		accent = styles.ColorError
	}
	if m.conflict != nil {
		hints = []key.Binding{widgets.Relabel(k.ReapplyEdits, "reload and reapply"), widgets.Relabel(k.Back, "keep editing")}
		accent = styles.ColorError
	}
	return widgets.Dialog{
		Title:    m.title,
		Subtitle: m.subtitle,
		Body:     body,
		Footer:   widgets.Hints(inner, hints...),
		Width:    m.width(),
		Accent:   accent,
	}.View()
}

func NewDiscardReleaseEditsModal(env Env, count int) ConfirmModal {
	return ConfirmModal{
		env:     env,
		title:   "Discard your changes?",
		message: "You have " + plural(count, "unsaved change") + ". Discarding them leaves the release pipeline as it is.",
		warning: "Your changes are lost.",
		action:  "discard",
		confirm: DiscardReleaseEditsConfirmedMsg{},
	}
}

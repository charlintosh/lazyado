package modals

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

type PullRequestVoteRequestMsg struct {
	PullRequest models.PullRequest
	Vote        models.Vote
}

type PullRequestCommentRequestMsg struct {
	PullRequest models.PullRequest
	ThreadID    int
	ParentID    int
	CommentID   int
	Anchor      *models.ThreadAnchor
	Text        string
}

type PullRequestCommentDeleteRequestMsg struct {
	PullRequest models.PullRequest
	ThreadID    int
	CommentID   int
}

func NewDeletePullRequestCommentModal(env Env, pr models.PullRequest, thread models.PullRequestThread, comment models.PullRequestComment) ConfirmModal {
	preview := ansi.Truncate(strings.Join(strings.Fields(comment.Content), " "), commentPreviewLen, "...")
	return ConfirmModal{
		env:     env,
		title:   "Delete comment",
		message: fmt.Sprintf("Are you sure you want to delete this comment?\n%q", preview),
		confirm: PullRequestCommentDeleteRequestMsg{PullRequest: pr, ThreadID: thread.ID, CommentID: comment.ID},
	}
}

func NewEditPullRequestCommentModal(env Env, pr models.PullRequest, thread models.PullRequestThread, comment models.PullRequestComment, members []models.TeamMember) PullRequestCommentModal {
	request := PullRequestCommentRequestMsg{PullRequest: pr, ThreadID: thread.ID, CommentID: comment.ID}
	m := newPullRequestCommentModal(env, "Edit comment", pullRequestLabel(pr), request, members)
	text := comment.Content
	for _, member := range members {
		if member.ID != "" {
			text = strings.ReplaceAll(text, "@<"+member.ID+">", "@"+member.DisplayName)
		}
	}
	m.area.SetValue(text)
	return m
}

func pullRequestLabel(pr models.PullRequest) string {
	return fmt.Sprintf("%d %s", pr.ID, pr.Title)
}

func NewVoteModal(env Env, pr models.PullRequest, current models.Vote, blocking int) ChoiceModal {
	options := make([]widgets.Option, len(models.Votes))
	for i, v := range models.Votes {
		options[i] = widgets.Option{Label: v.Action(), Value: strconv.Itoa(int(v)), Marked: v == current && v != models.VoteNone}
	}
	note := "Your vote: " + current.Label()
	if blocking > 0 {
		note += fmt.Sprintf(" · %d policies pending", blocking)
	}
	initial := ""
	if current != models.VoteNone {
		initial = strconv.Itoa(int(current))
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Vote",
		subtitle: pullRequestLabel(pr),
		note:     note,
		action:   "submit vote",
		options:  options,
		initial:  initial,
		noun:     "vote",
		nouns:    "votes",
		width:    styles.ModalWidthLG,
		rows:     len(models.Votes),
		choose: func(o widgets.Option) tea.Msg {
			vote, _ := strconv.Atoi(o.Value)
			return PullRequestVoteRequestMsg{PullRequest: pr, Vote: models.Vote(vote)}
		},
	})
}

type RepositoryChosenMsg struct {
	Repository models.Repository
	All        bool
}

type RepositoryFavoriteMsg struct {
	Repository models.Repository
	Favorite   bool
}

type RepositoryFavorites struct {
	Starred map[string]bool
	Local   map[string]bool
}

type RepositoryPickerModal struct {
	ChoiceModal
	repos     []models.Repository
	favorites RepositoryFavorites
	current   string
	single    bool
}

func NewRepositoryPickerModal(env Env, repos []models.Repository, favorites RepositoryFavorites, current string) RepositoryPickerModal {
	return newRepositoryPicker(env, repos, favorites, current, false)
}

func NewSingleRepositoryPickerModal(env Env, repos []models.Repository, favorites RepositoryFavorites, current string) RepositoryPickerModal {
	return newRepositoryPicker(env, repos, favorites, current, true)
}

func newRepositoryPicker(env Env, repos []models.Repository, favorites RepositoryFavorites, current string, single bool) RepositoryPickerModal {
	local := make(map[string]bool, len(favorites.Local))
	for id := range favorites.Local {
		local[id] = true
	}
	favorites.Local = local
	m := RepositoryPickerModal{repos: repos, favorites: favorites, current: current, single: single}
	m.ChoiceModal = m.build(env, current)
	return m
}

func (m RepositoryPickerModal) options() []widgets.Option {
	var favorites, rest []widgets.Option
	for _, r := range m.repos {
		o := widgets.Option{Label: r.Name, Value: r.ID, Marked: r.ID == m.current}
		switch {
		case m.favorites.Starred[strings.ToLower(r.ID)]:
			o.Detail = "starred in Azure DevOps"
			favorites = append(favorites, o)
		case m.favorites.Local[r.ID]:
			o.Detail = "favorite"
			favorites = append(favorites, o)
		default:
			rest = append(rest, o)
		}
	}
	options := append(favorites, rest...)
	if m.single {
		return options
	}
	return append(options, widgets.Option{
		Label: "All repositories", Value: models.AllRepositories, Detail: "whole project", Marked: m.current == models.AllRepositories,
	})
}

func (m RepositoryPickerModal) build(env Env, selected string) ChoiceModal {
	options := m.options()
	return newChoiceModal(env, choiceSpec{
		title:       "Repository",
		action:      "choose",
		options:     options,
		initial:     selected,
		search:      true,
		placeholder: "Type to filter repositories",
		noun:        "repository",
		nouns:       "repositories",
		width:       styles.ModalWidthLG,
		rows:        min(len(options), styles.PickerRows),
		hints:       []key.Binding{env.Keys.ToggleFavorite},
		choose: func(o widgets.Option) tea.Msg {
			if o.Value == models.AllRepositories {
				return RepositoryChosenMsg{All: true}
			}
			return RepositoryChosenMsg{Repository: models.Repository{ID: o.Value, Name: o.Label}}
		},
	})
}

func (m RepositoryPickerModal) toggleFavorite() (Modal, tea.Cmd) {
	o, ok := m.picker.Selected()
	if !ok || o.Value == models.AllRepositories || m.favorites.Starred[strings.ToLower(o.Value)] {
		return m, nil
	}
	favorite := !m.favorites.Local[o.Value]
	if favorite {
		m.favorites.Local[o.Value] = true
	} else {
		delete(m.favorites.Local, o.Value)
	}
	query := m.picker.Query()
	m.ChoiceModal = m.build(m.env, o.Value)
	if query != "" {
		m.picker.SetQuery(query)
		m.picker.Select(o.Value)
	}
	return m, send(RepositoryFavoriteMsg{Repository: models.Repository{ID: o.Value, Name: o.Label}, Favorite: favorite})
}

func (m RepositoryPickerModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && key.Matches(k, m.env.Keys.ToggleFavorite) {
		return m.toggleFavorite()
	}
	next, cmd := m.ChoiceModal.Update(msg)
	m.ChoiceModal = next.(ChoiceModal)
	return m, cmd
}

type WorkItemChosenMsg struct {
	Item models.WorkItem
}

type BuildChosenMsg struct {
	BuildID int
	Name    string
}

func NewPullRequestBuildsModal(env Env, pr models.PullRequest, builds []models.PullRequestCheck) ChoiceModal {
	options := make([]widgets.Option, len(builds))
	for i, b := range builds {
		options[i] = widgets.Option{Label: b.Name, Value: strconv.Itoa(i), Detail: checkStateLabel(b.State)}
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Builds",
		subtitle: pullRequestLabel(pr),
		action:   "open",
		options:  options,
		noun:     "build",
		nouns:    "builds",
		width:    styles.ModalWidthXL,
		rows:     min(len(options), styles.PickerRows),
		choose: func(o widgets.Option) tea.Msg {
			index, _ := strconv.Atoi(o.Value)
			return BuildChosenMsg{BuildID: builds[index].BuildID, Name: builds[index].Name}
		},
	})
}

func checkStateLabel(s models.CheckState) string {
	switch s {
	case models.CheckPassed:
		return "succeeded"
	case models.CheckFailed:
		return "failed"
	case models.CheckRunning:
		return "running"
	}
	return "pending"
}

func NewLinkedWorkItemsModal(env Env, pr models.PullRequest, items []models.WorkItem) ChoiceModal {
	options := make([]widgets.Option, len(items))
	for i, item := range items {
		options[i] = widgets.Option{Label: itemLabel(item), Value: strconv.Itoa(i), Detail: item.ShortType()}
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Linked work items",
		subtitle: pullRequestLabel(pr),
		action:   "open",
		options:  options,
		noun:     "work item",
		nouns:    "work items",
		width:    styles.ModalWidthXL,
		rows:     min(len(options), styles.PickerRows),
		choose: func(o widgets.Option) tea.Msg {
			index, _ := strconv.Atoi(o.Value)
			return WorkItemChosenMsg{Item: items[index]}
		},
	})
}

type PullRequestCommentModal struct {
	env        Env
	title      string
	subtitle   string
	request    PullRequestCommentRequestMsg
	area       textarea.Model
	members    []models.TeamMember
	mentions   widgets.Picker
	mentioning bool
	dismissed  string
	err        string
}

func pullRequestMentions(text string, members []models.TeamMember) string {
	sorted := append([]models.TeamMember(nil), members...)
	sort.SliceStable(sorted, func(i, j int) bool { return len(sorted[i].DisplayName) > len(sorted[j].DisplayName) })
	for _, member := range sorted {
		if member.ID != "" {
			text = replaceMentionWith(text, "@"+member.DisplayName, "@<"+member.ID+">")
		}
	}
	return text
}

func NewPullRequestCommentModal(env Env, pr models.PullRequest, anchor *models.ThreadAnchor, members []models.TeamMember) PullRequestCommentModal {
	title, subtitle := "Add comment", pullRequestLabel(pr)
	if anchor != nil {
		title, subtitle = "Comment on line", strings.TrimPrefix(anchor.FilePath, "/")+":"+anchor.Lines()
		if anchor.EndLine > anchor.Line {
			title = "Comment on lines"
		}
	}
	return newPullRequestCommentModal(env, title, subtitle, PullRequestCommentRequestMsg{PullRequest: pr, Anchor: anchor}, members)
}

func NewPullRequestReplyModal(env Env, pr models.PullRequest, thread models.PullRequestThread, comment models.PullRequestComment, members []models.TeamMember) PullRequestCommentModal {
	request := PullRequestCommentRequestMsg{PullRequest: pr, ThreadID: thread.ID}
	subtitle := pullRequestLabel(pr)
	if comment.ID == 0 && len(thread.Comments) > 0 {
		comment = thread.Comments[0]
	}
	if comment.ID != 0 {
		request.ParentID = comment.ID
		subtitle = "To " + comment.AuthorName
	}
	return newPullRequestCommentModal(env, "Reply", subtitle, request, members)
}

func newPullRequestCommentModal(env Env, title, subtitle string, request PullRequestCommentRequestMsg, members []models.TeamMember) PullRequestCommentModal {
	ta := textarea.New()
	ta.Placeholder = "Write a comment... (Markdown, @name to mention someone)"
	ta.CharLimit = styles.ContentCharLimitLG
	ta.ShowLineNumbers = false
	ta.Focus()
	options := make([]widgets.Option, len(members))
	for i, member := range members {
		options[i] = widgets.Option{Label: member.DisplayName, Value: member.ID}
	}
	m := PullRequestCommentModal{
		env: env, title: title, subtitle: subtitle, request: request, area: ta, members: members,
		mentions: widgets.NewPicker(env.Keys, options, widgets.PickerConfig{Rows: styles.MentionPickerRows}),
	}
	m.layout()
	return m
}

func (m PullRequestCommentModal) Init() tea.Cmd {
	return textarea.Blink
}

func (m PullRequestCommentModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		if m.mentioning {
			if next, handled := m.handleMentionKey(msg); handled {
				return next, nil
			}
		}
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Save):
			text := strings.TrimSpace(m.area.Value())
			if text == "" {
				m.err = "Comment cannot be empty"
				return m, nil
			}
			request := m.request
			request.Text = pullRequestMentions(text, m.members)
			return m, send(request)
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

func (m PullRequestCommentModal) handleMentionKey(msg tea.KeyPressMsg) (PullRequestCommentModal, bool) {
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
		return m, false
	}
	return m, true
}

func (m *PullRequestCommentModal) syncMentions() {
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

func (m PullRequestCommentModal) width() int {
	return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width)
}

func (m *PullRequestCommentModal) layout() {
	inner := widgets.DialogInnerWidth(m.width())
	m.area.SetWidth(inner)
	reserved := styles.MentionPickerRows + 3
	m.area.SetHeight(max(min(styles.TextareaHeightLG, widgets.DialogBodyHeight(m.env.Height, true)-reserved), 1))
	m.mentions.SetWidth(inner)
}

func (m PullRequestCommentModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	parts := []string{m.area.View()}
	k := m.env.Keys
	footer := widgets.Hints(inner, widgets.Relabel(k.Save, "post"), widgets.Relabel(k.Back, "cancel"))
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	switch {
	case m.mentioning:
		parts = append(parts, "", muted.Render("Mention"), m.mentions.View())
		footer = widgets.Hints(inner, widgets.Combine("move", k.PickerUp, k.PickerDown), widgets.Relabel(k.Select, "insert"), widgets.Relabel(k.Back, "dismiss"))
	case len(m.members) > 0:
		parts = append(parts, "", muted.Render("Tip: type @ to mention a team member"))
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

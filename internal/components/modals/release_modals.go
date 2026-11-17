package modals

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	releaseCommentLimit = 500
	labelWidth          = 14
)

type ReleaseApprovalRequestMsg struct {
	Approval models.ReleaseApproval
	Approve  bool
	Comment  string
}

type CancelDeploymentRequestMsg struct {
	ReleaseID int
	StageID   int
	Stage     string
	Comment   string
}

type DeployStageRequestMsg struct {
	ReleaseID int
	StageID   int
	Stage     string
	Comment   string
}

func fieldRows(rows [][2]string, width int) string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(labelWidth)
	value := lipgloss.NewStyle().Foreground(styles.ColorText).Width(max(width-labelWidth, 1))
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		if r[1] == "" {
			continue
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, label.Render(r[0]), value.Render(r[1])))
	}
	return strings.Join(lines, "\n")
}

func note(title, text string, accent lipgloss.Style, width int) string {
	return lipgloss.NewStyle().Width(width).Render(accent.Bold(true).Render(title) + " " + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(text))
}

type block struct {
	text string
	keep bool
}

func fitBlocks(blocks []block, budget int) string {
	height := func(bs []block) int {
		h := 0
		for i, b := range bs {
			if i > 0 {
				h++
			}
			h += lipgloss.Height(b.text)
		}
		return h
	}
	for height(blocks) > budget {
		drop := -1
		for i := len(blocks) - 1; i >= 0; i-- {
			if !blocks[i].keep {
				drop = i
				break
			}
		}
		if drop < 0 {
			break
		}
		blocks = append(blocks[:drop:drop], blocks[drop+1:]...)
	}
	parts := make([]string, len(blocks))
	for i, b := range blocks {
		parts[i] = b.text
	}
	return lipgloss.NewStyle().MaxHeight(max(budget, 1)).Render(strings.Join(parts, "\n"))
}

func commentInput(placeholder string, env Env) textinput.Model {
	in := newTextField("", placeholder, "", releaseCommentLimit, false).input
	in.Focus()
	sizeComment(&in, env)
	return in
}

func sizeComment(in *textinput.Model, env Env) {
	inner := widgets.DialogInnerWidth(widgets.DialogWidth(styles.ModalWidthXL, env.Width))
	in.SetWidth(max(inner-lipgloss.Width(inputPrompt)-1, 1))
}

type ApprovalModal struct {
	env      Env
	approval models.ReleaseApproval
	artifact string
	approve  bool
	onChoice bool
	comment  textinput.Model
	err      string
}

func NewApprovalModal(env Env, approval models.ReleaseApproval, release *models.Release) ApprovalModal {
	m := ApprovalModal{env: env, approval: approval, approve: true, onChoice: true, comment: commentInput("Why? Required to reject", env)}
	m.comment.Blur()
	if release != nil {
		m.artifact = release.ArtifactSummary()
	}
	return m
}

func (m ApprovalModal) Init() tea.Cmd { return nil }

func (m ApprovalModal) width() int { return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width) }

func (m ApprovalModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		sizeComment(&m.comment, m.env)
		return m, nil
	case tea.KeyPressMsg:
		k := m.env.Keys
		switch {
		case key.Matches(msg, k.Back):
			return m, closeModal
		case key.Matches(msg, k.Save):
			return m.submit()
		case key.Matches(msg, k.NextPanel), key.Matches(msg, k.PrevPanel), m.onChoice && key.Matches(msg, k.Select):
			return m.toggleFocus()
		case m.onChoice && (key.Matches(msg, k.Left) || key.Matches(msg, k.Right)):
			m.approve = !m.approve
			m.err = ""
			return m, nil
		case m.onChoice:
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m ApprovalModal) toggleFocus() (Modal, tea.Cmd) {
	m.onChoice = !m.onChoice
	if m.onChoice {
		m.comment.Blur()
		return m, nil
	}
	return m, m.comment.Focus()
}

func (m ApprovalModal) submit() (Modal, tea.Cmd) {
	comment := strings.TrimSpace(m.comment.Value())
	if !m.approve && comment == "" {
		m.err = "Add a comment to say why you reject it."
		return m, nil
	}
	return m, send(ReleaseApprovalRequestMsg{Approval: m.approval, Approve: m.approve, Comment: comment})
}

func (m ApprovalModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	a := m.approval
	approver := a.Approver.DisplayName
	if a.Group {
		approver += " (any one member)"
	}
	option := func(label string, on bool, c lipgloss.Style) string {
		if on {
			return c.Bold(true).Foreground(styles.ColorOnAccent).Padding(0, 1).Render(label)
		}
		return lipgloss.NewStyle().Background(styles.ColorChip).Foreground(styles.ColorLight).Padding(0, 1).Render(label)
	}
	decisionLabel := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	commentLabel := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	if m.onChoice {
		decisionLabel = decisionLabel.Foreground(styles.ColorPrimary).Bold(true)
	} else {
		commentLabel = commentLabel.Foreground(styles.ColorPrimary).Bold(true)
	}
	effect := "Approving lets the " + a.StageName + " deployment continue right away."
	if !m.approve {
		effect = "Rejecting stops " + a.StageName + ". Stages that wait on it don't run."
	}
	blocks := []block{
		{text: fieldRows([][2]string{{"Approvers", approver}, {"Requested", a.Created.Local().Format("Jan 2 15:04")}, {"Deploys", m.artifact}}, inner)},
		{text: decisionLabel.Render("Decision") + "\n" + option("Approve", m.approve, lipgloss.NewStyle().Background(styles.ColorSuccess)) + " " + option("Reject", !m.approve, lipgloss.NewStyle().Background(styles.ColorError)), keep: true},
		{text: commentLabel.Render("Comment") + "\n" + m.comment.View(), keep: true},
		{text: lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).Render(effect)},
	}
	if m.err != "" {
		blocks = append(blocks, block{text: lipgloss.NewStyle().Foreground(styles.ColorError).Width(inner).Render(m.err), keep: true})
	}
	k := m.env.Keys
	primary := widgets.Relabel(k.Save, "approve")
	if !m.approve {
		primary = widgets.Relabel(k.Save, "reject")
	}
	hints := []key.Binding{primary, widgets.Relabel(k.NextPanel, "next"), widgets.Relabel(k.Back, "cancel")}
	if m.onChoice {
		hints = append([]key.Binding{widgets.Combine("decide", k.Left, k.Right)}, hints...)
	}
	return widgets.Dialog{
		Title:    a.KindName() + " approval · " + a.StageName,
		Subtitle: a.ReleaseName + " · " + a.Definition,
		Body:     fitBlocks(blocks, widgets.DialogBodyHeight(m.env.Height, true)),
		Footer:   widgets.Hints(inner, hints...),
		Width:    m.width(),
	}.View()
}

type DeployStageModal struct {
	env     Env
	release models.Release
	stage   models.ReleaseStage
	comment textinput.Model
}

func NewDeployStageModal(env Env, release models.Release, stage models.ReleaseStage) DeployStageModal {
	return DeployStageModal{env: env, release: release, stage: stage, comment: commentInput("Optional, saved on the deployment", env)}
}

func (m DeployStageModal) Init() tea.Cmd { return textinput.Blink }

func (m DeployStageModal) width() int { return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width) }

func (m DeployStageModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		sizeComment(&m.comment, m.env)
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Save), key.Matches(msg, m.env.Keys.Select):
			return m, send(DeployStageRequestMsg{ReleaseID: m.release.ID, StageID: m.stage.ID, Stage: m.stage.Name, Comment: strings.TrimSpace(m.comment.Value())})
		}
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m DeployStageModal) verb() string {
	if m.stage.Attempt > 0 {
		return "Redeploy"
	}
	return "Deploy"
}

func (m DeployStageModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	s := m.stage
	stage := s.Name + " · " + s.State.Name()
	if s.Attempt > 0 {
		stage += fmt.Sprintf(" · attempt %d next", s.Attempt+1)
	}
	blocks := []block{{text: fieldRows([][2]string{{"Stage", stage}, {"Artifact", m.release.ArtifactSummary()}}, inner)}}
	if len(s.Approvers) > 0 {
		blocks = append(blocks, block{text: note("Approval", strings.Join(s.Approvers, ", ")+" must approve before it starts. The request is sent when you deploy.",
			lipgloss.NewStyle().Foreground(styles.ColorOrange), inner)})
	}
	if s.State == models.StageFailed {
		why := "the last attempt failed"
		if issues := models.RunIssues([]models.RunStage{s.RunStage()}); len(issues) > 0 {
			why = issues[0].Where + ": " + issues[0].Message
		}
		blocks = append(blocks, block{text: note("Last attempt failed", why, lipgloss.NewStyle().Foreground(styles.ColorError), inner)})
	}
	blocks = append(blocks,
		block{text: lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render("Comment") + "\n" + m.comment.View(), keep: true},
		block{text: lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).Render("Only " + s.Name + " deploys. Later stages follow their own triggers.")})
	k := m.env.Keys
	return widgets.Dialog{
		Title:    m.verb() + " " + s.Name,
		Subtitle: m.release.Name + " · " + m.release.Definition,
		Body:     fitBlocks(blocks, widgets.DialogBodyHeight(m.env.Height, true)),
		Footer:   widgets.Hints(inner, widgets.Relabel(k.Save, strings.ToLower(m.verb())), widgets.Relabel(k.Back, "cancel")),
		Width:    m.width(),
	}.View()
}

type CancelDeploymentModal struct {
	env     Env
	release models.Release
	stage   models.ReleaseStage
	comment textinput.Model
}

func NewCancelDeploymentModal(env Env, release models.Release, stage models.ReleaseStage) CancelDeploymentModal {
	return CancelDeploymentModal{env: env, release: release, stage: stage, comment: commentInput("Optional, saved on the deployment", env)}
}

func (m CancelDeploymentModal) Init() tea.Cmd { return textinput.Blink }

func (m CancelDeploymentModal) width() int {
	return widgets.DialogWidth(styles.ModalWidthXL, m.env.Width)
}

func (m CancelDeploymentModal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.env = m.env.resized(msg)
		sizeComment(&m.comment, m.env)
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.env.Keys.Back):
			return m, closeModal
		case key.Matches(msg, m.env.Keys.Save), key.Matches(msg, m.env.Keys.Select):
			return m, send(CancelDeploymentRequestMsg{ReleaseID: m.release.ID, StageID: m.stage.ID, Stage: m.stage.Name, Comment: strings.TrimSpace(m.comment.Value())})
		}
	}
	var cmd tea.Cmd
	m.comment, cmd = m.comment.Update(msg)
	return m, cmd
}

func (m CancelDeploymentModal) View() string {
	inner := widgets.DialogInnerWidth(m.width())
	s := m.stage
	effect := "Running tasks stop and " + s.Name + " ends as canceled. Stages that wait on it don't run."
	if _, waiting := s.PendingApproval(); waiting {
		effect = "The pending approval is withdrawn and " + s.Name + " ends as canceled. Stages that wait on it don't run."
	}
	blocks := []block{
		{text: fieldRows([][2]string{{"Stage", s.Name + " · " + s.State.Name()}, {"Artifact", m.release.ArtifactSummary()}}, inner)},
		{text: note("Cancel", effect, lipgloss.NewStyle().Foreground(styles.ColorError), inner), keep: true},
		{text: lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true).Render("Comment") + "\n" + m.comment.View(), keep: true},
		{text: lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(inner).Render("You can deploy " + s.Name + " again afterwards with " + m.env.Keys.DeployStage.Help().Key + ".")},
	}
	k := m.env.Keys
	return widgets.Dialog{
		Title:    "Cancel the " + s.Name + " deployment",
		Subtitle: m.release.Name + " · " + m.release.Definition,
		Body:     fitBlocks(blocks, widgets.DialogBodyHeight(m.env.Height, true)),
		Footer:   widgets.Hints(inner, widgets.Relabel(k.Save, "cancel deployment"), widgets.Relabel(k.Back, "keep it running")),
		Width:    m.width(),
		Accent:   styles.ColorError,
	}.View()
}

var errNoVersion = errors.New("Choose a version for every build artifact")

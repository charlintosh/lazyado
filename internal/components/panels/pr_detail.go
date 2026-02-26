package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PRDetailPanel struct {
	item     *models.PullRequest
	viewport viewport.Model
	styles   styles.Styles
	keys     keys.KeyMap
	width    int
	height   int
	focused  bool
	ready    bool
}

type ClosePRDetailMsg struct{}

type PRApproveRequestMsg struct {
	PR   models.PullRequest
	Vote models.PRVote
}

func NewPRDetailPanel(s styles.Styles, k keys.KeyMap) PRDetailPanel {
	return PRDetailPanel{
		styles: s,
		keys:   k,
	}
}

func (p *PRDetailPanel) SetItem(item *models.PullRequest) {
	p.item = item
	if p.ready {
		p.viewport.SetContent(p.buildContent())
		p.viewport.GotoTop()
	}
}

func (p *PRDetailPanel) Item() *models.PullRequest {
	return p.item
}

func (p *PRDetailPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	if !p.ready {
		p.viewport = viewport.New(width-2, height-2)
		p.viewport.Style = lipgloss.NewStyle()
		p.ready = true
	} else {
		p.viewport.Width = width - 2
		p.viewport.Height = height - 2
	}

	if p.item != nil {
		p.viewport.SetContent(p.buildContent())
	}
}

func (p *PRDetailPanel) SetFocused(focused bool) {
	p.focused = focused
}

func (p PRDetailPanel) Update(msg tea.Msg) (PRDetailPanel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, p.keys.Back) {
			return p, func() tea.Msg { return ClosePRDetailMsg{} }
		}
	case tea.MouseMsg:
		if p.ready {
			p.viewport, cmd = p.viewport.Update(msg)
		}
		return p, cmd
	}

	if p.ready {
		p.viewport, cmd = p.viewport.Update(msg)
	}
	return p, cmd
}

func (p PRDetailPanel) View() string {
	if p.item == nil {
		return ""
	}

	panel := p.styles.PanelInactive
	if p.focused {
		panel = p.styles.PanelActive
	}

	return panel.
		Width(p.width).
		Height(p.height).
		Render(p.viewport.View())
}

func (p PRDetailPanel) buildContent() string {
	if p.item == nil {
		return ""
	}

	pr := p.item
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorText)
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Width(16)
	valueStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorAccent).MarginTop(1)
	branchStyle := lipgloss.NewStyle().Foreground(styles.ColorSecondary)
	draftStyle := lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)

	b.WriteString(titleStyle.Render(fmt.Sprintf("#%d %s", pr.ID, pr.Title)))
	b.WriteString("\n")

	if pr.IsDraft {
		b.WriteString(draftStyle.Render("[DRAFT]"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Details"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Status") + valueStyle.Render(prStatusStyled(pr, p.styles)))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Repository") + valueStyle.Render(pr.Repository.Name))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Source") + branchStyle.Render(pr.ShortSourceBranch()))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Target") + branchStyle.Render(pr.ShortTargetBranch()))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Created by") + valueStyle.Render(pr.CreatedBy))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Created") + valueStyle.Render(formatPRTime(pr.CreationDate)))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Merge status") + valueStyle.Render(prMergeStatusLabel(pr.MergeStatus)))
	b.WriteString("\n")

	if pr.Description != "" {
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("Description"))
		b.WriteString("\n")
		b.WriteString(valueStyle.Render(pr.Description))
		b.WriteString("\n")
	}

	if len(pr.Reviewers) > 0 {
		b.WriteString("\n")
		b.WriteString(sectionStyle.Render("Reviewers"))
		b.WriteString("\n")
		for _, r := range pr.Reviewers {
			if r.IsContainer {
				continue
			}
			vote := models.PRVote(r.Vote)
			voteLabel := prVoteStyledLabel(vote, p.styles)
			reqLabel := ""
			if r.IsRequired {
				reqLabel = lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(" (required)")
			}
			b.WriteString(fmt.Sprintf("  %s %s%s\n", voteLabel, r.DisplayName, reqLabel))
		}
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	b.WriteString(helpStyle.Render("enter:open in browser  a:approve  r:reject  w:wait  0:reset vote  Esc:back"))

	return b.String()
}

func prStatusStyled(pr *models.PullRequest, s styles.Styles) string {
	switch pr.Status {
	case models.PRStatusActive:
		return lipgloss.NewStyle().Foreground(styles.ColorBlue).Render("Active")
	case models.PRStatusCompleted:
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("Completed")
	case models.PRStatusAbandoned:
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("Abandoned")
	default:
		return string(pr.Status)
	}
}

func prVoteStyledLabel(vote models.PRVote, s styles.Styles) string {
	switch {
	case vote >= 10:
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ Approved")
	case vote == 5:
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓ Approved*")
	case vote == 0:
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("- No vote")
	case vote == -5:
		return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("⏳ Waiting")
	case vote <= -10:
		return lipgloss.NewStyle().Foreground(styles.ColorError).Render("✗ Rejected")
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("? Unknown")
	}
}

func prMergeStatusLabel(status models.PRMergeStatus) string {
	switch status {
	case models.PRMergeSucceeded:
		return "Succeeded"
	case models.PRMergeConflicts:
		return "Conflicts"
	case models.PRMergeQueued:
		return "Queued"
	case models.PRMergeRejectedByPolicy:
		return "Rejected by policy"
	case models.PRMergeFailure:
		return "Failed"
	default:
		return "Not set"
	}
}

func formatPRTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006 15:04")
}

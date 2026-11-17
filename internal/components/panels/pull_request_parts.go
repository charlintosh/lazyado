package panels

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	branchJoin     = " into "
	draftLabel     = "DRAFT"
	noDescription  = "No description provided."
	loadingChecks  = "Checking policies"
	noChecks       = "No policies or status checks apply."
	noReviewers    = "No reviewers yet."
	requiredLabel  = "Required"
	optionalLabel  = "Optional"
	conflictsLabel = "Merge conflicts"
	groupNote      = "(group)"
)

func voteColor(v models.Vote) (color.Color, bool) {
	switch v {
	case models.VoteApproved, models.VoteApprovedWithSuggestions:
		return styles.ColorSuccess, true
	case models.VoteWaitingForAuthor:
		return styles.ColorOrange, true
	case models.VoteRejected:
		return styles.ColorError, true
	}
	return styles.ColorChip, false
}

func VoteChip(r models.Reviewer, mine bool) string {
	bg, voted := voteColor(r.Vote)
	fg := styles.ColorText
	if voted {
		fg = styles.ColorOnAccent
	}
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(mine).Underline(mine).Padding(0, 1).
		Render(widgets.Initials(r.Name()))
}

func VoteText(v models.Vote) string {
	c, voted := voteColor(v)
	if !voted {
		return Muted(v.Label())
	}
	return lipgloss.NewStyle().Foreground(c).Render(v.Label())
}

func PullRequestBadge(pr models.PullRequest) string {
	switch {
	case pr.Status == models.PullRequestCompleted:
		return Badge("COMPLETED", styles.ColorSuccess, styles.ColorOnAccent)
	case pr.Status == models.PullRequestAbandoned:
		return Badge("ABANDONED", styles.ColorChip, styles.ColorLight)
	case pr.IsDraft:
		return Badge(draftLabel, styles.ColorChip, styles.ColorLight)
	}
	return Badge("ACTIVE", styles.ColorPrimary, styles.ColorOnAccent)
}

func PullRequestBranches(pr models.PullRequest) string {
	accent := lipgloss.NewStyle().Foreground(styles.ColorAccent)
	return accent.Render(pr.SourceBranch) + Muted(branchJoin) + accent.Render(pr.TargetBranch)
}

func checkColor(s models.CheckState) color.Color {
	switch s {
	case models.CheckPassed:
		return styles.ColorSuccess
	case models.CheckFailed:
		return styles.ColorError
	case models.CheckRunning:
		return styles.ColorPrimary
	}
	return styles.ColorOrange
}

func sectionTitle(title string, width int) string {
	rule := lipgloss.NewStyle().Foreground(styles.ColorChip).Render(strings.Repeat(ruleGlyph, max(width, 0)))
	return lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(title) + "\n" + rule
}

func checkRows(checks []models.PullRequestCheck, width int) []string {
	label := styles.PullRequestCheckLabel
	text := max(width-label-1, 1)
	var rows []string
	for _, c := range checks {
		state := lipgloss.NewStyle().Foreground(checkColor(c.State)).Bold(true).Width(label).Render(c.State.Label())
		rows = append(rows, state+" "+ansi.Truncate(c.Name, text, "…"))
		if c.Detail != "" {
			rows = append(rows, strings.Repeat(" ", label+1)+Muted(ansi.Truncate(c.Detail, text, "…")))
		}
	}
	return rows
}

func RenderChecks(checks []models.PullRequestCheck, loaded bool, width int, spinner string) string {
	lines := []string{sectionTitle("Policies", width)}
	switch {
	case !loaded:
		return strings.Join(append(lines, Loading(spinner, loadingChecks)), "\n")
	case len(checks) == 0:
		return strings.Join(append(lines, Muted(noChecks)), "\n")
	}
	var required, optional []models.PullRequestCheck
	for _, c := range checks {
		if c.Required {
			required = append(required, c)
		} else {
			optional = append(optional, c)
		}
	}
	if len(required) > 0 {
		lines = append(lines, Muted(requiredLabel))
		lines = append(lines, checkRows(required, width)...)
	}
	if len(optional) > 0 {
		lines = append(lines, Muted(optionalLabel))
		lines = append(lines, checkRows(optional, width)...)
	}
	return strings.Join(lines, "\n")
}

func reviewerRows(reviewers []models.Reviewer, me models.Identity, width int) []string {
	rows := make([]string, 0, len(reviewers))
	for _, r := range reviewers {
		mine := me.Is(r.ID, r.UniqueName)
		chip := VoteChip(r, mine)
		vote := VoteText(r.Vote)
		room := max(width-lipgloss.Width(chip)-lipgloss.Width(vote)-2, 1)
		name := r.Name()
		switch {
		case mine:
			name += " " + youNote
		case r.Group:
			name += " " + groupNote
		}
		name = ansi.Truncate(name, room, "…")
		gap := strings.Repeat(" ", max(room-lipgloss.Width(name), 0))
		rows = append(rows, chip+" "+name+gap+" "+vote)
	}
	return rows
}

func RenderReviewers(pr models.PullRequest, me models.Identity, width int) string {
	lines := []string{sectionTitle("Reviewers", width)}
	if len(pr.Reviewers) == 0 {
		return strings.Join(append(lines, Muted(noReviewers)), "\n")
	}
	var required, optional []models.Reviewer
	for _, r := range pr.Reviewers {
		if r.Required {
			required = append(required, r)
		} else {
			optional = append(optional, r)
		}
	}
	if len(required) > 0 {
		lines = append(lines, Muted(requiredLabel))
		lines = append(lines, reviewerRows(required, me, width)...)
	}
	if len(optional) > 0 {
		lines = append(lines, Muted(optionalLabel))
		lines = append(lines, reviewerRows(optional, me, width)...)
	}
	return strings.Join(lines, "\n")
}

func PullRequestSummary(pr models.PullRequest, width int) string {
	title := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Width(width).Render(pr.Title)
	id := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render(fmt.Sprintf("%d", pr.ID))
	meta := PullRequestBadge(pr) + " " + id + " " + Muted(pr.Author.DisplayName+" · "+formatCommentDate(pr.CreatedDate))
	lines := []string{title, Line(meta, width), lipgloss.NewStyle().Width(width).Render(Muted(pr.RepositoryName+" · ") + PullRequestBranches(pr))}
	if pr.HasConflicts() {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColorError).Bold(true).Render(conflictsLabel))
	}
	return strings.Join(lines, "\n")
}

func PullRequestDescription(pr models.PullRequest, width int) string {
	if strings.TrimSpace(pr.Description) == "" {
		return Muted(noDescription)
	}
	return strings.TrimRight(components.RenderMarkdown(pr.Description, width), "\n")
}

type PullRequestPreviewProps struct {
	PullRequest  *models.PullRequest
	Checks       []models.PullRequestCheck
	ChecksLoaded bool
	Spinner      string
	Me           models.Identity
	Width        int
	Height       int
}

func RenderPullRequestPreview(p PullRequestPreviewProps) string {
	if p.PullRequest == nil {
		return Block(Muted("Select a pull request to preview it."), p.Width, p.Height)
	}
	pr := *p.PullRequest
	parts := []string{
		PullRequestSummary(pr, p.Width),
		RenderChecks(p.Checks, p.ChecksLoaded, p.Width, p.Spinner),
		RenderReviewers(pr, p.Me, p.Width),
		sectionTitle("Description", p.Width) + "\n" + PullRequestDescription(pr, p.Width),
	}
	return Block(strings.Join(parts, "\n\n"), p.Width, p.Height)
}

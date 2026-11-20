package panels

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

const (
	stageChipWidth   = 14
	stageChipGap     = " "
	stageStripGap    = 3
	stageStripArrows = 4
	releaseLayout    = "Jan 2 15:04"
)

type OpenReleaseMsg struct {
	ReleaseID int
}

type BrowseReleaseMsg struct {
	DefinitionID int
	ReleaseID    int
	URL          string
}

type ApproveReleaseMsg struct {
	Approval models.ReleaseApproval
}

type DeployStageMsg struct {
	Release models.Release
	Stage   models.ReleaseStage
}

type CancelDeploymentMsg struct {
	Release models.Release
	Stage   models.ReleaseStage
}

type CreateReleaseMsg struct {
	DefinitionID int
	Name         string
}

type ReleaseNoticeMsg struct {
	Text string
}

func stageColors(s models.StageState) (color.Color, color.Color) {
	switch s {
	case models.StageSucceeded:
		return styles.ColorSuccess, styles.ColorOnAccent
	case models.StagePartial:
		return styles.ColorWarning, styles.ColorOnAccent
	case models.StageFailed, models.StageRejected:
		return styles.ColorError, styles.ColorOnAccent
	case models.StageDeploying:
		return styles.ColorPrimary, styles.ColorOnAccent
	case models.StageWaitingApproval:
		return styles.ColorOrange, styles.ColorOnAccent
	case models.StageCanceled:
		return styles.ColorMuted, styles.ColorOnAccent
	case models.StageQueued, models.StageScheduled:
		return styles.ColorChip, styles.ColorLight
	}
	return styles.ColorChip, styles.ColorTextMuted
}

func StageBadge(s models.StageState) string {
	label := s.Label()
	label += strings.Repeat(" ", max(runBadgeWidth-len(label), 0))
	bg, fg := stageColors(s)
	return Badge(label, bg, fg)
}

func StageChip(s models.ReleaseStage, width int) string {
	text := s.Name
	if label := s.State.Label(); label != "" {
		text = label + " " + s.Name
	}
	text = ansi.Truncate(text, max(width-2, 1), "…")
	bg, fg := stageColors(s.State)
	return lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(s.State != models.StageNotDeployed).
		Padding(0, 1).Width(width).Render(text)
}

func StageChips(stages []models.ReleaseStage, width int, base lipgloss.Style) string {
	var out string
	for i, s := range stages {
		chip := StageChip(s, stageChipWidth)
		next := out
		if i > 0 {
			next += base.Render(stageChipGap)
		}
		next += chip
		if lipgloss.Width(next) > width {
			if lipgloss.Width(out)+2 <= width {
				out += base.Foreground(styles.ColorTextMuted).Render(" …")
			}
			break
		}
		out = next
	}
	return out
}

func StageStateText(s models.StageState) string {
	bg, _ := stageColors(s)
	if bg == styles.ColorChip {
		bg = styles.ColorTextMuted
	}
	return lipgloss.NewStyle().Foreground(bg).Render(s.Name())
}

func approvalBadge(a models.ReleaseApproval, mine bool) string {
	label := a.KindLabel()
	label += strings.Repeat(" ", max(runBadgeWidth-len(label), 0))
	if mine {
		return Badge(label, styles.ColorOrange, styles.ColorOnAccent)
	}
	return Badge(label, styles.ColorChip, styles.ColorLight)
}

func approverName(a models.ReleaseApproval) string {
	if a.Group {
		return a.Approver.DisplayName + " (any one member)"
	}
	return a.Approver.DisplayName
}

func releaseStamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(releaseLayout)
}

type ApprovalPreviewProps struct {
	Approval *models.ReleaseApproval
	Mine     bool
	Release  *models.Release
	Spinner  string
	Width    int
	Height   int
}

func RenderApprovalPreview(p ApprovalPreviewProps) string {
	a := p.Approval
	if a == nil {
		return Muted("Select an approval.")
	}
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	you := Muted("can't approve this one")
	if p.Mine {
		you = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("can approve")
	}
	rows := keptRows([][2]string{
		{"Approvers", approverName(*a)},
		{"You", you},
		{"Requested", releaseStamp(a.Created)},
	})
	parts := []string{bold.Render(a.KindName() + " approval · " + a.StageName), Muted(a.ReleaseName + " · " + a.Definition), "", FieldRows(rows, fieldLabel, p.Width)}
	switch {
	case p.Release == nil:
		parts = append(parts, "", Loading(p.Spinner, "Loading release"))
	default:
		parts = append(parts, "", Line(Muted(p.Release.ArtifactSummary()), p.Width), "", sectionTitle("Stages", p.Width))
		for _, s := range p.Release.Stages {
			parts = append(parts, spreadLine(StageBadge(s.State)+" "+s.Name, Muted(s.State.Name()), p.Width))
		}
	}
	return Block(strings.Join(parts, "\n"), p.Width, p.Height)
}

func stageBox(s models.ReleaseStage, selected bool, width int) string {
	border := styles.ColorBorder
	if selected {
		border = styles.ColorPrimary
	}
	inner := max(width-4, 1)
	status := StageStateText(s.State)
	if a, ok := s.PendingApproval(); ok {
		status = lipgloss.NewStyle().Foreground(styles.ColorOrange).Render(strings.ToLower(a.KindName()) + " approval")
	}
	lines := []string{
		Line(StageBadge(s.State)+" "+lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(s.Name), inner),
		Line(status, inner),
		Line(Muted(s.Trigger), inner),
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1).Width(width).
		Render(strings.Join(lines, "\n"))
}

func stagePick(s models.ReleaseStage, selected bool) string {
	dot, _ := stageColors(s.State)
	if dot == styles.ColorChip {
		dot = styles.ColorTextMuted
	}
	if !selected {
		return lipgloss.NewStyle().Foreground(dot).Render("●") + " " + lipgloss.NewStyle().Foreground(styles.ColorLight).Render(s.Name)
	}
	base := lipgloss.NewStyle().Background(styles.ColorHighlight)
	return base.Foreground(dot).Render(" ●") + base.Foreground(styles.ColorText).Bold(true).Render(" "+s.Name) +
		base.Foreground(styles.ColorPrimaryLight).Render(" · "+s.State.Name()+" ")
}

func stageStrip(stages []models.ReleaseStage, selected, width int, hint string) string {
	if len(stages) == 0 || width <= 0 {
		return ""
	}
	selected = min(max(selected, 0), len(stages)-1)
	picks := make([]string, len(stages))
	for i, s := range stages {
		picks[i] = stagePick(s, i == selected)
	}
	lead, tail := " "+Muted("Stage")+"  ", Muted(hint)+" "
	budget := width - lipgloss.Width(lead) - lipgloss.Width(tail) - stageStripArrows
	lo, hi := selected, selected+1
	used := lipgloss.Width(picks[selected])
	fits := func(i int) bool { return used+stageStripGap+lipgloss.Width(picks[i]) <= budget }
	for grew := true; grew; {
		grew = false
		if hi < len(picks) && fits(hi) {
			used += stageStripGap + lipgloss.Width(picks[hi])
			hi, grew = hi+1, true
		}
		if lo > 0 && fits(lo-1) {
			used += stageStripGap + lipgloss.Width(picks[lo-1])
			lo, grew = lo-1, true
		}
	}
	row := strings.Join(picks[lo:hi], strings.Repeat(" ", stageStripGap))
	if lo > 0 {
		row = Muted("‹ ") + row
	}
	if hi < len(picks) {
		row += Muted(" ›")
	}
	return spreadLine(lead+row, tail, width)
}

func stageBoxes(stages []models.ReleaseStage, selected, width int) (string, int) {
	boxWidth := min(max(width/max(len(stages), 1)-1, 20), 28)
	perRow := max((width+1)/(boxWidth+1), 1)
	var rows []string
	for start := 0; start < len(stages); start += perRow {
		var boxes []string
		for i := start; i < min(start+perRow, len(stages)); i++ {
			if i > start {
				boxes = append(boxes, " ")
			}
			boxes = append(boxes, stageBox(stages[i], i == selected, boxWidth))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, boxes...))
	}
	out := strings.Join(rows, "\n")
	return out, lipgloss.Height(out)
}

func durationOf(start, end time.Time) string {
	if start.IsZero() || end.IsZero() {
		return ""
	}
	return models.FormatDuration(end.Sub(start))
}

func plural1(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

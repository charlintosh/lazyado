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
	runBadgeWidth = 4
	stripCell     = "  "
	fieldLabel    = 14
	loadingStages = "Loading stages"
	noStages      = "No stages yet."
)

func runStateColor(s models.RunState) color.Color {
	switch s {
	case models.RunSucceeded:
		return styles.ColorSuccess
	case models.RunFailed:
		return styles.ColorError
	case models.RunRunning:
		return styles.ColorPrimary
	case models.RunPartial:
		return styles.ColorWarning
	case models.RunCanceled:
		return styles.ColorMuted
	}
	return styles.ColorChip
}

func RunBadge(s models.RunState) string {
	label := s.Label()
	label += strings.Repeat(" ", max(runBadgeWidth-len(label), 0))
	fg := styles.ColorOnAccent
	if runStateColor(s) == styles.ColorChip {
		fg = styles.ColorLight
	}
	return Badge(label, runStateColor(s), fg)
}

func RunStateText(s models.RunState, text string) string {
	c := runStateColor(s)
	if c == styles.ColorChip {
		c = styles.ColorTextMuted
	}
	return lipgloss.NewStyle().Foreground(c).Render(text)
}

func runStrip(runs []models.PipelineRun) string {
	cells := make([]string, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		cells = append(cells, lipgloss.NewStyle().Background(runStateColor(runs[i].State())).Render(stripCell))
	}
	return strings.Join(cells, " ")
}

func ShortAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("Jan 2")
}

func RunDuration(r models.PipelineRun, now time.Time) string {
	switch r.State() {
	case models.RunQueued:
		return "queued"
	case models.RunUnknown:
		return ""
	}
	return models.FormatDuration(r.Duration(now))
}

func RecordDuration(r models.TimelineRecord, now time.Time) string {
	if r.Started.IsZero() {
		return ""
	}
	return models.FormatDuration(r.Duration(now))
}

func runNumber(r models.PipelineRun) string {
	return lipgloss.NewStyle().Foreground(styles.ColorAccent).Bold(true).Render("#" + r.Number)
}

func RunMeta(r models.PipelineRun) string {
	parts := []string{r.ReasonLabel()}
	if b := r.Branch(); b != "" {
		parts = append(parts, b)
	}
	if c := r.ShortCommit(); c != "" {
		parts = append(parts, c)
	}
	if r.RequestedFor.DisplayName != "" {
		parts = append(parts, r.RequestedFor.DisplayName)
	}
	return strings.Join(parts, " · ")
}

func spreadLine(left, right string, width int) string {
	room := width - lipgloss.Width(right)
	if room < 1 {
		return Line(left, width)
	}
	left = ansi.Truncate(left, max(room-1, 0), "…")
	return left + strings.Repeat(" ", max(room-lipgloss.Width(left), 0)) + right
}

func RenderStageRows(stages []models.RunStage, now time.Time, width int) []string {
	rows := make([]string, 0, len(stages))
	for _, s := range stages {
		done, total := s.JobCounts()
		detail := fmt.Sprintf("%d/%d jobs", done, total)
		if d := RecordDuration(s.TimelineRecord, now); d != "" {
			detail += " · " + d
		}
		rows = append(rows, spreadLine(RunBadge(s.State)+" "+s.Name, Muted(detail), width))
	}
	return rows
}

type PipelinePreviewProps struct {
	Pipeline      *models.Pipeline
	Run           *models.PipelineRun
	History       []models.PipelineRun
	HistoryLoaded bool
	Stages        []models.RunStage
	StagesLoaded  bool
	ShowPipeline  bool
	Spinner       string
	Width         int
	Height        int
}

func RenderPipelinePreview(p PipelinePreviewProps) string {
	if p.Pipeline == nil && p.Run == nil {
		return Muted("Select a pipeline or a run.")
	}
	now := time.Now()
	bold := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true)
	var sections []string
	if p.Pipeline != nil && p.ShowPipeline {
		head := []string{Line(bold.Render(p.Pipeline.Name), p.Width), Muted(p.Pipeline.FolderLabel())}
		if p.HistoryLoaded && len(p.History) > 0 {
			stats := models.StatsOf(p.History)
			summary := fmt.Sprintf("%d of %d passed", stats.Passed, stats.Completed)
			if stats.Completed > 0 {
				summary += " · median " + models.FormatDuration(stats.Median)
			}
			head = append(head, "", bold.Render(fmt.Sprintf("Last %d runs", len(p.History))), runStrip(p.History), Muted(summary))
		} else if !p.HistoryLoaded {
			head = append(head, "", Loading(p.Spinner, "Loading recent runs"))
		}
		sections = append(sections, strings.Join(head, "\n"))
	}
	if r := p.Run; r != nil {
		title := "Run " + runNumber(*r)
		if p.ShowPipeline {
			title = "Latest run " + runNumber(*r)
		}
		rows := [][2]string{
			{"State", RunStateText(r.State(), r.State().Name())},
			{"Trigger", r.ReasonLabel()},
			{"Branch", r.Branch()},
			{"Commit", r.ShortCommit()},
			{"Requested by", r.RequestedFor.DisplayName},
			{"Queued", ShortAgo(r.When())},
			{"Duration", RunDuration(*r, now)},
		}
		if !p.ShowPipeline {
			rows = append([][2]string{{"Pipeline", r.PipelineName}}, rows...)
		}
		var kept [][2]string
		for _, row := range rows {
			if row[1] != "" {
				kept = append(kept, row)
			}
		}
		lines := []string{bold.Render(title), Line(r.Title(), p.Width), "", FieldRows(kept, fieldLabel, p.Width)}
		sections = append(sections, strings.Join(lines, "\n"))
		stages := []string{sectionTitle("Stages", p.Width)}
		switch {
		case !p.StagesLoaded:
			stages = append(stages, Loading(p.Spinner, loadingStages))
		case len(p.Stages) == 0:
			stages = append(stages, Muted(noStages))
		default:
			stages = append(stages, RenderStageRows(p.Stages, now, p.Width)...)
		}
		sections = append(sections, strings.Join(stages, "\n"))
	}
	return Block(strings.Join(sections, "\n\n"), p.Width, p.Height)
}

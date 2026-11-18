package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func manyStages(n int) []models.RunStage {
	start := time.Now().Add(-time.Hour)
	stages := make([]models.RunStage, n)
	for i := range stages {
		stages[i] = models.RunStage{TimelineRecord: models.TimelineRecord{ID: "s", Name: "Deploy stage with a rather long name", State: models.RunQueued}}
	}
	stages[0].State, stages[0].Started, stages[0].Finished = models.RunSucceeded, start, start.Add(time.Minute)
	stages[n-3].State, stages[n-3].Started = models.RunRunning, start
	return stages
}

func TestPickStagesKeepsWhatMattersInRunOrder(t *testing.T) {
	stages := manyStages(20)
	stages[5].State, stages[5].Started, stages[5].Finished = models.RunFailed, time.Now().Add(-time.Hour), time.Now()
	picked := pickStages(stages)
	want := []int{0, 1, 5, 17}
	if len(picked) != len(want) {
		t.Fatalf("picked %v, want %v", picked, want)
	}
	for i := range want {
		if picked[i] != want[i] {
			t.Fatalf("picked %v, want %v", picked, want)
		}
	}
	if more := moreStages(stages, picked); more != "+ 16 more queued" {
		t.Errorf("more line = %q", more)
	}
}

func TestStageStripFitsTheWidth(t *testing.T) {
	states := StageStates(manyStages(20))
	if got := ansi.Strip(StageStrip(states, 60)); got != strings.TrimSpace(got) || strings.Count(got, " ") != 19 {
		t.Errorf("wide strip should space the cells: %q", got)
	}
	if got := ansi.Strip(StageStrip(states, 30)); len([]rune(got)) != 20 {
		t.Errorf("narrow strip should use one column per stage: %q", got)
	}
	if got := ansi.Strip(StageStrip(StageStates(manyStages(40)), 30)); !strings.Contains(got, "✓ 1") || !strings.Contains(got, "○ 38") {
		t.Errorf("a strip wider than the pane should show counts: %q", got)
	}
}

func TestPreviewStagesStayShortForLargeRuns(t *testing.T) {
	run := models.PipelineRun{ID: 1, Number: "1", Status: "inProgress"}
	out := ansi.Strip(RenderPipelinePreview(PipelinePreviewProps{Run: &run, Stages: manyStages(40), StagesLoaded: true, Width: 50, Height: 30}))
	if n := strings.Count(out, "Deploy stage with"); n > previewStages {
		t.Errorf("the preview lists %d stages, at most %d expected:\n%s", n, previewStages, out)
	}
	if !strings.Contains(out, "more queued") {
		t.Errorf("the preview must fold the other stages:\n%s", out)
	}
}

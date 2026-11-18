package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charmbracelet/x/ansi"
)

const highlightSGR = "2;43;58;74"

type cellPaint int

const (
	paintNone cellPaint = iota
	paintHighlight
	paintOther
)

type cell struct {
	paint cellPaint
	blank bool
}

func paintedCells(line string) []cell {
	var cells []cell
	bg := paintNone
	for len(line) > 0 {
		if strings.HasPrefix(line, "\x1b[") {
			end := strings.IndexByte(line, 'm')
			if end < 0 {
				break
			}
			bg = applySGR(line[2:end], bg)
			line = line[end+1:]
			continue
		}
		if line[0] == '\x1b' {
			next := strings.IndexAny(line[1:], "\x1b")
			if next < 0 {
				break
			}
			line = line[next+1:]
			continue
		}
		cluster, width := ansi.FirstGraphemeCluster(line, ansi.GraphemeWidth)
		for range width {
			cells = append(cells, cell{paint: bg, blank: cluster == " "})
		}
		line = line[len(cluster):]
	}
	return cells
}

func applySGR(params string, bg cellPaint) cellPaint {
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		switch {
		case parts[i] == "" || n == 0:
			bg = paintNone
		case n == 49:
			bg = paintNone
		case n == 48 || n == 38:
			span := 2
			if i+1 < len(parts) && parts[i+1] == "2" {
				span = 4
			}
			if n == 48 {
				bg = paintOther
				if strings.Join(parts[i+1:min(i+1+span, len(parts))], ";") == highlightSGR {
					bg = paintHighlight
				}
			}
			i += span
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			bg = paintOther
		}
	}
	return bg
}

func highlightGaps(frame string) []string {
	var bad []string
	for i, line := range strings.Split(frame, "\n") {
		cells := paintedCells(line)
		var holes []string
		start := 0
		for end := 0; end <= len(cells); end++ {
			if end < len(cells) && (cells[end].paint != paintNone || cells[end].blank) {
				continue
			}
			holes = append(holes, segmentHoles(cells, start, end)...)
			start = end + 1
		}
		if len(holes) > 0 {
			bad = append(bad, strconv.Itoa(i)+" cols "+strings.Join(holes, ",")+": "+ansi.Strip(line))
		}
	}
	return bad
}

func segmentHoles(cells []cell, start, end int) []string {
	first, last := -1, -1
	for x := start; x < end; x++ {
		if cells[x].paint == paintHighlight {
			if first < 0 {
				first = x
			}
			last = x
		}
	}
	var holes []string
	for x := first; first >= 0 && x <= last; x++ {
		if cells[x].paint == paintNone {
			holes = append(holes, strconv.Itoa(x))
		}
	}
	return holes
}

func hasHighlight(frame string) bool {
	for _, line := range strings.Split(frame, "\n") {
		for _, c := range paintedCells(line) {
			if c.paint == paintHighlight {
				return true
			}
		}
	}
	return false
}

func assertSolidSelection(t *testing.T, name string, a App) {
	t.Helper()
	frame := a.render()
	if !hasHighlight(frame) {
		t.Errorf("%s: no selected row was drawn", name)
	}
	if gaps := highlightGaps(frame); len(gaps) > 0 {
		t.Errorf("%s: the selected row's background has gaps on lines\n%s", name, strings.Join(gaps, "\n"))
	}
}

func assertSolidModal(t *testing.T, name string, a App) {
	t.Helper()
	m, ok := a.modals.top()
	if !ok {
		t.Fatalf("%s: no dialog open", name)
	}
	if !hasHighlight(m.View()) {
		t.Errorf("%s: no selected row was drawn", name)
	}
	if gaps := highlightGaps(m.View()); len(gaps) > 0 {
		t.Errorf("%s: the selected row's background has gaps on lines\n%s", name, strings.Join(gaps, "\n"))
	}
}

func TestSelectedRowsAreHighlightedEdgeToEdge(t *testing.T) {
	for _, width := range []int{200, 120, 80} {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			assertSolidSelection(t, "backlog", newTestApp(t, width, 40))
			assertSolidSelection(t, "board", feed(t, newTestApp(t, width, 40), press("B")))
			assertSolidSelection(t, "sprint", feed(t, newTestApp(t, width, 40), press("S")))
			assertSolidSelection(t, "pull requests", openPullRequestHub(t, width, 40))
			assertSolidSelection(t, "review files", openFiles(t, width, 40))
			hub := openPipelinesHub(t, width, 40)
			assertSolidSelection(t, "recent pipelines", hub)
			runs := feed(t, hub, press("3"))
			runs = feed(t, runs, pipelineListMsg{seq: runs.pl.seq, mode: panels.ListRuns, runs: sampleRuns()})
			assertSolidSelection(t, "runs", runs)
			view := openRunView(t, width, 40)
			assertSolidSelection(t, "run summary", view)
			assertSolidSelection(t, "run logs", feed(t, view, press("2")))
			releases := openReleasesHub(t, width, 40)
			assertSolidSelection(t, "release pipelines", releases)
			assertSolidSelection(t, "releases", feed(t, releases, special(tea.KeyTab)))
			approvals := feed(t, releases, press("2"))
			approvals = feed(t, approvals, approvalsLoadedMsg{seq: approvals.rl.approvalsSeq, mine: sampleRelease().Stages[1].Approvals})
			assertSolidSelection(t, "approvals", approvals)
			assertSolidSelection(t, "release logs", feed(t, openReleaseView(t, width, 40), press("h"), special(tea.KeyEnter)))
			assertSolidModal(t, "run form", openRunForm(t, width, 40))
			assertSolidModal(t, "create release", openCreateRelease(t, width, 40))
			assertSolidModal(t, "filter picker", feed(t, newTestApp(t, width, 40), press("3")))
			assertSolidModal(t, "repository picker", feed(t, newTestApp(t, width, 40), press("P"), repositoriesLoadedMsg{repos: twoRepositories()}))
		})
	}
}

package panels

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

var sgr = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

func hasBackground(s string) bool {
	for _, m := range sgr.FindAllStringSubmatch(s, -1) {
		params := strings.FieldsFunc(m[1], func(r rune) bool { return r == ';' || r == ':' })
		for i := 0; i < len(params); i++ {
			n, _ := strconv.Atoi(params[i])
			switch {
			case n == 38 && i+1 < len(params) && params[i+1] == "5":
				i += 2
			case n == 38 && i+1 < len(params) && params[i+1] == "2":
				i += 4
			case n == 48, n >= 40 && n <= 47, n >= 100 && n <= 107:
				return true
			}
		}
	}
	return false
}

func detailFixture(width int) DetailView {
	child := &models.WorkItem{ID: 2147, Title: "Card vault migration", Type: models.WorkItemTypeTask, State: "To Do"}
	item := &models.WorkItem{
		ID:            2141,
		Title:         "Checkout: save card for later",
		Type:          models.WorkItemTypePBI,
		State:         "Committed",
		Reason:        "Commitment made by the team",
		AssignedTo:    "Ana Ruiz",
		AreaPath:      "Fabrikam Fiber\\Web",
		IterationPath: "Fabrikam Fiber\\Sprint 14",
		Tags:          []string{"checkout"},
		Children:      []*models.WorkItem{child},
	}
	d := NewDetailView(styles.DefaultStyles(), keys.DefaultKeyMap())
	d.SetSize(width, 40)
	d.SetItem(item)
	d.SetComments(item.ID, []models.Comment{{ID: 1, Text: "Looks good", CreatedBy: "Ben Okafor"}})
	return d
}

func headerLines(d DetailView) []string {
	return strings.Split(d.View(), "\n")[:d.headerHeight()]
}

func TestWorkItemHeaderPaintsNoBackground(t *testing.T) {
	if !hasBackground(lipgloss.NewStyle().Background(styles.ColorSurfaceAlt).Render("x")) {
		t.Fatal("the background check cannot see Lip Gloss backgrounds")
	}
	if !hasBackground(Badge("x", styles.ColorPrimary, styles.ColorOnAccent)) {
		t.Fatal("the background check cannot see truecolor backgrounds")
	}
	for _, width := range []int{160, 90} {
		for i, line := range headerLines(detailFixture(width)) {
			if hasBackground(line) {
				t.Errorf("width %d: header line %d sets a background: %q", width, i, line)
			}
		}
	}
}

func TestWorkItemHeaderFollowsTheAzureDevOpsForm(t *testing.T) {
	header := ansi.Strip(strings.Join(headerLines(detailFixture(160)), "\n"))
	for _, want := range []string{"PRODUCT BACKLOG ITEM 2141", "Checkout: save card for later", "AR Ana Ruiz", "1 comment", "[checkout]",
		"Edit e", "State s", "Assign a", "New task t", "Delete d", "State      ● Committed", "Reason     Commitment made by the team",
		"Area       Fabrikam Fiber\\Web", "Iteration  Fabrikam Fiber\\Sprint 14", "Details   Discussion 1"} {
		if !strings.Contains(header, want) {
			t.Errorf("header is missing %q:\n%s", want, header)
		}
	}
}

func TestWorkItemTabUnderlineSitsUnderTheActiveTab(t *testing.T) {
	d := detailFixture(160)
	for _, tab := range []string{"Details", "Discussion 1"} {
		lines := headerLines(d)
		band := []rune(ansi.Strip(lines[len(lines)-2]))
		rule := []rune(ansi.Strip(lines[len(lines)-1]))
		at := strings.Index(string(band), tab)
		col := len([]rune(string(band)[:at]))
		under := strings.Index(string(rule), "━")
		if under < 0 || len([]rune(string(rule)[:under])) != col || strings.Count(string(rule), "━") != len(tab) {
			t.Fatalf("%s starts at column %d but the underline is %q", tab, col, string(rule))
		}
		d.commentFocused = true
	}
}

func TestWorkItemHeaderStacksTheBandWhenNarrow(t *testing.T) {
	d := detailFixture(90)
	if got := len(headerLines(d)); got != styles.DetailHeaderLinesStacked {
		t.Fatalf("header has %d lines, want %d", got, styles.DetailHeaderLinesStacked)
	}
	header := ansi.Strip(strings.Join(headerLines(d), "\n"))
	if !strings.Contains(header, "Area       Fabrikam Fiber\\Web") || strings.Contains(header, "Delete d") {
		t.Fatalf("narrow header should stack fields and drop actions that don't fit:\n%s", header)
	}
	for i, line := range strings.Split(d.View(), "\n") {
		if w := ansi.StringWidth(line); w > 90 {
			t.Errorf("line %d is %d wide", i, w)
		}
	}
}

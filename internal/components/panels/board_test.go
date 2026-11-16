package panels

import (
	"strings"

	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

func press(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(text)[0], Text: text}
}

func boardFixture() []models.WorkItem {
	items := []models.WorkItem{
		{ID: 1, Type: models.WorkItemTypePBI, Title: "First", State: "New"},
		{ID: 2, Type: models.WorkItemTypePBI, Title: "Second", State: "Committed"},
		{ID: 3, Type: models.WorkItemTypeBug, Title: "Third", State: "Committed"},
		{ID: 4, Type: models.WorkItemTypeTask, Title: "Task A", State: "To Do", ParentID: 2},
		{ID: 5, Type: models.WorkItemTypeTask, Title: "Task B", State: "Done", ParentID: 2},
	}
	items[1].Children = []*models.WorkItem{&items[3], &items[4]}
	return items
}

func TestBoardGroupsBacklogItemsByState(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetItems(boardFixture(), nil)
	if len(b.columns) != 2 {
		t.Fatalf("got %d columns, want 2 (New, Committed)", len(b.columns))
	}
	if b.columns[0].state != "New" || b.columns[1].state != "Committed" {
		t.Fatalf("unexpected column order: %q, %q", b.columns[0].state, b.columns[1].state)
	}
	if b.Count() != 3 {
		t.Fatalf("tasks must not appear as cards, got %d cards", b.Count())
	}
}

func TestBoardUsesAzureDevOpsWorkflowOrder(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	states := map[string][]models.WorkItemStateInfo{
		"Product Backlog Item": {
			{Name: "New", Category: "Proposed"},
			{Name: "Approved", Category: "Proposed"},
			{Name: "Committed", Category: "InProgress"},
			{Name: "Done", Category: "Completed"},
			{Name: "Removed", Category: "Removed"},
		},
	}
	b.SetItems(boardFixture(), states)
	var got []string
	for _, c := range b.columns {
		got = append(got, c.state)
	}
	want := []string{"New", "Approved", "Committed", "Done"}
	if len(got) != len(want) {
		t.Fatalf("columns = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("columns = %v, want %v", got, want)
		}
	}
}

func TestBoardMoveCardRequestsNextState(t *testing.T) {
	b := NewBoardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	b.SetItems(boardFixture(), nil)
	b.SetSize(80, 30)
	b.SetFocused(true)

	b, cmd := b.Update(press(">"))
	if cmd == nil {
		t.Fatal("expected a move command")
	}
	msg, ok := cmd().(MoveWorkItemMsg)
	if !ok {
		t.Fatalf("got %T, want MoveWorkItemMsg", cmd())
	}
	if msg.Item.ID != 1 || msg.NewState != "Committed" {
		t.Fatalf("got item %d → %q, want 1 → Committed", msg.Item.ID, msg.NewState)
	}

	b, _ = b.Update(press("l"))
	b, _ = b.Update(press("j"))
	if sel := b.SelectedItem(); sel == nil || sel.ID != 3 {
		t.Fatalf("selected %v, want item 3", sel)
	}
	if _, cmd := b.Update(press(">")); cmd != nil {
		t.Fatal("moving past the last column must do nothing")
	}
}

func TestTaskboardNavigatesLanesAndMovesTasks(t *testing.T) {
	tb := NewTaskboardPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	tb.SetItems(boardFixture(), nil)
	tb.SetSize(120, 30)
	tb.SetFocused(true)

	tb, _ = tb.Update(press("j"))
	tb, _ = tb.Update(press("l"))
	if sel := tb.SelectedItem(); sel == nil || sel.ID != 4 {
		t.Fatalf("selected %v, want task 4", sel)
	}
	_, cmd := tb.Update(press(">"))
	if cmd == nil {
		t.Fatal("expected a move command")
	}
	msg := cmd().(MoveWorkItemMsg)
	if msg.Item.ID != 4 || msg.NewState != "Done" {
		t.Fatalf("got task %d → %q, want 4 → Done", msg.Item.ID, msg.NewState)
	}
	done, total := tb.Progress()
	if done != 1 || total != 2 {
		t.Fatalf("progress = %d/%d, want 1/2", done, total)
	}
}

func TestPaneIsExactSize(t *testing.T) {
	out := Pane(PaneTitle("5", "A very long title that will not fit", true), "body\nline", 20, 6, true)
	lines := strings.Split(out, "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 20 {
			t.Fatalf("line %d is %d cells, want 20", i, w)
		}
	}
}

func TestWorkItemsPanelKeepsSelectionAcrossResize(t *testing.T) {
	w := NewWorkItemsPanel(styles.DefaultStyles(), keys.DefaultKeyMap())
	w.SetSize(120, 20)
	w.SetItems([]models.WorkItem{{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}})
	w.SetFocused(true)
	w, _ = w.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	w.SetSize(80, 20)
	if sel := w.SelectedItem(); sel == nil || sel.ID != 2 {
		t.Fatalf("selection after resize = %v, want item 2", sel)
	}
}

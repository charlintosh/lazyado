package ui

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func loadedItemOf(t *testing.T, a App, id int) models.WorkItem {
	t.Helper()
	item, ok := a.loadedItem(id)
	if !ok {
		t.Fatalf("work item %d is not loaded", id)
	}
	return item
}

var dana = models.Identity{ID: "dana-id", DisplayName: "Dana Reyes"}

func queryingEveryone(a App) App {
	a.loadedQuery.assigned = models.FilterAll
	return a
}

func TestASavedMoveUpdatesTheCardWithoutReloading(t *testing.T) {
	a := queryingEveryone(droppedOnDone(t))
	seq := a.loadSeq
	saved := loadedItemOf(t, a, 1052)
	saved.State = "Done"
	a, _ = step(t, a, workItemMovedMsg{id: 1052, state: "Done", item: &saved})
	screen := ansi.Strip(a.render())
	if !strings.Contains(screen, "Moved to Done") || !strings.Contains(screen, "State changed to Done") {
		t.Fatalf("a saved move still flashes and notifies:\n%s", screen)
	}
	if a.loadSeq != seq || a.loading {
		t.Fatalf("a move the server confirmed must not reload the whole list (seq %d -> %d, loading=%v)", seq, a.loadSeq, a.loading)
	}
	if got := loadedItemOf(t, a, 1052); got.State != "Done" || got.ParentTitle == "" {
		t.Fatalf("the card must take the saved state and keep its parent title, got %+v", got)
	}
}

func TestAnEditUpdatesTheItemAndItsChildren(t *testing.T) {
	a := newTestAppAs(t, 160, 45, dana)
	seq := a.loadSeq
	before := loadedItemOf(t, a, 1043)
	saved := before
	saved.Title, saved.Children, saved.ParentTitle = "Checkout: guest purchase", nil, "from the reply"
	a, _ = step(t, a, workItemChangedMsg{notice: "Work item updated", item: &saved})
	if a.loadSeq != seq {
		t.Fatal("an edit the server confirmed must not reload the whole list")
	}
	parent := loadedItemOf(t, a, 1043)
	if parent.Title != "Checkout: guest purchase" || len(parent.Children) != 3 || parent.ParentTitle != before.ParentTitle {
		t.Fatalf("the edited item keeps its place in the tree, got %+v", parent)
	}
	if child := loadedItemOf(t, a, 1051); child.ParentTitle != "Checkout: guest purchase" {
		t.Fatalf("children show the new parent title, got %q", child.ParentTitle)
	}
}

func TestAnEditThatLeavesTheQueryReloads(t *testing.T) {
	a := newTestAppAs(t, 160, 45, dana)
	a.loadedQuery = workItemQuery{sprint: `storefront\Sprint 24`, state: "Committed", assigned: models.FilterAll, area: models.FilterAll}
	seq := a.loadSeq
	saved := loadedItemOf(t, a, 1043)
	saved.State = "Done"
	a, _ = step(t, a, workItemChangedMsg{notice: "State changed to Done", item: &saved})
	if a.loadSeq != seq+1 {
		t.Fatal("an item that no longer matches the server query must trigger a reload")
	}
}

func TestAnEditUnderTheServerMeFilterReloads(t *testing.T) {
	a := newTestApp(t, 160, 45)
	seq := a.loadSeq
	saved := loadedItemOf(t, a, 1043)
	a, _ = step(t, a, workItemChangedMsg{notice: "Assigned to Lena Ortiz", item: &saved})
	if a.loadSeq != seq+1 {
		t.Fatal("before the signed-in user is known the list is filtered by @me on the server, so it must reload")
	}
}

func TestASaveWithoutAReplyReloads(t *testing.T) {
	a := newTestApp(t, 160, 45)
	seq := a.loadSeq
	a, _ = step(t, a, workItemChangedMsg{notice: "Work item updated"})
	if a.loadSeq != seq+1 {
		t.Fatal("without the saved item the list must reload as before")
	}
}

func TestQueryAdmitsMatchesTheServerFilters(t *testing.T) {
	item := models.WorkItem{State: "Active", IterationPath: `web\Sprint 3`, AreaPath: `web\Checkout\Payments`}
	cases := []struct {
		q    workItemQuery
		want bool
	}{
		{workItemQuery{sprint: "all", state: "all", assigned: "all", area: "all"}, true},
		{workItemQuery{sprint: `web\Sprint 3`, state: "Active", area: `\web\Checkout\`}, true},
		{workItemQuery{area: `web\Checkout\Payments`}, true},
		{workItemQuery{area: `web\Check`}, false},
		{workItemQuery{state: "Closed"}, false},
		{workItemQuery{sprint: `web\Sprint 4`}, false},
		{workItemQuery{assigned: models.AssignedMe}, false},
	}
	for _, c := range cases {
		if got := c.q.admits(item); got != c.want {
			t.Errorf("%+v admits = %v, want %v", c.q, got, c.want)
		}
	}
}

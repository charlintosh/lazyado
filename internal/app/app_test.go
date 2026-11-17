package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/notification"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func press(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(text)[0], Text: text}
}

func ctrl(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

func special(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func notificationDismiss(id int) tea.Msg {
	return notification.AutoDismissMsg{ID: id}
}

func drain(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, drain(c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func step(t testing.TB, a App, msg tea.Msg) (App, []tea.Msg) {
	t.Helper()
	m, cmd := a.Update(msg)
	return m.(App), drain(cmd)
}

func find[T any](msgs []tea.Msg) (T, bool) {
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

func topModal[T any](t *testing.T, a App) T {
	t.Helper()
	m, ok := a.modals.top()
	if !ok {
		t.Fatalf("no modal open, want %T", *new(T))
	}
	v, ok := m.(T)
	if !ok {
		t.Fatalf("top modal is %T, want %T", m, *new(T))
	}
	return v
}

func TestPasteReachesFocusedInput(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a, _ = step(t, a, press("t"))
	topModal[modals.WorkItemForm](t, a)
	a, _ = step(t, a, tea.PasteMsg{Content: "Pasted task title"})
	_, msgs := step(t, a, ctrl('s'))
	req, ok := find[modals.TaskCreateRequestMsg](msgs)
	if !ok {
		t.Fatalf("save produced %v, want TaskCreateRequestMsg", msgs)
	}
	if req.Title != "Pasted task title" || req.ParentID != 1031 {
		t.Fatalf("got %+v, want pasted title on parent 1031", req)
	}
}

func TestStaleCommentResponsesAreDropped(t *testing.T) {
	a := newTestApp(t, 160, 45)
	if a.commentsFor != 1031 || !a.commentsPanel.Loading() {
		t.Fatalf("expected comments loading for 1031, got %d loading=%v", a.commentsFor, a.commentsPanel.Loading())
	}
	a, _ = step(t, a, commentsLoadedMsg{workItemID: 1043, comments: []models.Comment{{ID: 1, Text: "other"}}})
	if a.commentsPanel.Count() != 0 || !a.commentsPanel.Loading() {
		t.Fatal("comments for another work item must be ignored")
	}
	a, _ = step(t, a, commentsLoadedMsg{workItemID: 1031, comments: []models.Comment{{ID: 2, Text: "mine"}}})
	if a.commentsPanel.Count() != 1 || a.commentsPanel.Loading() {
		t.Fatal("comments for the selected work item must be shown")
	}
}

func TestStaleWorkItemResponsesAreDropped(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a, _ = step(t, a, ctrl('r'))
	a, _ = step(t, a, workItemsLoadedMsg{seq: a.loadSeq - 1})
	if a.workItemsPanel.Count() == 0 || !a.loading {
		t.Fatal("an outdated response must not replace the backlog")
	}
}

func TestErrorStacksOverOpenForm(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a, _ = step(t, a, press("e"))
	topModal[modals.WorkItemForm](t, a)
	a, _ = step(t, a, errMsg{err: errors.New("boom")})
	topModal[modals.ErrorModal](t, a)
	a, msgs := step(t, a, special(tea.KeyEscape))
	for _, m := range msgs {
		a, _ = step(t, a, m)
	}
	topModal[modals.WorkItemForm](t, a)
}

func TestCommentEditTargetsDetailItem(t *testing.T) {
	a := newTestAppAs(t, 160, 45, models.Identity{ID: "dana-id", DisplayName: "Dana Reyes"})
	other := sampleWorkItems()[4]
	a, _ = step(t, a, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a, _ = step(t, a, modals.QuickSearchSuccessMsg{Item: &other})
	if a.viewMode != ViewDetail || a.modals.open() {
		t.Fatal("quick search success should close the modal and open the detail view")
	}
	a, _ = step(t, a, commentsLoadedMsg{workItemID: other.ID, comments: []models.Comment{{ID: 7, Text: "hello", CreatedByID: "dana-id"}}})
	a, _ = step(t, a, special(tea.KeyTab))
	a, msgs := step(t, a, press("e"))
	edit, ok := find[panels.EditCommentMsg](msgs)
	if !ok || edit.WorkItemID != other.ID {
		t.Fatalf("got %v, want edit for work item %d", msgs, other.ID)
	}
	a, _ = step(t, a, edit)
	_, msgs = step(t, a, ctrl('s'))
	req, ok := find[modals.CommentUpdateRequestMsg](msgs)
	if !ok || req.WorkItemID != other.ID || req.CommentID != 7 {
		t.Fatalf("got %v, want update of comment 7 on %d", msgs, other.ID)
	}
}

func TestCreateFormHonoursChosenSprint(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a.iterations = append(a.iterations, models.Iteration{Name: "Sprint 25", Path: "storefront\\Sprint 25"})
	a, msgs := step(t, a, press("n"))
	a, msgs = step(t, a, special(tea.KeyEnter))
	for _, m := range msgs {
		a, _ = step(t, a, m)
	}
	topModal[modals.WorkItemForm](t, a)
	a, _ = step(t, a, tea.PasteMsg{Content: "New item"})
	for range 7 {
		a, _ = step(t, a, special(tea.KeyTab))
	}
	a, _ = step(t, a, special(tea.KeyDown))
	_, msgs = step(t, a, ctrl('s'))
	req, ok := find[modals.ParentCreateRequestMsg](msgs)
	if !ok {
		t.Fatalf("save produced %v", msgs)
	}
	if req.IterationPath != "storefront\\Sprint 25" || req.AreaPath != "storefront\\Checkout" {
		t.Fatalf("got iteration %q area %q", req.IterationPath, req.AreaPath)
	}
}

func TestNotificationIgnoresOlderTimers(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a, _ = step(t, a, copiedMsg{notice: "first"})
	a, _ = step(t, a, copiedMsg{notice: "second"})
	a, _ = step(t, a, notificationDismiss(1))
	if !a.notification.IsVisible() {
		t.Fatal("the first timer must not hide the second notification")
	}
	a, _ = step(t, a, notificationDismiss(2))
	if a.notification.IsVisible() {
		t.Fatal("the second timer must hide the second notification")
	}
}

func TestCancelledQuickSearchIgnoresLateResult(t *testing.T) {
	a := newTestApp(t, 120, 40)
	a, _ = step(t, a, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a, msgs := step(t, a, special(tea.KeyEscape))
	for _, m := range msgs {
		a, _ = step(t, a, m)
	}
	item := sampleWorkItems()[0]
	a, _ = step(t, a, modals.QuickSearchSuccessMsg{Item: &item})
	if a.viewMode != ViewMain {
		t.Fatal("a result for a cancelled quick search must not open the detail view")
	}
}

func typeText(t *testing.T, a App, text string) App {
	t.Helper()
	for _, r := range text {
		a, _ = step(t, a, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return a
}

func TestPersonPickerListsPeopleFromLoadedItems(t *testing.T) {
	a := newTestAppAs(t, 160, 45, models.Identity{DisplayName: "Dana Reyes"})
	a, _ = step(t, a, press("S"))
	a, _ = step(t, a, press("2"))
	picker := topModal[modals.FilterPickerModal](t, a)
	if picker.Group() != 1 || a.filterBar {
		t.Fatalf("2 opened group %d with filter bar %v, want the Person picker in the header", picker.Group(), a.filterBar)
	}
	view := ansi.Strip(picker.View())
	for _, want := range []string{"Person", "@Me", "Unassigned", "Active in Sprint 24", "Dana Reyes", "you", "Sam Kowal", "Team, nothing assigned", "Lena Ortiz"} {
		if !strings.Contains(view, want) {
			t.Errorf("person picker is missing %q:\n%s", want, view)
		}
	}
}

func TestPersonFilterAppliesWithoutReloading(t *testing.T) {
	a := newTestAppAs(t, 160, 45, models.Identity{DisplayName: "Dana Reyes"})
	seq := a.loadSeq
	a, _ = step(t, a, modals.FilterOptionChosenMsg{Group: 1, Value: "Sam Kowal"})
	if a.loadSeq != seq || a.loading {
		t.Fatal("choosing a person must filter the loaded items, not query again")
	}
	if a.workItemsPanel.Count() != 1 || a.workItemsPanel.SelectedItem().ID != 1060 {
		t.Fatalf("backlog shows %d rows, want only Sam's bug 1060", a.workItemsPanel.Count())
	}
	a, _ = step(t, a, press("x"))
	if a.workItemsPanel.Count() != 5 || a.filters.GetSelectedAssigned() != models.FilterAll {
		t.Fatalf("clear left %d rows and person %q", a.workItemsPanel.Count(), a.filters.GetSelectedAssigned())
	}
}

func TestMeFallsBackToServerWhenIdentityIsUnknown(t *testing.T) {
	fs := models.NewFilterState(nil, nil, nil)
	if q := queryFor(fs, models.Identity{}); q.assigned != models.AssignedMe {
		t.Fatalf("assigned = %q, want me sent to the server", q.assigned)
	}
	if q := queryFor(fs, models.Identity{DisplayName: "Dana Reyes"}); q.assigned != models.FilterAll {
		t.Fatalf("assigned = %q, want people filtered locally", q.assigned)
	}
}

func TestKeywordFilterNarrowsTheBacklog(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a, _ = step(t, a, press("/"))
	if !a.keyword.Focused() || !a.filterBar || a.modeLabel() != "FILTER" {
		t.Fatal("/ must open the filter bar and focus the keyword")
	}
	before := a.activeFilterCount()
	a = typeText(t, a, "guest")
	if a.workItemsPanel.Count() != 1 || a.workItemsPanel.SelectedItem().ID != 1043 {
		t.Fatalf("got %d rows, want the parent of the guest tasks", a.workItemsPanel.Count())
	}
	a, _ = step(t, a, special(tea.KeyEscape))
	if a.keyword.Focused() || a.keyword.Value() != "guest" {
		t.Fatal("esc must leave the keyword in place and return to the board")
	}
	if a.activeFilterCount() != before+1 || !strings.Contains(ansi.Strip(a.render()), "Filter 3") {
		t.Errorf("funnel counts %d filters, want the keyword added to %d", a.activeFilterCount(), before)
	}
}

func TestBarFiltersOpenTheFilterBar(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a, _ = step(t, a, press("S"))
	a, _ = step(t, a, press("3"))
	if !a.filterBar || topModal[modals.FilterPickerModal](t, a).Group() != 2 {
		t.Fatal("3 on Sprints must open the filter bar and the State picker")
	}
}

func TestRailCollapsesAndHides(t *testing.T) {
	cases := []struct {
		width   int
		rail    int
		compact bool
	}{
		{160, 28, false},
		{90, 5, true},
		{60, 0, false},
	}
	for _, c := range cases {
		l := newTestApp(t, c.width, 30).shell
		if l.railWidth != c.rail || l.compactRail != c.compact {
			t.Errorf("width %d: rail %d compact %v, want %d %v", c.width, l.railWidth, l.compactRail, c.rail, c.compact)
		}
	}
	a, _ := step(t, newTestApp(t, 160, 45), press("z"))
	if a.shell.railWidth != 0 {
		t.Error("z must hide the rail")
	}
}

func TestSprintPivotSwitchesToTheBacklogList(t *testing.T) {
	a := newTestApp(t, 160, 45)
	a, _ = step(t, a, press("S"))
	if a.mainView() != viewTaskboard {
		t.Fatal("Sprints opens on the taskboard")
	}
	a, _ = step(t, a, press("]"))
	if a.mainView() != viewList || a.selectedItem() == nil {
		t.Fatal("] must switch Sprints to its backlog list")
	}
	if !a.shell.showPane {
		t.Error("the sprint backlog list shows the details pane like Backlogs")
	}
	a, _ = step(t, a, press("["))
	if a.mainView() != viewTaskboard {
		t.Fatal("[ must switch back to the taskboard")
	}
}

func openDetailOf(t *testing.T, id int) App {
	t.Helper()
	a := newTestApp(t, 160, 45)
	item, ok := a.loadedItem(id)
	if !ok {
		t.Fatalf("work item %d is not loaded", id)
	}
	a, _ = step(t, a, panels.ViewWorkItemMsg{Item: item})
	if a.viewMode != ViewDetail {
		t.Fatal("v did not open the work item view")
	}
	return a
}

func submit(t *testing.T, a App, keys ...tea.Msg) []tea.Msg {
	t.Helper()
	var msgs []tea.Msg
	for _, k := range keys {
		a, msgs = step(t, a, k)
	}
	return msgs
}

func TestWorkItemViewRunsListActions(t *testing.T) {
	cases := map[string]struct {
		keys  []tea.Msg
		check func([]tea.Msg) bool
	}{
		"change state": {
			[]tea.Msg{press("s"), special(tea.KeyDown), special(tea.KeyEnter)},
			func(m []tea.Msg) bool { r, ok := find[modals.StateChangeRequestMsg](m); return ok && r.Item.ID == 1043 },
		},
		"assign": {
			[]tea.Msg{press("a"), special(tea.KeyEnter)},
			func(m []tea.Msg) bool { r, ok := find[modals.AssignRequestMsg](m); return ok && r.Item.ID == 1043 },
		},
		"edit": {
			[]tea.Msg{press("e"), ctrl('s')},
			func(m []tea.Msg) bool {
				r, ok := find[modals.WorkItemUpdateRequestMsg](m)
				return ok && r.ItemID == 1043
			},
		},
		"new task": {
			[]tea.Msg{press("t"), tea.PasteMsg{Content: "Follow-up"}, ctrl('s')},
			func(m []tea.Msg) bool { r, ok := find[modals.TaskCreateRequestMsg](m); return ok && r.ParentID == 1043 },
		},
		"delete": {
			[]tea.Msg{press("d"), press("y")},
			func(m []tea.Msg) bool { r, ok := find[modals.WorkItemDeleteRequestMsg](m); return ok && r.ID == 1043 },
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := openDetailOf(t, 1043)
			if msgs := submit(t, a, c.keys...); !c.check(msgs) {
				t.Fatalf("got %v", msgs)
			}
		})
	}
	a, _ := step(t, openDetailOf(t, 1043), press("i"))
	topModal[modals.ChoiceModal](t, a)
	if a.viewMode != ViewDetail {
		t.Fatal("info must open over the work item view")
	}
}

func TestWorkItemViewRefreshesAfterAChange(t *testing.T) {
	a := openDetailOf(t, 1043)
	a, _ = step(t, a, commentsLoadedMsg{workItemID: 1043, comments: []models.Comment{{ID: 1, Text: "kept"}}})
	a, _ = step(t, a, workItemChangedMsg{notice: "State changed to Done"})
	items := sampleWorkItems()
	items[0].State = "Done"
	a, _ = step(t, a, workItemsLoadedMsg{seq: a.loadSeq, items: items})
	if a.viewMode != ViewDetail || a.detailView.Item().State != "Done" {
		t.Fatalf("view shows state %q, want the refreshed Done", a.detailView.Item().State)
	}
	if !strings.Contains(ansi.Strip(a.render()), "kept") {
		t.Error("refreshing the item must keep its loaded discussion")
	}
}

func TestDeletingFromTheWorkItemViewClosesIt(t *testing.T) {
	a := openDetailOf(t, 1043)
	a, _ = step(t, a, workItemChangedMsg{notice: "Work item deleted", deletedID: 1043})
	if a.viewMode != ViewMain {
		t.Fatal("the view of a deleted work item must close")
	}
}

func TestCommentsCanOnlyBeChangedByTheirAuthor(t *testing.T) {
	me := models.Identity{ID: "dana-id", DisplayName: "Dana Reyes", UniqueName: "dana@example.com"}
	comments := []models.Comment{
		{ID: 1, Text: "mine", CreatedBy: "Dana Reyes", CreatedByID: "dana-id"},
		{ID: 2, Text: "not mine", CreatedBy: "Dana Reyes", CreatedByID: "namesake-id"},
	}
	places := map[string]func(App) App{
		"discussion pane": func(a App) App {
			a, _ = step(t, a, press("7"))
			return a
		},
		"work item view": func(a App) App {
			item, _ := a.loadedItem(a.commentsFor)
			a, _ = step(t, a, panels.ViewWorkItemMsg{Item: item})
			a, _ = step(t, a, commentsLoadedMsg{workItemID: item.ID, comments: comments})
			a, _ = step(t, a, special(tea.KeyTab))
			return a
		},
	}
	for name, open := range places {
		t.Run(name, func(t *testing.T) {
			a := newTestAppAs(t, 160, 45, me)
			a, _ = step(t, a, commentsLoadedMsg{workItemID: a.commentsFor, comments: comments})
			a = open(a)
			for _, k := range []string{"e", "d"} {
				_, msgs := step(t, a, press(k))
				next := a
				for _, m := range msgs {
					next, _ = step(t, next, m)
				}
				if !next.modals.open() {
					t.Fatalf("%s on my own comment opened nothing", k)
				}
			}
			a, _ = step(t, a, special(tea.KeyDown))
			for _, k := range []string{"e", "d"} {
				_, msgs := step(t, a, press(k))
				next := a
				for _, m := range msgs {
					next, _ = step(t, next, m)
				}
				if next.modals.open() || !next.notification.IsVisible() {
					t.Fatalf("%s on someone else's comment must be refused with a notice", k)
				}
			}
		})
	}
}

func TestOtherAuthorsCommentsAreRefusedEvenByMessage(t *testing.T) {
	a := newTestAppAs(t, 160, 45, models.Identity{ID: "dana-id"})
	theirs := models.Comment{ID: 9, CreatedBy: "Ben Okafor", CreatedByID: "ben-id"}
	for _, msg := range []tea.Msg{panels.EditCommentMsg{WorkItemID: 1043, Comment: theirs}, panels.DeleteCommentMsg{WorkItemID: 1043, Comment: theirs}} {
		next, _ := step(t, a, msg)
		if next.modals.open() {
			t.Fatalf("%T for another author's comment opened a modal", msg)
		}
	}
}

func TestIdentityResolvedLaterUnlocksOwnComments(t *testing.T) {
	a := newTestApp(t, 160, 45)
	mine := models.Comment{ID: 1, Text: "mine", CreatedBy: "Dana Reyes", CreatedByID: "dana-id"}
	a, _ = step(t, a, commentsLoadedMsg{workItemID: a.commentsFor, comments: []models.Comment{mine}})
	a, _ = step(t, a, press("7"))
	_, msgs := step(t, a, press("e"))
	if _, ok := find[panels.CommentReadOnlyMsg](msgs); !ok {
		t.Fatalf("without an identity e must be refused, got %v", msgs)
	}
	a, _ = step(t, a, identityLoadedMsg{me: models.Identity{ID: "dana-id"}})
	_, msgs = step(t, a, press("e"))
	if _, ok := find[panels.EditCommentMsg](msgs); !ok {
		t.Fatalf("after the identity resolves e must edit my comment, got %v", msgs)
	}
}

func TestWorkItemViewLikesSomeoneElsesComment(t *testing.T) {
	a := newTestAppAs(t, 160, 45, models.Identity{ID: "dana-id", DisplayName: "Dana Reyes"})
	item := sampleWorkItems()[4]
	a, _ = step(t, a, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	a, _ = step(t, a, modals.QuickSearchSuccessMsg{Item: &item})
	a, _ = step(t, a, commentsLoadedMsg{workItemID: item.ID, comments: []models.Comment{{ID: 9, Text: "Looks right", CreatedBy: "Ben Okafor", CreatedByID: "ben"}}})
	a, _ = step(t, a, special(tea.KeyTab))
	if out := ansi.Strip(a.render()); !strings.Contains(out, "╭─ BO Ben Okafor") || !strings.Contains(out, "╰─ + like") {
		t.Fatalf("the focused comment must be a card offering a like:\n%s", out)
	}
	a, msgs := step(t, a, press("+"))
	like, ok := find[panels.LikeCommentMsg](msgs)
	if !ok || !like.Like || like.WorkItemID != item.ID || like.Comment.ID != 9 {
		t.Fatalf("got %+v, want a like of comment 9 on %d", like, item.ID)
	}
	a, _ = step(t, a, like)
	if !a.loading {
		t.Fatal("a like must start the update")
	}
}

package modals

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func testEnv() Env {
	return Env{Keys: keys.DefaultKeyMap(), Width: 120, Height: 40}
}

func members() []models.TeamMember {
	return []models.TeamMember{
		{ID: "id-dana", DisplayName: "Dana Reyes", UniqueName: "dana@acme.io"},
		{ID: "id-marco", DisplayName: "Marco Lima", UniqueName: "marco@acme.io"},
	}
}

func typeInto(m Modal, text string) Modal {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func keyPress(m Modal, code rune, mod tea.KeyMod) (Modal, tea.Msg) {
	m, cmd := m.Update(tea.KeyPressMsg{Code: code, Mod: mod})
	return m, immediate(cmd)
}

func immediate(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(20 * time.Millisecond):
		return nil
	}
}

func TestFormRejectsPriorityOutsideRange(t *testing.T) {
	var m Modal = NewEditWorkItemForm(testEnv(), models.WorkItem{ID: 9, Title: "Item", Type: models.WorkItemTypePBI}, members())
	for range 2 {
		m, _ = keyPress(m, tea.KeyTab, 0)
	}
	m = typeInto(m, "5")
	m, msg := keyPress(m, 's', tea.ModCtrl)
	if msg != nil {
		t.Fatalf("save produced %#v, want a validation error", msg)
	}
	if !strings.Contains(m.View(), "Priority must be a number from 1 to 4") {
		t.Fatal("validation error is not shown")
	}
}

func TestBugFormCannotTurnIntoPBI(t *testing.T) {
	var m Modal = NewCreateBugForm(testEnv(), FormData{Members: members()})
	m = typeInto(m, "Crash")
	m, _ = keyPress(m, tea.KeyTab, tea.ModShift)
	m, _ = keyPress(m, tea.KeyRight, 0)
	_, msg := keyPress(m, 's', tea.ModCtrl)
	req, ok := msg.(ParentCreateRequestMsg)
	if !ok || req.WorkItemType != string(models.WorkItemTypeBug) || req.Title != "Crash" {
		t.Fatalf("got %#v, want a Bug titled Crash", msg)
	}
}

func TestEditFormKeepsUnassignedItemUnassigned(t *testing.T) {
	var m Modal = NewEditWorkItemForm(testEnv(), models.WorkItem{ID: 9, Title: "Item", Type: models.WorkItemTypeBug}, members())
	for range 5 {
		m, _ = keyPress(m, tea.KeyTab, 0)
	}
	m, _ = m.Update(tea.PasteMsg{})
	_, msg := keyPress(m, 's', tea.ModCtrl)
	req, ok := msg.(WorkItemUpdateRequestMsg)
	if !ok || req.AssignedTo != "" {
		t.Fatalf("got %#v, want no assignee change", msg)
	}
}

func TestEnterAdvancesFromSingleLineFields(t *testing.T) {
	m := NewCreateTaskForm(testEnv(), models.WorkItem{ID: 3, Title: "Parent"})
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(WorkItemForm).focus != 1 {
		t.Fatal("enter in the title should move to the description")
	}
	next, _ = next.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if next.(WorkItemForm).focus != 1 {
		t.Fatal("enter in the description should insert a newline, not move focus")
	}
}

func TestCommentMentionsAreSuggestedAndEncoded(t *testing.T) {
	var m Modal = NewCommentModal(testEnv(), models.WorkItem{ID: 4, Title: "Item"}, members())
	m = typeInto(m, "Hi @dan")
	if !m.(CommentModal).mentioning {
		t.Fatal("typing @dan should open mention suggestions")
	}
	m, _ = keyPress(m, tea.KeyEnter, 0)
	if got := m.(CommentModal).area.Value(); got != "Hi @Dana Reyes " {
		t.Fatalf("textarea = %q, want the inserted mention", got)
	}
	m = typeInto(m, "please review")
	_, msg := keyPress(m, 's', tea.ModCtrl)
	req, ok := msg.(CommentCreateRequestMsg)
	if !ok {
		t.Fatalf("got %#v, want CommentCreateRequestMsg", msg)
	}
	want := `Hi <a href="#" data-vss-mention="version:2.0,id-dana">Dana Reyes</a> please review`
	if req.WorkItemID != 4 || req.Text != want {
		t.Fatalf("got %+v, want %q", req, want)
	}
}

func TestChoiceModalClearsQueryBeforeClosing(t *testing.T) {
	var m Modal = NewAssignModal(testEnv(), models.WorkItem{ID: 1, Title: "Item"}, members())
	m = typeInto(m, "zz")
	m, msg := keyPress(m, tea.KeyEscape, 0)
	if msg != nil {
		t.Fatalf("first esc should clear the query, got %#v", msg)
	}
	_, msg = keyPress(m, tea.KeyEscape, 0)
	if _, ok := msg.(ModalClosedMsg); !ok {
		t.Fatalf("second esc should close, got %#v", msg)
	}
}

func TestQuickSearchAcceptsDigitsOnly(t *testing.T) {
	var m Modal = NewQuickSearchModal(testEnv())
	m = typeInto(m, "12a3")
	m, _ = m.Update(tea.PasteMsg{Content: "4-5"})
	_, msg := keyPress(m, tea.KeyEnter, 0)
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("got %#v, want a batch with the lookup request", msg)
	}
	for _, cmd := range batch {
		if req, ok := immediate(cmd).(QuickSearchRequestMsg); ok {
			if req.ID != 12345 {
				t.Fatalf("looked up %d, want 12345", req.ID)
			}
			return
		}
	}
	t.Fatal("no QuickSearchRequestMsg in the batch")
}

func mentionTeam() []models.TeamMember {
	return append(members(), models.TeamMember{ID: "id-priya", DisplayName: "Priya Nair", UniqueName: "priya@acme.io"})
}

func listedMembers(m Modal, team []models.TeamMember) []models.TeamMember {
	var out []models.TeamMember
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		for _, member := range team {
			if strings.Contains(line, member.DisplayName) {
				out = append(out, member)
			}
		}
	}
	return out
}

func postedMention(t *testing.T, m Modal) string {
	t.Helper()
	m, _ = keyPress(m, tea.KeyEnter, 0)
	_, msg := keyPress(m, 's', tea.ModCtrl)
	req, ok := msg.(CommentCreateRequestMsg)
	if !ok {
		t.Fatalf("save produced %#v", msg)
	}
	return req.Text
}

func TestMentionPickerKeepsSelectionWhileTheCursorBlinks(t *testing.T) {
	team := mentionTeam()
	var m Modal = NewCommentModal(testEnv(), models.WorkItem{ID: 7, Title: "Item"}, team)
	m = typeInto(m, "@a")
	listed := listedMembers(m, team)
	if len(listed) < 3 {
		t.Fatalf("@a should list all three people, got %v", listed)
	}
	for range 2 {
		m, _ = keyPress(m, tea.KeyDown, 0)
	}
	for range 3 {
		m, _ = m.Update(textarea.Blink())
		m, _ = m.Update(struct{}{})
	}
	if text := postedMention(t, m); !strings.Contains(text, "version:2.0,"+listed[2].ID) {
		t.Fatalf("posted %q, want a mention of %s, the third person listed", text, listed[2].DisplayName)
	}
}

func TestMentionPickerFollowsUpAndDown(t *testing.T) {
	team := mentionTeam()
	var m Modal = NewCommentModal(testEnv(), models.WorkItem{ID: 7, Title: "Item"}, team)
	m = typeInto(m, "hi @a")
	listed := listedMembers(m, team)
	for _, code := range []rune{tea.KeyDown, tea.KeyDown, tea.KeyUp} {
		m, _ = keyPress(m, code, 0)
		m, _ = m.Update(textarea.Blink())
	}
	text := postedMention(t, m)
	if !strings.HasPrefix(text, "hi ") || !strings.Contains(text, "version:2.0,"+listed[1].ID) {
		t.Fatalf("posted %q, want hi followed by a mention of %s", text, listed[1].DisplayName)
	}
}

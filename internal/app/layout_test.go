package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/api"
	"github.com/charlintosh/lazyado/internal/components/modals"
	"github.com/charlintosh/lazyado/internal/components/panels"
	"github.com/charlintosh/lazyado/internal/config"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

func sampleWorkItems() []models.WorkItem {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	item := func(id int, typ models.WorkItemType, title, state, assignee string, parent int, effort float64, tags ...string) models.WorkItem {
		return models.WorkItem{
			ID:            id,
			Title:         title,
			Type:          typ,
			State:         models.WorkItemState(state),
			AssignedTo:    assignee,
			IterationPath: "storefront\\Sprint 24",
			AreaPath:      "storefront\\Checkout",
			ParentID:      parent,
			Effort:        effort,
			Priority:      2,
			Tags:          tags,
			Description:   "Shoppers can complete a purchase with only an email address.",
			CreatedDate:   now.AddDate(0, 0, -16),
			ChangedDate:   now,
		}
	}
	items := []models.WorkItem{
		item(1043, models.WorkItemTypePBI, "Checkout: allow purchase without an account", "Committed", "Dana Reyes", 1012, 5, "checkout"),
		item(1051, models.WorkItemTypeTask, "Add guest session token to cart API", "In Progress", "Marco Lima", 1043, 3),
		item(1052, models.WorkItemTypeTask, "Email receipt for guest orders", "To Do", "", 1043, 2),
		item(1050, models.WorkItemTypeTask, "Guest option on sign-in step", "Done", "Dana Reyes", 1043, 0),
		item(1060, models.WorkItemTypeBug, "Address form drops apartment number", "Committed", "Sam Kowal", 1012, 2, "regression"),
		item(1044, models.WorkItemTypePBI, "Saved payment methods", "Approved", "Priya Nair", 1012, 8),
		item(1070, models.WorkItemTypePBI, "Typo tolerance in product search", "New", "", 1013, 3, "search"),
		item(1031, models.WorkItemTypePBI, "Cart persists across devices", "Done", "Dana Reyes", 1012, 5),
		item(1061, models.WorkItemTypeTask, "Preserve line 2 in address mapper", "Done", "Sam Kowal", 1060, 0),
		item(1062, models.WorkItemTypeTask, "Regression test for address line 2", "To Do", "Sam Kowal", 1060, 2),
	}
	index := map[int]int{}
	for i := range items {
		index[items[i].ID] = i
	}
	for i := range items {
		if p, ok := index[items[i].ParentID]; ok {
			items[i].ParentTitle = items[p].Title
			items[p].Children = append(items[p].Children, &items[i])
		}
	}
	return items
}

func newTestApp(t testing.TB, width, height int) App {
	t.Helper()
	return newTestAppAs(t, width, height, models.Identity{})
}

func newTestAppAs(t testing.TB, width, height int, me models.Identity) App {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	client := api.NewClient(&config.Config{Organization: "acme", Project: "storefront", Team: "Storefront Team"})
	var m tea.Model = NewApp(client, models.DiffLimits{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = m.Update(dataLoadedMsg{
		iterations:  []models.Iteration{{Name: "Sprint 24", Path: "storefront\\Sprint 24", TimeFrame: "current"}},
		areas:       []models.Area{{Name: "Checkout", Path: "storefront\\Checkout"}},
		teamMembers: []models.TeamMember{{DisplayName: "Dana Reyes"}, {DisplayName: "Lena Ortiz"}},
		me:          me,
	})
	m, _ = m.Update(workItemsLoadedMsg{seq: m.(App).loadSeq, items: sampleWorkItems()})
	return m.(App)
}

func assertFits(t *testing.T, out string, width, height int) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) > height {
		t.Errorf("rendered %d lines, terminal has %d", len(lines), height)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > width {
			t.Errorf("line %d is %d cells wide, terminal has %d: %q", i, w, width, l)
		}
	}
	status := ansi.Strip(lines[len(lines)-1])
	if !strings.Contains(status, "help") {
		t.Errorf("status bar at %dx%d lost the help hint: %q", width, height, status)
	}
	if strings.Contains(status, "…") || strings.HasSuffix(strings.TrimSpace(status), "...") {
		t.Errorf("status bar at %dx%d cuts a hint: %q", width, height, status)
	}
	header := ansi.Strip(lines[0])
	if strings.Contains(header, "Backlogs Search") || strings.Contains(header, "Boards Search") || strings.Contains(header, "Sprints Search") {
		t.Errorf("breadcrumb runs into the search hint at %dx%d: %q", width, height, header)
	}
	if dir := os.Getenv("LAYOUT_DUMP_DIR"); dir != "" {
		name := fmt.Sprintf("%s_%dx%d.ans", strings.ReplaceAll(t.Name(), "/", "_"), width, height)
		_ = os.WriteFile(filepath.Join(dir, name), []byte(out), 0o644)
	}
}

func TestMainLayoutFitsTerminal(t *testing.T) {
	sizes := [][2]int{{200, 50}, {160, 45}, {139, 40}, {120, 32}, {99, 30}, {90, 26}, {70, 22}, {60, 20}, {50, 14}}
	for _, sz := range sizes {
		app := newTestApp(t, sz[0], sz[1])
		assertFits(t, app.render(), sz[0], sz[1])
	}
}

func TestHubLayoutsFitTerminal(t *testing.T) {
	hubs := map[string]tea.KeyPressMsg{
		"board":  {Code: 'B', Text: "B"},
		"sprint": {Code: 'S', Text: "S"},
	}
	for name, k := range hubs {
		t.Run(name, func(t *testing.T) {
			for _, sz := range [][2]int{{180, 46}, {110, 30}, {90, 26}, {70, 22}, {55, 18}} {
				var m tea.Model = newTestApp(t, sz[0], sz[1])
				m, _ = m.Update(k)
				app := m.(App)
				assertFits(t, app.render(), sz[0], sz[1])
			}
		})
	}
}

func TestWorkItemFormFitsTerminal(t *testing.T) {
	for _, sz := range [][2]int{{180, 46}, {100, 30}, {60, 20}} {
		var m tea.Model = newTestApp(t, sz[0], sz[1])
		app := m.(App)
		item := app.selectedItem()
		if item == nil {
			t.Fatal("no selected item")
		}
		m, _ = m.Update(panels.ViewWorkItemMsg{Item: *item})
		app = m.(App)
		assertFits(t, app.render(), sz[0], sz[1])
	}
}

func TestFilterPickersFitTerminal(t *testing.T) {
	hubs := map[string]string{"backlog": "L", "board": "B", "sprint": "S"}
	for name, hub := range hubs {
		t.Run(name, func(t *testing.T) {
			for _, sz := range [][2]int{{200, 50}, {160, 40}, {99, 30}, {80, 24}, {50, 14}} {
				for _, group := range []string{"1", "2", "3", "4"} {
					a := newTestAppAs(t, sz[0], sz[1], models.Identity{DisplayName: "Dana Reyes"})
					a, _ = step(t, a, press(hub))
					a, _ = step(t, a, press(group))
					if !topIs[modals.FilterPickerModal](a.modals) {
						t.Fatalf("%s did not open a filter picker at %dx%d", group, sz[0], sz[1])
					}
					assertFits(t, a.render(), sz[0], sz[1])
				}
			}
		})
	}
}

func TestFilterBarFitsTerminal(t *testing.T) {
	for _, hub := range []string{"L", "B", "S"} {
		for _, sz := range [][2]int{{200, 50}, {139, 40}, {90, 26}, {60, 20}, {50, 14}} {
			a := newTestApp(t, sz[0], sz[1])
			a, _ = step(t, a, press(hub))
			a, _ = step(t, a, press("f"))
			assertFits(t, a.render(), sz[0], sz[1])
		}
	}
}

func TestModalsFitTerminal(t *testing.T) {
	openers := map[string][]tea.Msg{
		"state":         {tea.KeyPressMsg{Code: 's', Text: "s"}},
		"assign":        {tea.KeyPressMsg{Code: 'a', Text: "a"}},
		"comment":       {tea.KeyPressMsg{Code: 'c', Text: "c"}},
		"edit":          {tea.KeyPressMsg{Code: 'e', Text: "e"}},
		"task":          {tea.KeyPressMsg{Code: 't', Text: "t"}},
		"delete":        {tea.KeyPressMsg{Code: 'd', Text: "d"}},
		"info":          {tea.KeyPressMsg{Code: 'i', Text: "i"}},
		"help":          {tea.KeyPressMsg{Code: '?', Text: "?"}},
		"quick_search":  {tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}},
		"parent_type":   {tea.KeyPressMsg{Code: 'n', Text: "n"}},
		"pbi_form":      {tea.KeyPressMsg{Code: 'n', Text: "n"}, tea.KeyPressMsg{Code: tea.KeyEnter}},
		"bug_form":      {tea.KeyPressMsg{Code: 'n', Text: "n"}, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter}},
		"filter_picker": {tea.KeyPressMsg{Code: '1', Text: "1"}, tea.KeyPressMsg{Code: 'S', Text: "S"}},
		"person_picker": {tea.KeyPressMsg{Code: 'S', Text: "S"}, tea.KeyPressMsg{Code: '2', Text: "2"}},
		"error":         {errMsg{err: fmt.Errorf("TF401232: Work item 99 does not exist, or you do not have permissions to read it.")}},
	}
	for name, msgs := range openers {
		t.Run(name, func(t *testing.T) {
			for _, sz := range [][2]int{{160, 45}, {80, 24}, {50, 14}} {
				var m tea.Model = newTestApp(t, sz[0], sz[1])
				for _, msg := range msgs {
					var cmd tea.Cmd
					m, cmd = m.Update(msg)
					for _, follow := range drain(cmd) {
						m, _ = m.Update(follow)
					}
				}
				app := m.(App)
				if !app.modals.open() {
					t.Fatalf("%s did not open a modal at %dx%d", name, sz[0], sz[1])
				}
				assertFits(t, app.render(), sz[0], sz[1])
			}
		})
	}
}

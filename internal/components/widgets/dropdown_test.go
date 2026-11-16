package widgets

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charmbracelet/x/ansi"
)

func dropdownFixture() Dropdown {
	d := NewDropdown(keys.DefaultKeyMap(), []DropdownItem{
		{Label: "All", Value: "all", Count: "9"},
		{Label: "Ana Ruiz", Value: "ana", Group: "Active", Count: "2"},
		{Label: "Ben Okafor", Value: "ben", Group: "Active", Count: "3"},
		{Label: "Marco Silva", Value: "marco", Group: "Idle", Muted: true},
	}, 10, "Type to filter")
	d.SetWidth(30)
	d.Focus()
	return d
}

func TestDropdownMovesAcrossGroupsWithoutLandingOnHeaders(t *testing.T) {
	d := dropdownFixture()
	for range 3 {
		d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if it, _ := d.Selected(); it.Value != "marco" {
		t.Fatalf("selected %q, want marco", it.Value)
	}
	view := ansi.Strip(d.View())
	for _, want := range []string{"Active", "Idle", "Ana Ruiz", "3"} {
		if !strings.Contains(view, want) {
			t.Errorf("view is missing %q:\n%s", want, view)
		}
	}
}

func TestDropdownFiltersByLabel(t *testing.T) {
	d := dropdownFixture()
	for _, r := range "ok" {
		d, _ = d.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if it, ok := d.Selected(); !ok || it.Value != "ben" {
		t.Fatalf("selected %+v, want ben", it)
	}
	if view := ansi.Strip(d.View()); strings.Contains(view, "Ana Ruiz") || strings.Contains(view, "Idle") {
		t.Fatalf("filtered view still shows other rows:\n%s", view)
	}
}

func TestDropdownKeepsItsHeightWhileFiltering(t *testing.T) {
	d := dropdownFixture()
	before := len(strings.Split(d.View(), "\n"))
	d.SetQuery("zzz")
	if after := len(strings.Split(d.View(), "\n")); after != before {
		t.Fatalf("height changed from %d to %d", before, after)
	}
}

package models

import "testing"

func peopleFixture() []WorkItem {
	items := []WorkItem{
		{ID: 1, Type: WorkItemTypePBI, Title: "Checkout", AssignedTo: "Ana Ruiz", AssignedToUniqueName: "ana@acme.io"},
		{ID: 2, Type: WorkItemTypeTask, Title: "Card vault", AssignedTo: "Ben Okafor", AssignedToUniqueName: "ben@acme.io", ParentID: 1},
		{ID: 3, Type: WorkItemTypeTask, Title: "Token API", AssignedTo: "Ben Okafor", AssignedToUniqueName: "ben@acme.io", ParentID: 1},
		{ID: 4, Type: WorkItemTypeTask, Title: "Date picker", ParentID: 1, Tags: []string{"ui"}},
		{ID: 5, Type: WorkItemTypeBug, Title: "Invoice emails", AssignedTo: "Dana Kim", AssignedToUniqueName: "dana@acme.io"},
	}
	for i := range items {
		if items[i].ParentID == 1 {
			items[0].Children = append(items[0].Children, &items[i])
		}
	}
	return items
}

func TestFilterTreeKeepsParentsOfMatchingChildren(t *testing.T) {
	out := FilterTree(peopleFixture(), func(it *WorkItem) bool { return it.AssignedTo == "Ben Okafor" })
	if len(out) != 3 || out[0].ID != 1 {
		t.Fatalf("got %d items starting at %d, want parent 1 and its two matching tasks", len(out), out[0].ID)
	}
	if len(out[0].Children) != 2 || out[0].Children[0] != &out[1] || out[0].Children[1] != &out[2] {
		t.Fatalf("children must point into the filtered slice, got %v", out[0].Children)
	}
}

func TestFilterTreeDropsChildrenThatDoNotMatch(t *testing.T) {
	out := FilterTree(peopleFixture(), func(it *WorkItem) bool { return it.ID == 1 })
	if len(out) != 1 || len(out[0].Children) != 0 {
		t.Fatalf("got %+v, want only the parent without children", out)
	}
}

func TestPeopleOptionsListsActivePeopleThenIdleMembers(t *testing.T) {
	members := []TeamMember{
		{DisplayName: "Ana Ruiz", UniqueName: "ana@acme.io"},
		{DisplayName: "Marco Silva", UniqueName: "marco@acme.io"},
	}
	me := Identity{DisplayName: "Ben Okafor", UniqueName: "BEN@acme.io"}
	opts := PeopleOptions(peopleFixture(), members, me)
	want := []struct {
		value   string
		count   int
		section OptionSection
		you     bool
	}{
		{FilterAll, 5, SectionNone, false},
		{AssignedMe, 2, SectionNone, false},
		{AssignedNone, 1, SectionNone, false},
		{"ben@acme.io", 2, SectionActive, true},
		{"ana@acme.io", 1, SectionActive, false},
		{"dana@acme.io", 1, SectionActive, false},
		{"marco@acme.io", 0, SectionIdle, false},
	}
	if len(opts) != len(want) {
		t.Fatalf("got %d options %+v, want %d", len(opts), opts, len(want))
	}
	for i, w := range want {
		o := opts[i]
		if o.Value != w.value || o.Count != w.count || o.Section != w.section || o.You != w.you {
			t.Errorf("option %d = %+v, want %+v", i, o, w)
		}
	}
}

func TestPeopleOptionsWithoutIdentityHidesMyCount(t *testing.T) {
	opts := PeopleOptions(peopleFixture(), nil, Identity{})
	if opts[1].Value != AssignedMe || opts[1].Count != UnknownCount {
		t.Fatalf("got %+v, want @Me with an unknown count", opts[1])
	}
}

func TestSetPeopleKeepsSelectionAcrossScopes(t *testing.T) {
	fs := NewFilterState(nil, nil, nil)
	fs.Group(FilterTypeAssigned).SelectByValue(AssignedMe)
	fs.SetPeople(peopleFixture(), nil, Identity{})
	if got := fs.GetSelectedAssigned(); got != AssignedMe {
		t.Fatalf("selection = %q, want me", got)
	}
	fs.Group(FilterTypeAssigned).SelectByValue("dana@acme.io")
	fs.SetPeople(peopleFixture()[:2], nil, Identity{})
	g := fs.Group(FilterTypeAssigned)
	if got := g.SelectedOption(); got.Value != "dana@acme.io" || got.Label != "Dana Kim" || got.Section != SectionIdle {
		t.Fatalf("got %+v, want Dana kept as an idle option", got)
	}
}

func TestApplySavedSelectionsKeepsAPersonNotLoadedYet(t *testing.T) {
	fs := NewFilterState(nil, nil, nil)
	fs.ApplySavedSelections("", "", "dana@acme.io", "")
	if got := fs.GetSelectedAssigned(); got != "dana@acme.io" {
		t.Fatalf("assigned = %q, want the saved person", got)
	}
}

func TestFilterOrderMatchesNumberKeys(t *testing.T) {
	fs := NewFilterState(nil, nil, nil)
	want := []FilterType{FilterTypeSprint, FilterTypeAssigned, FilterTypeState, FilterTypeArea}
	for i, g := range fs.Groups {
		if g.Type != want[i] {
			t.Fatalf("group %d is %v, want %v", i, g.Type, want[i])
		}
	}
}

func TestMatchers(t *testing.T) {
	items := peopleFixture()
	me := Identity{DisplayName: "Dana Kim"}
	cases := []struct {
		name string
		got  bool
		want bool
	}{
		{"me by display name", MatchesAssignee(&items[4], AssignedMe, me), true},
		{"me excludes others", MatchesAssignee(&items[0], AssignedMe, me), false},
		{"unknown me defers to the server", MatchesAssignee(&items[0], AssignedMe, Identity{}), true},
		{"unassigned", MatchesAssignee(&items[3], AssignedNone, me), true},
		{"person by unique name", MatchesAssignee(&items[1], "BEN@acme.io", me), true},
		{"keyword in title", MatchesKeyword(&items[1], "vault"), true},
		{"keyword in id", MatchesKeyword(&items[4], "5"), true},
		{"keyword in tag", MatchesKeyword(&items[3], "UI"), true},
		{"keyword miss", MatchesKeyword(&items[0], "invoice"), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

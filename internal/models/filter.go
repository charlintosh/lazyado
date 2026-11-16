package models

import (
	"sort"
	"strconv"
	"strings"
)

// FilterType represents the type of filter
type FilterType int

const (
	FilterTypeSprint FilterType = iota
	FilterTypeAssigned
	FilterTypeState
	FilterTypeArea
)

const (
	FilterAll    = "all"
	AssignedMe   = "me"
	AssignedNone = "none"
)

type OptionSection int

const (
	SectionNone OptionSection = iota
	SectionActive
	SectionIdle
)

const UnknownCount = -1

// FilterOption represents a selectable filter option
type FilterOption struct {
	Label    string
	Value    string
	Selected bool
	Section  OptionSection
	Count    int
	You      bool
}

// FilterGroup represents a group of filter options
type FilterGroup struct {
	Type       FilterType
	Title      string
	Options    []FilterOption
	ShowCounts bool
}

// SelectedOption returns the currently selected option
func (f *FilterGroup) SelectedOption() *FilterOption {
	for i := range f.Options {
		if f.Options[i].Selected {
			return &f.Options[i]
		}
	}
	if len(f.Options) > 0 {
		return &f.Options[0]
	}
	return nil
}

func (f *FilterGroup) Value() string {
	if opt := f.SelectedOption(); opt != nil {
		return opt.Value
	}
	return FilterAll
}

func (f *FilterGroup) IsSet() bool {
	return f.Value() != FilterAll
}

// Select marks the option at the given index as selected
func (f *FilterGroup) Select(index int) {
	if index < 0 || index >= len(f.Options) {
		return
	}
	for i := range f.Options {
		f.Options[i].Selected = i == index
	}
}

// SelectByValue marks the option with the given value as selected
func (f *FilterGroup) SelectByValue(value string) bool {
	for i := range f.Options {
		if f.Options[i].Value == value {
			f.Select(i)
			return true
		}
	}
	return false
}

// FilterState holds the complete filter state
type FilterState struct {
	Groups []*FilterGroup
}

// NewFilterState creates a new filter state with default groups
func NewFilterState(iterations []Iteration, areas []Area, statesByType map[string][]WorkItemStateInfo) *FilterState {
	return &FilterState{
		Groups: []*FilterGroup{
			sprintGroup(iterations),
			assignedGroup(),
			stateGroup(statesByType),
			areaGroup(areas),
		},
	}
}

func sprintGroup(iterations []Iteration) *FilterGroup {
	options := []FilterOption{{Label: "All", Value: FilterAll}}
	current := 0
	for _, iter := range iterations {
		if iter.IsCurrent() && current == 0 {
			current = len(options)
		}
		options = append(options, FilterOption{Label: iter.DisplayName(), Value: iter.Path})
	}
	options[current].Selected = true
	return &FilterGroup{Type: FilterTypeSprint, Title: "Iteration", Options: options}
}

func assignedGroup() *FilterGroup {
	return &FilterGroup{
		Type:       FilterTypeAssigned,
		Title:      "Person",
		Options:    pinnedPeople(UnknownCount, UnknownCount, UnknownCount, AssignedMe),
		ShowCounts: true,
	}
}

func pinnedPeople(total, mine, unassigned int, selected string) []FilterOption {
	return []FilterOption{
		{Label: "All", Value: FilterAll, Count: total, Selected: selected == FilterAll},
		{Label: "@Me", Value: AssignedMe, Count: mine, Selected: selected == AssignedMe},
		{Label: "Unassigned", Value: AssignedNone, Count: unassigned, Selected: selected == AssignedNone},
	}
}

func stateGroup(statesByType map[string][]WorkItemStateInfo) *FilterGroup {
	options := []FilterOption{{Label: "All", Value: FilterAll, Selected: true}}
	seen := map[string]bool{}
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			options = append(options, FilterOption{Label: name, Value: name})
		}
	}
	for _, category := range []string{"Proposed", "InProgress", "Resolved", "Completed", "Removed"} {
		for _, states := range statesByType {
			for _, state := range states {
				if state.Category == category {
					add(state.Name)
				}
			}
		}
	}
	for _, states := range statesByType {
		for _, state := range states {
			add(state.Name)
		}
	}
	if len(options) == 1 {
		for _, state := range []string{"New", "Active", "Resolved", "Closed"} {
			add(state)
		}
	}
	return &FilterGroup{Type: FilterTypeState, Title: "State", Options: options}
}

func areaGroup(areas []Area) *FilterGroup {
	options := []FilterOption{{Label: "All", Value: FilterAll, Selected: true}}
	for _, area := range areas {
		options = append(options, FilterOption{Label: area.DisplayName(), Value: area.Path})
	}
	return &FilterGroup{Type: FilterTypeArea, Title: "Area", Options: options}
}

func (f *FilterState) Group(t FilterType) *FilterGroup {
	for _, g := range f.Groups {
		if g.Type == t {
			return g
		}
	}
	return nil
}

func (f *FilterState) value(t FilterType) string {
	if g := f.Group(t); g != nil {
		return g.Value()
	}
	return FilterAll
}

// GetSelectedSprint returns the selected sprint path
func (f *FilterState) GetSelectedSprint() string {
	return f.value(FilterTypeSprint)
}

// GetSelectedState returns the selected state filter
func (f *FilterState) GetSelectedState() string {
	return f.value(FilterTypeState)
}

// GetSelectedAssigned returns the selected assigned filter
func (f *FilterState) GetSelectedAssigned() string {
	return f.value(FilterTypeAssigned)
}

// GetSelectedArea returns the selected area path
func (f *FilterState) GetSelectedArea() string {
	return f.value(FilterTypeArea)
}

func (f *FilterState) ActiveCount(types ...FilterType) int {
	n := 0
	for _, t := range types {
		if g := f.Group(t); g != nil && g.IsSet() {
			n++
		}
	}
	return n
}

func (f *FilterState) Reset(types ...FilterType) {
	for _, t := range types {
		if g := f.Group(t); g != nil {
			g.SelectByValue(FilterAll)
		}
	}
}

// ApplySavedSelections applies saved filter selections
func (f *FilterState) ApplySavedSelections(sprint, state, assigned, area string) {
	saved := map[FilterType]string{
		FilterTypeSprint:   sprint,
		FilterTypeState:    state,
		FilterTypeAssigned: assigned,
		FilterTypeArea:     area,
	}
	for _, g := range f.Groups {
		value := saved[g.Type]
		if value == "" || g.SelectByValue(value) || g.Type != FilterTypeAssigned {
			continue
		}
		g.Options = append(g.Options, FilterOption{Label: value, Value: value, Section: SectionIdle, Count: UnknownCount})
		g.SelectByValue(value)
	}
}

func (f *FilterState) SetPeople(items []WorkItem, members []TeamMember, me Identity) {
	g := f.Group(FilterTypeAssigned)
	if g == nil {
		return
	}
	previous := *g.SelectedOption()
	g.Options = PeopleOptions(items, members, me)
	if g.SelectByValue(previous.Value) {
		return
	}
	previous.Section, previous.Count = SectionIdle, 0
	g.Options = append(g.Options, previous)
	g.SelectByValue(previous.Value)
}

type assigneeTally struct {
	name   string
	value  string
	count  int
	isSelf bool
}

func PeopleOptions(items []WorkItem, members []TeamMember, me Identity) []FilterOption {
	byValue := map[string]*assigneeTally{}
	byName := map[string]bool{}
	var tallies []*assigneeTally
	mine, unassigned := 0, 0
	for i := range items {
		it := &items[i]
		if it.AssignedTo == "" {
			unassigned++
			continue
		}
		self := me.Known() && me.Matches(it.AssignedTo, it.AssignedToUniqueName)
		if self {
			mine++
		}
		value := AssigneeValue(it.AssignedTo, it.AssignedToUniqueName)
		t, ok := byValue[strings.ToLower(value)]
		if !ok {
			t = &assigneeTally{name: it.AssignedTo, value: value, isSelf: self}
			byValue[strings.ToLower(value)] = t
			byName[it.AssignedTo] = true
			tallies = append(tallies, t)
		}
		t.count++
	}
	sort.SliceStable(tallies, func(i, j int) bool {
		if tallies[i].count != tallies[j].count {
			return tallies[i].count > tallies[j].count
		}
		return tallies[i].name < tallies[j].name
	})
	if !me.Known() {
		mine = UnknownCount
	}
	options := pinnedPeople(len(items), mine, unassigned, "")
	for _, t := range tallies {
		options = append(options, FilterOption{Label: t.name, Value: t.value, Section: SectionActive, Count: t.count, You: t.isSelf})
	}
	idle := make([]TeamMember, 0, len(members))
	for _, m := range members {
		value := AssigneeValue(m.DisplayName, m.UniqueName)
		if _, ok := byValue[strings.ToLower(value)]; ok || byName[m.DisplayName] {
			continue
		}
		idle = append(idle, m)
	}
	sort.SliceStable(idle, func(i, j int) bool { return idle[i].DisplayName < idle[j].DisplayName })
	for _, m := range idle {
		options = append(options, FilterOption{
			Label:   m.DisplayName,
			Value:   AssigneeValue(m.DisplayName, m.UniqueName),
			Section: SectionIdle,
			You:     me.Known() && me.Matches(m.DisplayName, m.UniqueName),
		})
	}
	return options
}

func AssigneeValue(displayName, uniqueName string) string {
	if uniqueName != "" {
		return uniqueName
	}
	return displayName
}

func MatchesAssignee(item *WorkItem, value string, me Identity) bool {
	switch value {
	case FilterAll, "":
		return true
	case AssignedMe:
		return !me.Known() || me.Matches(item.AssignedTo, item.AssignedToUniqueName)
	case AssignedNone:
		return item.AssignedTo == ""
	}
	return strings.EqualFold(value, item.AssignedToUniqueName) || value == item.AssignedTo
}

func MatchesKeyword(item *WorkItem, keyword string) bool {
	q := strings.ToLower(strings.TrimSpace(keyword))
	if q == "" {
		return true
	}
	if strings.Contains(strconv.Itoa(item.ID), q) || strings.Contains(strings.ToLower(item.Title), q) {
		return true
	}
	for _, tag := range item.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

func FilterTree(items []WorkItem, keep func(*WorkItem) bool) []WorkItem {
	index := make(map[int]int, len(items))
	for i := range items {
		index[items[i].ID] = i
	}
	visible := make([]bool, len(items))
	for i := range items {
		if !keep(&items[i]) {
			continue
		}
		for j, ok := i, true; ok && !visible[j]; j, ok = index[items[j].ParentID] {
			visible[j] = true
		}
	}
	out := make([]WorkItem, 0, len(items))
	position := make(map[int]int, len(items))
	for i := range items {
		if visible[i] {
			item := items[i]
			item.Children = nil
			position[item.ID] = len(out)
			out = append(out, item)
		}
	}
	for i := range out {
		if p, ok := position[out[i].ParentID]; ok && p != i {
			out[p].Children = append(out[p].Children, &out[i])
		}
	}
	return out
}

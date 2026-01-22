package models

// FilterType represents the type of filter
type FilterType int

const (
	FilterTypeSprint FilterType = iota
	FilterTypeState
	FilterTypeAssigned
	FilterTypeArea
)

// FilterOption represents a selectable filter option
type FilterOption struct {
	Label    string
	Value    string
	Selected bool
}

// FilterGroup represents a group of filter options
type FilterGroup struct {
	Type       FilterType
	Title      string
	Options    []FilterOption
	Cursor     int
	Offset     int            // Scroll offset for viewing
	allOptions []FilterOption // Original unfiltered options
	isFiltered bool           // Whether search filter is active
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

// Select marks the option at the given index as selected
func (f *FilterGroup) Select(index int) {
	if index < 0 || index >= len(f.Options) {
		return
	}
	for i := range f.Options {
		f.Options[i].Selected = i == index
	}
}

// SelectCurrent marks the option at the current cursor as selected
func (f *FilterGroup) SelectCurrent() {
	f.Select(f.Cursor)
}

// SelectByValue marks the option with the given value as selected
func (f *FilterGroup) SelectByValue(value string) {
	for i := range f.Options {
		if f.Options[i].Value == value {
			f.Options[i].Selected = true
			f.Cursor = i
		} else {
			f.Options[i].Selected = false
		}
	}
}

// MoveUp moves the cursor up
func (f *FilterGroup) MoveUp() {
	if f.Cursor > 0 {
		f.Cursor--
	}
}

// MoveDown moves the cursor down
func (f *FilterGroup) MoveDown() {
	if f.Cursor < len(f.Options)-1 {
		f.Cursor++
	}
}

// MoveToTop moves cursor to the first option
func (f *FilterGroup) MoveToTop() {
	f.Cursor = 0
}

// MoveToBottom moves cursor to the last option
func (f *FilterGroup) MoveToBottom() {
	if len(f.Options) > 0 {
		f.Cursor = len(f.Options) - 1
	}
}

// ApplySearchFilter filters options based on search query
func (f *FilterGroup) ApplySearchFilter(query string) {
	if query == "" {
		f.ClearSearchFilter()
		return
	}

	// Store original options on first filter
	if !f.isFiltered {
		f.allOptions = make([]FilterOption, len(f.Options))
		copy(f.allOptions, f.Options)
		f.isFiltered = true
	}

	// Filter options by query (case-insensitive substring match)
	filtered := []FilterOption{}
	lowerQuery := toLower(query)
	for _, opt := range f.allOptions {
		if contains(toLower(opt.Label), lowerQuery) {
			filtered = append(filtered, opt)
		}
	}

	f.Options = filtered
	f.Cursor = 0
	f.Offset = 0
}

// ClearSearchFilter restores all options
func (f *FilterGroup) ClearSearchFilter() {
	if f.isFiltered && f.allOptions != nil {
		f.Options = make([]FilterOption, len(f.allOptions))
		copy(f.Options, f.allOptions)
		f.isFiltered = false
	}
}

// Helper functions for case-insensitive string operations
func toLower(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r = r + 32
		}
		result = append(result, r)
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// FilterState holds the complete filter state
type FilterState struct {
	Groups      []*FilterGroup
	ActiveGroup int
	SearchQuery string
}

// NewFilterState creates a new filter state with default groups
func NewFilterState(iterations []Iteration, areas []Area, statesByType map[string][]WorkItemStateInfo) *FilterState {
	// Build sprint options from iterations
	sprintOptions := []FilterOption{
		{Label: "All", Value: "all", Selected: false},
	}

	for _, iter := range iterations {
		selected := iter.IsCurrent()
		sprintOptions = append(sprintOptions, FilterOption{
			Label:    iter.DisplayName(),
			Value:    iter.Path,
			Selected: selected,
		})
	}

	// If no current sprint found, select "All"
	hasSelected := false
	selectedIdx := 0
	for i, opt := range sprintOptions {
		if opt.Selected {
			hasSelected = true
			selectedIdx = i
			break
		}
	}
	if !hasSelected && len(sprintOptions) > 0 {
		sprintOptions[0].Selected = true
	}

	// Calculate initial cursor and offset for sprint group
	sprintCursor := selectedIdx
	sprintOffset := 0
	const maxVisibleOptions = 6
	if sprintCursor >= maxVisibleOptions {
		sprintOffset = sprintCursor - maxVisibleOptions + 1
	}

	// Build area options from areas
	areaOptions := []FilterOption{
		{Label: "All", Value: "all", Selected: true},
	}
	areaCursor := 0
	for i, area := range areas {
		areaOptions = append(areaOptions, FilterOption{
			Label:    area.DisplayName(),
			Value:    area.Path,
			Selected: false,
		})
		// Track if this area becomes selected later
		_ = i
	}
	areaOffset := 0
	if areaCursor >= maxVisibleOptions {
		areaOffset = areaCursor - maxVisibleOptions + 1
	}

	// Build state options from all work item types (unique states)
	stateOptions := []FilterOption{
		{Label: "All", Value: "all", Selected: true},
	}
	if len(statesByType) > 0 {
		// Collect unique states preserving order by category
		seenStates := make(map[string]bool)
		// Process in category order: Proposed, InProgress, Resolved, Completed
		categoryOrder := []string{"Proposed", "InProgress", "Resolved", "Completed", "Removed"}
		for _, category := range categoryOrder {
			for _, states := range statesByType {
				for _, state := range states {
					if state.Category == category && !seenStates[state.Name] {
						seenStates[state.Name] = true
						stateOptions = append(stateOptions, FilterOption{
							Label:    state.Name,
							Value:    state.Name,
							Selected: false,
						})
					}
				}
			}
		}
		// Also add any states that didn't match known categories
		for _, states := range statesByType {
			for _, state := range states {
				if !seenStates[state.Name] {
					seenStates[state.Name] = true
					stateOptions = append(stateOptions, FilterOption{
						Label:    state.Name,
						Value:    state.Name,
						Selected: false,
					})
				}
			}
		}
	}
	// Fallback to default states if we only have "All"
	if len(stateOptions) <= 1 {
		defaultStates := []string{"New", "Active", "Resolved", "Closed"}
		for _, state := range defaultStates {
			stateOptions = append(stateOptions, FilterOption{
				Label:    state,
				Value:    state,
				Selected: false,
			})
		}
	}

	// Calculate cursor and offset for state group
	stateCursor := 0
	for i, opt := range stateOptions {
		if opt.Selected {
			stateCursor = i
			break
		}
	}
	stateOffset := 0
	if stateCursor >= maxVisibleOptions {
		stateOffset = stateCursor - maxVisibleOptions + 1
	}

	// Calculate cursor for assigned group
	assignedOptions := []FilterOption{
		{Label: "All", Value: "all", Selected: false},
		{Label: "Me", Value: "me", Selected: true},
	}
	assignedCursor := 1 // "Me" is selected by default

	return &FilterState{
		Groups: []*FilterGroup{
			{
				Type:    FilterTypeSprint,
				Title:   "Sprint",
				Options: sprintOptions,
				Cursor:  sprintCursor,
				Offset:  sprintOffset,
			},
			{
				Type:    FilterTypeState,
				Title:   "State",
				Options: stateOptions,
				Cursor:  stateCursor,
				Offset:  stateOffset,
			},
			{
				Type:    FilterTypeAssigned,
				Title:   "Assigned",
				Options: assignedOptions,
				Cursor:  assignedCursor,
				Offset:  0,
			},
			{
				Type:    FilterTypeArea,
				Title:   "Area",
				Options: areaOptions,
				Cursor:  areaCursor,
				Offset:  areaOffset,
			},
		},
		ActiveGroup: 0,
	}
}

// ActiveFilterGroup returns the currently active filter group
func (f *FilterState) ActiveFilterGroup() *FilterGroup {
	if f.ActiveGroup >= 0 && f.ActiveGroup < len(f.Groups) {
		return f.Groups[f.ActiveGroup]
	}
	return nil
}

// NextGroup moves to the next filter group
func (f *FilterState) NextGroup() {
	if f.ActiveGroup < len(f.Groups)-1 {
		f.ActiveGroup++
	}
}

// PrevGroup moves to the previous filter group
func (f *FilterState) PrevGroup() {
	if f.ActiveGroup > 0 {
		f.ActiveGroup--
	}
}

// GetSelectedSprint returns the selected sprint path
func (f *FilterState) GetSelectedSprint() string {
	for _, g := range f.Groups {
		if g.Type == FilterTypeSprint {
			if opt := g.SelectedOption(); opt != nil {
				return opt.Value
			}
		}
	}
	return "all"
}

// GetSelectedState returns the selected state filter
func (f *FilterState) GetSelectedState() string {
	for _, g := range f.Groups {
		if g.Type == FilterTypeState {
			if opt := g.SelectedOption(); opt != nil {
				return opt.Value
			}
		}
	}
	return "all"
}

// GetSelectedAssigned returns the selected assigned filter
func (f *FilterState) GetSelectedAssigned() string {
	for _, g := range f.Groups {
		if g.Type == FilterTypeAssigned {
			if opt := g.SelectedOption(); opt != nil {
				return opt.Value
			}
		}
	}
	return "all"
}

// GetSelectedArea returns the selected area path
func (f *FilterState) GetSelectedArea() string {
	for _, g := range f.Groups {
		if g.Type == FilterTypeArea {
			if opt := g.SelectedOption(); opt != nil {
				return opt.Value
			}
		}
	}
	return "all"
}

// ApplySavedSelections applies saved filter selections
func (f *FilterState) ApplySavedSelections(sprint, state, assigned, area string) {
	const maxVisibleOptions = 6

	for _, g := range f.Groups {
		var targetValue string
		switch g.Type {
		case FilterTypeSprint:
			targetValue = sprint
		case FilterTypeState:
			targetValue = state
		case FilterTypeAssigned:
			targetValue = assigned
		case FilterTypeArea:
			targetValue = area
		}

		if targetValue == "" {
			continue
		}

		// Find and select the matching option
		found := false
		for i, opt := range g.Options {
			if opt.Value == targetValue {
				g.Select(i)
				g.Cursor = i
				// Adjust offset to make selected item visible
				if g.Cursor >= maxVisibleOptions {
					g.Offset = g.Cursor - maxVisibleOptions + 1
				} else {
					g.Offset = 0
				}
				found = true
				break
			}
		}

		// If not found and it's sprint with "current", keep the current selection
		if !found && g.Type == FilterTypeSprint && targetValue == "current" {
			// Already handled in NewFilterState - just ensure cursor is on selected
			for i, opt := range g.Options {
				if opt.Selected {
					g.Cursor = i
					if g.Cursor >= maxVisibleOptions {
						g.Offset = g.Cursor - maxVisibleOptions + 1
					} else {
						g.Offset = 0
					}
					break
				}
			}
		}
	}
}

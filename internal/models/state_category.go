package models

var stateCategories = map[string]string{
	"New":         "Proposed",
	"Proposed":    "Proposed",
	"To Do":       "Proposed",
	"Approved":    "Proposed",
	"Active":      "InProgress",
	"Committed":   "InProgress",
	"In Progress": "InProgress",
	"Testing":     "InProgress",
	"Resolved":    "Resolved",
	"Done":        "Completed",
	"Closed":      "Completed",
	"Removed":     "Removed",
}

var workflowOrder = []string{
	"New", "Proposed", "To Do", "Approved", "Active", "Committed",
	"In Progress", "Testing", "Resolved", "Done", "Closed",
}

func RegisterStateCategories(statesByType map[string][]WorkItemStateInfo) {
	for _, states := range statesByType {
		for _, st := range states {
			if st.Name != "" && st.Category != "" {
				stateCategories[st.Name] = st.Category
			}
		}
	}
}

func StateCategory(state string) string {
	return stateCategories[state]
}

func IsCompletedState(state string) bool {
	return stateCategories[state] == "Completed"
}

func BoardColumns(statesByType map[string][]WorkItemStateInfo, workItemType WorkItemType, present []string) []string {
	var cols []string
	seen := map[string]bool{}
	for _, st := range statesByType[string(workItemType)] {
		if st.Category == "Removed" || seen[st.Name] {
			continue
		}
		seen[st.Name] = true
		cols = append(cols, st.Name)
	}
	if len(cols) == 0 {
		presentSet := map[string]bool{}
		for _, p := range present {
			presentSet[p] = true
		}
		for _, name := range workflowOrder {
			if presentSet[name] {
				seen[name] = true
				cols = append(cols, name)
			}
		}
	}
	for _, p := range present {
		if !seen[p] && stateCategories[p] != "Removed" {
			seen[p] = true
			cols = append(cols, p)
		}
	}
	return cols
}

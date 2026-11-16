package modals

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

var defaultStates = []string{"New", "Active", "Resolved", "Closed"}

type StateChangeRequestMsg struct {
	Item     models.WorkItem
	NewState string
}

func NewStateModal(env Env, item models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) ChoiceModal {
	current := string(item.State)
	states := statesFor(item, statesByType)
	options := make([]widgets.Option, len(states))
	for i, s := range states {
		options[i] = widgets.Option{Label: s, Value: s, Marked: s == current}
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Change state",
		subtitle: itemLabel(item),
		note:     "Current: " + current,
		action:   "apply",
		options:  options,
		initial:  current,
		noun:     "state",
		nouns:    "states",
		width:    styles.ModalWidthSM,
		rows:     styles.PickerRows,
		choose: func(o widgets.Option) tea.Msg {
			return StateChangeRequestMsg{Item: item, NewState: o.Value}
		},
	})
}

func statesFor(item models.WorkItem, statesByType map[string][]models.WorkItemStateInfo) []string {
	infos := statesByType[string(item.Type)]
	if len(infos) == 0 {
		return defaultStates
	}
	states := make([]string, len(infos))
	for i, s := range infos {
		states[i] = s.Name
	}
	return states
}

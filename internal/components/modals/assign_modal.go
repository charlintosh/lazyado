package modals

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type AssignRequestMsg struct {
	Item      models.WorkItem
	UserEmail string
	UserName  string
}

func NewAssignModal(env Env, item models.WorkItem, members []models.TeamMember) ChoiceModal {
	options, initial := memberOptions(members, item.AssignedTo)
	current := item.AssignedTo
	if current == "" {
		current = "Unassigned"
	}
	return newChoiceModal(env, choiceSpec{
		title:       "Assign to",
		subtitle:    itemLabel(item),
		note:        "Current: " + current,
		action:      "assign",
		options:     options,
		initial:     initial,
		search:      true,
		placeholder: "Type to filter members...",
		noun:        "member",
		nouns:       "members",
		width:       styles.ModalWidthMD,
		rows:        styles.PickerRows,
		choose: func(o widgets.Option) tea.Msg {
			return AssignRequestMsg{Item: item, UserEmail: o.Value, UserName: o.Label}
		},
	})
}

func memberOptions(members []models.TeamMember, assignedTo string) ([]widgets.Option, string) {
	options := make([]widgets.Option, len(members))
	initial := ""
	for i, m := range members {
		marked := assignedTo != "" && m.DisplayName == assignedTo
		if marked {
			initial = m.UniqueName
		}
		options[i] = widgets.Option{Label: m.DisplayName, Value: m.UniqueName, Marked: marked}
	}
	return options, initial
}

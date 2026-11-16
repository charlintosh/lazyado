package modals

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type ParentTypeChosenMsg struct {
	Type models.WorkItemType
}

func NewParentTypeModal(env Env) ChoiceModal {
	types := []models.WorkItemType{models.WorkItemTypePBI, models.WorkItemTypeBug}
	options := make([]widgets.Option, len(types))
	for i, t := range types {
		options[i] = widgets.Option{Label: string(t), Value: string(t)}
	}
	return newChoiceModal(env, choiceSpec{
		title:   "Create parent work item",
		action:  "continue",
		options: options,
		width:   styles.ModalWidthSM,
		rows:    len(options),
		choose: func(o widgets.Option) tea.Msg {
			return ParentTypeChosenMsg{Type: models.WorkItemType(o.Value)}
		},
	})
}

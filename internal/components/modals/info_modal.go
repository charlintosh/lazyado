package modals

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

type CopyRequestMsg struct {
	Text   string
	Notice string
}

func NewInfoModal(env Env, item models.WorkItem) ChoiceModal {
	copies := map[string]CopyRequestMsg{
		"url": {Text: item.WebURL, Notice: "URL copied to clipboard"},
		"id":  {Text: strconv.Itoa(item.ID), Notice: "ID copied to clipboard"},
	}
	options := []widgets.Option{
		{Label: "Copy URL", Value: "url"},
		{Label: "Copy ID", Value: "id"},
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Work item info",
		subtitle: itemLabel(item),
		action:   "copy",
		options:  options,
		width:    styles.ModalWidthSM,
		rows:     len(options),
		choose: func(o widgets.Option) tea.Msg {
			return copies[o.Value]
		},
	})
}

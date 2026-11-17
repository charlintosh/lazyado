package modals

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/components/widgets"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
)

const (
	branchValuePrefix = "branch:"
	tagValuePrefix    = "tag:"
)

type VersionChosenMsg struct {
	Version models.GitVersion
}

func NewCloneModal(env Env, repo models.Repository) ChoiceModal {
	copies := map[string]CopyRequestMsg{
		"https": {Text: repo.RemoteURL, Notice: "HTTPS clone URL copied to clipboard"},
		"ssh":   {Text: repo.SSHURL, Notice: "SSH clone URL copied to clipboard"},
		"cmd":   {Text: "git clone " + repo.RemoteURL, Notice: "Clone command copied to clipboard"},
	}
	var options []widgets.Option
	if repo.RemoteURL != "" {
		options = append(options, widgets.Option{Label: "HTTPS    " + repo.RemoteURL, Value: "https"})
	}
	if repo.SSHURL != "" {
		options = append(options, widgets.Option{Label: "SSH      " + repo.SSHURL, Value: "ssh"})
	}
	if repo.RemoteURL != "" {
		options = append(options, widgets.Option{Label: "Command  git clone " + repo.RemoteURL, Value: "cmd"})
	}
	return newChoiceModal(env, choiceSpec{
		title:    "Clone " + repo.Name,
		note:     "SSH needs a public key in User settings › SSH public keys.",
		action:   "copy",
		options:  options,
		width:    styles.ModalWidthPR,
		rows:     len(options),
		choose:   func(o widgets.Option) tea.Msg { return copies[o.Value] },
		subtitle: "Copy a URL, then run git clone in your terminal",
	})
}

func NewVersionPickerModal(env Env, branches, tags []string, defaultBranch string, current models.GitVersion) ChoiceModal {
	options := versionOptions(branches, tags, defaultBranch, current)
	initial := versionValue(current)
	return newChoiceModal(env, choiceSpec{
		title:       "Branch or tag",
		action:      "choose",
		options:     options,
		initial:     initial,
		search:      true,
		placeholder: "Type to filter branches and tags",
		noun:        "ref",
		nouns:       "refs",
		width:       styles.ModalWidthLG,
		rows:        min(len(options), styles.PickerRows),
		choose: func(o widgets.Option) tea.Msg {
			return VersionChosenMsg{Version: versionFromValue(o.Value)}
		},
	})
}

package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charlintosh/lazyado/pkg/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InfoOption represents an option in the info modal
type InfoOption int

const (
	InfoOptionCopyURL InfoOption = iota
	InfoOptionCopyID
)

// InfoModal is a modal for showing work item info and copy options
type InfoModal struct {
	visible bool
	item    *models.WorkItem
	cursor  InfoOption
	styles  styles.Styles
	keys    keys.KeyMap
	width   int
	height  int
}

// NewInfoModal creates a new info modal
func NewInfoModal(st styles.Styles, keys keys.KeyMap) InfoModal {
	return InfoModal{
		styles: st,
		keys:   keys,
	}
}

// Init initializes the modal
func (m InfoModal) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m InfoModal) Update(msg tea.Msg) (InfoModal, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > InfoOptionCopyURL {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < InfoOptionCopyID {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Select):
			if m.item != nil {
				var text string
				var notifText string
				switch m.cursor {
				case InfoOptionCopyURL:
					text = m.item.WebURL
					notifText = "URL copied to clipboard"
				case InfoOptionCopyID:
					text = fmt.Sprintf("%d", m.item.ID)
					notifText = "ID copied to clipboard"
				}
				if err := clipboard.Copy(text); err != nil {
					return m, func() tea.Msg {
						return InfoCopyErrorMsg{Err: err}
					}
				}
				m.visible = false
				return m, func() tea.Msg {
					return InfoCopiedMsg{Text: notifText}
				}
			}
		case key.Matches(msg, m.keys.Back):
			m.visible = false
			return m, func() tea.Msg { return ModalClosedMsg{} }
		}
	}

	return m, nil
}

// View renders the modal
func (m InfoModal) View() string {
	if !m.visible {
		return ""
	}

	// Modal dimensions
	modalWidth := styles.ModalWidthSM
	modalHeight := styles.ModalHeightMD

	// Build content
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().Bold(true).Render("Work Item Info")
	if m.item != nil {
		itemInfo := lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Render("#" + components.Itoa(m.item.ID) + " " + components.TruncateStr(m.item.Title, 25))
		b.WriteString(title + "\n")
		b.WriteString(itemInfo + "\n\n")
	}

	// Options
	options := []string{"Copy URL", "Copy ID"}
	for i, opt := range options {
		cursor := "  "
		if InfoOption(i) == m.cursor {
			cursor = "▸ "
		}

		style := lipgloss.NewStyle()
		if InfoOption(i) == m.cursor {
			style = style.Bold(true).Foreground(styles.ColorPrimary)
		}

		b.WriteString(cursor + style.Render(opt) + "\n")
	}

	// Help text
	b.WriteString("\n")
	sel := m.keys.Select.Help()
	back := m.keys.Back.Help()
	instr := fmt.Sprintf("%s: %s  %s: %s", sel.Key, sel.Desc, back.Key, back.Desc)
	b.WriteString(m.styles.ModalInstructions.Render(instr))

	// Modal style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(styles.ModalPaddingV, styles.ModalPaddingH).
		Width(modalWidth).
		Height(modalHeight)

	modal := modalStyle.Render(b.String())

	// Center the modal
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// SetVisible sets the visibility
func (m *InfoModal) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.cursor = InfoOptionCopyURL
	}
}

// IsVisible returns whether the modal is visible
func (m *InfoModal) IsVisible() bool {
	return m.visible
}

// SetItem sets the work item to display info for
func (m *InfoModal) SetItem(item *models.WorkItem) {
	m.item = item
}

// SetSize sets the modal container size
func (m *InfoModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// InfoCopiedMsg is sent when text was copied successfully
type InfoCopiedMsg struct {
	Text string
}

// InfoCopyErrorMsg is sent when copying failed
type InfoCopyErrorMsg struct {
	Err error
}

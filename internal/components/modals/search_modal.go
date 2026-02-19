package modals

import (
	"fmt"
	"strings"

	"github.com/charlintosh/lazyado/internal/components"
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// searchMaxVisibleResults aliases styles.ModalListVisibleItems for local readability.
const searchMaxVisibleResults = styles.ModalListVisibleItems

// SearchModal is a modal for searching and selecting from filter options
type SearchModal struct {
	styles      styles.Styles
	keys        keys.KeyMap
	visible     bool
	width       int
	height      int
	searchInput textinput.Model
	filterGroup *models.FilterGroup
	cursor      int
	offset      int
}

// NewSearchModal creates a new search modal
func NewSearchModal(st styles.Styles, keys keys.KeyMap) SearchModal {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.CharLimit = styles.InputCharLimitMD
	ti.Prompt = "🔍 "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.ColorText)

	return SearchModal{
		styles:      st,
		keys:        keys,
		searchInput: ti,
	}
}

// Update handles messages for the search modal
func (s SearchModal) Update(msg tea.Msg) (SearchModal, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			// Cancel search
			return s, func() tea.Msg { return SearchCancelledMsg{} }
		case tea.KeyEnter:
			// Select current item
			if s.filterGroup != nil && len(s.filterGroup.Options) > 0 {
				return s, func() tea.Msg {
					return SearchSelectedMsg{
						SelectedOption: &s.filterGroup.Options[s.cursor],
					}
				}
			}
		case tea.KeyUp:
			if s.cursor > 0 {
				s.cursor--
				s.adjustOffset()
			}
		case tea.KeyDown:
			if s.filterGroup != nil && s.cursor < len(s.filterGroup.Options)-1 {
				s.cursor++
				s.adjustOffset()
			}
		default:
			// Update search input
			var cmd tea.Cmd
			s.searchInput, cmd = s.searchInput.Update(msg)
			// Apply search filter
			if s.filterGroup != nil {
				s.filterGroup.ApplySearchFilter(s.searchInput.Value())
				s.cursor = 0
				s.offset = 0
			}
			return s, cmd
		}
	}

	return s, nil
}

// adjustOffset ensures the cursor is visible
func (s *SearchModal) adjustOffset() {
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+searchMaxVisibleResults {
		s.offset = s.cursor - searchMaxVisibleResults + 1
	}
}

// View renders the search modal
func (s SearchModal) View() string {
	if !s.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		MarginBottom(1)

	title := "Search "
	if s.filterGroup != nil {
		title += s.filterGroup.Title
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(s.searchInput.View())
	b.WriteString("\n\n")

	// Results
	if s.filterGroup == nil || len(s.filterGroup.Options) == 0 {
		noResults := lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Italic(true).
			Render("No results found")
		b.WriteString(noResults)
	} else {
		// Show filtered results
		startIdx := s.offset
		endIdx := startIdx + searchMaxVisibleResults
		if endIdx > len(s.filterGroup.Options) {
			endIdx = len(s.filterGroup.Options)
		}

		// Scroll up indicator
		if s.offset > 0 {
			scrollStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
			b.WriteString(scrollStyle.Render("  ▲ more"))
			b.WriteString("\n")
		}

		for i := startIdx; i < endIdx; i++ {
			opt := s.filterGroup.Options[i]
			isCursor := i == s.cursor

			// Cursor indicator
			cursor := " "
			if isCursor {
				cursor = "▸"
			}

			// Selection indicator
			indicator := "○"
			if opt.Selected {
				indicator = "●"
			}

			// Style
			optionStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
			if opt.Selected {
				optionStyle = optionStyle.Foreground(styles.ColorSuccess)
			}
			if isCursor {
				optionStyle = optionStyle.Bold(true).Foreground(styles.ColorPrimary)
			}

			line := cursor + " " + indicator + " " + opt.Label
			b.WriteString(optionStyle.Render(line))
			b.WriteString("\n")
		}

		// Scroll down indicator
		if endIdx < len(s.filterGroup.Options) {
			scrollStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
			b.WriteString(scrollStyle.Render("  ▼ more"))
			b.WriteString("\n")
		}

		// Result count
		b.WriteString("\n")
		countStyle := lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Italic(true)
		b.WriteString(countStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left,
			"Showing ", components.Itoa(len(s.filterGroup.Options)), " result(s)")))
	}

	// Help text (derived from KeyMap)
	b.WriteString("\n\n")
	up := s.keys.Up.Help()
	down := s.keys.Down.Help()
	sel := s.keys.Select.Help()
	back := s.keys.Back.Help()
	instr := fmt.Sprintf("%s/%s: navigate  %s: %s  %s: %s", up.Key, down.Key, sel.Key, sel.Desc, back.Key, back.Desc)
	b.WriteString(s.styles.ModalInstructions.Render(instr))

	content := b.String()

	// Modal styling
	modalWidth := styles.ModalWidthMD
	modalHeight := styles.ModalHeightXL

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(styles.ModalPaddingV, styles.ModalPaddingH).
		Width(modalWidth).
		Height(modalHeight).
		Render(content)

	// Center the modal
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
	)
}

// SetVisible sets the visibility of the modal
func (s *SearchModal) SetVisible(visible bool) {
	s.visible = visible
	if visible {
		s.searchInput.Focus()
		s.cursor = 0
		s.offset = 0
	} else {
		s.searchInput.Blur()
		s.searchInput.SetValue("")
	}
}

// IsVisible returns whether the modal is visible
func (s SearchModal) IsVisible() bool {
	return s.visible
}

// SetSize sets the size of the modal
func (s *SearchModal) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetFilterGroup sets the filter group to search in
func (s *SearchModal) SetFilterGroup(group *models.FilterGroup) {
	s.filterGroup = group
	s.cursor = 0
	s.offset = 0
	if s.visible && group != nil {
		// Apply current search filter
		group.ApplySearchFilter(s.searchInput.Value())
	}
}

// Message types

// SearchSelectedMsg is sent when a search result is selected
type SearchSelectedMsg struct {
	SelectedOption *models.FilterOption
}

// SearchCancelledMsg is sent when search is cancelled
type SearchCancelledMsg struct{}

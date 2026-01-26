package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary      = lipgloss.Color("#7C3AED")
	ColorSecondary    = lipgloss.Color("#06B6D4")
	ColorSuccess      = lipgloss.Color("#10B981")
	ColorWarning      = lipgloss.Color("#F59E0B")
	ColorError        = lipgloss.Color("#EF4444")
	ColorMuted        = lipgloss.Color("#6B7280")
	ColorBorder       = lipgloss.Color("#374151")
	ColorHighlight    = lipgloss.Color("#1F2937")
	ColorText         = lipgloss.Color("#F9FAFB")
	ColorTextMuted    = lipgloss.Color("#9CA3AF")
	ColorAccent       = lipgloss.Color("#60A5FA")
	ColorTesting      = lipgloss.Color("#FBBF24")
	ColorLight        = lipgloss.Color("#D1D5DB")
	ColorOffWhite     = lipgloss.Color("#E5E7EB")
	ColorPrimaryLight = lipgloss.Color("#A78BFA")
	ColorBlue         = lipgloss.Color("#3B82F6")
	ColorViolet       = lipgloss.Color("#8B5CF6")
	ColorPink         = lipgloss.Color("#EC4899")
	ColorOrange       = lipgloss.Color("#F97316")
	ColorDarkBlue     = lipgloss.Color("#1E3A5F")
)

var typeColors = map[string]lipgloss.Color{
	"User Story":           ColorBlue,
	"Product Backlog Item": ColorBlue,
	"Story":                ColorBlue,
	"PBI":                  ColorBlue,
	"Task":                 ColorWarning,
	"Bug":                  ColorError,
	"Feature":              ColorViolet,
	"Epic":                 ColorPink,
}

var stateColors = map[string]lipgloss.Color{
	"New":         ColorMuted,
	"Active":      ColorBlue,
	"Resolved":    ColorSuccess,
	"Closed":      ColorMuted,
	"Committed":   ColorBlue,
	"To Do":       ColorOrange,
	"In Progress": ColorViolet,
	"Done":        ColorSuccess,
	"Testing":     ColorTesting,
	"Removed":     ColorError,
	"Approved":    ColorText,
}

type Styles struct {
	App       lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	StatusBar lipgloss.Style

	PanelActive   lipgloss.Style
	PanelInactive lipgloss.Style
	PanelTitle    lipgloss.Style

	FilterGroup      lipgloss.Style
	FilterGroupTitle lipgloss.Style
	FilterOption     lipgloss.Style
	FilterSelected   lipgloss.Style
	FilterCursor     lipgloss.Style

	ListHeader       lipgloss.Style
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListItemCursor   lipgloss.Style

	DetailTitle        lipgloss.Style
	DetailSection      lipgloss.Style
	DetailSectionTitle lipgloss.Style
	DetailLabel        lipgloss.Style
	DetailValue        lipgloss.Style
	DetailDescription  lipgloss.Style
	DetailTag          lipgloss.Style

	CommentBox         lipgloss.Style
	CommentBoxSelected lipgloss.Style
	CommentHeader      lipgloss.Style
	CommentAuthor      lipgloss.Style
	CommentDate        lipgloss.Style
	CommentText        lipgloss.Style
	CommentEdited      lipgloss.Style

	HelpKey   lipgloss.Style
	HelpDesc  lipgloss.Style
	HelpPanel lipgloss.Style

	ModalInstructions lipgloss.Style
	ModalTitle        lipgloss.Style
	ModalBox          lipgloss.Style

	TextMuted  lipgloss.Style
	ErrorText  lipgloss.Style
	AccentBold lipgloss.Style
	Selected   lipgloss.Style
	Accent     lipgloss.Style

	TypeBadge  func(string) lipgloss.Style
	StateBadge func(string) lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle(),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Padding(0, 1),

		Subtitle: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		StatusBar: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Padding(0, 1),

		PanelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1),

		PanelInactive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1),

		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorHighlight).
			Padding(0, 1),

		FilterGroup: lipgloss.NewStyle().
			MarginBottom(1),

		FilterGroupTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			MarginBottom(0),

		FilterOption: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			PaddingLeft(1),

		FilterSelected: lipgloss.NewStyle().
			Foreground(ColorSuccess).
			PaddingLeft(1),

		FilterCursor: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		ListHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextMuted).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder),

		ListItem: lipgloss.NewStyle().
			Foreground(ColorText),

		ListItemSelected: lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorHighlight),

		ListItemCursor: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			MarginBottom(1),

		DetailSection: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			MarginBottom(1),

		DetailSectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextMuted).
			MarginBottom(0),

		DetailLabel: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Width(12),

		DetailValue: lipgloss.NewStyle().
			Foreground(ColorText),

		DetailDescription: lipgloss.NewStyle().
			Foreground(ColorText),

		DetailTag: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Background(ColorDarkBlue).
			Padding(0, 1).
			MarginRight(1),

		CommentBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			MarginBottom(1),

		CommentBoxSelected: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1).
			MarginBottom(1),

		CommentHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			MarginBottom(0),

		CommentAuthor: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimaryLight),

		CommentDate: lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Italic(true),

		CommentText: lipgloss.NewStyle().
			Foreground(ColorOffWhite).
			MarginTop(0).
			MarginBottom(0),

		CommentEdited: lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			MarginTop(0),

		HelpKey: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		HelpPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2),

		ModalInstructions: lipgloss.NewStyle().
			Foreground(ColorMuted),

		ModalTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTesting).
			Padding(0, 1),

		ModalBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2),

		TextMuted: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		ErrorText: lipgloss.NewStyle().
			Foreground(ColorError),

		AccentBold: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary),

		Accent: lipgloss.NewStyle().
			Foreground(ColorAccent),

		Selected: lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorText).
			Bold(true),

		TypeBadge: func(t string) lipgloss.Style {
			color := typeColors[t]
			if color == "" {
				color = ColorMuted
			}
			return lipgloss.NewStyle().
				Foreground(color).
				Bold(true)
		},

		StateBadge: func(s string) lipgloss.Style {
			color := stateColors[s]
			if color == "" {
				color = ColorMuted
			}
			return lipgloss.NewStyle().
				Foreground(color)
		},
	}
}

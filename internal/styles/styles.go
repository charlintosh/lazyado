package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	ColorPrimary      = lipgloss.Color("#479EF5")
	ColorSecondary    = lipgloss.Color("#2AB0E0")
	ColorSuccess      = lipgloss.Color("#6CCB5F")
	ColorWarning      = lipgloss.Color("#F2CB1D")
	ColorError        = lipgloss.Color("#E0475B")
	ColorMuted        = lipgloss.Color("#8A8886")
	ColorBorder       = lipgloss.Color("#605E5C")
	ColorHighlight    = lipgloss.Color("#2B3A4A")
	ColorHighlightDim = lipgloss.Color("#262F38")
	ColorErrorTint    = lipgloss.Color("#2A1D20")
	ColorErrorCurrent = lipgloss.Color("#3A1A20")
	ColorWarningTint  = lipgloss.Color("#2A2718")
	ColorText         = lipgloss.Color("#F3F2F1")
	ColorTextMuted    = lipgloss.Color("#A19F9D")
	ColorAccent       = lipgloss.Color("#6CB8F6")
	ColorTesting      = lipgloss.Color("#F2CB1D")
	ColorLight        = lipgloss.Color("#C8C6C4")
	ColorOffWhite     = lipgloss.Color("#E1DFDD")
	ColorPrimaryLight = lipgloss.Color("#A9D3FA")
	ColorBlue         = lipgloss.Color("#479EF5")
	ColorViolet       = lipgloss.Color("#A66BD3")
	ColorPink         = lipgloss.Color("#E3008C")
	ColorOrange       = lipgloss.Color("#FF8C1A")
	ColorDarkBlue     = lipgloss.Color("#1F3A57")
	ColorSurface      = lipgloss.Color("#252423")
	ColorSurfaceAlt   = lipgloss.Color("#201F1E")
	ColorChip         = lipgloss.Color("#323130")
	ColorOnAccent     = lipgloss.Color("#1B1A19")
	ColorDropTint     = lipgloss.Color("#1A2633")
	ColorPhantom      = lipgloss.Color("#1E3247")
	ColorDiffAdd      = lipgloss.Color("#16301F")
	ColorDiffDelete   = lipgloss.Color("#3A1A20")
	ColorDiffHunk     = lipgloss.Color("#1F3A57")
	ColorMatch        = lipgloss.Color("#5C4A12")
	ColorVioletTint   = lipgloss.Color("#2E2440")
	ColorVioletLight  = lipgloss.Color("#C9A5E6")
	ColorTealTint     = lipgloss.Color("#173238")
	ColorTealLight    = lipgloss.Color("#8FD6EA")
	ColorSecretTint   = lipgloss.Color("#3A1A2E")
	ColorSecretLight  = lipgloss.Color("#F0A6D0")
	ColorStaleText    = lipgloss.Color("#F3B3BC")
	ColorChipSoft     = lipgloss.Color("#262524")
	ColorDeployedNow  = lipgloss.Color("#8FE388")
	ColorWaitTint     = lipgloss.Color("#2E2214")
	ColorWaitLight    = lipgloss.Color("#FFC48A")
	ColorTypePBI      = lipgloss.Color("#2AB0E0")
	ColorTypeTask     = lipgloss.Color("#F2CB1D")
	ColorTypeBug      = lipgloss.Color("#E0475B")
	ColorTypeFeature  = lipgloss.Color("#A66BD3")
	ColorTypeEpic     = lipgloss.Color("#FF8C1A")
)

var typeColors = map[string]color.Color{
	"User Story":           ColorTypePBI,
	"Product Backlog Item": ColorTypePBI,
	"Story":                ColorTypePBI,
	"PBI":                  ColorTypePBI,
	"Task":                 ColorTypeTask,
	"Bug":                  ColorTypeBug,
	"Feature":              ColorTypeFeature,
	"Epic":                 ColorTypeEpic,
	"Issue":                ColorTypeFeature,
	"Impediment":           ColorTypeFeature,
}

var stateColors = map[string]color.Color{
	"New":         ColorTextMuted,
	"Proposed":    ColorTextMuted,
	"To Do":       ColorTextMuted,
	"Approved":    ColorLight,
	"Active":      ColorBlue,
	"Committed":   ColorBlue,
	"In Progress": ColorBlue,
	"Testing":     ColorTesting,
	"Resolved":    ColorSuccess,
	"Done":        ColorSuccess,
	"Closed":      ColorSuccess,
	"Removed":     ColorMuted,
}

var typeLabels = map[string]string{
	"Product Backlog Item": "PRODUCT BACKLOG ITEM",
	"User Story":           "USER STORY",
}

func TypeColor(workItemType string) color.Color {
	if c, ok := typeColors[workItemType]; ok {
		return c
	}
	return ColorTextMuted
}

func StateColor(state string) color.Color {
	if c, ok := stateColors[state]; ok {
		return c
	}
	return ColorTextMuted
}

var avatarColors = []color.Color{ColorOrange, ColorSecondary, ColorSuccess, ColorViolet, ColorWarning, ColorPrimaryLight}

func AvatarColor(name string) color.Color {
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	return avatarColors[sum%len(avatarColors)]
}

func TypeLabel(workItemType string) string {
	if l, ok := typeLabels[workItemType]; ok {
		return l
	}
	return strings.ToUpper(workItemType)
}

func RegisterStateColor(state, hex string) {
	if state == "" || len(hex) != 6 {
		return
	}
	stateColors[state] = lipgloss.Color("#" + hex)
}

func RegisterTypeColor(workItemType, hex string) {
	if workItemType == "" || len(hex) != 6 {
		return
	}
	typeColors[workItemType] = lipgloss.Color("#" + hex)
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

	TextMuted lipgloss.Style
	Selected  lipgloss.Style

	TypeBadge  func(string) lipgloss.Style
	StateBadge func(string) lipgloss.Style
}

func DefaultStyles() *Styles {
	return &Styles{
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
			Foreground(ColorLight).
			Background(ColorChip).
			Padding(0, 1).
			MarginRight(1),

		TextMuted: lipgloss.NewStyle().
			Foreground(ColorTextMuted),

		Selected: lipgloss.NewStyle().
			Background(ColorHighlight).
			Foreground(ColorText).
			Bold(true),

		TypeBadge: func(t string) lipgloss.Style {
			return lipgloss.NewStyle().
				Foreground(TypeColor(t)).
				Bold(true)
		},

		StateBadge: func(s string) lipgloss.Style {
			return lipgloss.NewStyle().
				Foreground(StateColor(s))
		},
	}
}

package screens

import (
	"fmt"

	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SplashScreen is the loading screen component
type SplashScreen struct {
	styles  styles.Styles
	width   int
	height  int
	spinner spinner.Model
}

// NewSplashScreen creates a new splash screen
func NewSplashScreen(st styles.Styles) SplashScreen {
	sp := spinner.New()
	sp.Style = st.Accent
	sp.Spinner = spinner.Dot

	return SplashScreen{
		styles:  st,
		spinner: sp,
	}
}

// SetSize sets the dimensions of the splash screen
func (s *SplashScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Update handles messages for the splash screen
func (s SplashScreen) Update(msg tea.Msg) (SplashScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the splash screen
func (s SplashScreen) View() string {
	// ASCII art logo (provided banner)
	logo := []string{
		"                                                                       ",
		" ▄▄▄▄                                                    ▄▄           ",
		" ▀▀██                                                    ██           ",
		"   ██       ▄█████▄  ████████  ▀██  ███   ▄█████▄   ▄███▄██   ▄████▄  ",
		"   ██       ▀ ▄▄▄██      ▄█▀    ██▄ ██    ▀ ▄▄▄██  ██▀  ▀██  ██▀  ▀██ ",
		"   ██      ▄██▀▀▀██    ▄█▀       ████▀   ▄██▀▀▀██  ██    ██  ██    ██ ",
		"   ██▄▄▄   ██▄▄▄███  ▄██▄▄▄▄▄     ███    ██▄▄▄███  ▀██▄▄███  ▀██▄▄██▀ ",
		"    ▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀▀▀▀▀▀▀     ██      ▀▀▀▀ ▀▀    ▀▀▀ ▀▀    ▀▀▀▀   ",
		"                                ███                                   ",
		"                                                                      ",
	}

	// Style the logo
	logoStyle := s.styles.AccentBold

	var styledLogo []string
	for _, line := range logo {
		styledLogo = append(styledLogo, logoStyle.Render(line))
	}

	logoBlock := lipgloss.JoinVertical(lipgloss.Left, styledLogo...)

	// Loading animation (use bubbles spinner)
	spinnerView := s.spinner.View()

	loadingText := fmt.Sprintf("%s Loading Azure DevOps data...", spinnerView)
	loading := s.styles.Accent.MarginTop(2).Render(loadingText)

	// Info text
	info := s.styles.TextMuted.MarginTop(1).Render("• Fetching iterations, areas, and team members")

	// Combine all elements
	content := lipgloss.JoinVertical(lipgloss.Left,
		logoBlock,
		loading,
		info,
	)

	// Center everything
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// (spinner.Tick from bubbles provides periodic ticks)

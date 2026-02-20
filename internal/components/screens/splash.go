package screens

import (
	"math"
	"strings"
	"time"

	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

const (
	animFPS     = 60
	numLetters  = 7
	letterDelay = 10
	numBars     = 12
	maxBarH     = 4
)

var titleChars = []string{"l", "a", "z", "y", "a", "d", "o"}

var barGradient = []lipgloss.Color{
	styles.ColorDarkBlue,
	styles.ColorPrimary,
	styles.ColorViolet,
	styles.ColorAccent,
}

type FrameMsg time.Time

func AnimateCmd() tea.Cmd {
	return tea.Tick(time.Second/animFPS, func(t time.Time) tea.Msg {
		return FrameMsg(t)
	})
}

type SplashScreen struct {
	styles  styles.Styles
	width   int
	height  int
	spinner spinner.Model

	revealSpring harmonica.Spring
	barSpring    harmonica.Spring
	frame        int

	letterVal [numLetters]float64
	letterVel [numLetters]float64

	barH    [numBars]float64
	barHVel [numBars]float64
}

func NewSplashScreen(st styles.Styles) SplashScreen {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	sp.Spinner = spinner.MiniDot

	return SplashScreen{
		styles:       st,
		spinner:      sp,
		revealSpring: harmonica.NewSpring(harmonica.FPS(animFPS), 6.0, 0.5),
		barSpring:    harmonica.NewSpring(harmonica.FPS(animFPS), 5.0, 0.3),
	}
}

func (s *SplashScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
}

func (s SplashScreen) Tick() tea.Cmd {
	return s.spinner.Tick
}

func (s SplashScreen) Update(msg tea.Msg) (SplashScreen, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.(type) {
	case FrameMsg:
		s.frame++

		for i := 0; i < numLetters; i++ {
			if s.frame > i*letterDelay {
				s.letterVal[i], s.letterVel[i] = s.revealSpring.Update(
					s.letterVal[i], s.letterVel[i], 1.0,
				)
			}
		}

		for i := 0; i < numBars; i++ {
			phase := float64(s.frame)/14.0 + float64(i)*0.55
			target := (math.Sin(phase) + 1) / 2 * float64(maxBarH)
			s.barH[i], s.barHVel[i] = s.barSpring.Update(
				s.barH[i], s.barHVel[i], target,
			)
		}

		cmds = append(cmds, AnimateCmd())

	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s SplashScreen) View() string {
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorAccent)
	mutedStyle := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	dotStyle := lipgloss.NewStyle().Foreground(styles.ColorBorder)

	var titleParts []string
	for i, ch := range titleChars {
		v := s.letterVal[i]
		switch {
		case v < 0.15:
			titleParts = append(titleParts, dotStyle.Render("·"))
		case v < 0.7:
			titleParts = append(titleParts, mutedStyle.Render(ch))
		default:
			titleParts = append(titleParts, accentStyle.Render(ch))
		}
	}
	title := strings.Join(titleParts, " ")

	loadLine := lipgloss.NewStyle().
		Foreground(styles.ColorTextMuted).
		Render(s.spinner.View())

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		loadLine,
	)

	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

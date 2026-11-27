package screens

import (
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/colorprofile"
)

type SplashMode string

const (
	SplashAnimated SplashMode = "animated"
	SplashStatic   SplashMode = "static"
	SplashOff      SplashMode = "off"
)

func ParseSplashMode(value string) SplashMode {
	switch mode := SplashMode(strings.ToLower(strings.TrimSpace(value))); mode {
	case SplashStatic, SplashOff:
		return mode
	default:
		return SplashAnimated
	}
}

const (
	frameInterval  = 80 * time.Millisecond
	revealPerFrame = 1.5
	letterStagger  = 1.5
	letterFade     = 2
	sideGap        = 3
	hintMargin     = 2
)

const (
	splashWordmark = "lazyado"
	splashTagline  = "Boards · Repos · Pipelines · Releases"
	splashProduct  = "Azure DevOps"
	splashStatus   = "Connecting to Azure DevOps"
	splashSkipHint = "press any key to skip"
)

type splashLayout struct {
	logo int
	side bool
}

var splashLayouts = []struct {
	minWidth, minHeight int
	layout              splashLayout
}{
	{minWidth: 60, minHeight: 34, layout: splashLayout{logo: 48}},
	{minWidth: 50, minHeight: 22, layout: splashLayout{logo: 32}},
	{minWidth: 56, minHeight: 14, layout: splashLayout{logo: 24, side: true}},
}

func layoutFor(width, height int, mode SplashMode) splashLayout {
	if mode == SplashOff {
		return splashLayout{}
	}
	for _, tier := range splashLayouts {
		if width >= tier.minWidth && height >= tier.minHeight {
			return tier.layout
		}
	}
	return splashLayout{}
}

type FrameMsg time.Time

func AnimateCmd() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg {
		return FrameMsg(t)
	})
}

type SplashScreen struct {
	spinner spinner.Model
	mode    SplashMode
	mono    bool
	width   int
	height  int
	frame   int
	layout  splashLayout
	art     *logoArt
	artMono bool
}

func NewSplashScreen() SplashScreen {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)
	sp.Spinner = spinner.MiniDot
	return SplashScreen{spinner: sp, mode: SplashAnimated}
}

func (s SplashScreen) WithMode(mode SplashMode) SplashScreen {
	s.mode = mode
	s.refresh()
	return s
}

func (s *SplashScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.refresh()
}

func (s *SplashScreen) SetProfile(profile colorprofile.Profile) {
	s.mono = profile == colorprofile.ASCII || profile == colorprofile.NoTTY
	s.refresh()
}

func (s *SplashScreen) refresh() {
	s.layout = layoutFor(s.width, s.height, s.mode)
	switch {
	case s.layout.logo == 0:
		s.art = nil
	case s.art == nil || s.art.size != s.layout.logo || s.artMono != s.mono:
		s.art = buildLogo(s.layout.logo, s.mono)
		s.artMono = s.mono
	}
}

func (s SplashScreen) animated() bool {
	return s.mode == SplashAnimated
}

func (s SplashScreen) Init() tea.Cmd {
	if !s.animated() {
		return s.spinner.Tick
	}
	return tea.Batch(s.spinner.Tick, AnimateCmd())
}

func (s SplashScreen) Update(msg tea.Msg) (SplashScreen, tea.Cmd) {
	switch msg.(type) {
	case FrameMsg:
		if !s.animated() {
			return s, nil
		}
		s.frame++
		return s, AnimateCmd()
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s SplashScreen) View() string {
	body := s.body()
	hint := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(splashSkipHint)
	if lipgloss.Height(body) >= s.height || lipgloss.Width(hint)+hintMargin > s.width {
		return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, body)
	}
	top := lipgloss.Place(s.width, s.height-1, lipgloss.Center, lipgloss.Center, body)
	footer := lipgloss.PlaceHorizontal(s.width, lipgloss.Right, hint+strings.Repeat(" ", hintMargin))
	return top + "\n" + footer
}

func (s SplashScreen) body() string {
	wordmark, status := s.wordmark(), s.status()
	switch {
	case s.art == nil:
		return lipgloss.JoinVertical(lipgloss.Center, wordmark, "", status)
	case s.layout.side:
		product := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(splashProduct)
		text := lipgloss.JoinVertical(lipgloss.Left, wordmark, "", product, "", status)
		return lipgloss.JoinHorizontal(lipgloss.Center, s.logo(), strings.Repeat(" ", sideGap), text)
	default:
		tagline := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(splashTagline)
		return lipgloss.JoinVertical(lipgloss.Center, s.logo(), "", wordmark, tagline, "", status)
	}
}

func (s SplashScreen) logo() string {
	if !s.animated() {
		return s.art.view(math.Inf(1), -1)
	}
	phase := -1
	if !s.mono {
		phase = s.frame
	}
	return s.art.view(float64(s.frame)*revealPerFrame, phase)
}

func (s SplashScreen) wordmark() string {
	accent := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorAccent)
	muted := lipgloss.NewStyle().Foreground(styles.ColorTextMuted)
	pending := lipgloss.NewStyle().Foreground(styles.ColorBorder)
	start := s.wordmarkStart()
	letters := make([]string, 0, len(splashWordmark))
	for i, ch := range splashWordmark {
		at := start + float64(i)*letterStagger
		switch frame := float64(s.frame); {
		case !s.animated() || frame >= at+letterFade:
			letters = append(letters, accent.Render(string(ch)))
		case frame >= at:
			letters = append(letters, muted.Render(string(ch)))
		default:
			letters = append(letters, pending.Render("·"))
		}
	}
	return strings.Join(letters, " ")
}

func (s SplashScreen) wordmarkStart() float64 {
	if s.art == nil {
		return 0
	}
	return math.Ceil(float64(s.art.height()) / revealPerFrame)
}

func (s SplashScreen) status() string {
	text := lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render(splashStatus)
	return s.spinner.View() + " " + text
}

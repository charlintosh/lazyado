package panels

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

func lipglossBlock(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(s)
}

func lipglossPane(title, body string, width, height int, focused bool) string {
	if width < styles.BoxPaddedSize || height < styles.PaneChromeHeight {
		return lipglossBlock("", width, height)
	}
	innerWidth, innerHeight := PaneContentSize(width, height)
	header := lipgloss.NewStyle().
		Width(innerWidth).MaxWidth(innerWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(styles.ColorBorder).
		Render(Line(title, innerWidth))
	content := lipglossBlock(body, innerWidth, innerHeight)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(paneBorderColor(focused)).
		Padding(0, 1).
		Width(width).Height(height).
		MaxWidth(width).MaxHeight(height).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, content))
}

var blockInputs = map[string]string{
	"empty":          "",
	"styled short":   lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render("Taskboard") + " " + Muted("3 items"),
	"exact width":    strings.Repeat("─", 20),
	"many lines":     strings.Repeat("row\n", 30),
	"wide glyphs":    "┃ ■ ● ✕ ↻ ▾ ⤢ ⇄",
	"needs wrapping": "this line is much longer than the block is wide and must wrap like lipgloss does",
	"tabs":           "a\tb",
}

func TestBlockMatchesLipgloss(t *testing.T) {
	for name, in := range blockInputs {
		for _, size := range [][2]int{{20, 5}, {12, 3}, {40, 1}, {1, 1}, {0, 3}} {
			if got, want := Block(in, size[0], size[1]), lipglossBlock(in, size[0], size[1]); got != want {
				t.Errorf("%s at %v:\n got %q\nwant %q", name, size, got, want)
			}
		}
	}
}

func TestPaneMatchesLipgloss(t *testing.T) {
	for name, in := range blockInputs {
		for _, size := range [][2]int{{30, 8}, {12, 5}, {4, 4}, {3, 9}, {40, 3}} {
			for _, focused := range []bool{true, false} {
				got := Pane(PaneTitle("5", "Board", focused), in, size[0], size[1], focused)
				want := lipglossPane(PaneTitle("5", "Board", focused), in, size[0], size[1], focused)
				if size[1] == styles.PaneChromeHeight {
					if lines := strings.Split(ansi.Strip(got), "\n"); !strings.HasPrefix(lines[len(lines)-1], "╰") {
						t.Errorf("%s at %v: a pane with no room for content must still close its border:\n%s", name, size, ansi.Strip(got))
					}
				} else if ansi.Strip(got) != ansi.Strip(want) {
					t.Errorf("%s at %v focused=%v:\n got\n%s\nwant\n%s", name, size, focused, ansi.Strip(got), ansi.Strip(want))
				}
				if lipgloss.Width(got) != lipgloss.Width(want) || lipgloss.Height(got) != lipgloss.Height(want) {
					t.Errorf("%s at %v: size %dx%d, want %dx%d", name, size, lipgloss.Width(got), lipgloss.Height(got), lipgloss.Width(want), lipgloss.Height(want))
				}
			}
		}
	}
}

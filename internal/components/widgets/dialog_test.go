package widgets

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
	"github.com/charmbracelet/x/ansi"
)

func spliceFixture() (string, string) {
	tint := lipgloss.NewStyle().Background(styles.ColorDropTint).Foreground(styles.ColorPrimary)
	row := tint.Render(strings.Repeat("ab┃cd", 8))
	base := strings.TrimSuffix(strings.Repeat(row+"\n", 6), "\n")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(12).Render("chip\n→ Done")
	return base, box
}

func TestSpliceDrawsLikeTheCompositor(t *testing.T) {
	base, box := spliceFixture()
	for _, pos := range [][2]int{{0, 0}, {7, 1}, {35, 4}, {-3, -2}} {
		got := Splice(base, box, pos[0], pos[1], 40, 6)
		want := OverlayAt(base, box, pos[0], pos[1], 40, 6)
		if ansi.Strip(got) != ansi.Strip(want) {
			t.Errorf("at %v:\n got\n%s\nwant\n%s", pos, ansi.Strip(got), ansi.Strip(want))
		}
	}
}

func TestSpliceKeepsTheColoursAroundTheBox(t *testing.T) {
	base, box := spliceFixture()
	out := strings.Split(Splice(base, box, 10, 1, 40, 6), "\n")[2]
	right := out[strings.LastIndex(out, "│")+len("│"):]
	if !strings.Contains(right, "48;2;26;38;51") {
		t.Fatalf("cells right of the box lost their background: %q", right)
	}
	if !strings.HasPrefix(out[strings.Index(out, ansi.ResetStyle):], ansi.ResetStyle) {
		t.Fatal("the box must start from a clean style so the row's colours cannot bleed into it")
	}
}

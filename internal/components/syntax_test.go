package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/styles"
	xansi "github.com/charmbracelet/x/ansi"
)

func TestHighlightCodeKeepsOneLinePerSourceLineAndColorsTokens(t *testing.T) {
	src := "package cart\n\n/* a\n   block */\nfunc Total() int {\n\treturn 42\n}\n"
	lines := HighlightCode("src/cart.go", src)
	if len(lines) != 7 {
		t.Fatalf("got %d lines, want 7 (multi-line comments must stay split per line)", len(lines))
	}
	if lines[5].Plain() != strings.Repeat(" ", styles.DiffTabWidth)+"return 42" {
		t.Fatalf("tabs must expand, got %q", lines[5].Plain())
	}
	colored := 0
	for _, tok := range lines[4] {
		if tok.Color != nil {
			colored++
		}
	}
	if colored < 2 {
		t.Fatalf("Go keywords and names must get colors from the shared theme, got %+v", lines[4])
	}
}

func TestHighlightCodeFallsBackToPlainTextForUnknownFiles(t *testing.T) {
	lines := HighlightCode("LICENSE", "Permission is hereby granted\nfree of charge")
	if len(lines) != 2 || lines[0].Plain() != "Permission is hereby granted" {
		t.Fatalf("got %+v", lines)
	}
}

func TestCodeLineRenderFillsAndTruncatesToWidthOnTheGivenBackground(t *testing.T) {
	line := HighlightCode("a.go", "func longFunctionName() {}")[0]
	base := lipgloss.NewStyle().Background(styles.ColorDiffAdd)
	out := line.Render(base, 12)
	if w := lipgloss.Width(out); w != 12 {
		t.Fatalf("got width %d", w)
	}
	if !strings.HasSuffix(xansi.Strip(out), truncationMark) {
		t.Fatalf("a long line must end with an ellipsis, got %q", xansi.Strip(out))
	}
	if w := lipgloss.Width(line.Render(base, 60)); w != 60 {
		t.Fatalf("a short line must be padded to the width, got %d", w)
	}
}

func TestSanitizeCodeDropsControlCharacters(t *testing.T) {
	if got := SanitizeCode("a\x1b[31mb\x07"); got != "a[31mb" {
		t.Fatalf("got %q", got)
	}
}

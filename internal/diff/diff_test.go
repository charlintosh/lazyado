package diff

import (
	"fmt"
	"strings"
	"testing"
)

func render(lines []Line) string {
	var b strings.Builder
	for _, l := range lines {
		switch l.Kind {
		case Hunk:
			b.WriteString(l.Text)
		case Add:
			fmt.Fprintf(&b, "+%d %s", l.New, l.Text)
		case Delete:
			fmt.Fprintf(&b, "-%d %s", l.Old, l.Text)
		default:
			fmt.Fprintf(&b, " %d/%d %s", l.Old, l.New, l.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func apply(t *testing.T, oldText string, lines []Line) {
	t.Helper()
	old := splitLines(oldText)
	for _, l := range lines {
		if (l.Kind == Context || l.Kind == Delete) && old[l.Old-1] != l.Text {
			t.Errorf("old line %d is %q, diff says %q", l.Old, old[l.Old-1], l.Text)
		}
	}
}

func TestUnifiedNumbersBothSidesAndKeepsContext(t *testing.T) {
	oldText := "a\nb\nc\nd\ne\nf\ng\nh\n"
	newText := "a\nb\nC\nd\ne\nf\ng\nh\ni\n"
	got := render(Unified(oldText, newText, 1, Pins{}))
	want := "@@ -2,3 +2,3 @@\n 2/2 b\n-3 c\n+3 C\n 4/4 d\n@@ -8,1 +8,2 @@\n 8/8 h\n+9 i\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	apply(t, oldText, Unified(oldText, newText, 1, Pins{}))
}

func TestUnifiedHandlesAddedDeletedAndUnchangedFiles(t *testing.T) {
	if got := render(Unified("", "x\ny\n", 3, Pins{})); got != "@@ -0,0 +1,2 @@\n+1 x\n+2 y\n" {
		t.Errorf("added file: %q", got)
	}
	if got := render(Unified("x\r\ny\r\n", "", 3, Pins{})); got != "@@ -1,2 +0,0 @@\n-1 x\n-2 y\n" {
		t.Errorf("deleted file: %q", got)
	}
	if got := Unified("same\n", "same\n", 3, Pins{}); got != nil {
		t.Errorf("unchanged file produced %v", got)
	}
}

func TestUnifiedReconstructsTheNewFile(t *testing.T) {
	oldLines, newLines := []string{}, []string{}
	for i := 0; i < 400; i++ {
		oldLines = append(oldLines, fmt.Sprintf("line %d", i))
		switch {
		case i%37 == 0:
		case i%29 == 0:
			newLines = append(newLines, fmt.Sprintf("changed %d", i))
		case i%53 == 0:
			newLines = append(newLines, fmt.Sprintf("line %d", i), "inserted")
		default:
			newLines = append(newLines, fmt.Sprintf("line %d", i))
		}
	}
	lines := Unified(strings.Join(oldLines, "\n"), strings.Join(newLines, "\n"), len(oldLines)+len(newLines), Pins{})
	var rebuilt []string
	for _, l := range lines {
		if l.Kind == Context || l.Kind == Add {
			rebuilt = append(rebuilt, l.Text)
			if rebuilt[l.New-1] != l.Text {
				t.Fatalf("new line number %d is wrong", l.New)
			}
		}
	}
	if strings.Join(rebuilt, "\n") != strings.Join(newLines, "\n") {
		t.Fatal("applying the diff does not reproduce the new file")
	}
	added, deleted := Stats(lines)
	if added == 0 || deleted == 0 || added+deleted > 80 {
		t.Fatalf("diff is not minimal enough: +%d -%d", added, deleted)
	}
}

func TestUnifiedKeepsPinnedLinesOutsideTheChanges(t *testing.T) {
	var oldLines []string
	for i := 1; i <= 30; i++ {
		oldLines = append(oldLines, fmt.Sprintf("line %d", i))
	}
	newLines := append([]string{}, oldLines...)
	newLines[1] = "changed"
	oldText, newText := strings.Join(oldLines, "\n"), strings.Join(newLines, "\n")
	got := render(Unified(oldText, newText, 1, Pins{New: []int{20}}))
	want := "@@ -1,3 +1,3 @@\n 1/1 line 1\n-2 line 2\n+2 changed\n 3/3 line 3\n@@ -19,3 +19,3 @@\n 19/19 line 19\n 20/20 line 20\n 21/21 line 21\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := render(Unified(oldText, oldText, 0, Pins{Old: []int{7}})); got != "@@ -7,1 +7,1 @@\n 7/7 line 7\n" {
		t.Fatalf("an unchanged file with a pinned line: %q", got)
	}
}

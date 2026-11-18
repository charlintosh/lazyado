package widgets

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const blanks = "                                                                                                                                "

func Block(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if fitted, ok := fitBlock(s, width, height); ok {
		return fitted
	}
	return lipgloss.NewStyle().Width(width).Height(height).MaxWidth(width).MaxHeight(height).Render(s)
}

func fitBlock(s string, width, height int) (string, bool) {
	if strings.ContainsAny(s, "\t\r") {
		return "", false
	}
	var b strings.Builder
	b.Grow(len(s) + width*height)
	n := 0
	for line := range strings.SplitSeq(s, "\n") {
		if n == height {
			break
		}
		w := ansi.StringWidth(line)
		if w > width {
			return "", false
		}
		if n > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
		WriteBlanks(&b, width-w)
		n++
	}
	for ; n < height; n++ {
		if n > 0 {
			b.WriteByte('\n')
		}
		WriteBlanks(&b, width)
	}
	return b.String(), true
}

func WriteBlanks(b *strings.Builder, n int) {
	for n > len(blanks) {
		b.WriteString(blanks)
		n -= len(blanks)
	}
	b.WriteString(blanks[:max(n, 0)])
}

func JoinColumns(columns ...string) string {
	split := make([][]string, len(columns))
	rows := 0
	for i, c := range columns {
		split[i] = strings.Split(c, "\n")
		rows = max(rows, len(split[i]))
	}
	var b strings.Builder
	for r := range rows {
		if r > 0 {
			b.WriteByte('\n')
		}
		for _, lines := range split {
			if r < len(lines) {
				b.WriteString(lines[r])
			}
		}
	}
	return b.String()
}

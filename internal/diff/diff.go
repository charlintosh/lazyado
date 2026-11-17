package diff

import (
	"fmt"
	"slices"
	"strings"
)

type Kind int

const (
	Context Kind = iota
	Add
	Delete
	Hunk
)

type Line struct {
	Kind Kind
	Old  int
	New  int
	Text string
}

const maxEditDistance = 4000

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

type Pins struct {
	Old []int
	New []int
}

func (p Pins) holds(l Line) bool {
	return (l.Kind != Add && slices.Contains(p.Old, l.Old)) || (l.Kind != Delete && slices.Contains(p.New, l.New))
}

func Unified(oldText, newText string, context int, pins Pins) []Line {
	all := number(edits(splitLines(oldText), splitLines(newText)))
	return hunks(all, context, pins)
}

func Stats(lines []Line) (int, int) {
	added, deleted := 0, 0
	for _, l := range lines {
		switch l.Kind {
		case Add:
			added++
		case Delete:
			deleted++
		}
	}
	return added, deleted
}

func edits(a, b []string) []Line {
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	out := make([]Line, 0, len(a)+len(b))
	for _, s := range a[:prefix] {
		out = append(out, Line{Kind: Context, Text: s})
	}
	out = append(out, myers(a[prefix:len(a)-suffix], b[prefix:len(b)-suffix])...)
	for _, s := range a[len(a)-suffix:] {
		out = append(out, Line{Kind: Context, Text: s})
	}
	return out
}

func replaceAll(a, b []string) []Line {
	out := make([]Line, 0, len(a)+len(b))
	for _, s := range a {
		out = append(out, Line{Kind: Delete, Text: s})
	}
	for _, s := range b {
		out = append(out, Line{Kind: Add, Text: s})
	}
	return out
}

func myers(a, b []string) []Line {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return replaceAll(a, b)
	}
	limit := min(n+m, maxEditDistance)
	offset := limit + 1
	v := make([]int, 2*limit+3)
	var trace [][]int
	for d := 0; d <= limit; d++ {
		trace = append(trace, append([]int(nil), v[offset-d-1:offset+d+2]...))
		for k := -d; k <= d; k += 2 {
			x := v[offset+k-1] + 1
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return backtrack(a, b, trace)
			}
		}
	}
	return replaceAll(a, b)
}

func backtrack(a, b []string, trace [][]int) []Line {
	x, y := len(a), len(b)
	reversed := make([]Line, 0, x+y)
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d+1] }
		k := x - y
		prevK := k - 1
		if k == -d || (k != d && at(k-1) < at(k+1)) {
			prevK = k + 1
		}
		prevX := at(prevK)
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
			reversed = append(reversed, Line{Kind: Context, Text: a[x]})
		}
		if d == 0 {
			break
		}
		if x == prevX {
			y--
			reversed = append(reversed, Line{Kind: Add, Text: b[y]})
		} else {
			x--
			reversed = append(reversed, Line{Kind: Delete, Text: a[x]})
		}
	}
	out := make([]Line, len(reversed))
	for i, l := range reversed {
		out[len(reversed)-1-i] = l
	}
	return out
}

func number(lines []Line) []Line {
	oldNo, newNo := 0, 0
	for i := range lines {
		switch lines[i].Kind {
		case Context:
			oldNo++
			newNo++
			lines[i].Old, lines[i].New = oldNo, newNo
		case Delete:
			oldNo++
			lines[i].Old = oldNo
		case Add:
			newNo++
			lines[i].New = newNo
		}
	}
	return lines
}

func hunks(lines []Line, context int, pins Pins) []Line {
	keep := make([]bool, len(lines))
	changed := false
	for i, l := range lines {
		if l.Kind == Context && !pins.holds(l) {
			continue
		}
		changed = true
		for j := max(i-context, 0); j <= min(i+context, len(lines)-1); j++ {
			keep[j] = true
		}
	}
	if !changed {
		return nil
	}
	var out []Line
	for i := 0; i < len(lines); {
		if !keep[i] {
			i++
			continue
		}
		end := i
		for end < len(lines) && keep[end] {
			end++
		}
		out = append(out, header(lines[i:end]))
		out = append(out, lines[i:end]...)
		i = end
	}
	return out
}

func header(group []Line) Line {
	oldStart, newStart, oldCount, newCount := 0, 0, 0, 0
	for _, l := range group {
		if l.Old > 0 {
			if oldStart == 0 {
				oldStart = l.Old
			}
			oldCount++
		}
		if l.New > 0 {
			if newStart == 0 {
				newStart = l.New
			}
			newCount++
		}
	}
	return Line{Kind: Hunk, Text: fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)}
}

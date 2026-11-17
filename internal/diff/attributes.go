package diff

import (
	"regexp"
	"strings"
)

type Reason int

const (
	Shown Reason = iota
	Generated
	NoDiff
	Configured
)

type rule struct {
	match  *regexp.Regexp
	reason Reason
}

type Attributes struct {
	rules []rule
}

func ParseAttributes(text string) Attributes {
	var a Attributes
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "#") || strings.HasPrefix(fields[0], "!") {
			continue
		}
		reason, ok := attributeReason(fields[1:])
		if !ok {
			continue
		}
		if re, err := compilePattern(fields[0]); err == nil {
			a.rules = append(a.rules, rule{match: re, reason: reason})
		}
	}
	return a
}

func (a Attributes) WithPatterns(patterns []string) Attributes {
	rules := append([]rule{}, a.rules...)
	for _, p := range patterns {
		if re, err := compilePattern(strings.TrimSpace(p)); err == nil && strings.TrimSpace(p) != "" {
			rules = append(rules, rule{match: re, reason: Configured})
		}
	}
	return Attributes{rules: rules}
}

func (a Attributes) Reason(path string) Reason {
	path = strings.TrimPrefix(path, "/")
	reason := Shown
	for _, r := range a.rules {
		if r.match.MatchString(path) {
			reason = r.reason
		}
	}
	return reason
}

func attributeReason(attrs []string) (Reason, bool) {
	reason, found := Shown, false
	for _, attr := range attrs {
		switch attr {
		case "linguist-generated", "linguist-generated=true":
			reason, found = Generated, true
		case "-linguist-generated", "linguist-generated=false", "diff":
			reason, found = Shown, true
		case "-diff", "binary":
			reason, found = NoDiff, true
		}
	}
	return reason, found
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	anchored := strings.Contains(strings.TrimSuffix(pattern, "/"), "/")
	pattern = strings.TrimPrefix(pattern, "/")
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case strings.HasPrefix(pattern[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "/**") && i+3 == len(pattern):
			b.WriteString("/.*")
			i += 2
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i++
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	prefix := "^(?:.*/)?"
	if anchored {
		prefix = "^"
	}
	return regexp.Compile(prefix + b.String() + "$")
}

func LineCount(text string) int {
	return len(splitLines(text))
}

func Estimate(oldText, newText string) int {
	a, b := splitLines(oldText), splitLines(newText)
	start := 0
	for start < len(a) && start < len(b) && a[start] == b[start] {
		start++
	}
	endA, endB := len(a), len(b)
	for endA > start && endB > start && a[endA-1] == b[endB-1] {
		endA--
		endB--
	}
	return (endA - start) + (endB - start)
}

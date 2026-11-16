package widgets

import "strings"

func Initials(name string) string {
	fields := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '.' || r == '@' || r == '-' || r == '_'
	})
	var out []rune
	for _, f := range fields {
		if len(out) < 2 {
			out = append(out, []rune(f)[0])
		}
	}
	if len(out) < 2 {
		out = []rune(strings.TrimSpace(name))
		if len(out) > 2 {
			out = out[:2]
		}
	}
	return strings.ToUpper(string(out))
}

package components

import (
	"hash/fnv"
	"image/color"
	"path"
	"strings"
	"sync"
	"unicode"

	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charlintosh/lazyado/internal/styles"
	xansi "github.com/charmbracelet/x/ansi"
)

const (
	syntaxCacheLimit = 64
	syntaxMaxBytes   = 2 << 20
	truncationMark   = "…"
)

type CodeToken struct {
	Text   string
	Color  color.Color
	Bold   bool
	Italic bool
}

type CodeLine []CodeToken

type syntaxCache struct {
	mu     sync.Mutex
	style  *chroma.Style
	output map[uint64][]CodeLine
}

var syntax = &syntaxCache{style: codeStyle(markdownStyle().CodeBlock.Chroma), output: map[uint64][]CodeLine{}}

func HighlightCode(filePath, text string) []CodeLine {
	return syntax.highlight(filePath, text)
}

func (l CodeLine) Plain() string {
	var b strings.Builder
	for _, t := range l {
		b.WriteString(t.Text)
	}
	return b.String()
}

func (l CodeLine) Render(base lipgloss.Style, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for _, t := range l {
		st := base
		if t.Color != nil {
			st = st.Foreground(t.Color)
		}
		st = st.Bold(t.Bold).Italic(t.Italic)
		w := xansi.StringWidth(t.Text)
		if used+w > width {
			b.WriteString(st.Render(xansi.Truncate(t.Text, width-used, truncationMark)))
			used = width
			break
		}
		b.WriteString(st.Render(t.Text))
		used += w
	}
	if used < width {
		b.WriteString(base.Render(strings.Repeat(" ", width-used)))
	}
	return b.String()
}

func SanitizeCode(text string) string {
	text = strings.ReplaceAll(text, "\t", strings.Repeat(" ", styles.DiffTabWidth))
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}

func (c *syntaxCache) highlight(filePath, text string) []CodeLine {
	h := fnv.New64a()
	_, _ = h.Write([]byte(filePath))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(text))
	key := h.Sum64()
	c.mu.Lock()
	defer c.mu.Unlock()
	if out, ok := c.output[key]; ok {
		return out
	}
	out := c.tokenize(filePath, text)
	if len(c.output) >= syntaxCacheLimit {
		clear(c.output)
	}
	c.output[key] = out
	return out
}

func (c *syntaxCache) tokenize(filePath, text string) []CodeLine {
	text = strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if text == "" {
		return nil
	}
	lexer := lexerFor(filePath, text)
	if lexer == nil || len(text) > syntaxMaxBytes {
		return plainLines(text)
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, text)
	if err != nil {
		return plainLines(text)
	}
	lines := []CodeLine{nil}
	for tok := it(); tok != chroma.EOF; tok = it() {
		entry := c.style.Get(tok.Type)
		parts := strings.Split(tok.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				lines = append(lines, nil)
			}
			if part == "" {
				continue
			}
			last := len(lines) - 1
			lines[last] = append(lines[last], tokenFor(SanitizeCode(part), entry))
		}
	}
	return lines
}

func lexerFor(filePath, text string) chroma.Lexer {
	if l := lexers.Match(path.Base(filePath)); l != nil {
		return l
	}
	if l := lexers.Analyse(text); l != nil {
		return l
	}
	return nil
}

func plainLines(text string) []CodeLine {
	src := strings.Split(text, "\n")
	out := make([]CodeLine, len(src))
	for i, s := range src {
		if s != "" {
			out[i] = CodeLine{{Text: SanitizeCode(s)}}
		}
	}
	return out
}

func tokenFor(text string, e chroma.StyleEntry) CodeToken {
	t := CodeToken{Text: text, Bold: e.Bold == chroma.Yes, Italic: e.Italic == chroma.Yes}
	if e.Colour.IsSet() {
		t.Color = lipgloss.Color(e.Colour.String())
	}
	return t
}

func codeStyle(c *ansi.Chroma) *chroma.Style {
	if c == nil {
		return chroma.MustNewStyle("lazyado", chroma.StyleEntries{})
	}
	return chroma.MustNewStyle("lazyado", chroma.StyleEntries{
		chroma.Text:                chromaEntry(c.Text),
		chroma.Error:               chromaEntry(c.Error),
		chroma.Comment:             chromaEntry(c.Comment),
		chroma.CommentPreproc:      chromaEntry(c.CommentPreproc),
		chroma.Keyword:             chromaEntry(c.Keyword),
		chroma.KeywordReserved:     chromaEntry(c.KeywordReserved),
		chroma.KeywordNamespace:    chromaEntry(c.KeywordNamespace),
		chroma.KeywordType:         chromaEntry(c.KeywordType),
		chroma.Operator:            chromaEntry(c.Operator),
		chroma.Punctuation:         chromaEntry(c.Punctuation),
		chroma.Name:                chromaEntry(c.Name),
		chroma.NameBuiltin:         chromaEntry(c.NameBuiltin),
		chroma.NameTag:             chromaEntry(c.NameTag),
		chroma.NameAttribute:       chromaEntry(c.NameAttribute),
		chroma.NameClass:           chromaEntry(c.NameClass),
		chroma.NameConstant:        chromaEntry(c.NameConstant),
		chroma.NameDecorator:       chromaEntry(c.NameDecorator),
		chroma.NameException:       chromaEntry(c.NameException),
		chroma.NameFunction:        chromaEntry(c.NameFunction),
		chroma.NameOther:           chromaEntry(c.NameOther),
		chroma.Literal:             chromaEntry(c.Literal),
		chroma.LiteralNumber:       chromaEntry(c.LiteralNumber),
		chroma.LiteralDate:         chromaEntry(c.LiteralDate),
		chroma.LiteralString:       chromaEntry(c.LiteralString),
		chroma.LiteralStringEscape: chromaEntry(c.LiteralStringEscape),
		chroma.GenericDeleted:      chromaEntry(c.GenericDeleted),
		chroma.GenericEmph:         chromaEntry(c.GenericEmph),
		chroma.GenericInserted:     chromaEntry(c.GenericInserted),
		chroma.GenericStrong:       chromaEntry(c.GenericStrong),
		chroma.GenericSubheading:   chromaEntry(c.GenericSubheading),
	})
}

func chromaEntry(p ansi.StylePrimitive) string {
	var parts []string
	if p.Color != nil {
		parts = append(parts, *p.Color)
	}
	if p.Italic != nil && *p.Italic {
		parts = append(parts, "italic")
	}
	if p.Bold != nil && *p.Bold {
		parts = append(parts, "bold")
	}
	return strings.Join(parts, " ")
}

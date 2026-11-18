package components

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	glamourstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charlintosh/lazyado/internal/cache"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	xansi "github.com/charmbracelet/x/ansi"
)

const (
	markdownCacheBytes = 16 << 20
	markdownEntryBytes = 64
	renderersKept      = 3
)

type markdownKey struct {
	width  int
	source string
}

type markdownCache struct {
	mu        sync.Mutex
	style     ansi.StyleConfig
	renderers *cache.LRU[int, *glamour.TermRenderer]
	output    *cache.LRU[markdownKey, rendered]
}

type rendered struct {
	source string
	out    string
}

var markdown = &markdownCache{
	style:     markdownStyle(),
	renderers: cache.Count[int, *glamour.TermRenderer](renderersKept),
	output: cache.New[markdownKey, rendered](markdownCacheBytes, func(r rendered) int {
		return len(r.source) + len(r.out) + markdownEntryBytes
	}),
}

func RenderMarkdown(source string, width int) string {
	return markdown.render(source, max(width, 1))
}

func RenderHTML(htmlSource, fallback string, width int) string {
	source := fallback
	if htmlSource != "" {
		source = HTMLToMarkdown(htmlSource)
	}
	return RenderMarkdown(source, width)
}

func CommentMarkdown(c models.Comment) string {
	if strings.EqualFold(c.Format, "markdown") {
		return NormalizeMarkdownImages(c.Text)
	}
	source := c.RenderedText
	if source == "" {
		source = c.Text
	}
	return HTMLToMarkdown(source)
}

func PlainText(markdownSource string) string {
	return strings.Join(strings.Fields(xansi.Strip(RenderMarkdown(markdownSource, 1<<12))), " ")
}

func (c *markdownCache) render(source string, width int) string {
	k := markdownKey{width: width, source: source}
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.output.Get(k); ok {
		return r.out
	}
	out := tidyMarkdown(c.glamour(source, width))
	c.output.Put(k, rendered{source: source, out: out})
	return out
}

func (c *markdownCache) glamour(source string, width int) string {
	r, ok := c.renderers.Get(width)
	if !ok {
		var err error
		r, err = glamour.NewTermRenderer(glamour.WithStyles(c.style), glamour.WithWordWrap(width))
		if err != nil {
			return lipgloss.NewStyle().Width(width).Render(source)
		}
		c.renderers.Put(width, r)
	}
	out, err := r.Render(source)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(source)
	}
	return out
}

func tidyMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	blank := func(l string) bool { return strings.TrimSpace(xansi.Strip(l)) == "" }
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}
	for len(lines) > 0 && blank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for i, l := range lines {
		if blank(l) {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

func markdownStyle() ansi.StyleConfig {
	s := glamourstyles.DarkStyleConfig
	s.Document.BlockPrefix = ""
	s.Document.BlockSuffix = ""
	s.Document.Margin = uintPtr(0)
	s.Document.Color = hex(styles.ColorOffWhite)
	s.Heading.Color = hex(styles.ColorPrimary)
	s.H1 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: hex(styles.ColorPrimary), Bold: boolPtr(true)}}
	s.BlockQuote.Color = hex(styles.ColorTextMuted)
	s.Strong.Color = hex(styles.ColorText)
	s.Link.Color = hex(styles.ColorAccent)
	s.LinkText.Color = hex(styles.ColorAccent)
	s.ImageText = ansi.StylePrimitive{Color: hex(styles.ColorAccent), Underline: boolPtr(true), Format: "[{{.text}}]"}
	s.Image = ansi.StylePrimitive{Format: `{{""}}`}
	s.Code.Color = hex(styles.ColorWarning)
	s.Code.BackgroundColor = hex(styles.ColorChip)
	s.CodeBlock.Margin = uintPtr(1)
	s.HorizontalRule.Color = hex(styles.ColorBorder)
	return s
}

func hex(c color.Color) *string {
	r, g, b, _ := c.RGBA()
	s := fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	return &s
}

func boolPtr(b bool) *bool { return &b }

func uintPtr(u uint) *uint { return &u }

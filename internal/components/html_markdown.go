package components

import (
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

const mentionAttr = "data-vss-mention"

var (
	htmlConverter    = newHTMLConverter()
	markdownImage    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]*)[^)]*\)`)
	markdownSpecials = strings.NewReplacer(`\`, `\\`, "*", `\*`, "_", `\_`, "[", `\[`, "]", `\]`, "`", "\\`")
	urlUnsafe        = strings.NewReplacer(" ", "%20", "<", "%3C", ">", "%3E", "\n", "")
)

func newHTMLConverter() *converter.Converter {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	conv.Register.RendererFor("a", converter.TagTypeInline, renderAnchor, converter.PriorityEarly)
	conv.Register.RendererFor("img", converter.TagTypeInline, renderImage, converter.PriorityEarly)
	return conv
}

func HTMLToMarkdown(source string) string {
	if source == "" {
		return ""
	}
	md, err := htmlConverter.ConvertString(source)
	if err != nil {
		return source
	}
	return md
}

func NormalizeMarkdownImages(md string) string {
	return markdownImage.ReplaceAllStringFunc(md, func(m string) string {
		parts := markdownImage.FindStringSubmatch(m)
		return imageMarkdown(strings.Trim(parts[2], "<>"), parts[1])
	})
}

func renderAnchor(_ converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	if hasAttr(n, mentionAttr) {
		name := strings.TrimSpace(nodeText(n))
		if name == "" {
			return converter.RenderTryNext
		}
		if !strings.HasPrefix(name, "@") {
			name = "@" + name
		}
		_, _ = w.WriteString("**" + markdownSpecials.Replace(name) + "**")
		return converter.RenderSuccess
	}
	if img := onlyImage(n); img != nil {
		return renderImage(nil, w, img)
	}
	return converter.RenderTryNext
}

func renderImage(_ converter.Context, w converter.Writer, n *html.Node) converter.RenderStatus {
	_, _ = w.WriteString(imageMarkdown(attr(n, "src"), attr(n, "alt")))
	return converter.RenderSuccess
}

func imageMarkdown(src, alt string) string {
	label := "image: " + markdownSpecials.Replace(imageName(src, alt))
	src = urlUnsafe.Replace(strings.TrimSpace(src))
	if src == "" {
		return "*[" + label + "]*"
	}
	return "![" + label + "](<" + src + ">)"
}

func imageName(src, alt string) string {
	if u, err := url.Parse(src); err == nil {
		if name := strings.TrimSpace(u.Query().Get("fileName")); name != "" {
			return name
		}
		if base := path.Base(u.Path); base != "." && base != "/" && strings.Contains(base, ".") {
			return base
		}
	}
	if alt = strings.TrimSpace(alt); alt != "" {
		return alt
	}
	return "image"
}

func onlyImage(n *html.Node) *html.Node {
	var img *html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.ElementNode && c.Data == "img" && img == nil:
			img = c
		case c.Type == html.TextNode && strings.TrimSpace(c.Data) == "":
		default:
			return nil
		}
	}
	return img
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if a.Key == name {
			return true
		}
	}
	return false
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(nodeText(c))
	}
	return b.String()
}

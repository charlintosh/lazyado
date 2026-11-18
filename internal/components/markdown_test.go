package components

import (
	"strings"
	"testing"

	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charmbracelet/x/ansi"
)

const attachment = "https://dev.azure.com/acme/0a1b2c3d-1111-2222-3333-444455556666/_apis/wit/attachments/bbe6d401-65d2-49c0-8800-90f2ac8e7215?fileName=image.png"

func TestImagesRenderAsClickableLabels(t *testing.T) {
	cases := map[string]models.Comment{
		"html image":   {Text: `<div><b>Before:</b></div><div><img src="` + attachment + `" alt=Image></div>`},
		"linked image": {Text: `<div><a href="` + attachment + `"><img src="` + attachment + `" alt=Image></a></div>`},
		"markdown":     {Format: "markdown", Text: "Before:\n\n![Image](" + attachment + ")"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := RenderMarkdown(CommentMarkdown(c), 40)
			visible := ansi.Strip(out)
			if !strings.Contains(visible, "[image: image.png]") {
				t.Fatalf("missing image label:\n%s", visible)
			}
			for _, bad := range []string{"→", "attachments", "Image:"} {
				if strings.Contains(visible, bad) {
					t.Fatalf("visible output still contains %q:\n%s", bad, visible)
				}
			}
			if !strings.Contains(out, "\x1b]8;") || !strings.Contains(out, attachment) {
				t.Fatalf("the label must link to the attachment:\n%q", out)
			}
		})
	}
}

func TestImageLabelsKeepFileNamesAndHandleMissingSources(t *testing.T) {
	out := ansi.Strip(RenderMarkdown(HTMLToMarkdown(`<img src="https://example.com/a?fileName=my_file_v2.png">`), 60))
	if !strings.Contains(out, "[image: my_file_v2.png]") {
		t.Fatalf("file name with underscores was mangled:\n%s", out)
	}
	out = RenderMarkdown(HTMLToMarkdown(`<img alt="Diagram">`), 60)
	if !strings.Contains(ansi.Strip(out), "[image: Diagram]") || strings.Contains(out, "\x1b]8;") {
		t.Fatalf("an image without a source should render a plain label:\n%q", out)
	}
}

func TestImageNameFallsBackToAltText(t *testing.T) {
	if got := imageName("https://example.com/_apis/wit/attachments/abc", "Screenshot"); got != "Screenshot" {
		t.Fatalf("got %q, want the alt text", got)
	}
	if got := imageName("", ""); got != "image" {
		t.Fatalf("got %q, want the generic name", got)
	}
	if got := imageName("https://example.com/files/diagram.png", ""); got != "diagram.png" {
		t.Fatalf("got %q, want the file name from the path", got)
	}
}

func TestLinksAndMentionsStillRender(t *testing.T) {
	md := HTMLToMarkdown(`<div><a href="#" data-vss-mention="version:2.0,abc">@Dana_Reyes</a> see <a href="https://example.com/doc">the doc</a></div>`)
	plain := ansi.Strip(RenderMarkdown(md, 80))
	if !strings.Contains(plain, "@Dana_Reyes") || !strings.Contains(plain, "the doc") || !strings.Contains(plain, "https://example.com/doc") {
		t.Fatalf("unexpected render:\n%s", plain)
	}
}

func TestResizingKeepsOnlyAFewRenderers(t *testing.T) {
	for width := 40; width < 80; width++ {
		RenderMarkdown("# Release notes\n\n- faster boards", width)
	}
	if n := markdown.renderers.Len(); n > renderersKept {
		t.Fatalf("%d glamour renderers kept after resizing, want at most %d", n, renderersKept)
	}
}

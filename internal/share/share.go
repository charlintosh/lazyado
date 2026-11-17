package share

import (
	"fmt"
	"strings"
)

type Kind int

const (
	Link Kind = iota
	Permalink
	ID
	Name
	Path
	Snippet
	Markdown
	Title
	Mention
	GitHub
	RunID
	Logs
	PipelineLink
)

type Option struct {
	Kind  Kind
	Label string
	Text  string
}

type Target struct {
	Noun    string
	Subject string
	Options []Option
}

func (t Target) Empty() bool {
	return len(t.Options) == 0
}

func (t Target) Find(kind Kind) (Option, bool) {
	for _, o := range t.Options {
		if o.Kind == kind {
			return o, true
		}
	}
	return Option{}, false
}

func (t Target) Default() (Option, bool) {
	if t.Empty() {
		return Option{}, false
	}
	return t.Options[0], true
}

func (t Target) Notice(o Option) string {
	if t.Noun == "" {
		return o.Label + " copied"
	}
	return fmt.Sprintf("%s copied for %s", o.Label, t.Noun)
}

type builder struct {
	target Target
}

func newTarget(noun, subject string) *builder {
	return &builder{target: Target{Noun: noun, Subject: subject}}
}

func (b *builder) add(kind Kind, label, text string) *builder {
	if strings.TrimSpace(text) != "" {
		b.target.Options = append(b.target.Options, Option{Kind: kind, Label: label, Text: text})
	}
	return b
}

func (b *builder) build() Target {
	return b.target
}

func markdownLink(text, url string) string {
	if url == "" {
		return ""
	}
	return "[" + escapeMarkdown(text) + "](" + url + ")"
}

func escapeMarkdown(s string) string {
	return strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(s)
}

func titled(title, url string) string {
	if url == "" {
		return ""
	}
	return title + " — " + url
}

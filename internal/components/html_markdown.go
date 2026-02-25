package components

import (
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

var htmlConverter *converter.Converter

func init() {
	htmlConverter = converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
}

func HTMLToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	md, err := htmlConverter.ConvertString(html)
	if err != nil {
		return html
	}

	return md
}

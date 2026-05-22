package commentrender

import (
	"bytes"
	"html/template"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	markdownOnce sync.Once
	markdown     goldmark.Markdown
	policy       *bluemonday.Policy
)

func HTML(source string) template.HTML {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	markdownOnce.Do(func() {
		markdown = goldmark.New(
			goldmark.WithExtensions(extension.Strikethrough, extension.Table),
			goldmark.WithRendererOptions(goldhtml.WithHardWraps()),
		)
		policy = bluemonday.UGCPolicy()
		policy.RequireNoFollowOnLinks(true)
		policy.RequireNoReferrerOnLinks(true)
		policy.AddTargetBlankToFullyQualifiedLinks(true)
	})

	var buf bytes.Buffer
	if err := markdown.Convert([]byte(source), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(source))
	}
	return template.HTML(policy.SanitizeBytes(buf.Bytes()))
}

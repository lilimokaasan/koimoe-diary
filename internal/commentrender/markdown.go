package commentrender

import (
	"bytes"
	"html/template"
	"regexp"
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

var commentImagePattern = regexp.MustCompile(`(?is)\[img\](.*?)\[/img\]`)

func HTML(source string) template.HTML {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	source = renderImageBBCode(source)
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

func renderImageBBCode(source string) string {
	return commentImagePattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := commentImagePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		url := strings.TrimSpace(stripTags(parts[1]))
		if !safeImageURL(url) {
			return match
		}
		return "![Comment image](" + url + ")"
	})
}

func stripTags(value string) string {
	var out strings.Builder
	inTag := false
	for _, r := range value {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}

func safeImageURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return false
	}
	if strings.ContainsAny(value, " \t\r\n()") {
		return false
	}
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "/static/uploads/") ||
		strings.HasPrefix(lower, "/static/theme/")
}

package legacyshortcode

import (
	"html"
	"html/template"
	"regexp"
	"strings"
)

var (
	tocPattern      = regexp.MustCompile(`(?i)\[toc\]`)
	beginOpen       = regexp.MustCompile(`(?i)\[begin\]`)
	beginClose      = regexp.MustCompile(`(?i)\[/begin\]`)
	panelPattern    = regexp.MustCompile(`(?is)\[(task|warning|noway|buy)\](.*?)\[/\s*(task|warning|noway|buy)\]`)
	downloadPattern = regexp.MustCompile(`(?is)\[download\](.*?)\[/download\]`)
	collapsePattern = regexp.MustCompile(`(?is)\[collapse([^\]]*)\](.*?)\[/collapse\]`)
	titlePattern    = regexp.MustCompile(`(?i)\btitle\s*=\s*("([^"]*)"|'([^']*)'|([^\s\]]+))`)
)

func Apply(content template.HTML) template.HTML {
	source := string(content)
	if !strings.Contains(source, "[") {
		return content
	}
	source = tocPattern.ReplaceAllString(source, "")
	source = beginOpen.ReplaceAllString(source, `<span class="legacy-begin">`)
	source = beginClose.ReplaceAllString(source, `</span>`)
	source = panelPattern.ReplaceAllStringFunc(source, renderPanel)
	source = downloadPattern.ReplaceAllStringFunc(source, renderDownload)
	source = collapsePattern.ReplaceAllStringFunc(source, renderCollapse)
	return template.HTML(source)
}

func renderPanel(match string) string {
	parts := panelPattern.FindStringSubmatch(match)
	if len(parts) < 4 || !strings.EqualFold(parts[1], parts[3]) {
		return match
	}
	kind := strings.ToLower(parts[1])
	icon := map[string]string{
		"task":    "fa-tasks",
		"warning": "fa-exclamation-triangle",
		"noway":   "fa-times-circle",
		"buy":     "fa-check-square",
	}[kind]
	return `<div class="legacy-shortcode legacy-` + kind + `"><i class="fa ` + icon + `" aria-hidden="true"></i><div>` + parts[2] + `</div></div>`
}

func renderDownload(match string) string {
	parts := downloadPattern.FindStringSubmatch(match)
	if len(parts) < 2 {
		return match
	}
	href := strings.TrimSpace(stripTags(parts[1]))
	if href == "" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
		return match
	}
	escaped := html.EscapeString(href)
	return `<a class="legacy-download" href="` + escaped + `" rel="external noopener noreferrer" target="_blank"><i class="fa fa-download" aria-hidden="true"></i><span>Download</span></a>`
}

func renderCollapse(match string) string {
	parts := collapsePattern.FindStringSubmatch(match)
	if len(parts) < 3 {
		return match
	}
	title := collapseTitle(parts[1])
	if title == "" {
		title = "More"
	}
	return `<details class="legacy-collapse"><summary><i class="fa fa-angle-down" aria-hidden="true"></i><span>` + html.EscapeString(title) + `</span></summary><div class="legacy-collapse-content">` + parts[2] + `</div></details>`
}

func collapseTitle(attrs string) string {
	matches := titlePattern.FindStringSubmatch(attrs)
	if len(matches) == 0 {
		return ""
	}
	for _, value := range matches[2:] {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
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

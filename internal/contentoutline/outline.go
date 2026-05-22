package contentoutline

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"

	"sakurairo-go/internal/models"
)

const minHeadingCount = 2

func Apply(content template.HTML) (template.HTML, []models.ContentHeading) {
	source := string(content)
	nodes, err := html.ParseFragment(strings.NewReader(source), nil)
	if err != nil {
		return content, nil
	}

	usedIDs := make(map[string]int)
	var outline []models.ContentHeading
	for _, node := range nodes {
		collectHeadings(node, usedIDs, &outline)
	}
	if len(outline) < minHeadingCount {
		return content, nil
	}

	var buf bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&buf, node); err != nil {
			return content, nil
		}
	}
	return template.HTML(buf.String()), outline
}

func collectHeadings(node *html.Node, usedIDs map[string]int, outline *[]models.ContentHeading) {
	if node.Type == html.ElementNode {
		level := headingLevel(node.Data)
		if level >= 2 && level <= 4 {
			title := strings.TrimSpace(textContent(node))
			if title != "" {
				id := headingID(node)
				if id == "" {
					id = uniqueID(slugify(title), usedIDs)
					setAttr(node, "id", id)
				} else if usedIDs[id] > 0 {
					id = uniqueID(id, usedIDs)
					setAttr(node, "id", id)
				} else {
					usedIDs[id]++
				}
				*outline = append(*outline, models.ContentHeading{ID: id, Title: title, Level: level})
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectHeadings(child, usedIDs, outline)
	}
}

func headingLevel(tag string) int {
	if len(tag) != 2 || tag[0] != 'h' || tag[1] < '1' || tag[1] > '6' {
		return 0
	}
	return int(tag[1] - '0')
}

func headingID(node *html.Node) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, "id") {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

func setAttr(node *html.Node, key string, value string) {
	for i := range node.Attr {
		if strings.EqualFold(node.Attr[i].Key, key) {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}

func textContent(node *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			if text := strings.TrimSpace(n.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(parts, " ")
}

func slugify(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "section"
	}
	return "toc-" + slug
}

func uniqueID(base string, used map[string]int) string {
	count := used[base]
	used[base] = count + 1
	if count == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(count+1)
}

package contentoutline

import (
	"html/template"
	"strings"
	"testing"
)

func TestApplyAddsHeadingIDsAndOutline(t *testing.T) {
	content, outline := Apply(template.HTML(`<p>Intro</p><h2>First Step</h2><h3>细节</h3><h2>First Step</h2>`))

	if len(outline) != 3 {
		t.Fatalf("len(outline) = %d, want 3", len(outline))
	}
	if outline[0].ID != "toc-first-step" || outline[0].Title != "First Step" || outline[0].Level != 2 {
		t.Fatalf("first outline = %#v", outline[0])
	}
	if outline[1].ID != "toc-细节" || outline[1].Level != 3 {
		t.Fatalf("second outline = %#v", outline[1])
	}
	if outline[2].ID != "toc-first-step-2" {
		t.Fatalf("duplicate heading id = %q, want toc-first-step-2", outline[2].ID)
	}
	rendered := string(content)
	if !strings.Contains(rendered, `<h2 id="toc-first-step">First Step</h2>`) {
		t.Fatalf("rendered content missing generated id: %s", rendered)
	}
}

func TestApplyKeepsShortContentUntouched(t *testing.T) {
	source := template.HTML(`<h2>Lonely heading</h2><p>Body</p>`)
	content, outline := Apply(source)

	if len(outline) != 0 {
		t.Fatalf("len(outline) = %d, want 0", len(outline))
	}
	if content != source {
		t.Fatalf("content changed for short outline: %s", content)
	}
}

func TestApplyPreservesExistingID(t *testing.T) {
	content, outline := Apply(template.HTML(`<h2 id="kept">Kept</h2><h2 id="kept">Duplicate</h2><h2>Next</h2>`))

	if len(outline) != 3 {
		t.Fatalf("len(outline) = %d, want 3", len(outline))
	}
	if outline[0].ID != "kept" {
		t.Fatalf("existing id = %q, want kept", outline[0].ID)
	}
	if outline[1].ID != "kept-2" {
		t.Fatalf("duplicate existing id = %q, want kept-2", outline[1].ID)
	}
	if !strings.Contains(string(content), `<h2 id="kept">Kept</h2>`) {
		t.Fatalf("rendered content did not preserve id: %s", content)
	}
	if !strings.Contains(string(content), `<h2 id="kept-2">Duplicate</h2>`) {
		t.Fatalf("rendered content did not deduplicate id: %s", content)
	}
}

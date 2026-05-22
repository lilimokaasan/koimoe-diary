package legacyshortcode

import (
	"html/template"
	"strings"
	"testing"
)

func TestApplyConvertsLegacyShortcodes(t *testing.T) {
	got := string(Apply(template.HTML(`[toc]<p>[begin]Hello[/begin] world.</p>
[warning]<strong>Careful</strong>[/warning]
[collapse title="Spoiler"]<p>Hidden</p>[/collapse]
[download]https://example.com/file.zip[/download]
!{A soft image}(https://example.com/full.jpg)[https://example.com/thumb.jpg]`)))

	checks := []string{
		`<span class="legacy-begin">Hello</span>`,
		`<div class="legacy-shortcode legacy-warning">`,
		`<details class="legacy-collapse">`,
		`<span>Spoiler</span>`,
		`<a class="legacy-download" href="https://example.com/file.zip"`,
		`<figure class="legacy-image"><a href="https://example.com/full.jpg"`,
		`<img src="https://example.com/thumb.jpg" alt="A soft image">`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("Apply() missing %q in:\n%s", check, got)
		}
	}
	if strings.Contains(got, "[toc]") {
		t.Fatalf("Apply() left toc marker in output: %s", got)
	}
}

func TestApplyConvertsLegacyImageWithoutThumb(t *testing.T) {
	got := string(Apply(template.HTML(`!{Alt text}(/static/uploads/image.jpg)`)))
	want := `<figure class="legacy-image"><img src="/static/uploads/image.jpg" alt="Alt text"></figure>`
	if got != want {
		t.Fatalf("Apply() image = %s, want %s", got, want)
	}
}

func TestApplyKeepsUnsafeImageLiteral(t *testing.T) {
	got := string(Apply(template.HTML(`!{bad}(javascript:alert(1))`)))
	if got != `!{bad}(javascript:alert(1))` {
		t.Fatalf("Apply() should keep unsafe image literal, got %s", got)
	}
}

func TestApplyKeepsUnsafeDownloadLiteral(t *testing.T) {
	got := string(Apply(template.HTML(`[download]javascript:alert(1)[/download]`)))
	if !strings.Contains(got, `[download]javascript:alert(1)[/download]`) {
		t.Fatalf("Apply() should keep unsafe download literal, got %s", got)
	}
}

func TestApplyRequiresMatchingPanelTags(t *testing.T) {
	got := string(Apply(template.HTML(`[warning]Careful[/task]`)))
	if got != `[warning]Careful[/task]` {
		t.Fatalf("Apply() converted mismatched tags: %s", got)
	}
}

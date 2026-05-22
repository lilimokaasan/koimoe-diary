package commentrender

import (
	"strings"
	"testing"
)

func TestHTMLRendersMarkdownAndSanitizesScript(t *testing.T) {
	got := string(HTML("hello **koi**\n\n<script>alert(1)</script>"))
	if !strings.Contains(got, "<strong>koi</strong>") {
		t.Fatalf("rendered markdown missing strong tag: %s", got)
	}
	if strings.Contains(got, "<script") || strings.Contains(got, "alert(1)") {
		t.Fatalf("unsafe script survived: %s", got)
	}
}

func TestHTMLAddsSafeLinkAttributes(t *testing.T) {
	got := string(HTML("[KoiMoe](https://koimoe.com)"))
	if !strings.Contains(got, `nofollow`) || !strings.Contains(got, `noreferrer`) {
		t.Fatalf("link missing safe rel attributes: %s", got)
	}
	if !strings.Contains(got, `target="_blank"`) {
		t.Fatalf("external link missing blank target: %s", got)
	}
}

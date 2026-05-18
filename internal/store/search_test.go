package store

import "testing"

func TestNormalizeSearchQuery(t *testing.T) {
	got := normalizeSearchQuery("  Koi&nbsp;Moe\n\tDiary  ")
	want := "Koi Moe Diary"
	if got != want {
		t.Fatalf("normalizeSearchQuery() = %q, want %q", got, want)
	}
}

func TestLikePatternEscapesWildcards(t *testing.T) {
	got := likePattern(`100%_koi\moe`)
	want := `%100\%\_koi\\moe%`
	if got != want {
		t.Fatalf("likePattern() = %q, want %q", got, want)
	}
}

func TestSearchTextStripsHTML(t *testing.T) {
	got := searchText(`<p>Koi&nbsp;<strong>Moe</strong></p><script>ignored</script>`)
	want := "Koi Moe ignored"
	if got != want {
		t.Fatalf("searchText() = %q, want %q", got, want)
	}
}

package store

import (
	"testing"

	"sakurairo-go/internal/models"
)

func TestBuildCommentTreeNestsReplies(t *testing.T) {
	comments := []models.Comment{
		{ID: 1, Author: "Lilim", Content: "hello"},
		{ID: 2, ParentID: 1, Author: "Codex", Content: "reply"},
		{ID: 3, Author: "KoiMoe", Content: "another"},
	}

	got := buildCommentTree(comments)
	if len(got) != 2 {
		t.Fatalf("root comments = %d, want 2", len(got))
	}
	if len(got[0].Replies) != 1 {
		t.Fatalf("first root replies = %d, want 1", len(got[0].Replies))
	}
	if got[0].Replies[0].ParentAuthor != "Lilim" {
		t.Fatalf("reply parent author = %q, want Lilim", got[0].Replies[0].ParentAuthor)
	}
	if got[1].ID != 3 {
		t.Fatalf("second root id = %d, want 3", got[1].ID)
	}
}

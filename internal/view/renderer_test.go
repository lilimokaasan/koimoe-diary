package view

import (
	"path/filepath"
	"testing"
)

func TestNewDefaultRendererParsesTemplates(t *testing.T) {
	pattern := filepath.Join("..", "..", "web", "templates", "*.tmpl")
	if _, err := NewRenderer(pattern); err != nil {
		t.Fatalf("parse templates: %v", err)
	}
}

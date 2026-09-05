package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderUsesBundledDefault(t *testing.T) {
	content, err := NewResolver("").Render("new.md", Data{Name: "git", Date: "2026-09-05"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(content, "uchi: v1") ||
		!strings.Contains(content, "create: 2026-09-05") ||
		!strings.Contains(content, "# git\n") ||
		!strings.Contains(content, "- [git](https://github.com)") {
		t.Errorf("Render() = %q, want rendered default values", content)
	}
}

func TestRenderPrefersOverride(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "new.md.tmpl"), []byte("{{ .Date }}: {{ .Name }}"), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := NewResolver(directory).Render("new.md", Data{Name: "git", Date: "2026-09-05"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if content != "2026-09-05: git" {
		t.Errorf("Render() = %q, want override output", content)
	}
}

func TestRenderRejectsInvalidOrUnknownTemplate(t *testing.T) {
	resolver := NewResolver("")
	for _, name := range []string{"", "../new.md", "missing.md"} {
		if _, err := resolver.Render(name, Data{}); err == nil {
			t.Errorf("Render(%q) error = nil, want error", name)
		}
	}
}

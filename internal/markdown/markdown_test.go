package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.md")
	content := "---\ntitle: Test Document\nshell: bash\n---\n\n```bash\necho \"hello world\"\n```\n\n```sh {schema=env}\nGIT_PAGER=vim\n```\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	document, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if document.Frontmatter != "title: Test Document\nshell: bash" {
		t.Errorf("Frontmatter = %q", document.Frontmatter)
	}
	if len(document.CodeFences) != 3 {
		t.Fatalf("CodeFences count = %d, want 3", len(document.CodeFences))
	}
	if document.CodeFences[0].HasAnnotation {
		t.Errorf("expected first fence to have HasAnnotation=false, got true")
	}
	if document.CodeFences[0].Language != "bash" || document.CodeFences[0].Content != "echo \"hello world\"" {
		t.Errorf("first fence = %+v", document.CodeFences[0])
	}

	if !document.CodeFences[1].HasAnnotation {
		t.Errorf("expected second fence to have HasAnnotation=true")
	}
	if document.CodeFences[1].Language != "sh" || document.CodeFences[1].Schema() != "env" {
		t.Errorf("second fence = %+v", document.CodeFences[1])
	}

	if !document.CodeFences[2].HasAnnotation {
		t.Errorf("expected third fence to have HasAnnotation=true")
	}
	if document.CodeFences[2].Language != "sh" || document.CodeFences[2].Schema() != "alias" {
		t.Errorf("third fence = %+v", document.CodeFences[2])
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "valid v1 string",
			input:       "uchi: v1\ntitle: test",
			wantVersion: "v1",
			wantErr:     false,
		},
		{
			name:        "valid v1 quoted",
			input:       "uchi: \"v1\"\ntitle: test",
			wantVersion: "v1",
			wantErr:     false,
		},
		{
			name:        "unspecified uchi field",
			input:       "title: General Note",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "empty frontmatter",
			input:       "",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "yaml comment only",
			input:       "# uchi: v1\ntitle: test",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "uchi inside another key string",
			input:       "note: \"uchi: v1\"",
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:        "unknown version v2",
			input:       "uchi: v2",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "type mismatch int",
			input:       "uchi: 1",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "type mismatch bool",
			input:       "uchi: true",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "type mismatch sequence",
			input:       "uchi:\n  - v1",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "type mismatch map",
			input:       "uchi:\n  version: v1",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "type mismatch null",
			input:       "uchi: null",
			wantVersion: "",
			wantErr:     true,
		},
		{
			name:        "broken yaml syntax",
			input:       "uchi: [invalid",
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if fm.UchiVersion != tt.wantVersion {
				t.Errorf("ParseFrontmatter() UchiVersion = %q, want %q", fm.UchiVersion, tt.wantVersion)
			}
		})
	}
}

func TestParseFileFrontmatterValidation(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid v1 file", func(t *testing.T) {
		path := filepath.Join(dir, "valid.md")
		if err := os.WriteFile(path, []byte("---\nuchi: v1\n---\n# hello"), 0644); err != nil {
			t.Fatal(err)
		}
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if doc.UchiVersion != "v1" {
			t.Errorf("UchiVersion = %q, want %q", doc.UchiVersion, "v1")
		}
	})

	t.Run("broken yaml frontmatter includes filepath", func(t *testing.T) {
		path := filepath.Join(dir, "broken.md")
		if err := os.WriteFile(path, []byte("---\nuchi: [unclosed\n---\n# hello"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q should contain filepath %q", err.Error(), path)
		}
	})

	t.Run("type mismatch includes filepath", func(t *testing.T) {
		path := filepath.Join(dir, "typemismatch.md")
		if err := os.WriteFile(path, []byte("---\nuchi: 123\n---\n# hello"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q should contain filepath %q", err.Error(), path)
		}
	})

	t.Run("unknown version includes filepath", func(t *testing.T) {
		path := filepath.Join(dir, "unknown.md")
		if err := os.WriteFile(path, []byte("---\nuchi: v2\n---\n# hello"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q should contain filepath %q", err.Error(), path)
		}
	})
}

func TestWalkFindsMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"one.md":              "# one",
		"nested/two.markdown": "# two",
		"ignored.txt":         "ignored",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	documents, err := Walk(dir)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}
	if len(documents) != 2 {
		t.Errorf("Walk() returned %d documents, want 2", len(documents))
	}
}

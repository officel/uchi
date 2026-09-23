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

func TestParseFenceHeader(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLang   string
		wantAnnot  bool
		wantParams map[string]string
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "unannotated language",
			input:      "```bash",
			wantLang:   "bash",
			wantAnnot:  false,
			wantParams: map[string]string{},
			wantErr:    false,
		},
		{
			name:       "unannotated empty",
			input:      "```",
			wantLang:   "",
			wantAnnot:  false,
			wantParams: map[string]string{},
			wantErr:    false,
		},
		{
			name:       "simple schema=env",
			input:      "```sh {schema=env}",
			wantLang:   "sh",
			wantAnnot:  true,
			wantParams: map[string]string{"schema": "env"},
			wantErr:    false,
		},
		{
			name:       "boolean flag attribute",
			input:      "```sh {schema=alias flag}",
			wantLang:   "sh",
			wantAnnot:  true,
			wantParams: map[string]string{"schema": "alias", "flag": "true"},
			wantErr:    false,
		},
		{
			name:       "double quoted value with space",
			input:      "```bash {schema=\"env with space\"}",
			wantLang:   "bash",
			wantAnnot:  true,
			wantParams: map[string]string{"schema": "env with space"},
			wantErr:    false,
		},
		{
			name:       "single quoted value with space",
			input:      "```zsh {schema='env with single space'}",
			wantLang:   "zsh",
			wantAnnot:  true,
			wantParams: map[string]string{"schema": "env with single space"},
			wantErr:    false,
		},
		{
			name:       "no language before brace",
			input:      "```{schema=rc}",
			wantLang:   "",
			wantAnnot:  true,
			wantParams: map[string]string{"schema": "rc"},
			wantErr:    false,
		},
		{
			name:      "duplicate attribute key",
			input:     "```sh {schema=env schema=alias}",
			wantErr:   true,
			errSubstr: "duplicate attribute key",
		},
		{
			name:      "empty value unquoted",
			input:     "```sh {schema=}",
			wantErr:   true,
			errSubstr: "empty attribute value",
		},
		{
			name:      "empty value double quoted",
			input:     "```sh {schema=\"\"}",
			wantErr:   true,
			errSubstr: "empty attribute value",
		},
		{
			name:      "empty value single quoted",
			input:     "```sh {schema=''}",
			wantErr:   true,
			errSubstr: "empty attribute value",
		},
		{
			name:      "unclosed opening brace",
			input:     "```sh {schema=env",
			wantErr:   true,
			errSubstr: "unclosed '{'",
		},
		{
			name:      "unmatched closing brace",
			input:     "```sh schema=env}",
			wantErr:   true,
			errSubstr: "unmatched '}'",
		},
		{
			name:      "extra content after closing brace",
			input:     "```sh {schema=env} extra",
			wantErr:   true,
			errSubstr: "unexpected content",
		},
		{
			name:      "unclosed double quote",
			input:     "```sh {schema=\"env}",
			wantErr:   true,
			errSubstr: "unclosed double quote",
		},
		{
			name:      "unclosed single quote",
			input:     "```sh {schema='env}",
			wantErr:   true,
			errSubstr: "unclosed single quote",
		},
		{
			name:      "empty attribute key",
			input:     "```sh {=env}",
			wantErr:   true,
			errSubstr: "invalid or empty attribute name",
		},
		{
			name:      "invalid attribute key char",
			input:     "```sh {schema?=env}",
			wantErr:   true,
			errSubstr: "invalid character",
		},
		{
			name:      "missing space between attribute value and next token",
			input:     "```sh {schema=\"env\"extra}",
			wantErr:   true,
			errSubstr: "expected whitespace or '}'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fence, err := ParseFenceHeader(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFenceHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain substring %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if fence.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", fence.Language, tt.wantLang)
			}
			if fence.HasAnnotation != tt.wantAnnot {
				t.Errorf("HasAnnotation = %v, want %v", fence.HasAnnotation, tt.wantAnnot)
			}
			if len(fence.Params) != len(tt.wantParams) {
				t.Errorf("Params len = %d, want %d", len(fence.Params), len(tt.wantParams))
			}
			for k, wantVal := range tt.wantParams {
				if gotVal, ok := fence.Params[k]; !ok || gotVal != wantVal {
					t.Errorf("Params[%q] = %q (exists: %v), want %q", k, gotVal, ok, wantVal)
				}
			}
		})
	}
}

func TestParseFileCodeFenceValidation(t *testing.T) {
	dir := t.TempDir()

	t.Run("uchi v1 file with broken fence header returns error with path", func(t *testing.T) {
		path := filepath.Join(dir, "broken_fence.md")
		content := "---\nuchi: v1\n---\n```sh {schema=env\necho hi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for unclosed '{', got nil")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q should contain file path %q", err.Error(), path)
		}
	})

	t.Run("non uchi v1 file with broken fence header is skipped without error", func(t *testing.T) {
		path := filepath.Join(dir, "ignored_fence.md")
		content := "---\ntitle: Note\n---\n```sh {schema=env\necho hi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error for non-uchi file: %v", err)
		}
		if doc.UchiVersion != "" {
			t.Errorf("expected empty UchiVersion, got %q", doc.UchiVersion)
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

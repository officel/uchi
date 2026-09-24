package markdown

import (
	"errors"
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

	t.Run("uchi v1 file with unknown schema returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "unknown_schema.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=custom}\necho hi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for unknown schema, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `unknown schema "custom"`) {
			t.Errorf("message %q should contain unknown schema info", diag.Message)
		}
		if !strings.Contains(diag.Message, "valid schemas: alias, env, function, profile, rc") {
			t.Errorf("message %q should list valid schemas", diag.Message)
		}
	})

	t.Run("uchi v1 file with unknown attribute returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "unknown_attr.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias foo=bar}\nalias a=b\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for unknown attribute, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `unknown attribute "foo"`) {
			t.Errorf("message %q should contain unknown attribute info", diag.Message)
		}
		if !strings.Contains(diag.Message, "allowed attributes: schema, target") {
			t.Errorf("message %q should list allowed attributes", diag.Message)
		}
	})

	t.Run("uchi v1 file with annotated fence missing schema returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "missing_schema.md")
		content := "---\nuchi: v1\n---\n\n```sh {}\necho hi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for missing schema, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, "missing required 'schema' attribute") {
			t.Errorf("message %q should contain missing required schema info", diag.Message)
		}
	})

	t.Run("uchi v1 file with unannotated code fence succeeds without error", func(t *testing.T) {
		path := filepath.Join(dir, "unannotated.md")
		content := "---\nuchi: v1\n---\n\n```sh\necho hi\n```\n\n```sh {schema=alias}\nalias a=b\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error for file with unannotated fence: %v", err)
		}
		if len(doc.CodeFences) != 2 {
			t.Errorf("len(doc.CodeFences) = %d, want 2", len(doc.CodeFences))
		}
	})
}

func TestParseFileTargetValidation(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid single target", func(t *testing.T) {
		path := filepath.Join(dir, "single_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=bash}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.CodeFences) != 1 {
			t.Fatalf("CodeFences count = %d, want 1", len(doc.CodeFences))
		}
		if len(doc.CodeFences[0].Targets) != 1 || doc.CodeFences[0].Targets[0] != "bash" {
			t.Errorf("Targets = %v, want [bash]", doc.CodeFences[0].Targets)
		}
	})

	t.Run("valid comma-separated multiple targets", func(t *testing.T) {
		path := filepath.Join(dir, "multi_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=\"bash, zsh, powershell\"}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.CodeFences) != 1 {
			t.Fatalf("CodeFences count = %d, want 1", len(doc.CodeFences))
		}
		want := []string{"bash", "zsh", "powershell"}
		got := doc.CodeFences[0].Targets
		if len(got) != len(want) {
			t.Fatalf("Targets = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Targets[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("unspecified target defaults to all", func(t *testing.T) {
		path := filepath.Join(dir, "unspecified_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias}\nalias a=b\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(doc.CodeFences[0].Targets) != 1 || doc.CodeFences[0].Targets[0] != "all" {
			t.Errorf("Targets = %v, want [all]", doc.CodeFences[0].Targets)
		}
	})

	t.Run("unknown target returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "unknown_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=cmd}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for unknown target, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `unknown target shell "cmd"`) {
			t.Errorf("message %q should contain unknown target shell info", diag.Message)
		}
		if !strings.Contains(diag.Message, "valid targets: all, bash, fish, powershell, pwsh, sh, zsh") {
			t.Errorf("message %q should list valid targets", diag.Message)
		}
	})

	t.Run("duplicate target shell returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "duplicate_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=\"bash, bash\"}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for duplicate target shell, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `duplicate target shell "bash"`) {
			t.Errorf("message %q should contain duplicate target info", diag.Message)
		}
	})

	t.Run("combining all with specific targets returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "all_combined.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=\"all, bash\"}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for combining all with specific targets, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `target "all" cannot be combined with specific target shells`) {
			t.Errorf("message %q should contain all combination error info", diag.Message)
		}
	})

	t.Run("invalid empty target value returns Diagnostic error", func(t *testing.T) {
		path := filepath.Join(dir, "empty_target_comma.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env target=\",\"}\nexport FOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error for empty comma target, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path || diag.Line != 5 {
			t.Errorf("diag = %v, want path=%s line=5", diag, path)
		}
		if !strings.Contains(diag.Message, `invalid or empty target attribute value`) {
			t.Errorf("message %q should contain invalid/empty target info", diag.Message)
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

func TestWalkDeterministicRelativePathOrder(t *testing.T) {
	dir := t.TempDir()

	// Write files in reverse alphabetical / non-sequential creation order
	files := []string{
		"z_last.md",
		"sub/beta.md",
		"sub/alpha.md",
		"a_first.md",
	}

	for _, rel := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("---\nuchi: v1\n---\n# "+rel), 0644); err != nil {
			t.Fatal(err)
		}
	}

	documents, err := Walk(dir)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if len(documents) != len(files) {
		t.Fatalf("len(documents) = %d, want %d", len(documents), len(files))
	}

	wantOrder := []string{
		"a_first.md",
		"sub/alpha.md",
		"sub/beta.md",
		"z_last.md",
	}

	for i, doc := range documents {
		rel, err := filepath.Rel(dir, doc.FilePath)
		if err != nil {
			t.Fatalf("filepath.Rel failed: %v", err)
		}
		slashRel := filepath.ToSlash(rel)
		if slashRel != wantOrder[i] {
			t.Errorf("documents[%d] relative path = %q, want %q", i, slashRel, wantOrder[i])
		}
	}
}

func TestParseFileLineTracking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.md")
	content := "---\nuchi: v1\n---\n\nSome header\n```sh {schema=env}\nFOO=bar\n```\n\n```sh {schema=alias}\nalias x=y\n```\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if doc.FrontmatterStartLine != 1 {
		t.Errorf("FrontmatterStartLine = %d, want 1", doc.FrontmatterStartLine)
	}
	if len(doc.CodeFences) != 2 {
		t.Fatalf("CodeFences count = %d, want 2", len(doc.CodeFences))
	}
	if doc.CodeFences[0].StartLine != 6 {
		t.Errorf("CodeFences[0].StartLine = %d, want 6", doc.CodeFences[0].StartLine)
	}
	if doc.CodeFences[1].StartLine != 10 {
		t.Errorf("CodeFences[1].StartLine = %d, want 10", doc.CodeFences[1].StartLine)
	}
}

func TestDiagnosticErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("frontmatter error returns Diagnostic with line 1", func(t *testing.T) {
		path := filepath.Join(dir, "bad_fm.md")
		content := "---\nuchi: [invalid\n---\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path {
			t.Errorf("diag.Path = %q, want %q", diag.Path, path)
		}
		if diag.Line != 1 {
			t.Errorf("diag.Line = %d, want 1", diag.Line)
		}
		if !strings.HasPrefix(err.Error(), path+":1:") {
			t.Errorf("Error() = %q, want prefix %q", err.Error(), path+":1:")
		}
	})

	t.Run("code fence attribute error returns Diagnostic with exact line number", func(t *testing.T) {
		path := filepath.Join(dir, "bad_fence.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=env\nFOO=bar\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var diag *Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != path {
			t.Errorf("diag.Path = %q, want %q", diag.Path, path)
		}
		if diag.Line != 5 {
			t.Errorf("diag.Line = %d, want 5", diag.Line)
		}
		if !strings.HasPrefix(err.Error(), path+":5:") {
			t.Errorf("Error() = %q, want prefix %q", err.Error(), path+":5:")
		}
	})
}

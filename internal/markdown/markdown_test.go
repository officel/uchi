package markdown

import (
	"os"
	"path/filepath"
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

package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.md")
	content := "---\ntitle: Test Document\nshell: bash\n---\n\n```bash\necho \"hello world\"\n```\n\n```zsh\nexport FOO=bar\n```\n"
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
	if len(document.CodeFences) != 2 {
		t.Fatalf("CodeFences count = %d, want 2", len(document.CodeFences))
	}
	if document.CodeFences[0] != (CodeFence{Language: "bash", Content: "echo \"hello world\""}) {
		t.Errorf("first fence = %+v", document.CodeFences[0])
	}
	if document.CodeFences[1] != (CodeFence{Language: "zsh", Content: "export FOO=bar"}) {
		t.Errorf("second fence = %+v", document.CodeFences[1])
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

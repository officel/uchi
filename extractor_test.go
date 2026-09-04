package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `---
title: Test Document
shell: bash
---

# Title

Here is a paragraph.

` + "```bash\necho \"hello world\"\n```" + `

Another paragraph.

` + "```zsh\nexport FOO=bar\n```" + `
`

	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to write test md file: %v", err)
	}

	doc, err := ParseMarkdownFile(mdPath)
	if err != nil {
		t.Fatalf("failed to parse markdown file: %v", err)
	}

	expectedFM := "title: Test Document\nshell: bash"
	if doc.Frontmatter != expectedFM {
		t.Errorf("expected frontmatter %q, got %q", expectedFM, doc.Frontmatter)
	}

	if len(doc.CodeFences) != 2 {
		t.Fatalf("expected 2 code fences, got %d", len(doc.CodeFences))
	}

	if doc.CodeFences[0].Language != "bash" {
		t.Errorf("expected language 'bash', got %q", doc.CodeFences[0].Language)
	}
	if doc.CodeFences[0].Content != "echo \"hello world\"" {
		t.Errorf("expected content 'echo \"hello world\"', got %q", doc.CodeFences[0].Content)
	}

	if doc.CodeFences[1].Language != "zsh" {
		t.Errorf("expected language 'zsh', got %q", doc.CodeFences[1].Language)
	}
	if doc.CodeFences[1].Content != "export FOO=bar" {
		t.Errorf("expected content 'export FOO=bar', got %q", doc.CodeFences[1].Content)
	}
}

func TestRunExtractionPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	inDir := filepath.Join(tmpDir, "input")
	outDir := filepath.Join(tmpDir, "output")

	if err := os.MkdirAll(inDir, 0755); err != nil {
		t.Fatalf("failed to create input dir: %v", err)
	}

	mdContent := `---
shell: bash
---
` + "```bash\necho \"extracted\"\n```"

	mdPath := filepath.Join(inDir, "example.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to create md file: %v", err)
	}

	cfg := &Config{
		InputDir:  inDir,
		OutputDir: outDir,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	outFile := filepath.Join(outDir, "example.txt")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file %s: %v", outFile, err)
	}

	expected := "echo \"extracted\""
	if string(content) != expected {
		t.Errorf("expected file content %q, got %q", expected, string(content))
	}
}

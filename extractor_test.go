package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownFile(t *testing.T) {
	mdPath := filepath.Join("testdata", "sample.md")

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
	outDir := filepath.Join(tmpDir, "output")

	cfg := &Config{
		InputDir:  "testdata",
		OutputDir: outDir,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	outFile1 := filepath.Join(outDir, "sample_1.txt")
	content1, err := os.ReadFile(outFile1)
	if err != nil {
		t.Fatalf("failed to read output file %s: %v", outFile1, err)
	}

	expected1 := "echo \"hello world\""
	if string(content1) != expected1 {
		t.Errorf("expected file content %q, got %q", expected1, string(content1))
	}

	outFile2 := filepath.Join(outDir, "sample_2.txt")
	content2, err := os.ReadFile(outFile2)
	if err != nil {
		t.Fatalf("failed to read output file %s: %v", outFile2, err)
	}

	expected2 := "export FOO=bar"
	if string(content2) != expected2 {
		t.Errorf("expected file content %q, got %q", expected2, string(content2))
	}
}

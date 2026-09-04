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

func TestRunExtractionPipelineAndDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	inDir := filepath.Join(tmpDir, "non_existent_input")
	outDir := filepath.Join(tmpDir, "non_existent_output")

	// Ensure input/output dirs do not exist before Run
	if _, err := os.Stat(inDir); !os.IsNotExist(err) {
		t.Fatalf("input dir shouldn't exist initially")
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("output dir shouldn't exist initially")
	}

	cfg := &Config{
		InputDir:  inDir,
		OutputDir: outDir,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify input and output directories were created by Run
	if info, err := os.Stat(inDir); err != nil || !info.IsDir() {
		t.Errorf("expected input directory %s to be created", inDir)
	}
	if info, err := os.Stat(outDir); err != nil || !info.IsDir() {
		t.Errorf("expected output directory %s to be created", outDir)
	}
}

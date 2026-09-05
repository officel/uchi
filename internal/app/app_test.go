package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/officel/uchi/internal/config"
)

func TestRunExtractsCodeFences(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "toc")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "git.md"), []byte("```sh\ngit status\n```"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(outputDir, "git.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "git status" {
		t.Errorf("extracted content = %q", content)
	}
}

func TestRunNewUsesTemplateOverride(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "new.md.tmpl"), []byte("# {{ .Name }}\n{{ .Date }}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	inputDir := filepath.Join(dir, "toc")
	if err := Run(&config.Config{InputDir: inputDir, TemplateDir: templateDir, Command: "new", CommandArg: "git"}, &output); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(inputDir, "git.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# git\n" + time.Now().Format("2006-01-02") + "\n"
	if string(content) != want {
		t.Errorf("created content = %q, want %q", content, want)
	}
	if !strings.Contains(output.String(), "Created ") {
		t.Errorf("output = %q, want creation message", output.String())
	}
}

func TestRunInit(t *testing.T) {
	dir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	var output bytes.Buffer
	cfg := &config.Config{
		InputDir:  "./toc",
		OutputDir: "./dist",
		Command:   "init",
	}
	if err := Run(cfg, &output); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(".uchi.yaml")
	if err != nil {
		t.Fatalf("failed to read .uchi.yaml: %v", err)
	}
	if !strings.Contains(string(content), "input_dir: ./toc") {
		t.Errorf("content = %s, want input_dir: ./toc", string(content))
	}
	if strings.Contains(string(content), "config_file") {
		t.Errorf("content contains config_file: %s", string(content))
	}
	if !strings.Contains(output.String(), "Created ") {
		t.Errorf("output = %q, want creation message", output.String())
	}
}

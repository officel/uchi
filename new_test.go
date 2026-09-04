package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigNewSubcommand(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, err := LoadConfig([]string{"new", "git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Command != "new" {
		t.Errorf("expected Command 'new', got %q", cfg.Command)
	}
	if cfg.CommandArg != "git" {
		t.Errorf("expected CommandArg 'git', got %q", cfg.CommandArg)
	}

	// Test missing argument for 'new'
	_, err = LoadConfig([]string{"new"})
	if err == nil {
		t.Errorf("expected error when argument for 'new' is missing")
	}

	// Test flag override with 'new' command
	cfgWithFlags, err := LoadConfig([]string{"-i", "./custom_toc", "new", "tmux"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgWithFlags.Command != "new" || cfgWithFlags.CommandArg != "tmux" || cfgWithFlags.InputDir != "./custom_toc" {
		t.Errorf("expected custom input dir and new subcommand, got %+v", cfgWithFlags)
	}
}

func TestRunNewSubcommand(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(origDir)

	// Write custom new.md template in current directory
	tmpl := `# {name}

- {date}
- [name](https://github.com)

## note

## alias

` + "```text\n# alias\nalias g=\"git\"\n```" + `

## rc

` + "```text\n```" + `

## profile

` + "```text\n```\n"

	if err := os.WriteFile("new.md", []byte(tmpl), 0644); err != nil {
		t.Fatalf("failed to create new.md: %v", err)
	}

	inputDir := filepath.Join(tmpDir, "toc")
	cfg := &Config{
		InputDir:   inputDir,
		OutputDir:  filepath.Join(tmpDir, "dist"),
		Command:    "new",
		CommandArg: "git",
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run failed for new subcommand: %v", err)
	}

	targetPath := filepath.Join(inputDir, "git.md")
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read created markdown file %s: %v", targetPath, err)
	}

	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "# git\n") {
		t.Errorf("expected content to start with '# git', got:\n%s", contentStr)
	}

	today := time.Now().Format("2006-01-02")
	if !strings.Contains(contentStr, "- "+today) {
		t.Errorf("expected content to contain date %s, got:\n%s", today, contentStr)
	}
}

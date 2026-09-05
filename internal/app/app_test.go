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

	gitMd := "---\nuchi: v1\n---\n## Environment\n\n```sh {schema=env}\nGIT_PAGER=vim\n```\n\n## alias\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n\n```sh\n# unannotated code block ignored\necho test\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "git.md"), []byte(gitMd), 0644); err != nil {
		t.Fatal(err)
	}

	zoxideMd := "---\nuchi: v1\n---\n## alias\n\n```sh {schema=alias}\nalias z=\"zoxide\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "zoxide.md"), []byte(zoxideMd), 0644); err != nil {
		t.Fatal(err)
	}

	ignoredMd := "```sh {schema=alias}\nalias bad=\"bad\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "ignored.md"), []byte(ignoredMd), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	gitEnv, err := os.ReadFile(filepath.Join(outputDir, "git", "env"))
	if err != nil {
		t.Fatalf("failed to read git/env: %v", err)
	}
	if string(gitEnv) != "GIT_PAGER=vim" {
		t.Errorf("git/env = %q, want %q", string(gitEnv), "GIT_PAGER=vim")
	}

	gitAlias, err := os.ReadFile(filepath.Join(outputDir, "git", "alias"))
	if err != nil {
		t.Fatalf("failed to read git/alias: %v", err)
	}
	if string(gitAlias) != "alias g=\"git\"" {
		t.Errorf("git/alias = %q, want %q", string(gitAlias), "alias g=\"git\"")
	}

	zoxideAlias, err := os.ReadFile(filepath.Join(outputDir, "zoxide", "alias"))
	if err != nil {
		t.Fatalf("failed to read zoxide/alias: %v", err)
	}
	if string(zoxideAlias) != "alias z=\"zoxide\"" {
		t.Errorf("zoxide/alias = %q, want %q", string(zoxideAlias), "alias z=\"zoxide\"")
	}

	mergedEnv, err := os.ReadFile(filepath.Join(outputDir, "env"))
	if err != nil {
		t.Fatalf("failed to read merged env: %v", err)
	}
	if string(mergedEnv) != "GIT_PAGER=vim" {
		t.Errorf("merged env = %q, want %q", string(mergedEnv), "GIT_PAGER=vim")
	}

	mergedAlias, err := os.ReadFile(filepath.Join(outputDir, "alias"))
	if err != nil {
		t.Fatalf("failed to read merged alias: %v", err)
	}
	wantMergedAlias := "alias g=\"git\"\nalias z=\"zoxide\""
	if string(mergedAlias) != wantMergedAlias {
		t.Errorf("merged alias = %q, want %q", string(mergedAlias), wantMergedAlias)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "ignored")); !os.IsNotExist(err) {
		t.Errorf("expected ignored directory to not exist, got err = %v", err)
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

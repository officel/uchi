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

	gitMd := "---\nuchi: v1\n---\n## Environment\n\n```sh {schema=env}\nGIT_PAGER=vim\n```\n\n## alias\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n\n```sh {schema=profile}\numask 022\n```\n\n```sh {schema=rc}\nset -o vi\n```\n\n```sh {schema=function}\ngit_clean() { git clean -df; }\n```\n\n```sh {schema=unknown}\necho unknown\n```\n\n```sh\n# unannotated code block ignored\necho test\n```\n"
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

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	gitEnv, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "env"))
	if err != nil {
		t.Fatalf("failed to read git/env: %v", err)
	}
	if string(gitEnv) != "# git\nGIT_PAGER=vim\n" {
		t.Errorf("git/env = %q, want %q", string(gitEnv), "# git\nGIT_PAGER=vim\n")
	}

	gitAlias, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "alias"))
	if err != nil {
		t.Fatalf("failed to read git/alias: %v", err)
	}
	if string(gitAlias) != "# git\nalias g=\"git\"\n" {
		t.Errorf("git/alias = %q, want %q", string(gitAlias), "# git\nalias g=\"git\"\n")
	}

	zoxideAlias, err := os.ReadFile(filepath.Join(outputDir, "parts", "zoxide", "alias"))
	if err != nil {
		t.Fatalf("failed to read zoxide/alias: %v", err)
	}
	if string(zoxideAlias) != "# zoxide\nalias z=\"zoxide\"\n" {
		t.Errorf("zoxide/alias = %q, want %q", string(zoxideAlias), "# zoxide\nalias z=\"zoxide\"\n")
	}

	mergedEnv, err := os.ReadFile(filepath.Join(outputDir, "env"))
	if err != nil {
		t.Fatalf("failed to read merged env: %v", err)
	}
	if string(mergedEnv) != "# git\nGIT_PAGER=vim\n" {
		t.Errorf("merged env = %q, want %q", string(mergedEnv), "# git\nGIT_PAGER=vim\n")
	}

	gitProfile, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "profile"))
	if err != nil {
		t.Fatalf("failed to read git/profile: %v", err)
	}
	if string(gitProfile) != "# git\numask 022\n" {
		t.Errorf("git/profile = %q, want %q", string(gitProfile), "# git\numask 022\n")
	}

	gitRc, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "rc"))
	if err != nil {
		t.Fatalf("failed to read git/rc: %v", err)
	}
	if string(gitRc) != "# git\nset -o vi\n" {
		t.Errorf("git/rc = %q, want %q", string(gitRc), "# git\nset -o vi\n")
	}

	gitFunction, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "function"))
	if err != nil {
		t.Fatalf("failed to read git/function: %v", err)
	}
	if string(gitFunction) != "# git\ngit_clean() { git clean -df; }\n" {
		t.Errorf("git/function = %q, want %q", string(gitFunction), "# git\ngit_clean() { git clean -df; }\n")
	}

	mergedAlias, err := os.ReadFile(filepath.Join(outputDir, "alias"))
	if err != nil {
		t.Fatalf("failed to read merged alias: %v", err)
	}
	wantMergedAlias := "# git\nalias g=\"git\"\n\n# zoxide\nalias z=\"zoxide\"\n"
	if string(mergedAlias) != wantMergedAlias {
		t.Errorf("merged alias = %q, want %q", string(mergedAlias), wantMergedAlias)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "parts", "git", "unknown")); !os.IsNotExist(err) {
		t.Errorf("expected git/unknown file to not exist, got err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "unknown")); !os.IsNotExist(err) {
		t.Errorf("expected unknown schema file to not exist, got err = %v", err)
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

func TestRunCheck(t *testing.T) {
	t.Run("default command when config file is present", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			ConfigFile:  "./.uchi.yaml",
			InputDir:    "./toc",
			OutputDir:   "./dist",
			TemplateDir: "./templates",
			AutoComment: true,
			Command:     "",
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		out := output.String()
		if !strings.Contains(out, "Config file: found (./.uchi.yaml)") {
			t.Errorf("out = %q, want found config file message", out)
		}
		if !strings.Contains(out, "input_dir: ./toc") || !strings.Contains(out, "output_dir: ./dist") || !strings.Contains(out, "template_dir: ./templates") || !strings.Contains(out, "auto_comment: true") {
			t.Errorf("out = %q, want all configuration values displayed", out)
		}
	})

	t.Run("check command when config file is not present", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			ConfigFile:  "",
			InputDir:    "./toc",
			OutputDir:   "./dist",
			AutoComment: false,
			Command:     "check",
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		out := output.String()
		if !strings.Contains(out, "Config file: not found") {
			t.Errorf("out = %q, want not found config file message", out)
		}
		if !strings.Contains(out, "auto_comment: false") {
			t.Errorf("out = %q, want auto_comment: false", out)
		}
	})
}

func TestRunExtractsCodeFencesWithoutAutoComment(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "toc")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	gitMd := "---\nuchi: v1\n---\n## alias\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "git.md"), []byte(gitMd), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: false, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	gitAlias, err := os.ReadFile(filepath.Join(outputDir, "parts", "git", "alias"))
	if err != nil {
		t.Fatalf("failed to read git/alias: %v", err)
	}
	if string(gitAlias) != "alias g=\"git\"\n" {
		t.Errorf("git/alias = %q, want %q", string(gitAlias), "alias g=\"git\"\n")
	}
}

func TestRunEmptyAndEOFNewline(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "toc")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1Md := "---\nuchi: v1\n---\n```sh {schema=alias}\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "doc1.md"), []byte(doc1Md), 0644); err != nil {
		t.Fatal(err)
	}

	doc2Md := "---\nuchi: v1\n---\n```sh {schema=alias}\n\n\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "doc2.md"), []byte(doc2Md), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	doc1Alias, err := os.ReadFile(filepath.Join(outputDir, "parts", "doc1", "alias"))
	if err != nil {
		t.Fatalf("failed to read doc1/alias: %v", err)
	}
	if string(doc1Alias) != "\n" {
		t.Errorf("doc1/alias = %q, want %q", string(doc1Alias), "\n")
	}

	doc2Alias, err := os.ReadFile(filepath.Join(outputDir, "parts", "doc2", "alias"))
	if err != nil {
		t.Fatalf("failed to read doc2/alias: %v", err)
	}
	if string(doc2Alias) != "\n" {
		t.Errorf("doc2/alias = %q, want %q", string(doc2Alias), "\n")
	}

	mergedAlias, err := os.ReadFile(filepath.Join(outputDir, "alias"))
	if err != nil {
		t.Fatalf("failed to read merged alias: %v", err)
	}
	if string(mergedAlias) != "\n" {
		t.Errorf("merged alias = %q, want %q", string(mergedAlias), "\n")
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

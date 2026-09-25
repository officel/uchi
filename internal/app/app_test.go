package app

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/officel/uchi/internal/config"
	"github.com/officel/uchi/internal/markdown"
)

var update = flag.Bool("update", false, "update golden test fixtures")

func TestGolden(t *testing.T) {
	testdataDir := "testdata"
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fixtureName := entry.Name()
		t.Run(fixtureName, func(t *testing.T) {
			fixtureDir := filepath.Join(testdataDir, fixtureName)
			inputDir := filepath.Join(fixtureDir, "input")
			wantDir := filepath.Join(fixtureDir, "want")

			if _, err := os.Stat(inputDir); os.IsNotExist(err) {
				t.Skipf("skipping %s: input directory does not exist", fixtureName)
			}

			tmpDir := t.TempDir()
			tmpInput := filepath.Join(tmpDir, "input")
			tmpOutput := filepath.Join(tmpDir, "output")

			if err := copyDir(inputDir, tmpInput); err != nil {
				t.Fatalf("failed to copy input directory for %s: %v", fixtureName, err)
			}

			autoComment := true
			if fixtureName == "no_autocomment" {
				autoComment = false
			}

			cfg := &config.Config{
				InputDir:    tmpInput,
				OutputDir:   tmpOutput,
				AutoComment: autoComment,
				Command:     "gen",
			}

			var buf bytes.Buffer
			if err := Run(cfg, &buf); err != nil {
				t.Fatalf("Run() error for %s: %v", fixtureName, err)
			}

			if *update {
				if err := updateGolden(tmpOutput, wantDir); err != nil {
					t.Fatalf("failed to update golden fixture %s: %v", fixtureName, err)
				}
				return
			}

			compareDirTrees(t, tmpOutput, wantDir)
		})
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func updateGolden(gotDir, wantDir string) error {
	if err := os.RemoveAll(wantDir); err != nil {
		return err
	}
	return copyDir(gotDir, wantDir)
}

func compareDirTrees(t *testing.T, gotDir, wantDir string) {
	t.Helper()

	wantFiles := make(map[string]string)
	if _, err := os.Stat(wantDir); err == nil {
		err := filepath.Walk(wantDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(wantDir, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			wantFiles[filepath.ToSlash(rel)] = string(data)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk wantDir %s: %v", wantDir, err)
		}
	}

	gotFiles := make(map[string]string)
	if _, err := os.Stat(gotDir); err == nil {
		err := filepath.Walk(gotDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(gotDir, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			gotFiles[filepath.ToSlash(rel)] = string(data)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk gotDir %s: %v", gotDir, err)
		}
	}

	var wantKeys []string
	for k := range wantFiles {
		wantKeys = append(wantKeys, k)
	}
	sort.Strings(wantKeys)

	for _, rel := range wantKeys {
		if _, exists := gotFiles[rel]; !exists {
			t.Errorf("missing expected output file: %s", rel)
		}
	}

	var gotKeys []string
	for k := range gotFiles {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)

	for _, rel := range gotKeys {
		if _, exists := wantFiles[rel]; !exists {
			t.Errorf("unexpected output file: %s", rel)
		}
	}

	for _, rel := range wantKeys {
		gotContent, exists := gotFiles[rel]
		if !exists {
			continue
		}
		wantContent := wantFiles[rel]
		if gotContent != wantContent {
			oldLines := splitLines(wantContent)
			newLines := splitLines(gotContent)
			ops := computeLineDiff(oldLines, newLines)
			var diffBuf strings.Builder
			for _, op := range ops {
				switch op.Kind {
				case DiffEqual:
					diffBuf.WriteString(fmt.Sprintf("  %s\n", op.Line))
				case DiffDelete:
					diffBuf.WriteString(fmt.Sprintf("- %s\n", op.Line))
				case DiffInsert:
					diffBuf.WriteString(fmt.Sprintf("+ %s\n", op.Line))
				}
			}
			t.Errorf("content mismatch in %s:\n--- want\n+++ got\n%s", rel, diffBuf.String())
		}
	}
}

func TestRunExtractsCodeFences(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "toc")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	gitMd := "---\nuchi: v1\n---\n## Environment\n\n```sh {schema=env}\nGIT_PAGER=vim\n```\n\n## alias\n\n```sh {schema=alias}\nalias g=\"git\"\n```\n\n```sh {schema=profile}\numask 022\n```\n\n```sh {schema=rc}\nset -o vi\n```\n\n```sh {schema=function}\ngit_clean() { git clean -df; }\n```\n\n```sh\n# unannotated code block ignored\necho test\n```\n"
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

func TestRunColorOutput(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias a=\"app\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "doc1.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("check and init output ANSI escape codes when color=always", func(t *testing.T) {
		var buf bytes.Buffer
		cfgCheck := &config.Config{
			ConfigFile: filepath.Join(dir, ".uchi.yaml"),
			InputDir:   inputDir,
			OutputDir:  outputDir,
			Color:      "always",
			Command:    "check",
		}
		if err := Run(cfgCheck, &buf); err != nil {
			t.Fatalf("Run(check, color=always) error = %v", err)
		}
		if !strings.Contains(buf.String(), "\033[32mfound\033[0m") {
			t.Errorf("check output missing green 'found': %q", buf.String())
		}

		buf.Reset()
		cfgInit := &config.Config{
			ConfigFile: filepath.Join(dir, "init.yaml"),
			Color:      "always",
			Command:    "init",
		}
		if err := Run(cfgInit, &buf); err != nil {
			t.Fatalf("Run(init, color=always) error = %v", err)
		}
		if !strings.Contains(buf.String(), "\033[32mCreated\033[0m") {
			t.Errorf("init output missing green 'Created': %q", buf.String())
		}
	})

	t.Run("diff output ANSI escape codes when color=always", func(t *testing.T) {
		var buf bytes.Buffer
		cfgDiff := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Color:     "always",
			Command:   "diff",
		}
		if err := Run(cfgDiff, &buf); err != nil {
			t.Fatalf("Run(diff, color=always) error = %v", err)
		}
		outStr := buf.String()
		if !strings.Contains(outStr, "\033[32m[+]\033[0m") {
			t.Errorf("diff output missing green '[+]': %q", outStr)
		}
		if !strings.Contains(outStr, "\033[32m+ alias a=\"app\"\033[0m") {
			t.Errorf("diff output missing green insertion line: %q", outStr)
		}
	})

	t.Run("no ANSI escape codes when color=never", func(t *testing.T) {
		var buf bytes.Buffer
		cfgCheck := &config.Config{
			ConfigFile: filepath.Join(dir, ".uchi.yaml"),
			InputDir:   inputDir,
			OutputDir:  outputDir,
			Color:      "never",
			Command:    "check",
		}
		if err := Run(cfgCheck, &buf); err != nil {
			t.Fatalf("Run(check, color=never) error = %v", err)
		}
		if strings.Contains(buf.String(), "\033[") {
			t.Errorf("check output with color=never contains ANSI codes: %q", buf.String())
		}

		buf.Reset()
		cfgDiff := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Color:     "never",
			Command:   "diff",
		}
		if err := Run(cfgDiff, &buf); err != nil {
			t.Fatalf("Run(diff, color=never) error = %v", err)
		}
		if strings.Contains(buf.String(), "\033[") {
			t.Errorf("diff output with color=never contains ANSI codes: %q", buf.String())
		}
	})

	t.Run("diff with quiet and color=always shows color diff lines without noise", func(t *testing.T) {
		var buf bytes.Buffer
		cfgDiff := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Color:     "always",
			Quiet:     true,
			Command:   "diff",
		}
		if err := Run(cfgDiff, &buf); err != nil {
			t.Fatalf("Run(diff, quiet, color=always) error = %v", err)
		}
		outStr := buf.String()
		if !strings.Contains(outStr, "\033[32m[+]\033[0m") {
			t.Errorf("quiet diff output missing colorized diff header: %q", outStr)
		}
	})
}

func TestRunVerboseAndQuietModes(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias a=\"app\"\n```\n```sh\n# unannotated fence\necho hello\n```\n```sh {schema=env target=fish}\nFISH_VAR=1\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "doc1.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}

	doc2 := "# Not uchi v1\n```sh {schema=alias}\nalias x=\"y\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "doc2.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("check mode with quiet suppresses non-essential output", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
			Quiet:     true,
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run(check, quiet) error = %v", err)
		}
		if output.Len() != 0 {
			t.Errorf("expected empty output for quiet check, got %q", output.String())
		}
	})

	t.Run("check mode with verbose shows detailed logs", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
			Verbose:   true,
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run(check, verbose) error = %v", err)
		}
		outStr := output.String()
		if !strings.Contains(outStr, "[verbose] Running check for input directory") {
			t.Errorf("missing verbose check start message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "doc1.md (version v1)") {
			t.Errorf("missing analyzed document message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "missing or unsupported uchi version") {
			t.Errorf("missing skipped non-v1 document message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "unannotated") {
			t.Errorf("missing unannotated fence skip message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "Options:") {
			t.Errorf("verbose check should also include options summary, output:\n%s", outStr)
		}
	})

	t.Run("gen dry-run with verbose shows planned targets and skip reasons", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Shell:     "bash",
			Command:   "gen",
			DryRun:    true,
			Verbose:   true,
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run(gen dry-run, verbose) error = %v", err)
		}
		outStr := output.String()

		// Verify verbose details
		if !strings.Contains(outStr, "[verbose] Building generation plan") {
			t.Errorf("missing verbose build plan start message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "missing or unsupported uchi version") {
			t.Errorf("missing non-v1 document skip reason, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "unannotated") {
			t.Errorf("missing unannotated fence skip reason, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "target shell \"fish\" does not match selected shell \"bash\"") {
			t.Errorf("missing shell mismatch skip reason, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "[verbose] Executing dry-run for generation plan") {
			t.Errorf("missing dry-run start message, output:\n%s", outStr)
		}

		// Verify dry-run output target paths
		wantPart := filepath.Join(outputDir, "parts", "doc1", "alias")
		wantMerged := filepath.Join(outputDir, "alias")
		if !strings.Contains(outStr, wantPart) || !strings.Contains(outStr, wantMerged) {
			t.Errorf("missing dry-run target paths, output:\n%s", outStr)
		}
	})

	t.Run("gen dry-run with quiet outputs target paths without extra noise", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Shell:     "bash",
			Command:   "gen",
			DryRun:    true,
			Quiet:     true,
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run(gen dry-run, quiet) error = %v", err)
		}
		outStr := output.String()
		if strings.Contains(outStr, "[verbose]") || strings.Contains(outStr, "Config file") {
			t.Errorf("quiet dry-run contains verbose/info noise, output:\n%s", outStr)
		}

		wantPart := filepath.Join(outputDir, "parts", "doc1", "alias")
		wantMerged := filepath.Join(outputDir, "alias")
		wantOutput := wantPart + "\n" + wantMerged + "\n"
		if outStr != wantOutput {
			t.Errorf("got dry-run quiet output %q, want %q", outStr, wantOutput)
		}
	})

	t.Run("new and init subcommands respect quiet mode", func(t *testing.T) {
		newDir := t.TempDir()
		var outNew bytes.Buffer
		cfgNew := &config.Config{
			InputDir:   newDir,
			Command:    "new",
			CommandArg: "test_doc",
			Quiet:      true,
		}
		if err := Run(cfgNew, &outNew); err != nil {
			t.Fatalf("Run(new, quiet) error = %v", err)
		}
		if outNew.Len() != 0 {
			t.Errorf("quiet new output = %q, want empty", outNew.String())
		}

		initDir := t.TempDir()
		var outInit bytes.Buffer
		cfgInit := &config.Config{
			ConfigFile: filepath.Join(initDir, ".uchi.yaml"),
			Command:    "init",
			Quiet:      true,
		}
		if err := Run(cfgInit, &outInit); err != nil {
			t.Fatalf("Run(init, quiet) error = %v", err)
		}
		if outInit.Len() != 0 {
			t.Errorf("quiet init output = %q, want empty", outInit.String())
		}
	})

	t.Run("diff mode with verbose includes log details", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "diff",
			Verbose:   true,
		}
		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run(diff, verbose) error = %v", err)
		}
		outStr := output.String()
		if !strings.Contains(outStr, "[verbose] Executing diff comparison") {
			t.Errorf("missing verbose diff message, output:\n%s", outStr)
		}
		if !strings.Contains(outStr, "[+]") {
			t.Errorf("missing diff output lines, output:\n%s", outStr)
		}
	})

	t.Run("specifying both verbose and quiet returns error", func(t *testing.T) {
		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
			Verbose:   true,
			Quiet:     true,
		}
		err := Run(cfg, &output)
		if err == nil {
			t.Fatal("expected error for specifying both verbose and quiet, got nil")
		}
		if !strings.Contains(err.Error(), "cannot specify both --verbose and --quiet") {
			t.Errorf("error %q should mention both options cannot be specified", err.Error())
		}
	})
}

func TestRunSchemaAdapters(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("converts typeset -x to export for bash target", func(t *testing.T) {
		doc := "---\nuchi: v1\n---\n\n```sh {schema=env target=bash}\ntypeset -x MY_ENV=123\n```\n"
		mdPath := filepath.Join(inputDir, "env.md")
		if err := os.WriteFile(mdPath, []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(mdPath)

		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Shell:     "bash",
			Command:   "gen",
		}
		if err := Run(cfg, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(outputDir, "env"))
		if err != nil {
			t.Fatalf("failed to read env output: %v", err)
		}
		if !strings.Contains(string(got), "export MY_ENV=123") {
			t.Errorf("got %q, want export MY_ENV=123", string(got))
		}
		_ = os.RemoveAll(outputDir)
	})

	t.Run("rejects zsh global alias when target is bash", func(t *testing.T) {
		doc := "---\nuchi: v1\n---\n\n```sh {schema=alias target=bash}\nalias -g G='| grep'\n```\n"
		mdPath := filepath.Join(inputDir, "alias.md")
		if err := os.WriteFile(mdPath, []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(mdPath)

		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Shell:     "bash",
			Command:   "gen",
		}
		err := Run(cfg, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for zsh global alias on bash target, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
		}
		if diag.Path != mdPath {
			t.Errorf("diag.Path = %q, want %q", diag.Path, mdPath)
		}
		if diag.Line != 6 {
			t.Errorf("diag.Line = %d, want 6", diag.Line)
		}
		if !strings.Contains(err.Error(), "Zsh-specific alias option") {
			t.Errorf("err = %q, want Zsh-specific alias option error", err.Error())
		}
	})

	t.Run("rejects shopt when target is zsh", func(t *testing.T) {
		doc := "---\nuchi: v1\n---\n\n```sh {schema=rc target=zsh}\nshopt -s globstar\n```\n"
		mdPath := filepath.Join(inputDir, "rc.md")
		if err := os.WriteFile(mdPath, []byte(doc), 0644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(mdPath)

		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Shell:     "zsh",
			Command:   "gen",
		}
		err := Run(cfg, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for shopt on zsh target, got nil")
		}

		if !strings.Contains(err.Error(), "Bash option command 'shopt'") {
			t.Errorf("err = %q, want shopt error", err.Error())
		}
	})
}

func TestRunShellFilteringAndValidation(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	docContent := `---
uchi: v1
---

` + "```bash {schema=alias}\n" +
		`alias g="git"` + "\n" +
		"```\n\n" +
		"```bash {schema=alias target=bash}\n" +
		`alias b="bash_only"` + "\n" +
		"```\n\n" +
		"```zsh {schema=alias target=zsh}\n" +
		`alias z="zsh_only"` + "\n" +
		"```\n\n" +
		"```sh {schema=alias target=\"bash,zsh\"}\n" +
		`alias bz="shared_bash_zsh"` + "\n" +
		"```\n"

	mdPath := filepath.Join(inputDir, "shell.md")
	if err := os.WriteFile(mdPath, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Test Shell = "bash"
	{
		outDirBash := filepath.Join(tmpDir, "out_bash")
		cfg := &config.Config{
			InputDir:    inputDir,
			OutputDir:   outDirBash,
			Shell:       "bash",
			AutoComment: true,
			Command:     "gen",
		}
		var buf bytes.Buffer
		if err := Run(cfg, &buf); err != nil {
			t.Fatalf("Run(bash) unexpected error: %v", err)
		}

		mergedData, err := os.ReadFile(filepath.Join(outDirBash, "alias"))
		if err != nil {
			t.Fatalf("failed to read bash merged output: %v", err)
		}
		mergedStr := string(mergedData)

		if !strings.Contains(mergedStr, `alias g="git"`) {
			t.Errorf("bash output missing default target block: %s", mergedStr)
		}
		if !strings.Contains(mergedStr, `alias b="bash_only"`) {
			t.Errorf("bash output missing target=bash block: %s", mergedStr)
		}
		if !strings.Contains(mergedStr, `alias bz="shared_bash_zsh"`) {
			t.Errorf("bash output missing target=bash,zsh block: %s", mergedStr)
		}
		if strings.Contains(mergedStr, `alias z="zsh_only"`) {
			t.Errorf("bash output unexpectedly contains target=zsh block: %s", mergedStr)
		}
	}

	// 2. Test Shell = "zsh"
	{
		outDirZsh := filepath.Join(tmpDir, "out_zsh")
		cfg := &config.Config{
			InputDir:    inputDir,
			OutputDir:   outDirZsh,
			Shell:       "zsh",
			AutoComment: true,
			Command:     "gen",
		}
		var buf bytes.Buffer
		if err := Run(cfg, &buf); err != nil {
			t.Fatalf("Run(zsh) unexpected error: %v", err)
		}

		mergedData, err := os.ReadFile(filepath.Join(outDirZsh, "alias"))
		if err != nil {
			t.Fatalf("failed to read zsh merged output: %v", err)
		}
		mergedStr := string(mergedData)

		if !strings.Contains(mergedStr, `alias g="git"`) {
			t.Errorf("zsh output missing default target block: %s", mergedStr)
		}
		if !strings.Contains(mergedStr, `alias z="zsh_only"`) {
			t.Errorf("zsh output missing target=zsh block: %s", mergedStr)
		}
		if !strings.Contains(mergedStr, `alias bz="shared_bash_zsh"`) {
			t.Errorf("zsh output missing target=bash,zsh block: %s", mergedStr)
		}
		if strings.Contains(mergedStr, `alias b="bash_only"`) {
			t.Errorf("zsh output unexpectedly contains target=bash block: %s", mergedStr)
		}
	}

	// 3. Test Unknown Shell fails before generation
	{
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: filepath.Join(tmpDir, "out_unknown"),
			Shell:     "unknown_shell",
			Command:   "gen",
		}
		var buf bytes.Buffer
		err := Run(cfg, &buf)
		if err == nil {
			t.Fatal("expected error for unknown shell, got nil")
		}
		if !strings.Contains(err.Error(), "unknown shell") {
			t.Errorf("error %q should contain 'unknown shell'", err.Error())
		}
	}

	// 4. Test Unselected shell resulting in empty outputs
	{
		fishInputDir := filepath.Join(tmpDir, "input_fish")
		if err := os.MkdirAll(fishInputDir, 0755); err != nil {
			t.Fatal(err)
		}
		fishDoc := `---
uchi: v1
---

` + "```bash {schema=alias target=bash}\n" +
			`alias b="bash_only"` + "\n" +
			"```\n"
		if err := os.WriteFile(filepath.Join(fishInputDir, "shell.md"), []byte(fishDoc), 0644); err != nil {
			t.Fatal(err)
		}

		outDirFish := filepath.Join(tmpDir, "out_fish")
		cfg := &config.Config{
			InputDir:  fishInputDir,
			OutputDir: outDirFish,
			Shell:     "fish",
			Command:   "gen",
		}
		var buf bytes.Buffer
		if err := Run(cfg, &buf); err != nil {
			t.Fatalf("Run(fish) unexpected error: %v", err)
		}

		// Verify no alias target file was generated since no blocks matched fish
		aliasPath := filepath.Join(outDirFish, "alias")
		if _, err := os.Stat(aliasPath); !os.IsNotExist(err) {
			t.Errorf("file %s was generated unexpectedly for fish target", aliasPath)
		}
	}
}

func TestRunShellPortabilityInspection(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("rejects non-portable bash syntax on portable targets with line number and diagnostic details", func(t *testing.T) {
		path := filepath.Join(inputDir, "non_portable.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias}\nalias ok=\"ls -la\"\nif [[ $a == $b ]]; then\n  echo hi\nfi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "check"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error on non-portable bash syntax in portable target, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
		}
		if diag.Path != path {
			t.Errorf("diag.Path = %q, want %q", diag.Path, path)
		}
		if diag.Line != 7 {
			t.Errorf("diag.Line = %d, want 7", diag.Line)
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, `non-portable Bash syntax "[[ ... ]]" detected`) {
			t.Errorf("errMsg %q should contain construct diagnostic message", errMsg)
		}
		if !strings.Contains(errMsg, `confidence: high`) {
			t.Errorf("errMsg %q should contain confidence", errMsg)
		}
		if !strings.Contains(errMsg, `suggestion: use POSIX standard '[' or 'test'`) {
			t.Errorf("errMsg %q should contain suggestion", errMsg)
		}
		_ = os.Remove(path)
	})

	t.Run("allows non-portable bash syntax when target is explicitly bash", func(t *testing.T) {
		path := filepath.Join(inputDir, "bash_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias target=bash}\nalias ok=\"ls -la\"\nif [[ $a == $b ]]; then\n  arr=(1 2)\n  diff <(date) >(logger)\nfi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run() unexpected error = %v", err)
		}

		partsAlias, err := os.ReadFile(filepath.Join(outputDir, "parts", "bash_target", "alias"))
		if err != nil {
			t.Fatalf("failed to read parts file: %v", err)
		}
		if !strings.Contains(string(partsAlias), "if [[ $a == $b ]]; then") {
			t.Errorf("partsAlias %q should contain generated bash code", string(partsAlias))
		}
		_ = os.Remove(path)
		_ = os.RemoveAll(outputDir)
	})
}

func TestRunGenDiff(t *testing.T) {
	t.Run("displays diff for new output files without creating output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "non_existent_dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		docA := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias a=\"ls\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "a.md"), []byte(docA), 0644); err != nil {
			t.Fatal(err)
		}

		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:    inputDir,
			OutputDir:   outputDir,
			AutoComment: true,
			Command:     "gen",
			Diff:        true,
		}

		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		outStr := output.String()
		partPath := filepath.Join(outputDir, "parts", "a", "alias")
		mergedPath := filepath.Join(outputDir, "alias")

		if !strings.Contains(outStr, fmt.Sprintf("[+] %s (kind: part, status: new)", partPath)) {
			t.Errorf("output missing part header for %s. Output:\n%s", partPath, outStr)
		}
		if !strings.Contains(outStr, fmt.Sprintf("[+] %s (kind: merged, status: new)", mergedPath)) {
			t.Errorf("output missing merged header for %s. Output:\n%s", mergedPath, outStr)
		}
		if !strings.Contains(outStr, "+ # a\n+ alias a=\"ls\"") {
			t.Errorf("output missing line diffs. Output:\n%s", outStr)
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist after diff mode, got err = %v", outputDir, err)
		}
	})

	t.Run("reports unchanged, modified, new, and deleted files using manifest", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		file1 := filepath.Join(inputDir, "f1.md")
		doc1 := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias f1=\"f1\"\n```\n```sh {schema=env}\nF1_ENV=1\n```\n"
		if err := os.WriteFile(file1, []byte(doc1), 0644); err != nil {
			t.Fatal(err)
		}

		cfgGen := &config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}
		cfgDiff := &config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "diff"}

		// Initial gen writes output files and .uchi-manifest.json
		if err := Run(cfgGen, &bytes.Buffer{}); err != nil {
			t.Fatalf("initial Run(gen) error = %v", err)
		}

		// Check manifest existence
		manifestPath := filepath.Join(outputDir, ".uchi-manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			t.Fatalf("expected manifest file %s to exist, got %v", manifestPath, err)
		}

		// 1) Running diff right after gen should report all files as unchanged
		var bufUnchanged bytes.Buffer
		if err := Run(cfgDiff, &bufUnchanged); err != nil {
			t.Fatalf("Run(diff) error = %v", err)
		}

		partAliasPath := filepath.Join(outputDir, "parts", "f1", "alias")
		partEnvPath := filepath.Join(outputDir, "parts", "f1", "env")
		mergedAliasPath := filepath.Join(outputDir, "alias")
		mergedEnvPath := filepath.Join(outputDir, "env")

		outUnchanged := bufUnchanged.String()
		for _, p := range []string{partAliasPath, partEnvPath, mergedAliasPath, mergedEnvPath} {
			wantLine := fmt.Sprintf("[=] %s (kind:", p)
			if !strings.Contains(outUnchanged, wantLine) {
				t.Errorf("expected %s to be reported as unchanged with %q, got output:\n%s", p, wantLine, outUnchanged)
			}
		}

		// 2) Modify f1.md: change alias content, remove env fence, add function fence
		doc1Mod := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias f1=\"f1_modified\"\n```\n```sh {schema=function}\nf1_fn() { echo hi; }\n```\n"
		if err := os.WriteFile(file1, []byte(doc1Mod), 0644); err != nil {
			t.Fatal(err)
		}

		var bufMod bytes.Buffer
		if err := Run(cfgDiff, &bufMod); err != nil {
			t.Fatalf("Run(diff) modified error = %v", err)
		}

		outMod := bufMod.String()

		partFnPath := filepath.Join(outputDir, "parts", "f1", "function")
		mergedFnPath := filepath.Join(outputDir, "function")

		// Check statuses in diff output
		if !strings.Contains(outMod, fmt.Sprintf("[~] %s (kind: part, status: modified)", partAliasPath)) {
			t.Errorf("part alias diff header missing in:\n%s", outMod)
		}
		if !strings.Contains(outMod, "- alias f1=\"f1\"") || !strings.Contains(outMod, "+ alias f1=\"f1_modified\"") {
			t.Errorf("part alias line diff missing in:\n%s", outMod)
		}
		if !strings.Contains(outMod, fmt.Sprintf("[+] %s (kind: part, status: new)", partFnPath)) {
			t.Errorf("part function status new missing in:\n%s", outMod)
		}
		if !strings.Contains(outMod, fmt.Sprintf("[+] %s (kind: merged, status: new)", mergedFnPath)) {
			t.Errorf("merged function status new missing in:\n%s", outMod)
		}
		if !strings.Contains(outMod, fmt.Sprintf("[-] %s (kind: part, status: deleted)", partEnvPath)) {
			t.Errorf("part env status deleted missing in:\n%s", outMod)
		}

		// Verify that diff mode did NOT modify output files or manifest
		envContent, err := os.ReadFile(partEnvPath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", partEnvPath, err)
		}
		if string(envContent) != "# f1\nF1_ENV=1\n" {
			t.Errorf("diff mode modified file %s, got content %q", partEnvPath, string(envContent))
		}
	})

	t.Run("handles empty file content and newline normalization", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		docEmpty := "---\nuchi: v1\n---\n```sh {schema=alias}\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "empty.md"), []byte(docEmpty), 0644); err != nil {
			t.Fatal(err)
		}

		cfgGen := &config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}
		cfgDiff := &config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "diff"}

		// Generate initial empty files
		if err := Run(cfgGen, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}

		// Diff should show unchanged for empty files
		var bufDiff bytes.Buffer
		if err := Run(cfgDiff, &bufDiff); err != nil {
			t.Fatal(err)
		}

		out := bufDiff.String()
		if !strings.Contains(out, "status: unchanged") {
			t.Errorf("expected unchanged status for empty files, got:\n%s", out)
		}
	})
}

func TestRunGenDryRun(t *testing.T) {
	t.Run("displays planned output targets in deterministic order without creating files or output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "non_existent_dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		docA := "---\nuchi: v1\n---\n```sh {schema=env}\nENV_A=1\n```\n```sh {schema=alias}\nalias a=\"a\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "a.md"), []byte(docA), 0644); err != nil {
			t.Fatal(err)
		}

		docB := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias b=\"b\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "b.md"), []byte(docB), 0644); err != nil {
			t.Fatal(err)
		}

		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "gen",
			DryRun:    true,
		}

		if err := Run(cfg, &output); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		wantTargets := []string{
			filepath.Join(outputDir, "parts", "a", "env"),
			filepath.Join(outputDir, "parts", "a", "alias"),
			filepath.Join(outputDir, "parts", "b", "alias"),
			filepath.Join(outputDir, "env"),
			filepath.Join(outputDir, "alias"),
		}
		wantOutput := strings.Join(wantTargets, "\n") + "\n"

		if output.String() != wantOutput {
			t.Errorf("output = %q, want %q", output.String(), wantOutput)
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist after dry-run, got err = %v", outputDir, err)
		}
	})

	t.Run("fails on invalid input with same diagnostic error as normal gen", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		badFile := filepath.Join(inputDir, "bad.md")
		if err := os.WriteFile(badFile, []byte("---\nuchi: v1\n---\n```sh {schema=invalid_schema}\n```\n"), 0644); err != nil {
			t.Fatal(err)
		}

		var outputDry bytes.Buffer
		dryErr := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen", DryRun: true}, &outputDry)
		genErr := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen", DryRun: false}, &bytes.Buffer{})

		if dryErr == nil || genErr == nil {
			t.Fatalf("expected both dry-run and normal gen to fail, got dryErr=%v, genErr=%v", dryErr, genErr)
		}
		if dryErr.Error() != genErr.Error() {
			t.Errorf("dryErr = %q, genErr = %q; expected identical diagnostic errors", dryErr.Error(), genErr.Error())
		}
		if outputDry.Len() > 0 {
			t.Errorf("expected empty stdout on error, got %q", outputDry.String())
		}
		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist on failure, got err = %v", outputDir, err)
		}
	})
}

func TestRunDeterministicOutputWithVariedCreationOrder(t *testing.T) {
	dir := t.TempDir()

	fileSpecs := map[string]string{
		"z_last.md":    "---\nuchi: v1\n---\n```sh {schema=alias}\nalias z=\"zoxide\"\n```\n```sh {schema=env}\nZ_VAR=1\n```\n",
		"a_first.md":   "---\nuchi: v1\n---\n```sh {schema=alias}\nalias a=\"ls -a\"\n```\n",
		"sub/m_mid.md": "---\nuchi: v1\n---\n```sh {schema=alias}\nalias m=\"make\"\n```\n```sh {schema=env}\nM_VAR=2\n```\n",
	}

	// Set 1: Create in order z_last.md -> sub/m_mid.md -> a_first.md
	input1 := filepath.Join(dir, "input1")
	dist1 := filepath.Join(dir, "dist1")
	for _, name := range []string{"z_last.md", "sub/m_mid.md", "a_first.md"} {
		p := filepath.Join(input1, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(fileSpecs[name]), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Set 2: Create in order a_first.md -> z_last.md -> sub/m_mid.md
	input2 := filepath.Join(dir, "input2")
	dist2 := filepath.Join(dir, "dist2")
	for _, name := range []string{"a_first.md", "z_last.md", "sub/m_mid.md"} {
		p := filepath.Join(input2, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(fileSpecs[name]), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := Run(&config.Config{InputDir: input1, OutputDir: dist1, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(input1) error = %v", err)
	}
	if err := Run(&config.Config{InputDir: input2, OutputDir: dist2, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(input2) error = %v", err)
	}

	// Compare generated file structures and byte contents between dist1 and dist2
	var files1, files2 []string
	_ = filepath.Walk(dist1, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(dist1, path)
			files1 = append(files1, filepath.ToSlash(rel))
		}
		return nil
	})
	_ = filepath.Walk(dist2, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(dist2, path)
			files2 = append(files2, filepath.ToSlash(rel))
		}
		return nil
	})

	if len(files1) != len(files2) {
		t.Fatalf("files1 count = %d, files2 count = %d", len(files1), len(files2))
	}

	for i, rel := range files1 {
		if rel != files2[i] {
			t.Errorf("file[%d] mismatch: dist1 has %q, dist2 has %q", i, rel, files2[i])
			continue
		}
		c1, err1 := os.ReadFile(filepath.Join(dist1, rel))
		c2, err2 := os.ReadFile(filepath.Join(dist2, rel))
		if err1 != nil || err2 != nil {
			t.Fatalf("failed to read generated file %s: %v, %v", rel, err1, err2)
		}
		if !bytes.Equal(c1, c2) {
			t.Errorf("content mismatch for %s:\ndist1:\n%s\ndist2:\n%s", rel, string(c1), string(c2))
		}
	}
}

func TestRunDuplicateContentRetentionAndFenceOrder(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Doc A has duplicate alias blocks inside itself as well as same alias as Doc B
	docA := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias g=\"git\"\n```\n\n```sh {schema=env}\nEDITOR=vim\n```\n\n```sh {schema=alias}\nalias g=\"git\"\nalias status=\"git status\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "a.md"), []byte(docA), 0644); err != nil {
		t.Fatal(err)
	}

	docB := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias g=\"git\"\n```\n"
	if err := os.WriteFile(filepath.Join(inputDir, "b.md"), []byte(docB), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Part A alias should retain intra-document duplicate in line order
	partA, err := os.ReadFile(filepath.Join(outputDir, "parts", "a", "alias"))
	if err != nil {
		t.Fatalf("failed to read parts/a/alias: %v", err)
	}
	wantPartA := "# a\nalias g=\"git\"\nalias g=\"git\"\nalias status=\"git status\"\n"
	if string(partA) != wantPartA {
		t.Errorf("partA alias = %q, want %q", string(partA), wantPartA)
	}

	// Merged alias should retain all duplicates in document -> fence order
	mergedAlias, err := os.ReadFile(filepath.Join(outputDir, "alias"))
	if err != nil {
		t.Fatalf("failed to read merged alias: %v", err)
	}
	wantMergedAlias := "# a\nalias g=\"git\"\nalias g=\"git\"\nalias status=\"git status\"\n\n# b\nalias g=\"git\"\n"
	if string(mergedAlias) != wantMergedAlias {
		t.Errorf("merged alias = %q, want %q", string(mergedAlias), wantMergedAlias)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	t.Run("successfully writes new file and preserves default permission", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "out", "test.txt")

		err := writeFileAtomic(target, []byte("hello atomic\n"), 0644)
		if err != nil {
			t.Fatalf("writeFileAtomic failed: %v", err)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(got) != "hello atomic\n" {
			t.Errorf("content = %q, want %q", string(got), "hello atomic\n")
		}

		entries, err := os.ReadDir(filepath.Dir(target))
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), ".tmp-") {
				t.Errorf("found leftover temp file: %s", entry.Name())
			}
		}
	})

	t.Run("preserves existing file permissions", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "test.txt")

		if err := os.WriteFile(target, []byte("initial\n"), 0600); err != nil {
			t.Fatal(err)
		}

		err := writeFileAtomic(target, []byte("updated\n"), 0644)
		if err != nil {
			t.Fatalf("writeFileAtomic failed: %v", err)
		}

		fi, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0600 {
			t.Errorf("mode = %o, want %o", fi.Mode().Perm(), 0600)
		}

		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "updated\n" {
			t.Errorf("content = %q, want %q", string(got), "updated\n")
		}
	})

	t.Run("preserves existing content on atomic write failure and cleans temp file", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatal(err)
		}

		gitMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias g=\"git\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "git.md"), []byte(gitMd), 0644); err != nil {
			t.Fatal(err)
		}

		aliasPath := filepath.Join(outputDir, "alias")
		if err := os.WriteFile(aliasPath, []byte("PREVIOUS ALIAS CONTENT\n"), 0644); err != nil {
			t.Fatal(err)
		}

		partAliasPath := filepath.Join(outputDir, "parts", "git", "alias")
		if err := os.MkdirAll(filepath.Dir(partAliasPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(partAliasPath, []byte("PREVIOUS PART CONTENT\n"), 0644); err != nil {
			t.Fatal(err)
		}

		origWriter := atomicFileWriter
		defer func() { atomicFileWriter = origWriter }()

		atomicFileWriter = func(path string, content []byte, perm os.FileMode) error {
			if strings.HasSuffix(path, "alias") && !strings.Contains(path, "parts") {
				return fmt.Errorf("failed to write file %s: simulated write error", path)
			}
			return origWriter(path, content, perm)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error from atomic writer failure, got nil")
		}
		if !strings.Contains(err.Error(), aliasPath) {
			t.Errorf("error %q should contain target file path %q", err.Error(), aliasPath)
		}

		gotMerged, err := os.ReadFile(aliasPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotMerged) != "PREVIOUS ALIAS CONTENT\n" {
			t.Errorf("merged alias content = %q, want %q", string(gotMerged), "PREVIOUS ALIAS CONTENT\n")
		}

		var tempFiles []string
		_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.Contains(info.Name(), ".tmp-") {
				tempFiles = append(tempFiles, path)
			}
			return nil
		})

		if len(tempFiles) > 0 {
			t.Errorf("found temp files after failure: %v", tempFiles)
		}
	})

	t.Run("returns error containing file path when target is invalid directory", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "is_a_dir")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatal(err)
		}

		err := writeFileAtomic(target, []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error when writing to directory target, got nil")
		}
		if !strings.Contains(err.Error(), target) {
			t.Errorf("error %q should contain target path %q", err.Error(), target)
		}
	})
}

func TestRunCheck(t *testing.T) {
	t.Run("default command when config file is present", func(t *testing.T) {
		var output bytes.Buffer
		dir := t.TempDir()
		cfg := &config.Config{
			ConfigFile:  "./.uchi.yaml",
			InputDir:    dir,
			OutputDir:   filepath.Join(dir, "dist"),
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
		if !strings.Contains(out, "input_dir: ") || !strings.Contains(out, "output_dir: ") || !strings.Contains(out, "template_dir: ./templates") || !strings.Contains(out, "auto_comment: true") {
			t.Errorf("out = %q, want all configuration values displayed", out)
		}
	})

	t.Run("check command when config file is not present", func(t *testing.T) {
		var output bytes.Buffer
		dir := t.TempDir()
		cfg := &config.Config{
			ConfigFile:  "",
			InputDir:    dir,
			OutputDir:   filepath.Join(dir, "dist"),
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

	t.Run("check fails on invalid frontmatter and does not create output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		badPath := filepath.Join(inputDir, "bad.md")
		badContent := "---\nuchi: [invalid yaml\n---\n"
		if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
			t.Fatal(err)
		}

		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
		}
		err := Run(cfg, &output)
		if err == nil {
			t.Fatal("expected error on invalid frontmatter, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != badPath {
			t.Errorf("diag.Path = %q, want %q", diag.Path, badPath)
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist, got err = %v", outputDir, err)
		}
	})

	t.Run("check fails on invalid code fence attributes and does not create output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		badPath := filepath.Join(inputDir, "bad_fence.md")
		badContent := "---\nuchi: v1\n---\n```sh {schema=env\nFOO=bar\n```\n"
		if err := os.WriteFile(badPath, []byte(badContent), 0644); err != nil {
			t.Fatal(err)
		}

		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
		}
		err := Run(cfg, &output)
		if err == nil {
			t.Fatal("expected error on invalid code fence attributes, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != badPath || diag.Line != 4 {
			t.Errorf("diag = %v, want path=%s line=4", diag, badPath)
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist, got err = %v", outputDir, err)
		}
	})

	t.Run("check aggregates multiple errors", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		file1 := filepath.Join(inputDir, "file1.md")
		content1 := "---\nuchi: [broken\n---\n"
		if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
			t.Fatal(err)
		}

		file2 := filepath.Join(inputDir, "file2.md")
		content2 := "---\nuchi: v1\n---\n```sh {schema=env\nBAR=baz\n```\n"
		if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
			t.Fatal(err)
		}

		var output bytes.Buffer
		cfg := &config.Config{
			InputDir:  inputDir,
			OutputDir: outputDir,
			Command:   "check",
		}
		err := Run(cfg, &output)
		if err == nil {
			t.Fatal("expected error on multiple invalid files, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, file1) {
			t.Errorf("error message %q should contain %q", errMsg, file1)
		}
		if !strings.Contains(errMsg, file2) {
			t.Errorf("error message %q should contain %q", errMsg, file2)
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist, got err = %v", outputDir, err)
		}
	})

	t.Run("gen and check produce consistent validation outcomes", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDirCheck := filepath.Join(dir, "dist_check")
		outputDirGen := filepath.Join(dir, "dist_gen")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		badFile := filepath.Join(inputDir, "bad.md")
		if err := os.WriteFile(badFile, []byte("---\nuchi: v1\n---\n```sh {schema=}\n```\n"), 0644); err != nil {
			t.Fatal(err)
		}

		checkErr := Run(&config.Config{InputDir: inputDir, OutputDir: outputDirCheck, Command: "check"}, &bytes.Buffer{})
		genErr := Run(&config.Config{InputDir: inputDir, OutputDir: outputDirGen, Command: "gen"}, &bytes.Buffer{})

		if checkErr == nil || genErr == nil {
			t.Fatalf("expected both check and gen to fail, got checkErr=%v, genErr=%v", checkErr, genErr)
		}
		if checkErr.Error() != genErr.Error() {
			t.Errorf("checkErr = %q, genErr = %q; expected identical errors", checkErr.Error(), genErr.Error())
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

func TestRunFrontmatterValidationAndFiltering(t *testing.T) {
	t.Run("skips files with comments or note containing uchi v1", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		commentMd := "---\n# uchi: v1\ntitle: comment test\n---\n```sh {schema=alias}\nalias c=\"comment\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "comment.md"), []byte(commentMd), 0644); err != nil {
			t.Fatal(err)
		}

		noteMd := "---\nnote: \"uchi: v1\"\n---\n```sh {schema=alias}\nalias n=\"note\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "note.md"), []byte(noteMd), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run() unexpected error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(outputDir, "parts", "comment", "alias")); !os.IsNotExist(err) {
			t.Errorf("expected comment/alias to not exist, got err = %v", err)
		}
		if _, err := os.Stat(filepath.Join(outputDir, "parts", "note", "alias")); !os.IsNotExist(err) {
			t.Errorf("expected note/alias to not exist, got err = %v", err)
		}
		if _, err := os.Stat(filepath.Join(outputDir, "alias")); !os.IsNotExist(err) {
			t.Errorf("expected merged alias to not exist, got err = %v", err)
		}
	})

	t.Run("returns diagnostic with filepath for broken frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		badMd := "---\nuchi: [broken\n---\n```sh {schema=alias}\nalias b=\"bad\"\n```\n"
		badPath := filepath.Join(inputDir, "bad.md")
		if err := os.WriteFile(badPath, []byte(badMd), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for broken frontmatter, got nil")
		}
		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic error, got %T (%v)", err, err)
		}
		if diag.Path != badPath {
			t.Errorf("diag.Path = %q, want %q", diag.Path, badPath)
		}
		if diag.Line != 1 {
			t.Errorf("diag.Line = %d, want 1", diag.Line)
		}
		if !strings.HasPrefix(err.Error(), badPath+":1:") {
			t.Errorf("error %q should have prefix %q", err.Error(), badPath+":1:")
		}
	})
}

func TestRunTargetAttributeValidationAndGeneration(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("successfully generates files for valid single and multiple targets", func(t *testing.T) {
		path := filepath.Join(inputDir, "targets.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias target=bash}\nalias b=\"bash\"\n```\n\n```sh {schema=alias target=\"zsh, fish\"}\nalias z=\"zsh\"\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run() unexpected error = %v", err)
		}

		partsAlias, err := os.ReadFile(filepath.Join(outputDir, "parts", "targets", "alias"))
		if err != nil {
			t.Fatalf("failed to read parts/targets/alias: %v", err)
		}
		wantPart := "# targets\nalias b=\"bash\"\nalias z=\"zsh\"\n"
		if string(partsAlias) != wantPart {
			t.Errorf("partsAlias = %q, want %q", string(partsAlias), wantPart)
		}
		_ = os.Remove(path)
		_ = os.RemoveAll(outputDir)
	})

	t.Run("fails check on unknown target shell", func(t *testing.T) {
		path := filepath.Join(inputDir, "bad_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias target=invalid_shell}\nalias x=\"y\"\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "check"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for unknown target shell, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
		}
		if !strings.Contains(err.Error(), `unknown target shell "invalid_shell"`) {
			t.Errorf("error %q should contain unknown target info", err.Error())
		}
		_ = os.Remove(path)
	})

	t.Run("fails gen on duplicate target shell in list", func(t *testing.T) {
		path := filepath.Join(inputDir, "dup_target.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias target=\"bash, bash\"}\nalias x=\"y\"\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for duplicate target shell, got nil")
		}

		if !strings.Contains(err.Error(), `duplicate target shell "bash"`) {
			t.Errorf("error %q should contain duplicate target info", err.Error())
		}
		_ = os.Remove(path)
	})
}

func TestRunSchemaAndAttributeValidation(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("fails on unknown schema and includes line number and valid schema list", func(t *testing.T) {
		path := filepath.Join(inputDir, "unknown_schema.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=unknown}\necho unknown\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for unknown schema, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
		}
		if diag.Path != path {
			t.Errorf("diag.Path = %q, want %q", diag.Path, path)
		}
		if diag.Line != 5 {
			t.Errorf("diag.Line = %d, want 5", diag.Line)
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, `unknown schema "unknown"`) {
			t.Errorf("err %q should contain unknown schema message", errMsg)
		}
		if !strings.Contains(errMsg, "valid schemas: alias, env, function, profile, rc") {
			t.Errorf("err %q should list valid schemas", errMsg)
		}
		_ = os.Remove(path)
	})

	t.Run("fails on unknown attribute and includes line number and allowed attributes list", func(t *testing.T) {
		path := filepath.Join(inputDir, "unknown_attr.md")
		content := "---\nuchi: v1\n---\n\n```sh {schema=alias invalid_attr=val}\nalias x=y\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "check"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for unknown attribute, got nil")
		}

		var diag *markdown.Diagnostic
		if !errors.As(err, &diag) {
			t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
		}
		if diag.Path != path {
			t.Errorf("diag.Path = %q, want %q", diag.Path, path)
		}
		if diag.Line != 5 {
			t.Errorf("diag.Line = %d, want 5", diag.Line)
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, `unknown attribute "invalid_attr"`) {
			t.Errorf("err %q should contain unknown attribute message", errMsg)
		}
		if !strings.Contains(errMsg, "allowed attributes: schema, target") {
			t.Errorf("err %q should list allowed attributes", errMsg)
		}
		_ = os.Remove(path)
	})

	t.Run("fails on missing schema attribute in annotated fence", func(t *testing.T) {
		path := filepath.Join(inputDir, "missing_schema.md")
		content := "---\nuchi: v1\n---\n\n```sh {lang=bash}\necho hi\n```\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for missing schema attribute, got nil")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "missing required 'schema' attribute") {
			t.Errorf("err %q should contain missing required schema message", errMsg)
		}
		if !strings.Contains(errMsg, "valid schemas: alias, env, function, profile, rc") {
			t.Errorf("err %q should list valid schemas", errMsg)
		}
		_ = os.Remove(path)
	})
}

func TestRunOutputPathSafety(t *testing.T) {
	t.Run("nested input creates parts in nested dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		nestedDir := filepath.Join(inputDir, "nested")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatal(err)
		}

		toolMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias tool=\"nested_tool\"\n```\n"
		if err := os.WriteFile(filepath.Join(nestedDir, "tool.md"), []byte(toolMd), 0644); err != nil {
			t.Fatal(err)
		}

		if err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, AutoComment: true, Command: "gen"}, &bytes.Buffer{}); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		partPath := filepath.Join(outputDir, "parts", "nested", "tool", "alias")
		content, err := os.ReadFile(partPath)
		if err != nil {
			t.Fatalf("failed to read nested parts file %s: %v", partPath, err)
		}
		want := "# nested/tool\nalias tool=\"nested_tool\"\n"
		if string(content) != want {
			t.Errorf("parts content = %q, want %q", string(content), want)
		}

		mergedPath := filepath.Join(outputDir, "alias")
		mergedContent, err := os.ReadFile(mergedPath)
		if err != nil {
			t.Fatalf("failed to read merged alias file: %v", err)
		}
		if string(mergedContent) != want {
			t.Errorf("merged content = %q, want %q", string(mergedContent), want)
		}
	})

	t.Run("rejects empty base name input file like .md without creating output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		emptyBaseMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias dot=\"dot\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, ".md"), []byte(emptyBaseMd), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for input file with empty base name, got nil")
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist, got err = %v", outputDir, err)
		}
	})

	t.Run("rejects path traversal with double dots in nested input without creating output dir", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		subDir := filepath.Join(inputDir, "..bad")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		badMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias bad=\"bad\"\n```\n"
		if err := os.WriteFile(filepath.Join(subDir, "file.md"), []byte(badMd), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error for path traversal input, got nil")
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist on validation failure, got err = %v", outputDir, err)
		}
	})

	t.Run("atomic failure ensures no partial output when one file fails path validation", func(t *testing.T) {
		dir := t.TempDir()
		inputDir := filepath.Join(dir, "input")
		outputDir := filepath.Join(dir, "dist")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			t.Fatal(err)
		}

		goodMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias good=\"good\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, "a_good.md"), []byte(goodMd), 0644); err != nil {
			t.Fatal(err)
		}

		badMd := "---\nuchi: v1\n---\n```sh {schema=alias}\nalias bad=\"bad\"\n```\n"
		if err := os.WriteFile(filepath.Join(inputDir, ".md"), []byte(badMd), 0644); err != nil {
			t.Fatal(err)
		}

		err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
		if err == nil {
			t.Fatal("expected error on invalid document stem, got nil")
		}

		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			t.Errorf("expected output directory %s to not exist, got err = %v", outputDir, err)
		}
	})
}

func TestBuildOutputPath(t *testing.T) {
	outDir := filepath.Join(os.TempDir(), "uchi_test_dist")

	validPath, err := buildOutputPath(outDir, "parts", "nested/tool", "alias")
	if err != nil {
		t.Fatalf("unexpected error for valid elements: %v", err)
	}
	wantPath := filepath.Join(outDir, "parts", "nested", "tool", "alias")
	if validPath != wantPath {
		t.Errorf("buildOutputPath() = %q, want %q", validPath, wantPath)
	}

	invalidCases := []struct {
		name     string
		elements []string
	}{
		{"dot dot segment", []string{"parts", "../outside", "alias"}},
		{"absolute path", []string{"/etc/passwd"}},
		{"empty element", []string{"parts", "", "alias"}},
		{"invalid base", []string{"parts", "..", "alias"}},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildOutputPath(outDir, tc.elements...)
			if err == nil {
				t.Errorf("expected error for case %q, got nil", tc.name)
			}
		})
	}
}

func TestRunDiagnosticPropagation(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "input")
	outputDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatal(err)
	}

	badFencePath := filepath.Join(inputDir, "bad_fence.md")
	badFenceContent := "---\nuchi: v1\n---\n\n```sh {schema=env\nFOO=bar\n```\n"
	if err := os.WriteFile(badFencePath, []byte(badFenceContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := Run(&config.Config{InputDir: inputDir, OutputDir: outputDir, Command: "gen"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for bad code fence attribute, got nil")
	}

	var diag *markdown.Diagnostic
	if !errors.As(err, &diag) {
		t.Fatalf("expected *markdown.Diagnostic, got %T (%v)", err, err)
	}
	if diag.Path != badFencePath {
		t.Errorf("diag.Path = %q, want %q", diag.Path, badFencePath)
	}
	if diag.Line != 5 {
		t.Errorf("diag.Line = %d, want 5", diag.Line)
	}
	wantPrefix := badFencePath + ":5:"
	if !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Errorf("error %q should start with %q", err.Error(), wantPrefix)
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
		InputDir:  ".",
		OutputDir: "../dist",
		Command:   "init",
	}
	if err := Run(cfg, &output); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(".uchi.yaml")
	if err != nil {
		t.Fatalf("failed to read .uchi.yaml: %v", err)
	}
	if !strings.Contains(string(content), "input_dir: .") {
		t.Errorf("content = %s, want input_dir: .", string(content))
	}
	if strings.Contains(string(content), "config_file") {
		t.Errorf("content contains config_file: %s", string(content))
	}
	if !strings.Contains(output.String(), "Created ") {
		t.Errorf("output = %q, want creation message", output.String())
	}
}

package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/officel/uchi/internal/config"
	"github.com/officel/uchi/internal/markdown"
)

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
		if !strings.Contains(errMsg, "allowed attributes: schema") {
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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	changeToTempDir(t)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "." || cfg.OutputDir != "../dist" {
		t.Fatalf("Load() = %+v, want default input and output directories", cfg)
	}
	if !cfg.AutoComment {
		t.Fatalf("Load() AutoComment = false, want true by default")
	}

	if _, err := os.Stat(".uchi.yaml"); !os.IsNotExist(err) {
		t.Errorf(".uchi.yaml was created by default, expected no file creation")
	}
}

func TestLoadInitCommand(t *testing.T) {
	changeToTempDir(t)

	cfg, err := Load([]string{"init"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Command != "init" {
		t.Errorf("cfg.Command = %q, want 'init'", cfg.Command)
	}

	if _, err := Load([]string{"init", "extra_arg"}); err == nil {
		t.Errorf("Load() error = nil, want error for extra arguments to init")
	}
}

func TestLoadGenAndCheckCommands(t *testing.T) {
	changeToTempDir(t)

	cfgGen, err := Load([]string{"gen"})
	if err != nil {
		t.Fatalf("Load(gen) error = %v", err)
	}
	if cfgGen.Command != "gen" {
		t.Errorf("cfgGen.Command = %q, want 'gen'", cfgGen.Command)
	}

	if _, err := Load([]string{"gen", "extra_arg"}); err == nil {
		t.Errorf("Load(gen extra_arg) error = nil, want error")
	}

	cfgCheck, err := Load([]string{"check"})
	if err != nil {
		t.Fatalf("Load(check) error = %v", err)
	}
	if cfgCheck.Command != "check" {
		t.Errorf("cfgCheck.Command = %q, want 'check'", cfgCheck.Command)
	}

	if _, err := Load([]string{"check", "extra_arg"}); err == nil {
		t.Errorf("Load(check extra_arg) error = nil, want error")
	}
}

func TestLoadUsesDotConfigFile(t *testing.T) {
	changeToTempDir(t)
	if err := os.MkdirAll(".config", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".config", ".uchi.yaml"), []byte("input_dir: yaml_in\noutput_dir: yaml_out\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "yaml_in" || cfg.OutputDir != "yaml_out" {
		t.Errorf("Load() = %+v, want YAML values", cfg)
	}
	if _, err := os.Stat(".uchi.yaml"); !os.IsNotExist(err) {
		t.Error(".uchi.yaml was created despite .config/.uchi.yaml existing")
	}
}

func TestLoadFlagsOverrideFileAndSupportTemplateDirectory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("input_dir: yaml_in\noutput_dir: yaml_out\ntemplate_dir: yaml_templates\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load([]string{"-c", configPath, "--output", "flag_out", "-t", "flag_templates"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.InputDir != "yaml_in" || cfg.OutputDir != "flag_out" || cfg.TemplateDir != "flag_templates" {
		t.Errorf("Load() = %+v, want YAML input plus flag overrides", cfg)
	}
}

func TestLoadValidatesFlagSyntaxAndNewCommand(t *testing.T) {
	for _, args := range [][]string{{"-input", "dir"}, {"--i", "dir"}, {"-config", "file.yaml"}, {"--c", "file.yaml"}, {"--a"}, {"-auto-comment"}} {
		if _, err := Load(args); err == nil {
			t.Errorf("Load(%v) error = nil, want syntax error", args)
		}
	}

	changeToTempDir(t)
	cfg, err := Load([]string{"-i", "custom_toc", "--auto-comment", "new", "git"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Command != "new" || cfg.CommandArg != "git" || cfg.InputDir != "custom_toc" || !cfg.AutoComment {
		t.Errorf("Load() = %+v, want parsed new command with auto-comment", cfg)
	}
}

func TestLoadAutoComment(t *testing.T) {
	changeToTempDir(t)
	if err := os.WriteFile(".uchi.yaml", []byte("auto_comment: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfgFile, err := Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfgFile.AutoComment {
		t.Errorf("AutoComment = true, want false from YAML file")
	}

	cfgFlagOverride, err := Load([]string{"--auto-comment"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfgFlagOverride.AutoComment {
		t.Errorf("AutoComment = false, want true from CLI flag override")
	}

	cfgFlagDisable, err := Load([]string{"--auto-comment=false"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfgFlagDisable.AutoComment {
		t.Errorf("AutoComment = true, want false from CLI flag --auto-comment=false")
	}
}

func changeToTempDir(t *testing.T) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaultsAndAutoCreate(t *testing.T) {
	// Change working directory to a clean temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InputDir != "./toc" {
		t.Errorf("expected InputDir to be './toc', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "./dist" {
		t.Errorf("expected OutputDir to be './dist', got %s", cfg.OutputDir)
	}

	// Verify ./.uchi.yaml was created when neither existed
	createdConfigPath := "./.uchi.yaml"
	if _, err := os.Stat(createdConfigPath); os.IsNotExist(err) {
		t.Errorf("expected %s to be created, but it does not exist", createdConfigPath)
	}
}

func TestLoadConfigFromDotConfigLocation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(origDir)

	// Create ./.config/.uchi.yaml
	configDir := filepath.Join(tmpDir, ".config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create .config dir: %v", err)
	}
	configPath := filepath.Join(configDir, ".uchi.yaml")
	configData := "input_dir: yaml_dot_config_in\noutput_dir: yaml_dot_config_out\n"
	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InputDir != "yaml_dot_config_in" {
		t.Errorf("expected InputDir 'yaml_dot_config_in', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "yaml_dot_config_out" {
		t.Errorf("expected OutputDir 'yaml_dot_config_out', got %s", cfg.OutputDir)
	}

	// ./.uchi.yaml should NOT have been auto-generated since ./.config/.uchi.yaml existed
	if _, err := os.Stat("./.uchi.yaml"); !os.IsNotExist(err) {
		t.Errorf("./.uchi.yaml should not exist when ./.config/.uchi.yaml exists")
	}
}

func TestLoadConfigFlags(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(origDir)

	// Short flags
	argsShort := []string{"-i", "input_dir", "-o", "output_dir"}
	cfgShort, err := LoadConfig(argsShort)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgShort.InputDir != "input_dir" || cfgShort.OutputDir != "output_dir" {
		t.Errorf("short flags failed: %+v", cfgShort)
	}

	// Long flags
	argsLong := []string{"--input", "input_dir_long", "--output", "output_dir_long"}
	cfgLong, err := LoadConfig(argsLong)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgLong.InputDir != "input_dir_long" || cfgLong.OutputDir != "output_dir_long" {
		t.Errorf("long flags failed: %+v", cfgLong)
	}
}

func TestLoadConfigInvalidFlagSyntax(t *testing.T) {
	invalidCases := [][]string{
		{"-input", "dir"}, // long option with single hyphen
		{"--i", "dir"},     // short option with double hyphen
		{"-config", "file.yaml"},
		{"--c", "file.yaml"},
	}

	for _, args := range invalidCases {
		_, err := LoadConfig(args)
		if err == nil {
			t.Errorf("expected error for args %v, got nil", args)
		}
	}
}

func TestLoadConfigFileAndFlagsOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configData := "input_dir: yaml_in\noutput_dir: yaml_out\n"

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test config file loading without overrides
	cfg, err := LoadConfig([]string{"-c", configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InputDir != "yaml_in" {
		t.Errorf("expected InputDir 'yaml_in', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "yaml_out" {
		t.Errorf("expected OutputDir 'yaml_out', got %s", cfg.OutputDir)
	}

	// Test CLI flag overrides config file
	cfgOverride, err := LoadConfig([]string{"-c", configPath, "-o", "flag_out"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgOverride.InputDir != "yaml_in" {
		t.Errorf("expected InputDir 'yaml_in', got %s", cfgOverride.InputDir)
	}
	if cfgOverride.OutputDir != "flag_out" {
		t.Errorf("expected OutputDir 'flag_out', got %s", cfgOverride.OutputDir)
	}
}

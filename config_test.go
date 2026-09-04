package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
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
}

func TestLoadConfigFlags(t *testing.T) {
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
		{"-config", "file.json"},
		{"--c", "file.json"},
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
	configPath := filepath.Join(tmpDir, "config.json")
	configData := `{"input_dir": "json_in", "output_dir": "json_out"}`

	if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test config file loading without overrides
	cfg, err := LoadConfig([]string{"-c", configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.InputDir != "json_in" {
		t.Errorf("expected InputDir 'json_in', got %s", cfg.InputDir)
	}
	if cfg.OutputDir != "json_out" {
		t.Errorf("expected OutputDir 'json_out', got %s", cfg.OutputDir)
	}

	// Test CLI flag overrides config file
	cfgOverride, err := LoadConfig([]string{"-c", configPath, "-o", "flag_out"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfgOverride.InputDir != "json_in" {
		t.Errorf("expected InputDir 'json_in', got %s", cfgOverride.InputDir)
	}
	if cfgOverride.OutputDir != "flag_out" {
		t.Errorf("expected OutputDir 'flag_out', got %s", cfgOverride.OutputDir)
	}
}

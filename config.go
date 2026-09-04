package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	ConfigFile string `json:"config_file"`
	InputDir   string `json:"input_dir"`
	OutputDir  string `json:"output_dir"`
}

// DefaultConfig returns the configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		InputDir:  "./toc",
		OutputDir: "./dist",
	}
}

// LoadConfig creates a configuration with default values, loads overrides from
// a config file if specified, and finally overrides with CLI flags.
func LoadConfig(args []string) (*Config, error) {
	cfg := DefaultConfig()

	fs := flag.NewFlagSet("uchi", flag.ContinueOnError)
	var configFileFlag string
	var inputDirFlag string
	var outputDirFlag string

	fs.StringVar(&configFileFlag, "c", "", "Path to configuration file")
	fs.StringVar(&configFileFlag, "config", "", "Path to configuration file")
	fs.StringVar(&inputDirFlag, "i", "", "Input directory containing markdown files")
	fs.StringVar(&inputDirFlag, "input", "", "Input directory containing markdown files")
	fs.StringVar(&outputDirFlag, "o", "", "Output directory for extracted files")
	fs.StringVar(&outputDirFlag, "output", "", "Output directory for extracted files")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// If config file is provided via flag, load it first
	if configFileFlag != "" {
		cfg.ConfigFile = configFileFlag
		if err := loadConfigFile(cfg.ConfigFile, cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
		}
	}

	// CLI explicit flags override config file and defaults
	if inputDirFlag != "" {
		cfg.InputDir = inputDirFlag
	}
	if outputDirFlag != "" {
		cfg.OutputDir = outputDirFlag
	}

	return cfg, nil
}

func loadConfigFile(filePath string, cfg *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

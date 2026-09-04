package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
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
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fmt.Fprintf(fs.Output(), "  -c, --config string\n\tPath to configuration file\n")
		fmt.Fprintf(fs.Output(), "  -i, --input string\n\tInput directory containing markdown files\n")
		fmt.Fprintf(fs.Output(), "  -o, --output string\n\tOutput directory for extracted files\n")
	}

	var configFileFlag string
	var inputDirFlag string
	var outputDirFlag string

	fs.StringVar(&configFileFlag, "c", "", "Path to configuration file")
	fs.StringVar(&configFileFlag, "config", "", "Path to configuration file")
	fs.StringVar(&inputDirFlag, "i", "", "Input directory containing markdown files")
	fs.StringVar(&inputDirFlag, "input", "", "Input directory containing markdown files")
	fs.StringVar(&outputDirFlag, "o", "", "Output directory for extracted files")
	fs.StringVar(&outputDirFlag, "output", "", "Output directory for extracted files")

	normalizedArgs, err := preprocessArgs(args)
	if err != nil {
		return nil, err
	}

	if err := fs.Parse(normalizedArgs); err != nil {
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

func preprocessArgs(args []string) ([]string, error) {
	result := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			result = append(result, args[i:]...)
			break
		}
		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			if idx := strings.Index(name, "="); idx != -1 {
				name = name[:idx]
			}
			if len(name) == 1 {
				return nil, fmt.Errorf("invalid option syntax '%s': short option must use single hyphen", arg)
			}
		} else if strings.HasPrefix(arg, "-") && arg != "-" {
			name := arg[1:]
			if idx := strings.Index(name, "="); idx != -1 {
				name = name[:idx]
			}
			if len(name) > 1 {
				return nil, fmt.Errorf("invalid option syntax '%s': long option must use double hyphen", arg)
			}
		}
		result = append(result, arg)
	}
	return result, nil
}

func loadConfigFile(filePath string, cfg *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

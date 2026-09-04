package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	ConfigFile string `json:"-" yaml:"-"`
	InputDir   string `json:"input_dir" yaml:"input_dir"`
	OutputDir  string `json:"output_dir" yaml:"output_dir"`
	Command    string `json:"-" yaml:"-"`
	CommandArg string `json:"-" yaml:"-"`
}

// DefaultConfigPaths defines candidate configuration file paths in priority order.
var DefaultConfigPaths = []string{
	"./.uchi.yaml",
	"./.config/.uchi.yaml",
}

// DefaultConfig returns the configuration with default values.
func DefaultConfig() *Config {
	return &Config{
		InputDir:  "./toc",
		OutputDir: "./dist",
	}
}

// SaveConfigFile writes the configuration to a file in YAML format.
func SaveConfigFile(filePath string, cfg *Config) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// InitConfigFile creates a default configuration file at targetPath and notifies the user.
// This function can also be used directly by a future `init` subcommand.
func InitConfigFile(targetPath string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.ConfigFile = targetPath
	if err := SaveConfigFile(targetPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to create configuration file %s: %w", targetPath, err)
	}
	fmt.Printf("Created config file: %s\n", targetPath)
	return cfg, nil
}

// LoadConfig creates a configuration with default values, loads overrides from
// a config file if specified or found, and finally overrides with CLI flags.
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

	flagArgs, positionalArgs := partitionArgs(normalizedArgs)

	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}

	if len(positionalArgs) > 0 {
		switch positionalArgs[0] {
		case "new":
			cfg.Command = "new"
			if len(positionalArgs) == 2 {
				cfg.CommandArg = positionalArgs[1]
			} else {
				return nil, fmt.Errorf("subcommand 'new' requires exactly 1 argument")
			}
		default:
			return nil, fmt.Errorf("unknown command '%s'", positionalArgs[0])
		}
	}

	if configFileFlag != "" {
		cfg.ConfigFile = configFileFlag
		if err := loadConfigFile(cfg.ConfigFile, cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
		}
	} else {
		// Look for existing default configuration files
		var foundPath string
		for _, path := range DefaultConfigPaths {
			if _, err := os.Stat(path); err == nil {
				foundPath = path
				break
			}
		}

		if foundPath != "" {
			cfg.ConfigFile = foundPath
			if err := loadConfigFile(cfg.ConfigFile, cfg); err != nil {
				return nil, fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
			}
		} else {
			// Neither ./.uchi.yaml nor ./.config/.uchi.yaml exists, generate default ./.uchi.yaml
			defaultPath := DefaultConfigPaths[0]
			initCfg, err := InitConfigFile(defaultPath)
			if err != nil {
				return nil, err
			}
			cfg.ConfigFile = initCfg.ConfigFile
			cfg.InputDir = initCfg.InputDir
			cfg.OutputDir = initCfg.OutputDir
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

func partitionArgs(args []string) ([]string, []string) {
	var flagArgs []string
	var positionalArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionalArgs = append(positionalArgs, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs
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
	return yaml.Unmarshal(data, cfg)
}

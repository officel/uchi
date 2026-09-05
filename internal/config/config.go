package config

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
	ConfigFile  string `json:"-" yaml:"-"`
	InputDir    string `json:"input_dir" yaml:"input_dir"`
	OutputDir   string `json:"output_dir" yaml:"output_dir"`
	TemplateDir string `json:"template_dir" yaml:"template_dir"`
	AutoComment bool   `json:"auto_comment" yaml:"auto_comment"`
	Command     string `json:"-" yaml:"-"`
	CommandArg  string `json:"-" yaml:"-"`
}

// DefaultConfigPaths defines candidate configuration file paths in priority order.
var DefaultConfigPaths = []string{
	"./.uchi.yaml",
	"./.config/.uchi.yaml",
}

// Default returns the configuration with default values.
func Default() *Config {
	return &Config{
		InputDir:    "./toc",
		OutputDir:   "./dist",
		AutoComment: true,
	}
}

// Save writes the configuration to a file in YAML format.
func Save(filePath string, cfg *Config) error {
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

// Initialize creates a default configuration file at targetPath.
func Initialize(targetPath string) (*Config, error) {
	cfg := Default()
	cfg.ConfigFile = targetPath
	if err := Save(targetPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to create configuration file %s: %w", targetPath, err)
	}
	return cfg, nil
}

// Load creates a configuration with default values, loads overrides from a
// config file if specified or found, and finally overrides with CLI flags.
func Load(args []string) (*Config, error) {
	cfg := Default()

	fs := flag.NewFlagSet("uchi", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
		fmt.Fprintf(fs.Output(), "  -c, --config string\n\tPath to configuration file\n")
		fmt.Fprintf(fs.Output(), "  -i, --input string\n\tInput directory containing markdown files\n")
		fmt.Fprintf(fs.Output(), "  -o, --output string\n\tOutput directory for extracted files\n")
		fmt.Fprintf(fs.Output(), "  -t, --template-dir string\n\tDirectory containing template overrides\n")
		fmt.Fprintf(fs.Output(), "  --auto-comment\n\tOutput tool name as comment at header (default true)\n")
	}

	var configFileFlag string
	var inputDirFlag string
	var outputDirFlag string
	var templateDirFlag string
	var autoCommentFlag bool

	fs.StringVar(&configFileFlag, "c", "", "Path to configuration file")
	fs.StringVar(&configFileFlag, "config", "", "Path to configuration file")
	fs.StringVar(&inputDirFlag, "i", "", "Input directory containing markdown files")
	fs.StringVar(&inputDirFlag, "input", "", "Input directory containing markdown files")
	fs.StringVar(&outputDirFlag, "o", "", "Output directory for extracted files")
	fs.StringVar(&outputDirFlag, "output", "", "Output directory for extracted files")
	fs.StringVar(&templateDirFlag, "t", "", "Directory containing template overrides")
	fs.StringVar(&templateDirFlag, "template-dir", "", "Directory containing template overrides")
	fs.BoolVar(&autoCommentFlag, "auto-comment", true, "Output tool name as comment at header")

	normalizedArgs, err := preprocessArgs(args)
	if err != nil {
		return nil, err
	}

	flagArgs, positionalArgs := partitionArgs(normalizedArgs)
	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}

	autoCommentSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "auto-comment" {
			autoCommentSet = true
		}
	})

	if len(positionalArgs) > 0 {
		switch positionalArgs[0] {
		case "gen":
			cfg.Command = "gen"
			if len(positionalArgs) != 1 {
				return nil, fmt.Errorf("subcommand 'gen' does not take positional arguments")
			}
		case "check":
			cfg.Command = "check"
			if len(positionalArgs) != 1 {
				return nil, fmt.Errorf("subcommand 'check' does not take positional arguments")
			}
		case "new":
			cfg.Command = "new"
			if len(positionalArgs) == 2 {
				cfg.CommandArg = positionalArgs[1]
			} else {
				return nil, fmt.Errorf("subcommand 'new' requires exactly 1 argument")
			}
		case "init":
			cfg.Command = "init"
			if len(positionalArgs) != 1 {
				return nil, fmt.Errorf("subcommand 'init' does not take positional arguments")
			}
		default:
			return nil, fmt.Errorf("unknown command '%s'", positionalArgs[0])
		}
	}

	if configFileFlag != "" {
		cfg.ConfigFile = configFileFlag
		if err := loadFile(cfg.ConfigFile, cfg); err != nil {
			if !(cfg.Command == "init" && os.IsNotExist(err)) {
				return nil, fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
			}
		}
	} else if foundPath := findDefaultPath(); foundPath != "" {
		cfg.ConfigFile = foundPath
		if err := loadFile(cfg.ConfigFile, cfg); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
		}
	}

	if inputDirFlag != "" {
		cfg.InputDir = inputDirFlag
	}
	if outputDirFlag != "" {
		cfg.OutputDir = outputDirFlag
	}
	if templateDirFlag != "" {
		cfg.TemplateDir = templateDirFlag
	}
	if autoCommentSet {
		cfg.AutoComment = autoCommentFlag
	}

	return cfg, nil
}

func findDefaultPath() string {
	for _, path := range DefaultConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func partitionArgs(args []string) ([]string, []string) {
	var flagArgs []string
	var positionalArgs []string

	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			positionalArgs = append(positionalArgs, args[index+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			if !isBoolFlag(arg) && !strings.Contains(arg, "=") && index+1 < len(args) && !strings.HasPrefix(args[index+1], "-") {
				index++
				flagArgs = append(flagArgs, args[index])
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}
	return flagArgs, positionalArgs
}

func isBoolFlag(arg string) bool {
	name := strings.TrimPrefix(arg, "--")
	name = strings.TrimPrefix(name, "-")
	name = strings.SplitN(name, "=", 2)[0]
	return name == "auto-comment"
}

func preprocessArgs(args []string) ([]string, error) {
	result := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			result = append(result, args[index:]...)
			break
		}
		if strings.HasPrefix(arg, "--") {
			name := strings.SplitN(arg[2:], "=", 2)[0]
			if len(name) == 1 {
				return nil, fmt.Errorf("invalid option syntax '%s': short option must use single hyphen", arg)
			}
		} else if strings.HasPrefix(arg, "-") && arg != "-" {
			name := strings.SplitN(arg[1:], "=", 2)[0]
			if len(name) > 1 {
				return nil, fmt.Errorf("invalid option syntax '%s': long option must use double hyphen", arg)
			}
		}
		result = append(result, arg)
	}
	return result, nil
}

func loadFile(filePath string, cfg *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

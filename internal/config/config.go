package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/officel/uchi/internal/color"
	"github.com/officel/uchi/internal/schema"
	"gopkg.in/yaml.v3"
)

// ErrHelp is returned when help output is requested.
var ErrHelp = flag.ErrHelp

// Config holds the application configuration.
type Config struct {
	ConfigFile  string    `json:"-" yaml:"-"`
	InputDir    string    `json:"input_dir" yaml:"input_dir"`
	OutputDir   string    `json:"output_dir" yaml:"output_dir"`
	TemplateDir string    `json:"template_dir" yaml:"template_dir"`
	Shell       string    `json:"shell" yaml:"shell"`
	AutoComment bool      `json:"auto_comment" yaml:"auto_comment"`
	DryRun      bool      `json:"dry_run" yaml:"dry_run"`
	Diff        bool      `json:"diff" yaml:"diff"`
	Verbose     bool      `json:"verbose" yaml:"verbose"`
	Quiet       bool      `json:"quiet" yaml:"quiet"`
	Interactive bool      `json:"interactive" yaml:"interactive"`
	Color       string    `json:"color" yaml:"color"`
	Stdin       io.Reader `json:"-" yaml:"-"`
	Command     string    `json:"-" yaml:"-"`
	CommandArg  string    `json:"-" yaml:"-"`
}

// DefaultConfigPaths defines candidate configuration file paths in priority order.
var DefaultConfigPaths = []string{
	"./.uchi.yaml",
	"./.config/.uchi.yaml",
}

// Default returns the configuration with default values.
func Default() *Config {
	return &Config{
		InputDir:    ".",
		OutputDir:   "../dist",
		Shell:       schema.TargetAll,
		AutoComment: true,
		Color:       color.ModeAuto,
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

	normalizedArgs, err := preprocessArgs(args)
	if err != nil {
		return nil, err
	}

	flagArgs, positionalArgs := partitionArgs(normalizedArgs)

	if len(positionalArgs) > 0 && positionalArgs[0] == "help" {
		subcmd := ""
		if len(positionalArgs) > 1 {
			subcmd = positionalArgs[1]
		}
		PrintHelp(os.Stdout, subcmd)
		return nil, ErrHelp
	}

	subcmd := ""
	if len(positionalArgs) > 0 {
		subcmd = positionalArgs[0]
	}

	fs := flag.NewFlagSet("uchi", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		PrintHelp(fs.Output(), subcmd)
	}

	var configFileFlag string
	var inputDirFlag string
	var outputDirFlag string
	var templateDirFlag string
	var shellFlag string
	var autoCommentFlag bool
	var dryRunFlag bool
	var diffFlag bool
	var verboseFlag bool
	var quietFlag bool
	var interactiveFlag bool
	var colorFlag string

	fs.StringVar(&configFileFlag, "c", "", "Path to configuration file")
	fs.StringVar(&configFileFlag, "config", "", "Path to configuration file")
	fs.StringVar(&inputDirFlag, "i", "", "Input directory containing markdown files")
	fs.StringVar(&inputDirFlag, "input", "", "Input directory containing markdown files")
	fs.StringVar(&outputDirFlag, "o", "", "Output directory for extracted files")
	fs.StringVar(&outputDirFlag, "output", "", "Output directory for extracted files")
	fs.StringVar(&templateDirFlag, "t", "", "Directory containing template overrides")
	fs.StringVar(&templateDirFlag, "template-dir", "", "Directory containing template overrides")
	fs.StringVar(&shellFlag, "s", "", "Target shell for generation")
	fs.StringVar(&shellFlag, "shell", "", "Target shell for generation")
	fs.BoolVar(&autoCommentFlag, "auto-comment", true, "Output tool name as comment at header")
	fs.BoolVar(&dryRunFlag, "dry-run", false, "Perform a dry run without writing output files")
	fs.BoolVar(&diffFlag, "d", false, "Show diff between generation plan and current output")
	fs.BoolVar(&diffFlag, "diff", false, "Show diff between generation plan and current output")
	fs.BoolVar(&verboseFlag, "v", false, "Enable verbose output")
	fs.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output")
	fs.BoolVar(&quietFlag, "q", false, "Suppress non-essential output")
	fs.BoolVar(&quietFlag, "quiet", false, "Suppress non-essential output")
	fs.BoolVar(&interactiveFlag, "interactive", false, "Enable interactive diagnostic recovery")
	fs.StringVar(&colorFlag, "color", "", "Colorize output (auto, always, never)")

	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, err
	}

	autoCommentSet := false
	dryRunSet := false
	diffSet := false
	verboseSet := false
	quietSet := false
	interactiveSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "auto-comment" {
			autoCommentSet = true
		}
		if f.Name == "dry-run" {
			dryRunSet = true
		}
		if f.Name == "diff" || f.Name == "d" {
			diffSet = true
		}
		if f.Name == "verbose" || f.Name == "v" {
			verboseSet = true
		}
		if f.Name == "quiet" || f.Name == "q" {
			quietSet = true
		}
		if f.Name == "interactive" {
			interactiveSet = true
		}
	})

	if len(positionalArgs) > 0 {
		switch positionalArgs[0] {
		case "gen":
			cfg.Command = "gen"
			if len(positionalArgs) != 1 {
				return nil, fmt.Errorf("subcommand 'gen' does not take positional arguments")
			}
		case "diff":
			cfg.Command = "diff"
			if len(positionalArgs) != 1 {
				return nil, fmt.Errorf("subcommand 'diff' does not take positional arguments")
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
	if shellFlag != "" {
		cfg.Shell = shellFlag
	}
	if autoCommentSet {
		cfg.AutoComment = autoCommentFlag
	}
	if dryRunSet {
		cfg.DryRun = dryRunFlag
	}
	if diffSet {
		cfg.Diff = diffFlag
	}
	if verboseSet {
		cfg.Verbose = verboseFlag
	}
	if quietSet {
		cfg.Quiet = quietFlag
	}
	if interactiveSet {
		cfg.Interactive = interactiveFlag
	}
	if colorFlag != "" {
		cfg.Color = colorFlag
	}

	if !color.IsValidMode(cfg.Color) {
		return nil, fmt.Errorf("unknown color setting %q (valid settings: %s)", cfg.Color, strings.Join(color.ValidModes(), ", "))
	}

	if !schema.IsValidTarget(cfg.Shell) {
		return nil, fmt.Errorf("unknown shell %q (valid targets: %s)", cfg.Shell, strings.Join(schema.ValidTargets(), ", "))
	}

	if cfg.Verbose && cfg.Quiet {
		return nil, fmt.Errorf("cannot specify both --verbose and --quiet")
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
	return name == "auto-comment" || name == "dry-run" || name == "diff" || name == "d" || name == "verbose" || name == "v" || name == "quiet" || name == "q" || name == "interactive"
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

// PrintHelp outputs the CLI or subcommand help text to the provided writer.
func PrintHelp(w io.Writer, command string) {
	switch command {
	case "gen":
		fmt.Fprintln(w, "Usage of gen:")
		fmt.Fprintln(w, "  Extract code blocks from Markdown files and generate output files.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi gen [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -i, --input <dir>         Input directory containing markdown files (default \".\")")
		fmt.Fprintln(w, "  -o, --output <dir>        Output directory for extracted files (default \"../dist\")")
		fmt.Fprintln(w, "  -c, --config <path>       Path to YAML configuration file")
		fmt.Fprintln(w, "  -s, --shell <target>      Target shell filter (all, bash, fish, powershell, pwsh, sh, zsh) (default \"all\")")
		fmt.Fprintln(w, "      --auto-comment        Output tool name as comment at header (default true)")
		fmt.Fprintln(w, "      --dry-run             Perform a dry run without writing output files")
		fmt.Fprintln(w, "  -d, --diff                Show diff between generation plan and current output")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
	case "check":
		fmt.Fprintln(w, "Usage of check:")
		fmt.Fprintln(w, "  Validate input Markdown files and generation plan without writing files.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi check [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -i, --input <dir>         Input directory containing markdown files (default \".\")")
		fmt.Fprintln(w, "  -o, --output <dir>        Output directory for extracted files (default \"../dist\")")
		fmt.Fprintln(w, "  -c, --config <path>       Path to YAML configuration file")
		fmt.Fprintln(w, "  -s, --shell <target>      Target shell filter (all, bash, fish, powershell, pwsh, sh, zsh) (default \"all\")")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --interactive         Enable interactive diagnostic recovery")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
	case "diff":
		fmt.Fprintln(w, "Usage of diff:")
		fmt.Fprintln(w, "  Compare generation plan and manifest against current output files on disk.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi diff [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -i, --input <dir>         Input directory containing markdown files (default \".\")")
		fmt.Fprintln(w, "  -o, --output <dir>        Output directory for extracted files (default \"../dist\")")
		fmt.Fprintln(w, "  -c, --config <path>       Path to YAML configuration file")
		fmt.Fprintln(w, "  -s, --shell <target>      Target shell filter (all, bash, fish, powershell, pwsh, sh, zsh) (default \"all\")")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
	case "init":
		fmt.Fprintln(w, "Usage of init:")
		fmt.Fprintln(w, "  Create a default configuration file (.uchi.yaml) in the current directory.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi init [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -c, --config <path>       Path to configuration file to create")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
	case "new":
		fmt.Fprintln(w, "Usage of new:")
		fmt.Fprintln(w, "  Create a new uchi: v1 Markdown document in input directory.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi new <NAME> [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -i, --input <dir>         Input directory containing markdown files (default \".\")")
		fmt.Fprintln(w, "  -t, --template-dir <dir>  Directory containing template overrides")
		fmt.Fprintln(w, "  -c, --config <path>       Path to YAML configuration file")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
	default:
		fmt.Fprintln(w, "uchi - Dotfiles Literate Configuration CLI tool")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "  uchi [command] [flags]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Commands:")
		fmt.Fprintln(w, "  check        Validate input Markdown files and generation plan (default)")
		fmt.Fprintln(w, "  gen          Extract code blocks and generate output files")
		fmt.Fprintln(w, "  diff         Compare generation plan and manifest against output files")
		fmt.Fprintln(w, "  init         Create a default .uchi.yaml configuration file")
		fmt.Fprintln(w, "  new <NAME>   Create a new uchi: v1 Markdown document (<NAME>.md)")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		fmt.Fprintln(w, "  -c, --config <path>       Path to YAML configuration file")
		fmt.Fprintln(w, "  -i, --input <dir>         Input directory containing markdown files (default \".\")")
		fmt.Fprintln(w, "  -o, --output <dir>        Output directory for extracted files (default \"../dist\")")
		fmt.Fprintln(w, "  -t, --template-dir <dir>  Directory containing template overrides")
		fmt.Fprintln(w, "  -s, --shell <target>      Target shell filter (all, bash, fish, powershell, pwsh, sh, zsh) (default \"all\")")
		fmt.Fprintln(w, "      --auto-comment        Output tool name as comment at header (default true)")
		fmt.Fprintln(w, "      --dry-run             Perform a dry run without writing output files")
		fmt.Fprintln(w, "  -d, --diff                Show diff between generation plan and current output")
		fmt.Fprintln(w, "  -v, --verbose             Enable verbose output")
		fmt.Fprintln(w, "  -q, --quiet               Suppress non-essential output")
		fmt.Fprintln(w, "      --interactive         Enable interactive diagnostic recovery")
		fmt.Fprintln(w, "      --color <mode>        Colorize output (auto, always, never) (default \"auto\")")
		fmt.Fprintln(w, "  -h, --help                Show help for uchi or a subcommand")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Use \"uchi <command> --help\" or \"uchi help <command>\" for detailed help on a subcommand.")
	}
}

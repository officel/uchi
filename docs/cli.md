# uchi CLI Reference & Manual

This document serves as the complete reference manual for the `uchi` command-line interface.

---

## SYNOPSIS

```bash
uchi [command] [flags]
```

When no command is provided, `uchi` defaults to the `check` command.

---

## COMMANDS

### `check` (default)

Validates input Markdown files (`uchi: v1` frontmatter, code fence attributes, shell portability rules, output path boundaries, and path conflict checks) without writing files to disk.

```bash
uchi check [flags]
```

- **Exit Status**: `0` on successful validation; `1` if diagnostic errors or conflicts are found.

---

### `gen`

Extracts code blocks according to schema annotations (`alias`, `env`, `function`, `profile`, `rc`) and target shell criteria, writing atomic output files to `output_dir` (`parts/` and merged schema outputs) along with `.uchi-manifest.json`.

```bash
uchi gen [flags]
```

- **Dry-Run Mode (`--dry-run`)**: Builds and verifies the generation plan and prints planned output paths line-by-line without writing disk files.

---

### `diff`

Compares the generation plan and previously recorded `.uchi-manifest.json` against existing files in `output_dir`, displaying line-by-line diffs and status headers:

- `[+]`: New file in generation plan.
- `[~]`: Modified file with content differences.
- `[=]`: Unchanged file.
- `[-]`: Removed file previously tracked in manifest.

```bash
uchi diff [flags]
```

*(Alternatively, `uchi gen --diff` or `uchi gen -d` performs diff comparison).*

---

### `init`

Creates a default configuration file (`.uchi.yaml`) in the current directory.

```bash
uchi init [flags]
```

- If `.uchi.yaml` already exists, `init` returns an error to prevent accidental overwriting.

---

### `new <NAME>`

Creates a new `uchi: v1` Markdown file (`<NAME>.md`) in `input_dir` using standard or custom templates.

```bash
uchi new <NAME> [flags]
```

- **Positional Argument**: `<NAME>` (required). The created file will be `<input_dir>/<NAME>.md`.

---

## OPTIONS & FLAGS

CLI flags take precedence over configuration file settings and built-in defaults.

| Flag | Long Option | Description | Default |
|---|---|---|---|
| `-c` | `--config <path>` | Path to YAML configuration file | candidate search (`./.uchi.yaml`, `./.config/.uchi.yaml`) |
| `-i` | `--input <dir>` | Directory containing input Markdown files | `.` |
| `-o` | `--output <dir>` | Directory for extracted output files | `../dist` |
| `-t` | `--template-dir <dir>` | Directory containing custom template overrides (`new.md.tmpl`) | `""` (bundled template) |
| `-s` | `--shell <target>` | Filter generation by target shell (`all`, `bash`, `fish`, `powershell`, `pwsh`, `sh`, `zsh`) | `all` |
| | `--auto-comment` | Prepend `# <tool_name>` header comment to non-empty schema files | `true` |
| | `--dry-run` | Perform generation plan verification and list target paths without disk writes | `false` |
| `-d` | `--diff` | Show diff between generation plan and current output files | `false` |
| `-v` | `--verbose` | Enable verbose execution logging (analyzed files, fence extraction, skip reasons) | `false` |
| `-q` | `--quiet` | Suppress non-essential stdout output | `false` |
| | `--color <mode>` | Colorize output (`auto`, `always`, `never`) | `auto` |
| `-h` | `--help` | Show usage help for `uchi` or a subcommand | |

---

## CONFIGURATION PRECEDENCE

Configuration values are resolved in the following order (highest to lowest precedence):

1. **CLI Flags**: Passed explicitly on the command line (e.g. `-i ./src -o ./dist`).
2. **Configuration File**: Values defined in `.uchi.yaml` or `.config/.uchi.yaml` (or specified via `-c <path>`).
3. **Default Values**: Built-in defaults (`input_dir: "."`, `output_dir: "../dist"`, `shell: "all"`, `auto_comment: true`, `color: "auto"`).

### `.uchi.yaml` Example

```yaml
input_dir: .
output_dir: dist
shell: all
auto_comment: true
color: auto
```

---

## ENVIRONMENT VARIABLES

- **`NO_COLOR`**: When set to any non-empty value, disables ANSI escape codes when `--color=auto` is active.

```bash
NO_COLOR=1 uchi check
```

---

## EXIT CODES

- **`0`**: Command completed successfully (all inputs valid, generation completed, diff executed, or help displayed).
- **`1`**: Command failed due to validation errors, syntax/portability errors, path safety/conflict violations, or invalid flag combinations (such as specifying both `--verbose` and `--quiet`).

---

## REPRESENTATIVE WORKFLOWS & EXAMPLES

All examples below can be executed using the `uchi` binary built with `go build -o uchi ./cmd/uchi`.

### 1. Initial Setup and Basic Execution

Create a default config, create a new Markdown document, validate, and generate settings:

```bash
# 1. Initialize configuration file (.uchi.yaml)
./uchi init

# 2. Create a new document shell.md
./uchi new shell

# 3. Validate Markdown inputs
./uchi check

# 4. Generate extracted dotfile outputs into ./dist
./uchi gen -o ./dist
```

---

### 2. Previewing Generation Output (`--dry-run`)

Build and verify the generation plan without touching disk files:

```bash
./uchi gen --dry-run -o ./dist
```

*Output Example:*

```text
dist/parts/shell/alias
dist/parts/shell/env
dist/parts/shell/profile
dist/parts/shell/rc
dist/parts/shell/function
dist/alias
dist/env
dist/profile
dist/rc
dist/function
```

---

### 3. Inspecting Differences (`diff`)

Compare Markdown source changes against generated files and `.uchi-manifest.json`:

```bash
./uchi diff -o ./dist
```

*Output Example (when modifying an alias):*

```text
[~] dist/parts/shell/alias (kind: part, status: modified)
  # shell
  alias g="git"
+ alias ll="ls -la"
```

---

### 4. Target Shell Filtering (`-s` / `--shell`)

Filter code fences to extract shell-specific configurations:

```bash
# Extract Zsh-compatible snippets only
./uchi gen -s zsh -o ./dist/zsh

# Extract Bash-compatible snippets only
./uchi gen -s bash -o ./dist/bash
```

---

### 5. Error Diagnostics Checking (`check`)

When an input Markdown document contains invalid frontmatter or code fence attributes (e.g. `{schema=}` empty value):

```bash
./uchi check
```

*Error Output:*

```text
Error executing command: shell.md:12: failed to parse code fence: empty attribute value for key "schema"
```

*Exit status:* `1` (`echo $?` returns `1`).

---

## RELATED DOCUMENTATION

- [README.md](../README.md): Project overview and quickstart entry point.
- [BUILD.md](../BUILD.md): Build, test, and developer guide.
- [format-v1.md](format-v1.md): `uchi: v1` Markdown document format specification.
- [compatibility.md](compatibility.md): Compatibility guarantees and golden test policies.

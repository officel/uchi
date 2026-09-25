# Build and Run Instructions

This document provides build, test, and complete CLI usage reference for `uchi`.

## Prerequisites

- [Go](https://go.dev/) (version 1.24 or later)

## Building the CLI

To compile the application binary in the root repository directory:

```bash
go build -o uchi ./cmd/uchi
```

This creates an executable named `uchi` (or `uchi.exe` on Windows).

## Running Tests and Linter

To run unit and integration tests across all packages:

```bash
go test -v ./...
```

To run static analysis:

```bash
go vet ./...
```

---

## Command Reference & Workflow

`uchi` operates on Markdown files matching the `uchi: v1` specification ([docs/format-v1.md](docs/format-v1.md)).

### Subcommands

| Subcommand | Description | Purpose | Expected Exit Code |
|---|---|---|---|
| `check` (default) | Validates input Markdown files and checks generation plan integrity without writing disk files. | Input validation & syntax error detection | `0` (valid) / `1` (error) |
| `gen` | Extracts code blocks and atomically generates output files in `output_dir`, saving `.uchi-manifest.json`. | Artifact generation | `0` (success) / `1` (error) |
| `gen --dry-run` | Builds and verifies generation plan, outputting planned output target file paths line-by-line without disk writes. | Generation plan preview | `0` (success) / `1` (error) |
| `diff` / `gen --diff` | Compares generation plan and manifest against existing files in `output_dir`. | Difference inspection | `0` (success) / `1` (error) |
| `init` | Generates a default configuration file (`.uchi.yaml`) in the current directory. | Initial configuration setup | `0` (success) / `1` (error) |
| `new <NAME>` | Creates a new Markdown file (`<NAME>.md`) in `input_dir` using default or custom template. | New document creation | `0` (success) / `1` (error) |

---

## Options & Flags

CLI options take precedence over configuration files (`.uchi.yaml`) and defaults.

| Short Flag | Long Flag | Description | Default Value |
|---|---|---|---|
| `-i` | `--input <dir>` | Directory containing input Markdown files | `.` |
| `-o` | `--output <dir>` | Directory where extracted files will be generated | `../dist` |
| `-c` | `--config <path>` | Path to YAML configuration file | candidate search (`./.uchi.yaml`, `./.config/.uchi.yaml`) |
| `-t` | `--template-dir <dir>` | Directory containing custom Go templates (`new.md.tmpl`) | `""` (bundled template) |
| `-s` | `--shell <target>` | Filter generation plan by target shell (`all`, `bash`, `fish`, `powershell`, `pwsh`, `sh`, `zsh`) | `all` |
| | `--auto-comment` | Prepend `# <tool_name>` header comment to non-empty part schema files | `true` |
| | `--dry-run` | Perform plan verification and output target path listing without disk writes | `false` |
| `-d` | `--diff` | Perform diff comparison against existing output files and manifest | `false` |
| `-v` | `--verbose` | Enable verbose execution output (analyzed documents, fence extraction, skip reasons, plan details) | `false` |
| `-q` | `--quiet` | Suppress non-essential informational messages | `false` |

---

## Usage Examples & Exit Expectations

### 1. Initialize Configuration (`init`)

Creates `.uchi.yaml` with default settings:

```bash
./uchi init
```

*Output:*

```text
Created ./.uchi.yaml
```

*Exit status:* `0`

---

### 2. Create a Document (`new`)

Generates a Markdown file pre-configured with `uchi: v1` frontmatter:

```bash
./uchi new shell
```

*Output:*

```text
Created shell.md
```

*Exit status:* `0`

---

### 3. Validate Inputs (`check`)

Validates Markdown syntax, frontmatter (`uchi: v1`), code fence attributes, and path safety:

```bash
./uchi check
```

*Output (Success):*

```text
Config file: found (./.uchi.yaml)
Options:
  input_dir: .
  output_dir: ../dist
  template_dir:
  auto_comment: true
```

*Exit status:* `0`

#### Failure Example (`check`)

If a Markdown file contains invalid fence attributes (e.g., an empty attribute value `{schema=}` in `invalid.md` at line 5):

````markdown
---
uchi: v1
---

```bash {schema=}
alias foo='bar'
```
````

Running `./uchi check` fails early with a line-numbered diagnostic error:

```bash
./uchi check
```

*Error Output:*

```text
Error executing command: invalid.md:5: failed to parse code fence: empty attribute value for key "schema"
```

*Exit status:* `1` (`echo $?` returns `1`)

---

### 4. Preview Output Paths (`dry-run`)

Prints planned target output paths line-by-line without writing to disk:

```bash
./uchi gen --dry-run
```

*Output:*

```text
../dist/parts/shell/alias
../dist/parts/shell/env
../dist/parts/shell/profile
../dist/parts/shell/rc
../dist/parts/shell/function
../dist/alias
../dist/env
../dist/profile
../dist/rc
../dist/function
```

*Exit status:* `0`

---

### 5. Generate Extracted Files (`gen`)

Constructs generation plan, verifies path safety and conflict rules, and atomically writes output files:

```bash
./uchi gen -o ./dist
```

*Exit status:* `0`

During generation, `.uchi-manifest.json` is saved in the output directory tracking generated target paths.

---

### 6. Inspect Differences (`diff`)

Compares the generation plan and `.uchi-manifest.json` against current files in `output_dir`:

```bash
./uchi diff -o ./dist
```

*Status Indicators:*

- `[+]` (`new`): Target present in generation plan but missing on disk.
- `[~]` (`modified`): Target present in generation plan and on disk, but content differs (displays concise line diff).
- `[=]` (`unchanged`): Target present in generation plan and on disk with identical content.
- `[-]` (`deleted`): Target recorded in `.uchi-manifest.json` but no longer present in generation plan.

*Exit status:* `0` on successful comparison execution.

---

## Output Levels & Stream Policy

`uchi` supports three output levels to control message verbosity:

- **Normal Level** (default): Displays command results and informational messages (such as `check` configuration overview or `new`/`init` creation messages). Machine-readable outputs (e.g., `gen --dry-run` target paths, `diff` lines) are output cleanly.
- **Quiet Level** (`-q`, `--quiet`): Suppresses non-essential informational stdout output on success (such as `check` configuration summary or `new`/`init` creation messages), while preserving machine-readable command results (e.g., `gen --dry-run` paths, `diff` lines).
- **Verbose Level** (`-v`, `--verbose`): Outputs detailed execution logs on stdout (`[verbose] ...`), detailing input directory traversal, analyzed Markdown files, skipped documents/code blocks with reasons (e.g., unannotated blocks, shell mismatch, missing schema), and generation plan construction details.

*Mutual Exclusion*: `--verbose` and `--quiet` cannot be specified simultaneously. Doing so returns an error.

*Stream Separation*:
- **Standard Output (stdout)**: Primary command results (target path lists, diff outputs, configuration overviews) and verbose execution logs.
- **Standard Error (stderr)**: Diagnostic errors (e.g., line-numbered YAML/frontmatter syntax errors, fence attribute errors, portability violations) and CLI errors.

---

## Target Shell Selection & Conversion Limitations

- **Default Target**: If `-s` / `--shell` is omitted, the target shell defaults to `all`.
- **Supported Shell Options**: `all`, `bash`, `fish`, `powershell`, `pwsh`, `sh`, `zsh`.
- **Syntax Transformation Rules**:
  - `bash` and `zsh` targets apply minimal safe syntax transformations for recognized schemas (`alias`, `env`, `function`, `profile`, `rc`). For example, `typeset -x` in `env` blocks is automatically converted to `export` for `bash`.
  - Incompatible syntax (e.g. Zsh-specific `alias -g` on Bash or `export -f` on Zsh) emits line-numbered diagnostic errors.
  - Automatic syntax conversion for `powershell` / `pwsh` and other shells is currently **not implemented**.

---

## Version Control Caution for Output Directory

All files written to `output_dir` (default: `../dist`) and the manifest `.uchi-manifest.json` are generated build artifacts.

To prevent accidentally committing generated outputs to version control, add the output directory to your `.gitignore`:

```gitignore
# uchi generated outputs
/dist/
../dist/
.uchi-manifest.json
```

---

## Format Specification Link

For complete details on input Markdown frontmatter, code fence annotations, schemas, portability rules, and output structure, refer to:

- [docs/format-v1.md](docs/format-v1.md)

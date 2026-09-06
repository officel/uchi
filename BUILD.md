# Build and Run Instructions

This document provides instructions for building, testing, and running the `uchi` Go CLI application.

## Prerequisites

- [Go](https://go.dev/) (version 1.24 or later)

## Building the CLI

To compile the application binary in the root repository directory:

```bash
go build -o uchi ./cmd/uchi
```

This will create an executable named `uchi` (or `uchi.exe` on Windows).

## Running Tests

To run all unit and integration tests across the project:

```bash
go test -v ./...
```

## Usage

The CLI supports reading markdown files, parsing frontmatter and code blocks (code fences), and exporting extracted contents to an output directory.

### Options

- `-i`, `--input`: Path to the input directory containing Markdown files (Default: `./toc`)
- `-o`, `--output`: Path to the output directory where extracted files will be saved (Default: `./dist`)
- `-c`, `--config`: Path to a YAML configuration file specifying default values.
- `-t`, `--template-dir`: Directory containing templates that override bundled defaults.

### Example Commands

Check configuration status and current options (default command):

```bash
./uchi
```

or explicitly:

```bash
./uchi check
```

Generate extracted configuration files to output directory (`./dist`). Files split by Markdown document are written below `./dist/parts`, while merged schema files are written directly below `./dist`:

```bash
./uchi gen
```

Initialize a configuration file (`.uchi.yaml`):

```bash
./uchi init
```

Specify custom input and output directories:

```bash
./uchi -i ./docs -o ./out
```

Specify a configuration file:

```bash
./uchi -c .uchi.yaml
```

Override options in a configuration file with CLI flags:

```bash
./uchi -c .uchi.yaml -o ./override_out
```

### Templates

`uchi new NAME` uses the template bundled in the binary by default. To override it,
place a Go template named `new.md.tmpl` in a template directory and pass that
directory with `-t` or configure it with `template_dir`:

```yaml
input_dir: ./toc
output_dir: ./dist
template_dir: ./templates
```

Templates receive `.Name` and `.Date`. For example:

```text
# {{ .Name }}

- {{ .Date }}
```

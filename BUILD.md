# Build and Run Instructions

This document provides instructions for building, testing, and running the `uchi` Go CLI application.

## Prerequisites

- [Go](https://go.dev/) (version 1.24 or later)

## Building the CLI

To compile the application binary in the root repository directory:

```bash
go build -o uchi .
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
- `-c`, `--config`: Path to a JSON configuration file specifying default values.

### Example Commands

Run with defaults:

```bash
./uchi
```

Specify custom input and output directories:

```bash
./uchi -i ./docs -o ./out
```

Specify a configuration file:

```bash
./uchi -c config.json
```

Override options in a configuration file with CLI flags:

```bash
./uchi -c config.json -o ./override_out
```

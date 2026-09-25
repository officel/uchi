package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// Defined schema constants.
const (
	SchemaAlias    = "alias"
	SchemaEnv      = "env"
	SchemaProfile  = "profile"
	SchemaRc       = "rc"
	SchemaFunction = "function"
)

// Defined target shell constants.
const (
	TargetAll        = "all"
	TargetBash       = "bash"
	TargetFish       = "fish"
	TargetPowerShell = "powershell"
	TargetPwsh       = "pwsh"
	TargetSh         = "sh"
	TargetZsh        = "zsh"
)

var definedTargets = map[string]bool{
	TargetAll:        true,
	TargetBash:       true,
	TargetFish:       true,
	TargetPowerShell: true,
	TargetPwsh:       true,
	TargetSh:         true,
	TargetZsh:        true,
}

// ValidationIssue represents a diagnostic error during schema content validation.
type ValidationIssue struct {
	Line    int    // 1-based relative line number in snippet content
	Schema  string // Schema name, e.g. "alias", "env"
	Message string // Diagnostic message
}

func (v ValidationIssue) Error() string {
	return fmt.Sprintf("schema %q: %s", v.Schema, v.Message)
}

// AdapterIssue represents a diagnostic error for schema output adapters.
type AdapterIssue struct {
	Line    int    // 1-based relative line number in snippet content
	Schema  string // Schema name, e.g. "alias", "env"
	Target  string // Target shell, e.g. "bash", "zsh"
	Message string // Diagnostic message
}

func (a AdapterIssue) Error() string {
	return a.Message
}

// ValidTargets returns a sorted slice of valid target shell names.
func ValidTargets() []string {
	return []string{TargetAll, TargetBash, TargetFish, TargetPowerShell, TargetPwsh, TargetSh, TargetZsh}
}

// IsValidTarget reports whether the given name is a valid target shell name.
func IsValidTarget(name string) bool {
	return definedTargets[name]
}

var definedValidators = map[string]func(string) []ValidationIssue{
	SchemaAlias:    ValidateAlias,
	SchemaEnv:      ValidateEnv,
	SchemaProfile:  ValidateProfile,
	SchemaRc:       ValidateRc,
	SchemaFunction: ValidateFunction,
}

var definedSchemas = map[string]func(string, string) (string, []AdapterIssue){
	SchemaAlias:    ProcessAlias,
	SchemaEnv:      ProcessEnv,
	SchemaProfile:  ProcessProfile,
	SchemaRc:       ProcessRc,
	SchemaFunction: ProcessFunction,
}

// ValidSchemas returns a sorted slice of defined schema names.
func ValidSchemas() []string {
	return []string{SchemaAlias, SchemaEnv, SchemaFunction, SchemaProfile, SchemaRc}
}

// IsDefined reports whether the given schema name is defined.
func IsDefined(name string) bool {
	_, ok := definedSchemas[name]
	return ok
}

// Validate validates content using the schema-specific validator for name.
// It returns validation issues and true if defined, or (nil, false) if undefined.
func Validate(name string, content string) ([]ValidationIssue, bool) {
	fn, ok := definedValidators[name]
	if !ok {
		return nil, false
	}
	issues := fn(content)
	return issues, true
}

// Process processes content using the schema-specific function for name and target shell.
// It returns processed string, adapter issues, and true if defined, or ("", nil, false) if undefined.
func Process(name string, content string, target string) (string, []AdapterIssue, bool) {
	fn, ok := definedSchemas[name]
	if !ok {
		return "", nil, false
	}
	res, issues := fn(content, target)
	return res, issues, true
}

var (
	reZshAliasFlags = regexp.MustCompile(`\balias\s+-[^ ]*`)
	reExportN       = regexp.MustCompile(`\bexport\s+-n\b`)
	reExportF       = regexp.MustCompile(`\bexport\s+-f\b`)
	reZshFuncFlags  = regexp.MustCompile(`\bfunction\s+-[a-zA-Z]+\b`)
	reShopt         = regexp.MustCompile(`\bshopt\b`)
	reSetopt        = regexp.MustCompile(`\b(setopt|unsetopt)\b`)
	reTypesetLine   = regexp.MustCompile(`^(\s*)typeset\s+(.*)$`)

	// Validation regexes
	reAliasSyntax = regexp.MustCompile(`^(\s*)alias(\s+.*)?$`)
	reAliasAssign = regexp.MustCompile(`\b[a-zA-Z0-9_.-]+=\S+`)

	reEnvAssign   = regexp.MustCompile(`^\s*(export|typeset|declare|local|readonly)\b`)
	reVarAssign   = regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)=`)
	reKWVarAssign = regexp.MustCompile(`^\s*(export|typeset|declare|local|readonly)\s+.*?\b([a-zA-Z_][a-zA-Z0-9_]*)=`)

	reFuncDefine = regexp.MustCompile(`^\s*(function\s+[a-zA-Z_][a-zA-Z0-9_]*|[a-zA-Z_][a-zA-Z0-9_]*\s*\(\s*\))`)
)

// ValidateAlias validates code fence content with schema=alias.
func ValidateAlias(content string) []ValidationIssue {
	lines := strings.Split(content, "\n")
	var issues []ValidationIssue

	for i, rawLine := range lines {
		lineNum := i + 1
		masked := MaskShellLine(rawLine)
		trimmedMasked := strings.TrimSpace(masked)
		if trimmedMasked == "" {
			continue
		}

		if !reAliasSyntax.MatchString(trimmedMasked) {
			issues = append(issues, ValidationIssue{
				Line:    lineNum,
				Schema:  SchemaAlias,
				Message: fmt.Sprintf("invalid alias statement %q: must start with 'alias'", strings.TrimSpace(rawLine)),
			})
			continue
		}

		if !reAliasAssign.MatchString(trimmedMasked) {
			issues = append(issues, ValidationIssue{
				Line:    lineNum,
				Schema:  SchemaAlias,
				Message: fmt.Sprintf("invalid alias statement %q: missing '=' assignment", strings.TrimSpace(rawLine)),
			})
		}
	}

	return issues
}

// ValidateEnv validates code fence content with schema=env.
func ValidateEnv(content string) []ValidationIssue {
	lines := strings.Split(content, "\n")
	var issues []ValidationIssue

	for i, rawLine := range lines {
		lineNum := i + 1
		masked := MaskShellLine(rawLine)
		trimmedMasked := strings.TrimSpace(masked)
		if trimmedMasked == "" {
			continue
		}

		if !reEnvAssign.MatchString(trimmedMasked) && !reVarAssign.MatchString(trimmedMasked) {
			issues = append(issues, ValidationIssue{
				Line:    lineNum,
				Schema:  SchemaEnv,
				Message: fmt.Sprintf("invalid env statement %q: expected environment variable assignment or export", strings.TrimSpace(rawLine)),
			})
			continue
		}

		if !reVarAssign.MatchString(trimmedMasked) && !reKWVarAssign.MatchString(trimmedMasked) {
			issues = append(issues, ValidationIssue{
				Line:    lineNum,
				Schema:  SchemaEnv,
				Message: fmt.Sprintf("invalid env statement %q: missing variable assignment '='", strings.TrimSpace(rawLine)),
			})
		}
	}

	return issues
}

// ValidateFunction validates code fence content with schema=function.
func ValidateFunction(content string) []ValidationIssue {
	lines := strings.Split(content, "\n")
	var issues []ValidationIssue

	// Check if content contains at least one function definition
	hasFuncDef := false
	for _, rawLine := range lines {
		masked := MaskShellLine(rawLine)
		trimmedMasked := strings.TrimSpace(masked)
		if reFuncDefine.MatchString(trimmedMasked) {
			hasFuncDef = true
			break
		}
	}

	if !hasFuncDef && strings.TrimSpace(content) != "" {
		issues = append(issues, ValidationIssue{
			Line:    1,
			Schema:  SchemaFunction,
			Message: "missing function definition in function schema block",
		})
	}

	return issues
}

// ValidateProfile validates code fence content with schema=profile.
func ValidateProfile(content string) []ValidationIssue {
	return nil
}

// ValidateRc validates code fence content with schema=rc.
func ValidateRc(content string) []ValidationIssue {
	return nil
}

// ProcessAlias processes code fences with schema=alias for the target shell.
func ProcessAlias(content string, target string) (string, []AdapterIssue) {
	if target != TargetBash {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	var issues []AdapterIssue

	for i, rawLine := range lines {
		masked := MaskShellLine(rawLine)
		if loc := reZshAliasFlags.FindString(masked); loc != "" {
			if strings.Contains(loc, "-g") || strings.Contains(loc, "-s") {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  SchemaAlias,
					Target:  target,
					Message: fmt.Sprintf("Zsh-specific alias option in %q is not supported in Bash", strings.TrimSpace(rawLine)),
				})
			}
		}
	}

	return content, issues
}

// ProcessEnv processes code fences with schema=env for the target shell.
func ProcessEnv(content string, target string) (string, []AdapterIssue) {
	lines := strings.Split(content, "\n")
	var issues []AdapterIssue
	modified := false

	for i, rawLine := range lines {
		masked := MaskShellLine(rawLine)

		if target == TargetZsh {
			if reExportN.MatchString(masked) {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  SchemaEnv,
					Target:  target,
					Message: fmt.Sprintf("Bash-specific option 'export -n' in %q is not supported in Zsh", strings.TrimSpace(rawLine)),
				})
			}
			continue
		}

		if target == TargetBash {
			m := reTypesetLine.FindStringSubmatch(rawLine)
			if m != nil {
				indent := m[1]
				rest := strings.TrimSpace(m[2])
				tokens := strings.Fields(rest)

				hasExportFlag := false
				hasUnsupportedFlag := false

				var nonFlagTokens []string

				for _, tok := range tokens {
					if strings.HasPrefix(tok, "-") && len(tok) > 1 && !strings.Contains(tok, "=") {
						flags := tok[1:]
						for _, fl := range flags {
							if fl == 'x' {
								hasExportFlag = true
							} else if fl == 'g' {
								// -g is global in Zsh, harmless for export conversion
							} else {
								hasUnsupportedFlag = true
							}
						}
					} else {
						nonFlagTokens = append(nonFlagTokens, tok)
					}
				}

				if hasUnsupportedFlag || !hasExportFlag {
					issues = append(issues, AdapterIssue{
						Line:    i + 1,
						Schema:  SchemaEnv,
						Target:  target,
						Message: fmt.Sprintf("unsupported typeset option in %q for Bash export in env schema", strings.TrimSpace(rawLine)),
					})
				} else {
					lines[i] = indent + "export " + strings.Join(nonFlagTokens, " ")
					modified = true
				}
			}
		}
	}

	if modified {
		return strings.Join(lines, "\n"), issues
	}
	return content, issues
}

// ProcessFunction processes code fences with schema=function for the target shell.
func ProcessFunction(content string, target string) (string, []AdapterIssue) {
	lines := strings.Split(content, "\n")
	var issues []AdapterIssue

	for i, rawLine := range lines {
		masked := MaskShellLine(rawLine)

		if target == TargetZsh {
			if reExportF.MatchString(masked) {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  SchemaFunction,
					Target:  target,
					Message: fmt.Sprintf("Bash-specific function export 'export -f' in %q is not supported in Zsh", strings.TrimSpace(rawLine)),
				})
			}
		} else if target == TargetBash {
			if reZshFuncFlags.MatchString(masked) {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  SchemaFunction,
					Target:  target,
					Message: fmt.Sprintf("Zsh-specific function flags in %q are not supported in Bash", strings.TrimSpace(rawLine)),
				})
			}
		}
	}

	return content, issues
}

// ProcessProfile processes code fences with schema=profile for the target shell.
func ProcessProfile(content string, target string) (string, []AdapterIssue) {
	return processOptionCommands(content, target, SchemaProfile)
}

// ProcessRc processes code fences with schema=rc for the target shell.
func ProcessRc(content string, target string) (string, []AdapterIssue) {
	return processOptionCommands(content, target, SchemaRc)
}

func processOptionCommands(content string, target string, schemaName string) (string, []AdapterIssue) {
	lines := strings.Split(content, "\n")
	var issues []AdapterIssue

	for i, rawLine := range lines {
		masked := MaskShellLine(rawLine)

		if target == TargetBash {
			if reSetopt.MatchString(masked) {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  schemaName,
					Target:  target,
					Message: fmt.Sprintf("Zsh option command 'setopt/unsetopt' in %q is not supported in Bash", strings.TrimSpace(rawLine)),
				})
			}
		} else if target == TargetZsh {
			if reShopt.MatchString(masked) {
				issues = append(issues, AdapterIssue{
					Line:    i + 1,
					Schema:  schemaName,
					Target:  target,
					Message: fmt.Sprintf("Bash option command 'shopt' in %q is not supported in Zsh", strings.TrimSpace(rawLine)),
				})
			}
		}
	}

	return content, issues
}

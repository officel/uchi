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
)

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

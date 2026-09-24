package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// PortabilityIssue represents a detected non-portable shell syntax construct.
type PortabilityIssue struct {
	Line       int    // 1-based relative line number in snippet content
	Construct  string // Construct description, e.g. "[[ ... ]]"
	Confidence string // Confidence level, e.g. "high"
	Message    string // Diagnostic message
	Suggestion string // Workaround recommendation
}

func (p PortabilityIssue) Error() string {
	return fmt.Sprintf("non-portable Bash syntax %q detected (confidence: %s, suggestion: %s)",
		p.Construct, p.Confidence, p.Suggestion)
}

// IsPortableTarget reports whether the given target shells represent a portable target context.
// A target is considered portable if it is not strictly limited to Bash alone (e.g. target=bash).
func IsPortableTarget(targets []string) bool {
	if len(targets) == 1 && targets[0] == TargetBash {
		return false
	}
	return true
}

var (
	reDoubleBracket    = regexp.MustCompile(`\[\[`)
	reProcSubst        = regexp.MustCompile(`<\(|>\(`)
	reArrayAssignParen = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*=\(`)
	reArrayAssignIndex = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\[[^\]]+\]=`)
	reFunctionKeyword  = regexp.MustCompile(`\bfunction\s+[a-zA-Z_][a-zA-Z0-9_]*`)
)

// CheckPortability scans shell snippet content line-by-line for non-portable Bash-only constructs.
// It masks string literals and comments on each line prior to checking patterns to minimize false positives.
func CheckPortability(content string) []PortabilityIssue {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	var issues []PortabilityIssue

	for i, rawLine := range lines {
		lineNum := i + 1
		masked := MaskShellLine(rawLine)

		if reDoubleBracket.MatchString(masked) {
			issues = append(issues, PortabilityIssue{
				Line:       lineNum,
				Construct:  "[[ ... ]]",
				Confidence: "high",
				Message:    "non-portable Bash conditional construct '[[' detected",
				Suggestion: "use POSIX standard '[' or 'test'",
			})
		}

		if reProcSubst.MatchString(masked) {
			issues = append(issues, PortabilityIssue{
				Line:       lineNum,
				Construct:  "<(...) or >(...)",
				Confidence: "high",
				Message:    "non-portable Bash process substitution '<(...)' or '>(...)' detected",
				Suggestion: "use temporary files, named pipes (mkfifo), or standard pipelines",
			})
		}

		if reArrayAssignParen.MatchString(masked) || reArrayAssignIndex.MatchString(masked) {
			issues = append(issues, PortabilityIssue{
				Line:       lineNum,
				Construct:  "array assignment",
				Confidence: "high",
				Message:    "non-portable Bash array assignment detected",
				Suggestion: "use space-separated strings or positional parameters 'set --'",
			})
		}

		if reFunctionKeyword.MatchString(masked) {
			issues = append(issues, PortabilityIssue{
				Line:       lineNum,
				Construct:  "function keyword",
				Confidence: "high",
				Message:    "non-portable Bash 'function' keyword detected",
				Suggestion: "use POSIX standard 'name() { ... }' syntax",
			})
		}
	}

	return issues
}

// MaskShellLine masks contents of single-quoted strings, double-quoted strings,
// and comments in a single line of shell script to prevent false positives.
func MaskShellLine(line string) string {
	var sb strings.Builder
	sb.Grow(len(line))

	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			if inSingleQuote || inDoubleQuote {
				sb.WriteByte(' ')
			} else {
				sb.WriteByte(ch)
			}
			continue
		}

		if ch == '\\' && !inSingleQuote {
			escaped = true
			if inDoubleQuote {
				sb.WriteByte(' ')
			} else {
				sb.WriteByte(ch)
			}
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
				sb.WriteByte('\'')
			} else {
				sb.WriteByte(' ')
			}
			continue
		}

		if inDoubleQuote {
			if ch == '"' {
				inDoubleQuote = false
				sb.WriteByte('"')
			} else {
				sb.WriteByte(' ')
			}
			continue
		}

		if ch == '#' {
			isCommentStart := false
			if i == 0 {
				isCommentStart = true
			} else {
				prev := line[i-1]
				if prev == ' ' || prev == '\t' || prev == ';' || prev == '&' || prev == '|' || prev == '(' || prev == ')' || prev == '{' || prev == '}' {
					isCommentStart = true
				}
			}
			if isCommentStart {
				for j := i; j < len(line); j++ {
					sb.WriteByte(' ')
				}
				break
			}
		}

		if ch == '\'' {
			inSingleQuote = true
			sb.WriteByte('\'')
			continue
		}

		if ch == '"' {
			inDoubleQuote = true
			sb.WriteByte('"')
			continue
		}

		sb.WriteByte(ch)
	}

	return sb.String()
}

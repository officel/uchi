package schema

import (
	"strings"
	"testing"
)

func TestValidTargets(t *testing.T) {
	want := []string{"all", "bash", "fish", "powershell", "pwsh", "sh", "zsh"}
	got := ValidTargets()
	if len(got) != len(want) {
		t.Fatalf("ValidTargets() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidTargets()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsValidTarget(t *testing.T) {
	valid := ValidTargets()
	for _, target := range valid {
		if !IsValidTarget(target) {
			t.Errorf("IsValidTarget(%q) = false, want true", target)
		}
	}

	invalid := []string{"", "unknown", "BASH", "zsh1", "cmd"}
	for _, target := range invalid {
		if IsValidTarget(target) {
			t.Errorf("IsValidTarget(%q) = true, want false", target)
		}
	}
}

func TestValidSchemas(t *testing.T) {
	want := []string{"alias", "env", "function", "profile", "rc"}
	got := ValidSchemas()
	if len(got) != len(want) {
		t.Fatalf("ValidSchemas() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ValidSchemas()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsDefined(t *testing.T) {
	validSchemas := ValidSchemas()
	for _, s := range validSchemas {
		if !IsDefined(s) {
			t.Errorf("IsDefined(%q) = false, want true", s)
		}
	}

	invalidSchemas := []string{"", "unknown", "ALIAS", "shell", "bash"}
	for _, s := range invalidSchemas {
		if IsDefined(s) {
			t.Errorf("IsDefined(%q) = true, want false", s)
		}
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name       string
		schema     string
		input      string
		target     string
		want       string
		wantIssues int
		wantOk     bool
	}{
		{
			name:       "alias standard bash",
			schema:     "alias",
			input:      "alias ll='ls -la'",
			target:     TargetBash,
			want:       "alias ll='ls -la'",
			wantIssues: 0,
			wantOk:     true,
		},
		{
			name:       "alias zsh global alias on bash target fails",
			schema:     "alias",
			input:      "alias -g G='| grep'",
			target:     TargetBash,
			want:       "alias -g G='| grep'",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "alias zsh global alias on zsh target allowed",
			schema:     "alias",
			input:      "alias -g G='| grep'",
			target:     TargetZsh,
			want:       "alias -g G='| grep'",
			wantIssues: 0,
			wantOk:     true,
		},
		{
			name:       "env typeset -x conversion on bash target",
			schema:     "env",
			input:      "typeset -x FOO=bar\n  typeset -gx BAR=\"baz\"",
			target:     TargetBash,
			want:       "export FOO=bar\n  export BAR=\"baz\"",
			wantIssues: 0,
			wantOk:     true,
		},
		{
			name:       "env unsupported typeset flag on bash target",
			schema:     "env",
			input:      "typeset -T PATH path",
			target:     TargetBash,
			want:       "typeset -T PATH path",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "env export -n on zsh target fails",
			schema:     "env",
			input:      "export -n FOO",
			target:     TargetZsh,
			want:       "export -n FOO",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "function export -f on zsh target fails",
			schema:     "function",
			input:      "my_func() { echo hi; }\nexport -f my_func",
			target:     TargetZsh,
			want:       "my_func() { echo hi; }\nexport -f my_func",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "profile setopt on bash target fails",
			schema:     "profile",
			input:      "setopt AUTO_CD",
			target:     TargetBash,
			want:       "setopt AUTO_CD",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "rc shopt on zsh target fails",
			schema:     "rc",
			input:      "shopt -s globstar",
			target:     TargetZsh,
			want:       "shopt -s globstar",
			wantIssues: 1,
			wantOk:     true,
		},
		{
			name:       "all target maintains untouched output",
			schema:     "alias",
			input:      "alias -g G='| grep'",
			target:     TargetAll,
			want:       "alias -g G='| grep'",
			wantIssues: 0,
			wantOk:     true,
		},
		{
			name:       "undefined schema",
			schema:     "invalid",
			input:      "echo test",
			target:     TargetBash,
			want:       "",
			wantIssues: 0,
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, issues, ok := Process(tt.schema, tt.input, tt.target)
			if ok != tt.wantOk {
				t.Fatalf("Process(%q, %q, %q) ok = %v, want %v", tt.schema, tt.input, tt.target, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("Process(%q, %q, %q) content = %q, want %q", tt.schema, tt.input, tt.target, got, tt.want)
			}
			if len(issues) != tt.wantIssues {
				t.Errorf("Process(%q, %q, %q) issues count = %d, want %d", tt.schema, tt.input, tt.target, len(issues), tt.wantIssues)
			}
		})
	}
}

func TestAdapterIssueError(t *testing.T) {
	issue := AdapterIssue{
		Line:    3,
		Schema:  SchemaAlias,
		Target:  TargetBash,
		Message: "unsupported alias flag",
	}
	if issue.Error() != "unsupported alias flag" {
		t.Errorf("issue.Error() = %q, want %q", issue.Error(), "unsupported alias flag")
	}
}

func TestMaskedLinesInAdapters(t *testing.T) {
	// Comments should not trigger adapter issues
	commentInput := "# alias -g foo='bar'\n# setopt AUTO_CD\n# shopt -s globstar"
	_, issuesBash, _ := Process("alias", commentInput, TargetBash)
	if len(issuesBash) != 0 {
		t.Errorf("expected 0 issues for commentInput in bash, got %d", len(issuesBash))
	}

	_, issuesRc, _ := Process("rc", commentInput, TargetBash)
	if len(issuesRc) != 0 {
		t.Errorf("expected 0 issues for commentInput in rc, got %d", len(issuesRc))
	}

	// Double quote strings containing keywords should be masked
	stringInput := "echo \"setopt AUTO_CD\""
	_, issuesString, _ := Process("profile", stringInput, TargetBash)
	if len(issuesString) != 0 {
		t.Errorf("expected 0 issues for stringInput in profile, got %d", len(issuesString))
	}

	_ = strings.TrimSpace
}

package schema

import (
	"testing"
)

func TestIsPortableTarget(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		want    bool
	}{
		{"default all", []string{TargetAll}, true},
		{"single sh", []string{TargetSh}, true},
		{"single zsh", []string{TargetZsh}, true},
		{"multiple bash zsh", []string{TargetBash, TargetZsh}, true},
		{"multiple sh bash", []string{TargetSh, TargetBash}, true},
		{"strictly bash", []string{TargetBash}, false},
		{"empty targets", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPortableTarget(tt.targets)
			if got != tt.want {
				t.Errorf("IsPortableTarget(%v) = %v, want %v", tt.targets, got, tt.want)
			}
		})
	}
}

func TestMaskShellLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain code",
			input: "if [ $x = $y ]; then",
			want:  "if [ $x = $y ]; then",
		},
		{
			name:  "comment masked",
			input: "echo hello # comment containing [[ test ]]",
			want:  "echo hello                                ",
		},
		{
			name:  "single quotes masked",
			input: "echo '[[ text ]]' arr=(1 2)",
			want:  "echo '          ' arr=(1 2)",
		},
		{
			name:  "double quotes masked",
			input: "echo \"arr=(1 2)\" <(date)",
			want:  "echo \"         \" <(date)",
		},
		{
			name:  "hash inside quotes preserved as string content masked",
			input: "echo '# not a comment' # real comment",
			want:  "echo '               '               ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskShellLine(tt.input)
			if got != tt.want {
				t.Errorf("MaskShellLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckPortability(t *testing.T) {
	t.Run("detects double bracket", func(t *testing.T) {
		content := "alias x=y\nif [[ $a == $b ]]; then\n  echo hi\nfi"
		issues := CheckPortability(content)
		if len(issues) != 1 {
			t.Fatalf("len(issues) = %d, want 1", len(issues))
		}
		if issues[0].Line != 2 {
			t.Errorf("issues[0].Line = %d, want 2", issues[0].Line)
		}
		if issues[0].Construct != "[[ ... ]]" {
			t.Errorf("issues[0].Construct = %q, want %q", issues[0].Construct, "[[ ... ]]")
		}
		if issues[0].Confidence != "high" {
			t.Errorf("issues[0].Confidence = %q, want %q", issues[0].Confidence, "high")
		}
	})

	t.Run("detects process substitution", func(t *testing.T) {
		content := "diff <(cmd1) >(cmd2)"
		issues := CheckPortability(content)
		if len(issues) != 1 {
			t.Fatalf("len(issues) = %d, want 1", len(issues))
		}
		if issues[0].Line != 1 {
			t.Errorf("issues[0].Line = %d, want 1", issues[0].Line)
		}
		if issues[0].Construct != "<(...) or >(...)" {
			t.Errorf("issues[0].Construct = %q, want %q", issues[0].Construct, "<(...) or >(...)")
		}
	})

	t.Run("detects array assignment paren and index", func(t *testing.T) {
		content := "arr=(val1 val2)\narr[0]=val1"
		issues := CheckPortability(content)
		if len(issues) != 2 {
			t.Fatalf("len(issues) = %d, want 2", len(issues))
		}
		if issues[0].Line != 1 || issues[0].Construct != "array assignment" {
			t.Errorf("issue 0 = %v", issues[0])
		}
		if issues[1].Line != 2 || issues[1].Construct != "array assignment" {
			t.Errorf("issue 1 = %v", issues[1])
		}
	})

	t.Run("detects function keyword", func(t *testing.T) {
		content := "function my_fn() {\n  echo hi\n}"
		issues := CheckPortability(content)
		if len(issues) != 1 {
			t.Fatalf("len(issues) = %d, want 1", len(issues))
		}
		if issues[0].Line != 1 || issues[0].Construct != "function keyword" {
			t.Errorf("issue = %v", issues[0])
		}
	})

	t.Run("ignores valid POSIX shell code", func(t *testing.T) {
		content := "if [ \"$a\" = \"$b\" ]; then\n  echo hi\nfi\nmy_func() {\n  cat < file > out\n  set -- 1 2 3\n}"
		issues := CheckPortability(content)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for valid POSIX code, got %v", issues)
		}
	})

	t.Run("ignores false positives in comments and string literals", func(t *testing.T) {
		content := "# check [[ $x == $y ]]\n# arr=(1 2)\necho \"[[ test ]]\"\necho '<(cmd)'\nalias test='arr=(1 2)'"
		issues := CheckPortability(content)
		if len(issues) != 0 {
			t.Errorf("expected 0 issues for comments and string literals, got %v", issues)
		}
	})
}

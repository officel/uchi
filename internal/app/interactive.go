package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/officel/uchi/internal/color"
	"github.com/officel/uchi/internal/markdown"
	"github.com/officel/uchi/internal/schema"
)

// InteractiveSession holds the I/O streams and terminal status for interactive recovery.
type InteractiveSession struct {
	Reader     io.Reader
	Writer     io.Writer
	IsTerminal bool
	Colorizer  *color.Colorizer
}

// NewInteractiveSession initializes an interactive recovery session.
func NewInteractiveSession(r io.Reader, w io.Writer, colorMode string) *InteractiveSession {
	isTerm := false
	if f, ok := r.(*os.File); ok {
		fi, err := f.Stat()
		if err == nil {
			isTerm = (fi.Mode() & os.ModeCharDevice) != 0
		}
	}

	return &InteractiveSession{
		Reader:     r,
		Writer:     w,
		IsTerminal: isTerm,
		Colorizer:  color.New(colorMode, w),
	}
}

// DiagnosticCategory represents the type of issue found.
type DiagnosticCategory string

const (
	CategorySchema      DiagnosticCategory = "schema"
	CategoryShell       DiagnosticCategory = "shell"
	CategoryFrontmatter DiagnosticCategory = "frontmatter"
	CategoryConflict    DiagnosticCategory = "conflict"
	CategoryGeneric     DiagnosticCategory = "generic"
)

// DiagnosticGuide provides causes, fix suggestions, and documentation links.
type DiagnosticGuide struct {
	Category DiagnosticCategory
	Cause    string
	Fixes    []string
	DocLink  string
	Fixable  bool
}

// ExtractDiagnostics decomposes errors into individual markdown.Diagnostic pointers.
func ExtractDiagnostics(err error) []*markdown.Diagnostic {
	if err == nil {
		return nil
	}

	var result []*markdown.Diagnostic

	var diag *markdown.Diagnostic
	if errors.As(err, &diag) {
		// Single diagnostic or part of unwrap chain
		// Handle joined errors
		type multiErr interface {
			Unwrap() []error
		}
		if me, ok := err.(multiErr); ok {
			for _, sub := range me.Unwrap() {
				result = append(result, ExtractDiagnostics(sub)...)
			}
			return result
		}
		return []*markdown.Diagnostic{diag}
	}

	type multiErr interface {
		Unwrap() []error
	}
	if me, ok := err.(multiErr); ok {
		for _, sub := range me.Unwrap() {
			result = append(result, ExtractDiagnostics(sub)...)
		}
		return result
	}

	return []*markdown.Diagnostic{{Message: err.Error()}}
}

// ClassifyDiagnostic classifies a diagnostic error into a category and guide.
func ClassifyDiagnostic(d *markdown.Diagnostic) DiagnosticGuide {
	msg := d.Message

	switch {
	case strings.Contains(msg, "unknown schema") || strings.Contains(msg, "missing required 'schema' attribute"):
		return DiagnosticGuide{
			Category: CategorySchema,
			Cause:    "コードブロックのアノテーション内に定義済みの schema 属性が含まれていないか、未定義の schema 名が指定されています。",
			Fixes: []string{
				fmt.Sprintf("既知の schema (%s) から選択してアノテーション {schema=<名>} を更新する", strings.Join(schema.ValidSchemas(), ", ")),
				"タイポを修正する",
			},
			DocLink: "docs/format-v1.md#既知-schema",
			Fixable: d.Path != "" && d.Line > 0,
		}

	case strings.Contains(msg, "unknown target shell") || strings.Contains(msg, "unknown shell"):
		return DiagnosticGuide{
			Category: CategoryShell,
			Cause:    "コードブロックまたは設定で未対応のシェル名が指定されています。",
			Fixes: []string{
				fmt.Sprintf("許可されたシェル (%s) から有効なシェルを指定する", strings.Join(schema.ValidTargets(), ", ")),
			},
			DocLink: "docs/format-v1.md#アノテーション構文",
			Fixable: d.Path != "" && d.Line > 0,
		}

	case strings.Contains(msg, "frontmatter") || strings.Contains(msg, "uchi") || strings.Contains(msg, "YAML"):
		return DiagnosticGuide{
			Category: CategoryFrontmatter,
			Cause:    "Frontmatter の YAML 構文エラー、または `uchi: v1` の定義が不足・不適切です。",
			Fixes: []string{
				"ファイルの先頭に `---` で囲まれた `uchi: v1` frontmatter を追加・修正する",
			},
			DocLink: "docs/format-v1.md#frontmatter-仕様",
			Fixable: d.Path != "",
		}

	case strings.Contains(msg, "conflicting target file path") || strings.Contains(msg, "conflicts with reserved directory") || strings.Contains(msg, "conflicts with target subpath"):
		return DiagnosticGuide{
			Category: CategoryConflict,
			Cause:    "複数のドキュメントまたは schema が同じ出力ファイルパスにマッピングされているか、予約済みディレクトリ (`parts/`) と衝突しています。",
			Fixes: []string{
				"入力ファイル名または相対パスを変更して衝突を回避する",
				"重複している schema 指定を統一または分離する",
			},
			DocLink: "docs/format-v1.md#出力構造と出力ルール",
			Fixable: false,
		}

	default:
		return DiagnosticGuide{
			Category: CategoryGeneric,
			Cause:    "構文または属性の不備が検出されました。",
			Fixes: []string{
				"エラーメッセージと該当行を確認して記法を修正する",
			},
			DocLink: "docs/format-v1.md",
			Fixable: false,
		}
	}
}

// RunInteractiveRecovery guides the user through diagnostic recovery options.
func (sess *InteractiveSession) RunInteractiveRecovery(origErr error) error {
	if !sess.IsTerminal {
		fmt.Fprintf(sess.Writer, "%s\n", sess.Colorizer.Yellow("[interactive] 非対話環境 (stdin が端末ではありません) のため対話リカバリーをスキップします。"))
		return origErr
	}

	diags := ExtractDiagnostics(origErr)
	if len(diags) == 0 {
		return origErr
	}

	fmt.Fprintf(sess.Writer, "\n%s %d 件の診断エラーが検出されました。対話的リカバリーを開始します。\n\n",
		sess.Colorizer.Bold(sess.Colorizer.Cyan("[interactive]")), len(diags))

	scanner := bufio.NewScanner(sess.Reader)

	for idx, diag := range diags {
		guide := ClassifyDiagnostic(diag)

		fmt.Fprintf(sess.Writer, "--------------------------------------------------\n")
		fmt.Fprintf(sess.Writer, "%s [%d/%d] %s\n",
			sess.Colorizer.Bold("診断"), idx+1, len(diags), diag.Error())
		fmt.Fprintf(sess.Writer, "%s %s\n", sess.Colorizer.Yellow("原因:"), guide.Cause)
		fmt.Fprintf(sess.Writer, "%s %s\n", sess.Colorizer.Cyan("参照:"), guide.DocLink)

		for {
			fmt.Fprintf(sess.Writer, "\nアクションを選択してください:\n")
			fmt.Fprintf(sess.Writer, "  1. 修正案と解説を表示\n")
			if guide.Fixable {
				fmt.Fprintf(sess.Writer, "  2. 修正候補の差分を確認して自動修正を適用\n")
				fmt.Fprintf(sess.Writer, "  3. スキップして次へ\n")
			} else {
				fmt.Fprintf(sess.Writer, "  2. スキップして次へ\n")
			}
			fmt.Fprintf(sess.Writer, "選択 [1-%d]: ", map[bool]int{true: 3, false: 2}[guide.Fixable])

			if !scanner.Scan() {
				return origErr
			}
			input := strings.TrimSpace(scanner.Text())

			if input == "1" {
				fmt.Fprintf(sess.Writer, "\n%s\n", sess.Colorizer.Bold("【修正案】"))
				for i, fix := range guide.Fixes {
					fmt.Fprintf(sess.Writer, "  %d. %s\n", i+1, fix)
				}
				continue
			}

			if guide.Fixable && input == "2" {
				applied, err := sess.promptAndApplyFix(diag, guide, scanner)
				if err != nil {
					fmt.Fprintf(sess.Writer, "%s 修正の適用に失敗しました: %v\n", sess.Colorizer.Red("[エラー]"), err)
				} else if applied {
					fmt.Fprintf(sess.Writer, "%s 修正を適用しました。\n", sess.Colorizer.Green("[完了]"))
					break
				}
				continue
			}

			skipChoice := "2"
			if guide.Fixable {
				skipChoice = "3"
			}

			if input == skipChoice || input == "" {
				fmt.Fprintf(sess.Writer, "この診断をスキップします。\n")
				break
			}

			fmt.Fprintf(sess.Writer, "無効な選択肢です。\n")
		}
	}

	return nil
}

func (sess *InteractiveSession) promptAndApplyFix(diag *markdown.Diagnostic, guide DiagnosticGuide, scanner *bufio.Scanner) (bool, error) {
	if diag.Path == "" {
		return false, fmt.Errorf("対象ファイルのパスがありません")
	}

	contentBytes, err := os.ReadFile(diag.Path)
	if err != nil {
		return false, fmt.Errorf("ファイルの読み込みに失敗しました: %w", err)
	}
	oldContent := string(contentBytes)
	var newContent string

	switch guide.Category {
	case CategorySchema:
		fmt.Fprintf(sess.Writer, "\n適用する schema を選択してください (%s):\n", strings.Join(schema.ValidSchemas(), ", "))
		for i, s := range schema.ValidSchemas() {
			fmt.Fprintf(sess.Writer, "  %d. %s\n", i+1, s)
		}
		fmt.Fprintf(sess.Writer, "選択 [1-%d]: ", len(schema.ValidSchemas()))
		if !scanner.Scan() {
			return false, nil
		}
		sel := strings.TrimSpace(scanner.Text())
		var chosenSchema string
		for i, s := range schema.ValidSchemas() {
			if sel == fmt.Sprintf("%d", i+1) || sel == s {
				chosenSchema = s
				break
			}
		}
		if chosenSchema == "" {
			fmt.Fprintf(sess.Writer, "無効な schema が選択されました。\n")
			return false, nil
		}

		newContent, err = fixSchemaInContent(oldContent, diag.Line, chosenSchema)
		if err != nil {
			return false, err
		}

	case CategoryShell:
		fmt.Fprintf(sess.Writer, "\n適用する target shell を選択してください (%s):\n", strings.Join(schema.ValidTargets(), ", "))
		for i, t := range schema.ValidTargets() {
			fmt.Fprintf(sess.Writer, "  %d. %s\n", i+1, t)
		}
		fmt.Fprintf(sess.Writer, "選択 [1-%d]: ", len(schema.ValidTargets()))
		if !scanner.Scan() {
			return false, nil
		}
		sel := strings.TrimSpace(scanner.Text())
		var chosenShell string
		for i, t := range schema.ValidTargets() {
			if sel == fmt.Sprintf("%d", i+1) || sel == t {
				chosenShell = t
				break
			}
		}
		if chosenShell == "" {
			fmt.Fprintf(sess.Writer, "無効な target shell が選択されました。\n")
			return false, nil
		}

		newContent, err = fixShellInContent(oldContent, diag.Line, chosenShell)
		if err != nil {
			return false, err
		}

	case CategoryFrontmatter:
		newContent = fixFrontmatterInContent(oldContent)

	default:
		return false, fmt.Errorf("この問題に対する自動修正機能は未実装です")
	}

	if oldContent == newContent {
		fmt.Fprintf(sess.Writer, "変更箇所はありませんでした。\n")
		return false, nil
	}

	// Display line diff
	fmt.Fprintf(sess.Writer, "\n%s\n", sess.Colorizer.Bold("【修正差分】"))
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	ops := computeLineDiff(oldLines, newLines)

	for _, op := range ops {
		switch op.Kind {
		case DiffEqual:
			fmt.Fprintf(sess.Writer, "  %s\n", op.Line)
		case DiffDelete:
			fmt.Fprintf(sess.Writer, "%s\n", sess.Colorizer.Red("- "+op.Line))
		case DiffInsert:
			fmt.Fprintf(sess.Writer, "%s\n", sess.Colorizer.Green("+ "+op.Line))
		}
	}

	// Confirmation prompt
	fmt.Fprintf(sess.Writer, "\nこの修正を %s に適用しますか？ [y/N]: ", diag.Path)
	if !scanner.Scan() {
		return false, nil
	}
	confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Fprintf(sess.Writer, "修正の適用をキャンセルしました。\n")
		return false, nil
	}

	// Atomic write
	if err := writeFileAtomic(diag.Path, []byte(newContent), 0644); err != nil {
		return false, fmt.Errorf("ファイルの書き込みに失敗しました: %w", err)
	}

	return true, nil
}

func fixSchemaInContent(content string, lineNum int, newSchema string) (string, error) {
	lines := splitLines(content)
	if lineNum < 1 || lineNum > len(lines) {
		return "", fmt.Errorf("対象行 (行番号 %d) が見つかりません", lineNum)
	}

	idx := lineNum - 1
	line := lines[idx]

	if !strings.Contains(line, "```") {
		return "", fmt.Errorf("指定行 (行番号 %d) はコードフェンスヘッダーではありません", lineNum)
	}

	// Handle missing schema attribute or replacing existing schema
	if strings.Contains(line, "schema=") {
		// Replace existing schema value
		start := strings.Index(line, "schema=")
		end := start + len("schema=")
		for end < len(line) && line[end] != ' ' && line[end] != '}' && line[end] != '\t' {
			end++
		}
		lines[idx] = line[:start] + "schema=" + newSchema + line[end:]
	} else if strings.Contains(line, "{") {
		// Insert schema into existing attribute block
		braceIdx := strings.Index(line, "{")
		lines[idx] = line[:braceIdx+1] + "schema=" + newSchema + " " + line[braceIdx+1:]
	} else {
		// Add attribute block
		lines[idx] = line + " {schema=" + newSchema + "}"
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func fixShellInContent(content string, lineNum int, newShell string) (string, error) {
	lines := splitLines(content)
	if lineNum < 1 || lineNum > len(lines) {
		return "", fmt.Errorf("対象行 (行番号 %d) が見つかりません", lineNum)
	}

	idx := lineNum - 1
	line := lines[idx]

	if !strings.Contains(line, "```") {
		return "", fmt.Errorf("指定行 (行番号 %d) はコードフェンスヘッダーではありません", lineNum)
	}

	if strings.Contains(line, "target=") {
		start := strings.Index(line, "target=")
		end := start + len("target=")
		for end < len(line) && line[end] != ' ' && line[end] != '}' && line[end] != '\t' {
			end++
		}
		lines[idx] = line[:start] + "target=" + newShell + line[end:]
	} else if strings.Contains(line, "{") {
		braceIdx := strings.Index(line, "{")
		lines[idx] = line[:braceIdx+1] + "target=" + newShell + " " + line[braceIdx+1:]
	} else {
		lines[idx] = line + " {target=" + newShell + "}"
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func fixFrontmatterInContent(content string) string {
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "---") {
		lines := splitLines(content)
		if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
			// Find closing ---
			closingIdx := -1
			for i := 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) == "---" {
					closingIdx = i
					break
				}
			}

			if closingIdx != -1 {
				// Frontmatter block exists but missing uchi: v1 or malformed YAML
				var newFmLines []string
				newFmLines = append(newFmLines, "---", "uchi: v1")
				for i := 1; i < closingIdx; i++ {
					l := lines[i]
					if !strings.HasPrefix(strings.TrimSpace(l), "uchi:") {
						newFmLines = append(newFmLines, l)
					}
				}
				newFmLines = append(newFmLines, "---")
				rest := lines[closingIdx+1:]
				allLines := append(newFmLines, rest...)
				return strings.Join(allLines, "\n") + "\n"
			}
		}
	}

	// No frontmatter present: prepend valid uchi: v1 frontmatter
	fm := "---\nuchi: v1\n---\n\n"
	return fm + content
}

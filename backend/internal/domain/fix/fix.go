// Package fix provides composable fix method tree for text replacement.
// Each Node implements Fix(matchStr, ctx) — ctx carries surrounding text,
// span, and detect capture groups (B1/E4).
package fix

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Error represents a fix package error.
type Error struct {
	Op      string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("fix.%s: %s", e.Op, e.Message)
}

// Context carries match environment for context-aware and capture-group fixes (B1/E4).
type Context struct {
	Text   string   // text unit (usually paragraph) containing the match
	Start  int      // match start in Text
	End    int      // match end in Text (exclusive)
	Groups []string // detect regex submatches: [0]=full, [1]=$1, …
}

// Before returns up to n bytes immediately before the match.
func (c Context) Before(n int) string {
	if n <= 0 || c.Start <= 0 {
		return ""
	}
	from := c.Start - n
	if from < 0 {
		from = 0
	}
	return c.Text[from:c.Start]
}

// After returns up to n bytes immediately after the match.
func (c Context) After(n int) string {
	if n <= 0 || c.End >= len(c.Text) {
		return ""
	}
	to := c.End + n
	if to > len(c.Text) {
		to = len(c.Text)
	}
	return c.Text[c.End:to]
}

// Node is the fix interface.
type Node interface {
	Fix(matchStr string, ctx Context) string
}

// ---------------------------------------------------------------------------
//  Registry
// ---------------------------------------------------------------------------

type Builder func(args map[string]interface{}, children []Node) (Node, error)

var registry = map[string]Builder{}

// Register registers a fix method builder.
func Register(name string, b Builder) { registry[name] = b }

// Build constructs a Node from a fix method descriptor.
func Build(method string, args map[string]interface{}, children []Node) (Node, error) {
	b, ok := registry[method]
	if !ok {
		return nil, &Error{Op: "Build", Message: "unknown fix method: " + method}
	}
	return b(args, children)
}

// Registered returns all registered method names.
func Registered() []string {
	var names []string
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// ---------------------------------------------------------------------------
//  Helpers
// ---------------------------------------------------------------------------

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(args map[string]interface{}, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return def
}

func strSliceArg(args map[string]interface{}, key string) []string {
	if v, ok := args[key]; ok {
		switch vs := v.(type) {
		case []string:
			return vs
		case []interface{}:
			out := make([]string, 0, len(vs))
			for _, item := range vs {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// expandGroups substitutes $0/$1/${1} from detect capture groups into template.
func expandGroups(template string, groups []string) string {
	if len(groups) == 0 {
		return template
	}
	var b strings.Builder
	b.Grow(len(template))
	for i := 0; i < len(template); i++ {
		if template[i] != '$' || i+1 >= len(template) {
			b.WriteByte(template[i])
			continue
		}
		j := i + 1
		if template[j] == '{' {
			j++
			k := j
			for k < len(template) && template[k] >= '0' && template[k] <= '9' {
				k++
			}
			if k < len(template) && template[k] == '}' && k > j {
				idx, _ := strconv.Atoi(template[j:k])
				if idx >= 0 && idx < len(groups) {
					b.WriteString(groups[idx])
					i = k
					continue
				}
			}
			b.WriteByte('$')
			continue
		}
		k := j
		for k < len(template) && template[k] >= '0' && template[k] <= '9' {
			k++
		}
		if k > j {
			idx, _ := strconv.Atoi(template[j:k])
			if idx >= 0 && idx < len(groups) {
				b.WriteString(groups[idx])
				i = k - 1
				continue
			}
		}
		b.WriteByte('$')
	}
	return b.String()
}

// ---------------------------------------------------------------------------
//  Built-in fix methods
// ---------------------------------------------------------------------------

// ReplaceNode replaces match with a fixed string.
type ReplaceNode struct {
	With string
}

func (n *ReplaceNode) Fix(matchStr string, ctx Context) string {
	return n.With
}

// RemoveNode removes the match entirely.
type RemoveNode struct{}

func (n *RemoveNode) Fix(matchStr string, ctx Context) string {
	return ""
}

// RegexReplaceNode applies regex replacement on the match string.
// If detect capture groups are present (len(Groups) > 1), replacement is expanded
// with those groups ($1, $2, …) instead of re-matching (E4).
// Always operates on the match span only — never on the full document (B3).
type RegexReplaceNode struct {
	Pattern     *regexp.Regexp
	Replacement string
}

func (n *RegexReplaceNode) Fix(matchStr string, ctx Context) string {
	if len(ctx.Groups) > 1 {
		return expandGroups(n.Replacement, ctx.Groups)
	}
	if n.Pattern == nil {
		return matchStr
	}
	return n.Pattern.ReplaceAllString(matchStr, n.Replacement)
}

// UppercaseNode uppercases the entire match.
type UppercaseNode struct{}

func (n *UppercaseNode) Fix(matchStr string, ctx Context) string {
	return strings.ToUpper(matchStr)
}

// LowercaseNode lowercases the entire match.
type LowercaseNode struct{}

func (n *LowercaseNode) Fix(matchStr string, ctx Context) string {
	return strings.ToLower(matchStr)
}

// CapitalizeNode uppercases first letter, keeps rest.
type CapitalizeNode struct{}

func (n *CapitalizeNode) Fix(matchStr string, ctx Context) string {
	return capitalizeFirst(matchStr)
}

// SentenceCaseNode capitalizes first letter, lowercases rest.
type SentenceCaseNode struct{}

func (n *SentenceCaseNode) Fix(matchStr string, ctx Context) string {
	if matchStr == "" {
		return ""
	}
	first, size := utf8.DecodeRuneInString(matchStr)
	rest := matchStr[size:]
	return string(unicode.ToUpper(first)) + strings.ToLower(rest)
}

// TitleCaseNode capitalizes first letter of each word.
type TitleCaseNode struct{}

func (n *TitleCaseNode) Fix(matchStr string, ctx Context) string {
	return cases.Title(language.Und).String(matchStr)
}

// PrependNode adds text before the match (preserving match).
type PrependNode struct {
	With string
}

func (n *PrependNode) Fix(matchStr string, ctx Context) string {
	return n.With + matchStr
}

// AppendNode adds text after the match (preserving match).
type AppendNode struct {
	With string
}

func (n *AppendNode) Fix(matchStr string, ctx Context) string {
	return matchStr + n.With
}

// WrapNode adds prefix and suffix around the match.
type WrapNode struct {
	Prefix string
	Suffix string
}

func (n *WrapNode) Fix(matchStr string, ctx Context) string {
	return n.Prefix + matchStr + n.Suffix
}

// TrimNode removes leading and trailing whitespace.
type TrimNode struct{}

func (n *TrimNode) Fix(matchStr string, ctx Context) string {
	return strings.TrimSpace(matchStr)
}

// CollapseWhitespaceNode collapses multiple whitespace chars into single space.
type CollapseWhitespaceNode struct{}

var wsRe = regexp.MustCompile(`\s+`)

func (n *CollapseWhitespaceNode) Fix(matchStr string, ctx Context) string {
	return wsRe.ReplaceAllString(matchStr, " ")
}

// AndNode applies all children sequentially.
type AndNode struct {
	Children []Node
}

func (n *AndNode) Fix(matchStr string, ctx Context) string {
	result := matchStr
	for _, child := range n.Children {
		result = child.Fix(result, ctx)
	}
	return result
}

// WhenNode applies the Then fix only if Condition detects a match on the match string.
type WhenNode struct {
	Condition detect.Node
	Then      Node
}

func (n *WhenNode) Fix(matchStr string, ctx Context) string {
	if n.Condition != nil {
		matches := n.Condition.Detect(matchStr)
		if len(matches) == 0 {
			return matchStr
		}
	}
	if n.Then != nil {
		return n.Then.Fix(matchStr, ctx)
	}
	return matchStr
}

// ---------------------------------------------------------------------------
//  init — register built-in fix methods
// ---------------------------------------------------------------------------

func init() {
	Register("replace", func(args map[string]interface{}, children []Node) (Node, error) {
		with := strArg(args, "with")
		if with == "" && args["with"] == nil {
			return nil, &Error{Op: "replace", Message: "with is required"}
		}
		return &ReplaceNode{With: with}, nil
	})

	Register("remove", func(args map[string]interface{}, children []Node) (Node, error) {
		return &RemoveNode{}, nil
	})

	Register("regex_replace", func(args map[string]interface{}, children []Node) (Node, error) {
		pattern := strArg(args, "pattern")
		replacement := strArg(args, "replacement")
		if pattern == "" && replacement == "" {
			return nil, &Error{Op: "regex_replace", Message: "pattern or replacement is required"}
		}
		var re *regexp.Regexp
		if pattern != "" {
			var err error
			re, err = regexp.Compile(pattern)
			if err != nil {
				return nil, &Error{Op: "regex_replace", Message: "invalid RE2 pattern: " + err.Error()}
			}
		}
		return &RegexReplaceNode{Pattern: re, Replacement: replacement}, nil
	})

	Register("uppercase", func(args map[string]interface{}, children []Node) (Node, error) {
		return &UppercaseNode{}, nil
	})

	Register("lowercase", func(args map[string]interface{}, children []Node) (Node, error) {
		return &LowercaseNode{}, nil
	})

	Register("capitalize", func(args map[string]interface{}, children []Node) (Node, error) {
		return &CapitalizeNode{}, nil
	})

	Register("sentence_case", func(args map[string]interface{}, children []Node) (Node, error) {
		return &SentenceCaseNode{}, nil
	})

	Register("title_case", func(args map[string]interface{}, children []Node) (Node, error) {
		return &TitleCaseNode{}, nil
	})

	Register("prepend", func(args map[string]interface{}, children []Node) (Node, error) {
		with := strArg(args, "with")
		return &PrependNode{With: with}, nil
	})

	Register("append", func(args map[string]interface{}, children []Node) (Node, error) {
		with := strArg(args, "with")
		return &AppendNode{With: with}, nil
	})

	Register("wrap", func(args map[string]interface{}, children []Node) (Node, error) {
		prefix := strArg(args, "prefix")
		suffix := strArg(args, "suffix")
		return &WrapNode{Prefix: prefix, Suffix: suffix}, nil
	})

	Register("trim", func(args map[string]interface{}, children []Node) (Node, error) {
		return &TrimNode{}, nil
	})

	Register("collapse_whitespace", func(args map[string]interface{}, children []Node) (Node, error) {
		return &CollapseWhitespaceNode{}, nil
	})

	Register("and", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "and", Message: "requires at least 1 child node"}
		}
		return &AndNode{Children: children}, nil
	})
}

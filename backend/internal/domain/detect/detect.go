// Package detect provides a composable detection method tree.
// Each Node independently scans text and returns MatchRanges.
// Logical nodes (And, Or, Not) combine results from child nodes.
// See SPEC §8, A26, Q33 for the method system design.
package detect

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Error represents a detect package error.
type Error struct {
	Op      string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("detect.%s: %s", e.Op, e.Message)
}

// MatchRange represents a text match position [Start, End).
// Start is inclusive, End is exclusive (like Go slice bounds).
type MatchRange struct {
	Start int
	End   int
}

// Node is the detection interface. Every detection method implements this.
// All methods are self-contained: Detect(text) returns matches for that method.
// Logical methods (And/Or/Not) combine child results via set operations.
type Node interface {
	Detect(text string) []MatchRange
}

// ---------------------------------------------------------------------------
//  Registry — maps method YAML names to builders
// ---------------------------------------------------------------------------

type Builder func(args map[string]interface{}, children []Node) (Node, error)

var registry = map[string]Builder{}

// Register registers a detection method builder.
func Register(name string, b Builder) { registry[name] = b }

// Build constructs a Node from a YAML method descriptor.
func Build(method string, args map[string]interface{}, children []Node) (Node, error) {
	b, ok := registry[method]
	if !ok {
		return nil, &Error{Op: "Build", Message: "unknown detect method: " + method}
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

// wordBoundary checks that [start, end) is not surrounded by Unicode letters.
func wordBoundary(s string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:start])
		if unicode.IsLetter(r) {
			return false
		}
	}
	if end < len(s) {
		r, _ := utf8.DecodeRuneInString(s[end:])
		if unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// capitalizeFirst uppercases the first Unicode letter of s.
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// intersectRanges returns ranges from a that have an exact-match in b.
func intersectRanges(a, b []MatchRange) []MatchRange {
	set := make(map[[2]int]struct{}, len(b))
	for _, r := range b {
		set[[2]int{r.Start, r.End}] = struct{}{}
	}
	var out []MatchRange
	for _, r := range a {
		if _, ok := set[[2]int{r.Start, r.End}]; ok {
			out = append(out, r)
		}
	}
	return out
}

// unionRanges merges all ranges from a and b, deduped.
func unionRanges(a, b []MatchRange) []MatchRange {
	set := make(map[[2]int]struct{}, len(a)+len(b))
	for _, r := range a {
		set[[2]int{r.Start, r.End}] = struct{}{}
	}
	for _, r := range b {
		set[[2]int{r.Start, r.End}] = struct{}{}
	}
	out := make([]MatchRange, 0, len(set))
	for k := range set {
		out = append(out, MatchRange{Start: k[0], End: k[1]})
	}
	return out
}

// subtractRanges returns ranges from a that do not appear in b.
func subtractRanges(a, b []MatchRange) []MatchRange {
	set := make(map[[2]int]struct{}, len(b))
	for _, r := range b {
		set[[2]int{r.Start, r.End}] = struct{}{}
	}
	var out []MatchRange
	for _, r := range a {
		if _, ok := set[[2]int{r.Start, r.End}]; !ok {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
//  sentenceDetector — shared logic for sentence/paragraph boundaries
// ---------------------------------------------------------------------------

// sentenceBoundaries finds positions that end a sentence (. ! ? at end of word).
func sentenceBoundaries(text string) []int {
	re := regexp.MustCompile(`[.!?](?:\s|$)`)
	matches := re.FindAllStringIndex(text, -1)
	bounds := make([]int, 0, len(matches)+1)
	for _, m := range matches {
		// The match includes the period: . or ? or ! — end is m[1]
		// But we want the position of the punctuation: m[1]-1 or wherever
		bounds = append(bounds, m[0]+1) // after the punctuation
	}
	return bounds
}

// ---------------------------------------------------------------------------
//  String helpers for args parsing
// ---------------------------------------------------------------------------

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
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

func boolArg(args map[string]interface{}, key string, def bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
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

// AndNode — intersection of all child results.
type AndNode struct {
	Children []Node
}

func (n *AndNode) Detect(text string) []MatchRange {
	if len(n.Children) == 0 {
		return nil
	}

	var result []MatchRange
	var exclusions []MatchRange

	for _, child := range n.Children {
		if not, ok := child.(*NotNode); ok {
			// NotNode's Detect returns inner matches (what to exclude)
			exclusions = append(exclusions, not.Children[0].Detect(text)...)
			continue
		}
		matches := child.Detect(text)
		if result == nil {
			result = matches
		} else {
			result = intersectRanges(result, matches)
		}
		if len(result) == 0 {
			return nil
		}
	}

	if len(exclusions) > 0 {
		result = subtractRanges(result, exclusions)
	}
	return result
}

// OrNode — union of all child results.
type OrNode struct {
	Children []Node
}

func (n *OrNode) Detect(text string) []MatchRange {
	var result []MatchRange
	for _, child := range n.Children {
		matches := child.Detect(text)
		result = unionRanges(result, matches)
	}
	return result
}

// NotNode — wraps a single child for exclusion in And context.
// When used standalone, returns child's matches (identical to child).
type NotNode struct {
	Children []Node
}

func (n *NotNode) Detect(text string) []MatchRange {
	if len(n.Children) == 0 {
		return nil
	}
	return n.Children[0].Detect(text)
}

// RegexNode matches text using a compiled RE2 regexp.
type RegexNode struct {
	Pattern *regexp.Regexp
}

func (n *RegexNode) Detect(text string) []MatchRange {
	matches := n.Pattern.FindAllStringSubmatchIndex(text, -1)
	if matches == nil {
		return nil
	}
	out := make([]MatchRange, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		start, end := m[0], m[1]
		if start == end {
			continue
		}
		out = append(out, MatchRange{Start: start, End: end})
	}
	return out
}

// WordlistNode matches words from a list with word boundary checks.
type WordlistNode struct {
	Words         []string
	CaseSensitive bool
}

func (n *WordlistNode) Detect(text string) []MatchRange {
	if len(n.Words) == 0 || text == "" {
		return nil
	}

	var out []MatchRange
	searchText := text
	if !n.CaseSensitive {
		searchText = strings.ToLower(text)
	}

	for _, word := range n.Words {
		searchWord := word
		if !n.CaseSensitive {
			searchWord = strings.ToLower(word)
		}

		offset := 0
		for {
			idx := strings.Index(searchText[offset:], searchWord)
			if idx == -1 {
				break
			}
			start := offset + idx
			end := start + len(word)

			if !wordBoundary(searchText, start, end) {
				offset = start + 1
				continue
			}

			out = append(out, MatchRange{Start: start, End: end})
			offset = end
		}
	}
	return out
}

// ContainsNode matches if text contains value as substring.
type ContainsNode struct {
	Value         string
	CaseSensitive bool
}

func (n *ContainsNode) Detect(text string) []MatchRange {
	if n.Value == "" {
		return nil
	}
	searchText := text
	searchVal := n.Value
	if !n.CaseSensitive {
		searchText = strings.ToLower(text)
		searchVal = strings.ToLower(n.Value)
	}

	var out []MatchRange
	offset := 0
	for {
		idx := strings.Index(searchText[offset:], searchVal)
		if idx == -1 {
			break
		}
		start := offset + idx
		end := start + len(n.Value)
		out = append(out, MatchRange{Start: start, End: end})
		offset = end
	}
	return out
}

// EqNode matches exact equality of the string.
type EqNode struct {
	Value         string
	CaseSensitive bool
}

func (n *EqNode) Detect(text string) []MatchRange {
	if n.Value == "" {
		return nil
	}
	var match bool
	if n.CaseSensitive {
		match = text == n.Value
	} else {
		match = strings.EqualFold(text, n.Value)
	}
	if match {
		return []MatchRange{{Start: 0, End: len(text)}}
	}
	return nil
}

// PrefixNode matches start of string.
type PrefixNode struct {
	Value         string
	CaseSensitive bool
}

func (n *PrefixNode) Detect(text string) []MatchRange {
	if n.Value == "" {
		return nil
	}
	var match bool
	if n.CaseSensitive {
		match = strings.HasPrefix(text, n.Value)
	} else {
		match = strings.HasPrefix(strings.ToLower(text), strings.ToLower(n.Value))
	}
	if match {
		return []MatchRange{{Start: 0, End: len(n.Value)}}
	}
	return nil
}

// SuffixNode matches end of string.
type SuffixNode struct {
	Value         string
	CaseSensitive bool
}

func (n *SuffixNode) Detect(text string) []MatchRange {
	if n.Value == "" {
		return nil
	}
	var match bool
	if n.CaseSensitive {
		match = strings.HasSuffix(text, n.Value)
	} else {
		match = strings.HasSuffix(strings.ToLower(text), strings.ToLower(n.Value))
	}
	if match {
		return []MatchRange{{Start: len(text) - len(n.Value), End: len(text)}}
	}
	return nil
}

// SentenceStartNode matches beginning of each sentence.
type SentenceStartNode struct{}

func (n *SentenceStartNode) Detect(text string) []MatchRange {
	if text == "" {
		return nil
	}
	// First sentence always starts at 0
	out := []MatchRange{{Start: 0, End: 0}}

	bounds := sentenceBoundaries(text)
	for _, pos := range bounds {
		// Skip trailing whitespace after punctuation
		skip := pos
		for skip < len(text) && (text[skip] == ' ' || text[skip] == '\t' || text[skip] == '\n' || text[skip] == '\r') {
			skip++
		}
		if skip < len(text) {
			out = append(out, MatchRange{Start: skip, End: skip})
		}
	}
	return out
}

// SentenceEndNode matches end of each sentence.
type SentenceEndNode struct{}

func (n *SentenceEndNode) Detect(text string) []MatchRange {
	if text == "" {
		return nil
	}
	re := regexp.MustCompile(`[.!?]`)
	matches := re.FindAllStringIndex(text, -1)
	out := make([]MatchRange, 0, len(matches))
	for _, m := range matches {
		// End at the punctuation character
		out = append(out, MatchRange{Start: m[0], End: m[1]})
	}
	return out
}

// ParagraphStartNode matches beginning of each paragraph (after \n\n).
type ParagraphStartNode struct{}

func (n *ParagraphStartNode) Detect(text string) []MatchRange {
	if text == "" {
		return nil
	}
	out := []MatchRange{{Start: 0, End: 0}}
	re := regexp.MustCompile(`\n\n+`)
	matches := re.FindAllStringIndex(text, -1)
	for _, m := range matches {
		pos := m[1] // after the newlines
		if pos < len(text) {
			out = append(out, MatchRange{Start: pos, End: pos})
		}
	}
	return out
}

// ParagraphEndNode matches end of each paragraph (before \n\n).
type ParagraphEndNode struct{}

func (n *ParagraphEndNode) Detect(text string) []MatchRange {
	if text == "" {
		return nil
	}
	re := regexp.MustCompile(`\n\n+`)
	matches := re.FindAllStringIndex(text, -1)
	out := make([]MatchRange, 0, len(matches)+1)
	for _, m := range matches {
		// End before the newlines
		out = append(out, MatchRange{Start: m[0], End: m[0]})
	}
	return out
}

// WordBoundaryNode matches word boundaries (zero-length positions between words).
type WordBoundaryNode struct{}

func (n *WordBoundaryNode) Detect(text string) []MatchRange {
	if text == "" {
		return nil
	}
	var out []MatchRange
	// Start boundary of first word
	if len(text) > 0 && unicode.IsLetter(rune(text[0])) {
		out = append(out, MatchRange{Start: 0, End: 0})
	}
	inWord := unicode.IsLetter(rune(text[0]))
	for i := 1; i < len(text); i++ {
		r := rune(text[i])
		isLetter := unicode.IsLetter(r)
		if inWord != isLetter {
			out = append(out, MatchRange{Start: i, End: i})
			inWord = isLetter
		}
	}
	return out
}

// LengthNode matches text ranges whose length falls within [Min, Max].
// Scans for non-whitespace runs (word-level granularity).
type LengthNode struct {
	Min int
	Max int // 0 = no limit
}

func (n *LengthNode) Detect(text string) []MatchRange {
	wordRe := regexp.MustCompile(`\S+`)
	matches := wordRe.FindAllStringIndex(text, -1)
	var out []MatchRange
	for _, m := range matches {
		length := m[1] - m[0]
		if length >= n.Min && (n.Max == 0 || length <= n.Max) {
			out = append(out, MatchRange{Start: m[0], End: m[1]})
		}
	}
	return out
}

// CaseMode represents case matching mode for CaseNode.
type CaseMode string

const (
	CaseAllCaps    CaseMode = "all_caps"
	CaseAllLower   CaseMode = "all_lower"
	CaseCapitalized CaseMode = "capitalized"
	CaseHasUpper   CaseMode = "has_upper"
	CaseHasLower   CaseMode = "has_lower"
)

// CaseNode matches text ranges with specific case properties.
type CaseNode struct {
	Mode CaseMode
}

func (n *CaseNode) Detect(text string) []MatchRange {
	wordRe := regexp.MustCompile(`\S+`)
	matches := wordRe.FindAllStringIndex(text, -1)
	var out []MatchRange

	for _, m := range matches {
		word := text[m[0]:m[1]]
		if checkCase(n.Mode, word) {
			out = append(out, MatchRange{Start: m[0], End: m[1]})
		}
	}
	return out
}

func checkCase(mode CaseMode, s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	switch mode {
	case CaseAllCaps:
		for _, r := range runes {
			if unicode.IsLetter(r) && !unicode.IsUpper(r) {
				return false
			}
		}
		return true
	case CaseAllLower:
		for _, r := range runes {
			if unicode.IsLetter(r) && !unicode.IsLower(r) {
				return false
			}
		}
		return true
	case CaseCapitalized:
		if !unicode.IsUpper(runes[0]) {
			return false
		}
		for _, r := range runes[1:] {
			if unicode.IsLetter(r) && unicode.IsUpper(r) {
				return false
			}
		}
		return true
	case CaseHasUpper:
		for _, r := range runes {
			if unicode.IsUpper(r) {
				return true
			}
		}
		return false
	case CaseHasLower:
		for _, r := range runes {
			if unicode.IsLower(r) {
				return true
			}
		}
		return false
	}
	return false
}

// BeforeNode matches the character immediately before each match of the inner node.
type BeforeNode struct {
	Inner Node
}

func (n *BeforeNode) Detect(text string) []MatchRange {
	inner := n.Inner.Detect(text)
	var out []MatchRange
	for _, m := range inner {
		if m.Start > 0 {
			out = append(out, MatchRange{Start: m.Start - 1, End: m.Start})
		}
	}
	return out
}

// AfterNode matches the character immediately after each match of the inner node.
type AfterNode struct {
	Inner Node
}

func (n *AfterNode) Detect(text string) []MatchRange {
	inner := n.Inner.Detect(text)
	var out []MatchRange
	for _, m := range inner {
		if m.End < len(text) {
			out = append(out, MatchRange{Start: m.End, End: m.End + 1})
		}
	}
	return out
}

// SurroundedByNode matches text between left and right markers.
type SurroundedByNode struct {
	Left  string
	Right string
}

func (n *SurroundedByNode) Detect(text string) []MatchRange {
	if n.Left == "" || n.Right == "" {
		return nil
	}
	var out []MatchRange
	offset := 0
	for {
		leftIdx := strings.Index(text[offset:], n.Left)
		if leftIdx == -1 {
			break
		}
		leftEnd := offset + leftIdx + len(n.Left)
		rightIdx := strings.Index(text[leftEnd:], n.Right)
		if rightIdx == -1 {
			break
		}
		rightStart := leftEnd + rightIdx
		out = append(out, MatchRange{Start: leftEnd, End: rightStart})
		offset = rightStart + len(n.Right)
	}
	return out
}

// PositionNode matches at specific character position range.
type PositionNode struct {
	From int
	To   int // 0 = end of text
}

func (n *PositionNode) Detect(text string) []MatchRange {
	if n.From < 0 {
		return nil
	}
	to := n.To
	if to == 0 || to > len(text) {
		to = len(text)
	}
	if n.From >= to || n.From >= len(text) {
		return nil
	}
	return []MatchRange{{Start: n.From, End: to}}
}

// ---------------------------------------------------------------------------
//  init — register all built-in methods
// ---------------------------------------------------------------------------

func init() {
	Register("regex", func(args map[string]interface{}, children []Node) (Node, error) {
		pattern := strArg(args, "pattern")
		if pattern == "" {
			return nil, &Error{Op: "regex", Message: "pattern is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		compilePattern := pattern
		if !caseSensitive {
			compilePattern = "(?i)" + compilePattern
		}
		re, err := regexp.Compile(compilePattern)
		if err != nil {
			return nil, &Error{Op: "regex", Message: "invalid RE2 pattern: " + err.Error()}
		}
		return &RegexNode{Pattern: re}, nil
	})

	Register("wordlist", func(args map[string]interface{}, children []Node) (Node, error) {
		words := strSliceArg(args, "list")
		if len(words) == 0 {
			return nil, &Error{Op: "wordlist", Message: "list is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &WordlistNode{Words: words, CaseSensitive: caseSensitive}, nil
	})

	Register("contains", func(args map[string]interface{}, children []Node) (Node, error) {
		value := strArg(args, "value")
		if value == "" {
			return nil, &Error{Op: "contains", Message: "value is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &ContainsNode{Value: value, CaseSensitive: caseSensitive}, nil
	})

	Register("eq", func(args map[string]interface{}, children []Node) (Node, error) {
		value := strArg(args, "value")
		if value == "" {
			return nil, &Error{Op: "eq", Message: "value is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &EqNode{Value: value, CaseSensitive: caseSensitive}, nil
	})

	Register("prefix", func(args map[string]interface{}, children []Node) (Node, error) {
		value := strArg(args, "value")
		if value == "" {
			return nil, &Error{Op: "prefix", Message: "value is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &PrefixNode{Value: value, CaseSensitive: caseSensitive}, nil
	})

	Register("suffix", func(args map[string]interface{}, children []Node) (Node, error) {
		value := strArg(args, "value")
		if value == "" {
			return nil, &Error{Op: "suffix", Message: "value is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &SuffixNode{Value: value, CaseSensitive: caseSensitive}, nil
	})

	Register("sentence_start", func(args map[string]interface{}, children []Node) (Node, error) {
		return &SentenceStartNode{}, nil
	})

	Register("sentence_end", func(args map[string]interface{}, children []Node) (Node, error) {
		return &SentenceEndNode{}, nil
	})

	Register("paragraph_start", func(args map[string]interface{}, children []Node) (Node, error) {
		return &ParagraphStartNode{}, nil
	})

	Register("paragraph_end", func(args map[string]interface{}, children []Node) (Node, error) {
		return &ParagraphEndNode{}, nil
	})

	Register("word_boundary", func(args map[string]interface{}, children []Node) (Node, error) {
		return &WordBoundaryNode{}, nil
	})

	Register("length", func(args map[string]interface{}, children []Node) (Node, error) {
		min := intArg(args, "min", 1)
		max := intArg(args, "max", 0)
		return &LengthNode{Min: min, Max: max}, nil
	})

	Register("case", func(args map[string]interface{}, children []Node) (Node, error) {
		modeStr := strArg(args, "mode")
		if modeStr == "" {
			return nil, &Error{Op: "case", Message: "mode is required (all_caps, all_lower, capitalized, has_upper, has_lower)"}
		}
		mode := CaseMode(modeStr)
		switch mode {
		case CaseAllCaps, CaseAllLower, CaseCapitalized, CaseHasUpper, CaseHasLower:
		default:
			return nil, &Error{Op: "case", Message: "unknown case mode: " + modeStr}
		}
		return &CaseNode{Mode: mode}, nil
	})

	Register("before", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "before", Message: "requires a child node (use args) to define what to look before"}
		}
		return &BeforeNode{Inner: children[0]}, nil
	})

	Register("after", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "after", Message: "requires a child node (use args) to define what to look after"}
		}
		return &AfterNode{Inner: children[0]}, nil
	})

	Register("surrounded_by", func(args map[string]interface{}, children []Node) (Node, error) {
		left := strArg(args, "left")
		right := strArg(args, "right")
		if left == "" || right == "" {
			return nil, &Error{Op: "surrounded_by", Message: "left and right are required"}
		}
		return &SurroundedByNode{Left: left, Right: right}, nil
	})

	Register("position", func(args map[string]interface{}, children []Node) (Node, error) {
		from := intArg(args, "from", -1)
		to := intArg(args, "to", 0)
		if from < 0 {
			return nil, &Error{Op: "position", Message: "from is required"}
		}
		return &PositionNode{From: from, To: to}, nil
	})

	Register("and", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) < 2 {
			return nil, &Error{Op: "and", Message: "requires at least 2 child nodes"}
		}
		return &AndNode{Children: children}, nil
	})

	Register("or", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) < 2 {
			return nil, &Error{Op: "or", Message: "requires at least 2 child nodes"}
		}
		return &OrNode{Children: children}, nil
	})

	Register("not", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "not", Message: "requires at least 1 child node"}
		}
		return &NotNode{Children: children}, nil
	})
}

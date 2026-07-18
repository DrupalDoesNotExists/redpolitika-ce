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
// Groups holds optional regex submatches: [0]=full match, [1]=first capture, …
type MatchRange struct {
	Start  int
	End    int
	Groups []string
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
		mr := MatchRange{Start: start, End: end}
		if len(m) > 2 {
			nGroups := len(m) / 2
			groups := make([]string, nGroups)
			for i := 0; i < nGroups; i++ {
				gs, ge := m[i*2], m[i*2+1]
				if gs >= 0 && ge >= gs && ge <= len(text) {
					groups[i] = text[gs:ge]
				}
			}
			mr.Groups = groups
		}
		out = append(out, mr)
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
// If Inner is set, filters Inner matches by length; otherwise scans \S+ runs.
type LengthNode struct {
	Min   int  // minimum length (inclusive)
	Max   int  // maximum length (inclusive, 0 = no limit)
	Inner Node // optional child filter
}

func (n *LengthNode) Detect(text string) []MatchRange {
	var matches [][]int
	if n.Inner != nil {
		inner := n.Inner.Detect(text)
		matches = make([][]int, len(inner))
		for i, m := range inner {
			matches[i] = []int{m.Start, m.End}
		}
	} else {
		wordRe := regexp.MustCompile(`\S+`)
		matches = wordRe.FindAllStringIndex(text, -1)
	}
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
// If Inner is set, filters Inner matches by case; otherwise scans \S+ runs.
type CaseNode struct {
	Mode  CaseMode
	Inner Node // optional child filter
}

func (n *CaseNode) Detect(text string) []MatchRange {
	var matches [][]int
	if n.Inner != nil {
		inner := n.Inner.Detect(text)
		matches = make([][]int, len(inner))
		for i, m := range inner {
			matches[i] = []int{m.Start, m.End}
		}
	} else {
		wordRe := regexp.MustCompile(`\S+`)
		matches = wordRe.FindAllStringIndex(text, -1)
	}
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

// BeforeNode matches text within max_chars before an anchor pattern, using child detection.
// children ordering: [0]=Inner (detection to apply), [1]=Pattern (anchor).
// If Pattern is nil, falls back to returning 1 char before each Inner match.
type BeforeNode struct {
	Inner    Node
	Pattern  Node
	MaxChars int
}

func (n *BeforeNode) Detect(text string) []MatchRange {
	if n.Pattern == nil {
		// Backward compat: 1 char before each inner match
		inner := n.Inner.Detect(text)
		var out []MatchRange
		for _, m := range inner {
			if m.Start > 0 {
				out = append(out, MatchRange{Start: m.Start - 1, End: m.Start})
			}
		}
		return out
	}

	anchors := n.Pattern.Detect(text)
	seen := make(map[int]bool)
	var out []MatchRange
	maxChars := n.MaxChars
	if maxChars <= 0 {
		maxChars = 80
	}
	for _, a := range anchors {
		windowEnd := a.Start
		windowStart := windowEnd - maxChars
		if windowStart < 0 {
			windowStart = 0
		}
		if windowStart >= windowEnd {
			continue
		}
		window := text[windowStart:windowEnd]
		innerMatches := n.Inner.Detect(window)
		for _, m := range innerMatches {
			adj := MatchRange{Start: m.Start + windowStart, End: m.End + windowStart}
			if seen[adj.Start] {
				continue
			}
			seen[adj.Start] = true
			out = append(out, adj)
		}
	}
	return out
}

// AfterNode matches text within max_chars after an anchor pattern, using child detection.
// children ordering: [0]=Inner (detection to apply), [1]=Pattern (anchor).
// If Pattern is nil, falls back to returning 1 char after each Inner match.
type AfterNode struct {
	Inner    Node
	Pattern  Node
	MaxChars int
}

func (n *AfterNode) Detect(text string) []MatchRange {
	if n.Pattern == nil {
		// Backward compat: 1 char after each inner match
		inner := n.Inner.Detect(text)
		var out []MatchRange
		for _, m := range inner {
			if m.End < len(text) {
				out = append(out, MatchRange{Start: m.End, End: m.End + 1})
			}
		}
		return out
	}

	anchors := n.Pattern.Detect(text)
	seen := make(map[int]bool)
	var out []MatchRange
	maxChars := n.MaxChars
	if maxChars <= 0 {
		maxChars = 80
	}
	for _, a := range anchors {
		windowStart := a.End
		windowEnd := windowStart + maxChars
		if windowEnd > len(text) {
			windowEnd = len(text)
		}
		if windowStart >= windowEnd {
			continue
		}
		window := text[windowStart:windowEnd]
		innerMatches := n.Inner.Detect(window)
		for _, m := range innerMatches {
			adj := MatchRange{Start: m.Start + windowStart, End: m.End + windowStart}
			if seen[adj.Start] {
				continue
			}
			seen[adj.Start] = true
			out = append(out, adj)
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

// PositionNode matches at specific character position range or paragraph type.
type PositionNode struct {
	From  int    // character position, -1 when using type
	To    int    // 0 = end of text
	Type  string // "first_paragraph", "last_paragraph", or "" for numeric From/To
	Inner Node   // child detection (used with type-based position)
}

func (n *PositionNode) Detect(text string) []MatchRange {
	switch n.Type {
	case "first_paragraph":
		end := strings.Index(text, "\n\n")
		if end < 0 {
			end = len(text)
		}
		if n.Inner != nil {
			return filterByRange(n.Inner.Detect(text), 0, end)
		}
		return []MatchRange{{Start: 0, End: end}}

	case "last_paragraph":
		start := strings.LastIndex(text, "\n\n")
		if start < 0 {
			start = 0
		} else {
			start += 2 // skip past \n\n
		}
		if n.Inner != nil {
			return filterByRange(n.Inner.Detect(text), start, len(text))
		}
		return []MatchRange{{Start: start, End: len(text)}}
	}

	// Numeric position (original From/To behavior)
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

// filterByRange returns matches whose range falls entirely within [lo, hi).
func filterByRange(matches []MatchRange, lo, hi int) []MatchRange {
	var out []MatchRange
	for _, m := range matches {
		if m.Start >= lo && m.End <= hi {
			out = append(out, m)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
//  ThresholdNode — flags matches only when Inner matches >= Count times
//  per window (words, paragraph, or whole text).
// ---------------------------------------------------------------------------

// ThresholdMode represents the window type for threshold detection.
type ThresholdMode string

const (
	ThresholdPerWords     ThresholdMode = "words"
	ThresholdPerParagraph ThresholdMode = "paragraph"
	ThresholdPerText      ThresholdMode = "text"
)

// ThresholdNode flags matches only when Inner matches >= Count times per window.
type ThresholdNode struct {
	Count  int
	Per    ThresholdMode
	Window int // word window size (for per=words, default 100)
	Inner  Node
}

func (n *ThresholdNode) Detect(text string) []MatchRange {
	if n.Inner == nil {
		return nil
	}

	// Run inner detection
	allMatches := n.Inner.Detect(text)
	if len(allMatches) == 0 {
		return nil
	}

	switch n.Per {
	case ThresholdPerText:
		// Check if total matches >= Count across whole text
		if len(allMatches) >= n.Count {
			return allMatches
		}
		return nil

	case ThresholdPerParagraph:
		return n.detectPerParagraph(text, allMatches)

	default: // ThresholdPerWords
		return n.detectPerWords(text, allMatches)
	}
}

func (n *ThresholdNode) detectPerParagraph(text string, allMatches []MatchRange) []MatchRange {
	bounds := paragraphBoundaries(text)
	var result []MatchRange
	for _, p := range bounds {
		var countInPara int
		var matchesInPara []MatchRange
		for _, m := range allMatches {
			if m.Start >= p.Start && m.End <= p.End {
				countInPara++
				matchesInPara = append(matchesInPara, m)
			}
		}
		if countInPara >= n.Count {
			result = append(result, matchesInPara...)
		}
	}
	return result
}

func (n *ThresholdNode) detectPerWords(text string, allMatches []MatchRange) []MatchRange {
	words := strings.Fields(text)
	window := n.Window
	if window <= 0 {
		window = 100
	}
	if len(words) == 0 {
		return nil
	}

	// Map each match to word index (word containing its start position)
	matchToWord := make([]int, len(allMatches))
	for i, m := range allMatches {
		matchToWord[i] = wordCountIn(text[:m.Start])
	}

	// Sliding window with 50% overlap
	step := window / 2
	if step < 1 {
		step = 1
	}
	var result []MatchRange
	seen := make(map[int]bool) // dedup by Start position
	for wi := 0; wi < len(words); wi += step {
		endWord := wi + window
		if endWord > len(words) {
			endWord = len(words)
		}

		// Collect matches in this window
		var inWindow []MatchRange
		for i, m := range allMatches {
			if matchToWord[i] >= wi && matchToWord[i] < endWord && !seen[m.Start] {
				inWindow = append(inWindow, m)
			}
		}

		if len(inWindow) >= n.Count {
			for _, m := range inWindow {
				if !seen[m.Start] {
					seen[m.Start] = true
					result = append(result, m)
				}
			}
		}
	}

	return result
}

// ---------------------------------------------------------------------------
//  NearNode — flags Pattern matches that have a Near match within a window
// ---------------------------------------------------------------------------

// NearWindowMode is the proximity window for NearNode.
type NearWindowMode string

const (
	NearWindowSentence NearWindowMode = "sentence"
	NearWindowChars    NearWindowMode = "chars"
)

// NearNode returns Pattern matches that have at least one Near match in the same
// sentence or within Chars characters (either direction).
type NearNode struct {
	Pattern Node
	Near    Node
	Mode    NearWindowMode
	Chars   int // used when Mode == NearWindowChars
}

func (n *NearNode) Detect(text string) []MatchRange {
	if n.Pattern == nil || n.Near == nil {
		return nil
	}
	patterns := n.Pattern.Detect(text)
	nears := n.Near.Detect(text)
	if len(patterns) == 0 || len(nears) == 0 {
		return nil
	}

	var out []MatchRange
	seen := make(map[int]bool)

	switch n.Mode {
	case NearWindowChars:
		chars := n.Chars
		if chars <= 0 {
			chars = 80
		}
		for _, p := range patterns {
			for _, q := range nears {
				if rangesOverlapOrTouch(p, q) {
					continue // same/overlapping span — not "near"
				}
				dist := rangeDistance(p, q)
				if dist <= chars && !seen[p.Start] {
					seen[p.Start] = true
					out = append(out, p)
					break
				}
			}
		}
	default: // sentence
		bounds := sentenceSpanBoundaries(text)
		for _, p := range patterns {
			si := sentenceIndexOf(bounds, p.Start)
			if si < 0 {
				continue
			}
			for _, q := range nears {
				if sentenceIndexOf(bounds, q.Start) == si && !seen[p.Start] {
					seen[p.Start] = true
					out = append(out, p)
					break
				}
			}
		}
	}
	return out
}

func rangeDistance(a, b MatchRange) int {
	if a.End <= b.Start {
		return b.Start - a.End
	}
	if b.End <= a.Start {
		return a.Start - b.End
	}
	return 0 // overlapping
}

func rangesOverlapOrTouch(a, b MatchRange) bool {
	return a.Start < b.End && b.Start < a.End
}

// sentenceSpanBoundaries returns [start,end) spans for each sentence.
func sentenceSpanBoundaries(text string) []MatchRange {
	if text == "" {
		return nil
	}
	var spans []MatchRange
	start := 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '.' || c == '!' || c == '?' {
			end := i + 1
			spans = append(spans, MatchRange{Start: start, End: end})
			// skip whitespace after punct
			j := end
			for j < len(text) && (text[j] == ' ' || text[j] == '\t' || text[j] == '\n' || text[j] == '\r') {
				j++
			}
			start = j
			i = j - 1
		}
	}
	if start < len(text) {
		spans = append(spans, MatchRange{Start: start, End: len(text)})
	}
	if len(spans) == 0 {
		spans = []MatchRange{{Start: 0, End: len(text)}}
	}
	return spans
}

func sentenceIndexOf(spans []MatchRange, pos int) int {
	for i, s := range spans {
		if pos >= s.Start && pos < s.End {
			return i
		}
	}
	if len(spans) > 0 && pos == spans[len(spans)-1].End {
		return len(spans) - 1
	}
	return -1
}

// parboundary represents a paragraph boundary range.
type parboundary struct {
	Start int
	End   int
}

// splitParagraphs returns start positions of each paragraph.
func splitParagraphs(text string) []int {
	var starts []int
	starts = append(starts, 0)
	for i := 0; i < len(text); i++ {
		if i+1 < len(text) && text[i] == '\n' && text[i+1] == '\n' {
			j := i + 2
			for j < len(text) && (text[j] == '\n' || text[j] == '\r') {
				j++
			}
			if j < len(text) {
				starts = append(starts, j)
			}
			i = j - 1
		}
	}
	return starts
}

// paragraphBoundaries returns start/end ranges for each paragraph.
func paragraphBoundaries(text string) []parboundary {
	starts := splitParagraphs(text)
	bounds := make([]parboundary, len(starts))
	for i, s := range starts {
		end := len(text)
		if i+1 < len(starts) {
			end = starts[i+1]
			// Trim trailing whitespace from end
			for end > s && (text[end-1] == '\n' || text[end-1] == '\r' || text[end-1] == ' ') {
				end--
			}
		}
		bounds[i] = parboundary{Start: s, End: end}
	}
	return bounds
}

// wordCountIn returns number of whitespace-separated tokens in text.
func wordCountIn(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(strings.Fields(text))
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
		n := &LengthNode{Min: min, Max: max}
		if len(children) > 0 {
			n.Inner = children[0]
		}
		return n, nil
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
		n := &CaseNode{Mode: mode}
		if len(children) > 0 {
			n.Inner = children[0]
		}
		return n, nil
	})

	Register("before", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "before", Message: "requires a child node (use args) to define what to look before"}
		}
		n := &BeforeNode{Inner: children[0], MaxChars: intArg(args, "max", 80)}
		if len(children) > 1 {
			n.Pattern = children[1]
		}
		return n, nil
	})

	Register("after", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "after", Message: "requires a child node (use args) to define what to look after"}
		}
		n := &AfterNode{Inner: children[0], MaxChars: intArg(args, "max", 80)}
		if len(children) > 1 {
			n.Pattern = children[1]
		}
		return n, nil
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
		posType := strArg(args, "type")
		if posType == "first_paragraph" || posType == "last_paragraph" {
			var inner Node
			if len(children) > 0 {
				inner = children[0]
			}
			return &PositionNode{Type: posType, Inner: inner}, nil
		}
		from := intArg(args, "from", -1)
		to := intArg(args, "to", 0)
		if from < 0 {
			return nil, &Error{Op: "position", Message: "from is required"}
		}
		return &PositionNode{From: from, To: to}, nil
	})

	Register("threshold", func(args map[string]interface{}, children []Node) (Node, error) {
		count := intArg(args, "count", 0)
		if count < 1 {
			return nil, &Error{Op: "threshold", Message: "count is required (min 1)"}
		}

		per := strArg(args, "per")
		if per == "" {
			per = "words"
		}

		window := intArg(args, "window", 100)
		if window <= 0 {
			window = 100
		}

		var inner Node
		if len(children) > 0 {
			inner = children[0]
		}
		if inner == nil {
			return nil, &Error{Op: "threshold", Message: "requires a child node"}
		}

		return &ThresholdNode{
			Count:  count,
			Per:    ThresholdMode(per),
			Window: window,
			Inner:  inner,
		}, nil
	})

	Register("near", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) < 2 {
			return nil, &Error{Op: "near", Message: "requires pattern and near child nodes"}
		}
		n := &NearNode{Pattern: children[0], Near: children[1]}

		windowArg := args["window"]
		switch v := windowArg.(type) {
		case string:
			if v == "sentence" {
				n.Mode = NearWindowSentence
			} else if strings.HasPrefix(v, "chars:") {
				var chars int
				if _, err := fmt.Sscanf(v, "chars:%d", &chars); err == nil && chars > 0 {
					n.Mode = NearWindowChars
					n.Chars = chars
				} else {
					return nil, &Error{Op: "near", Message: "invalid window: " + v}
				}
			} else {
				return nil, &Error{Op: "near", Message: "window must be 'sentence', 'chars:N', or integer"}
			}
		case int:
			if v <= 0 {
				return nil, &Error{Op: "near", Message: "window chars must be > 0"}
			}
			n.Mode = NearWindowChars
			n.Chars = v
		case float64:
			chars := int(v)
			if chars <= 0 {
				return nil, &Error{Op: "near", Message: "window chars must be > 0"}
			}
			n.Mode = NearWindowChars
			n.Chars = chars
		case nil:
			n.Mode = NearWindowSentence
		default:
			return nil, &Error{Op: "near", Message: "invalid window type"}
		}
		return n, nil
	})

	Register("exclude", func(args map[string]interface{}, children []Node) (Node, error) {
		if len(children) == 0 {
			return nil, &Error{Op: "exclude", Message: "requires a child node"}
		}
		list := strSliceArg(args, "list")
		if len(list) == 0 {
			list = strSliceArg(args, "words")
		}
		if len(list) == 0 {
			return nil, &Error{Op: "exclude", Message: "list (whitelist) is required"}
		}
		caseSensitive := boolArg(args, "case_sensitive", false)
		return &AndNode{Children: []Node{
			children[0],
			&NotNode{Children: []Node{
				&WordlistNode{Words: list, CaseSensitive: caseSensitive},
			}},
		}}, nil
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

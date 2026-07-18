package model

import (
	"regexp"
	"strings"
)

// Suppression is an inline range where a rule (or all rules) must not raise flags.
// Parsed from HTML-style comments in the text (E3).
type Suppression struct {
	RuleID string // empty = all rules
	Start  int    // byte offset in full text (inclusive)
	End    int    // byte offset in full text (exclusive)
}

var (
	reDisable     = regexp.MustCompile(`<!--\s*rp:disable(?:\s+([^\s>]+))?\s*-->`)
	reEnable      = regexp.MustCompile(`<!--\s*rp:enable(?:\s+([^\s>]+))?\s*-->`)
	reDisableLine = regexp.MustCompile(`<!--\s*rp:disable-line(?:\s+([^\s>]+))?\s*-->`)
)

// ParseSuppressions extracts rp:disable / rp:enable / rp:disable-line ranges from text.
func ParseSuppressions(content string) []Suppression {
	if content == "" {
		return nil
	}

	var out []Suppression

	// disable-line: suppress the whole line containing the comment
	for _, m := range reDisableLine.FindAllStringSubmatchIndex(content, -1) {
		ruleID := ""
		if m[2] >= 0 {
			ruleID = content[m[2]:m[3]]
		}
		lineStart, lineEnd := lineBounds(content, m[0])
		out = append(out, Suppression{RuleID: ruleID, Start: lineStart, End: lineEnd})
	}

	// block disable/enable pairs
	type openBlock struct {
		ruleID string
		start  int
	}
	var stack []openBlock

	type marker struct {
		kind   string // "disable" | "enable"
		ruleID string
		pos    int
		end    int
	}
	var markers []marker
	for _, m := range reDisable.FindAllStringSubmatchIndex(content, -1) {
		ruleID := ""
		if m[2] >= 0 {
			ruleID = content[m[2]:m[3]]
		}
		markers = append(markers, marker{kind: "disable", ruleID: ruleID, pos: m[0], end: m[1]})
	}
	for _, m := range reEnable.FindAllStringSubmatchIndex(content, -1) {
		ruleID := ""
		if m[2] >= 0 {
			ruleID = content[m[2]:m[3]]
		}
		markers = append(markers, marker{kind: "enable", ruleID: ruleID, pos: m[0], end: m[1]})
	}
	// sort by position
	for i := 0; i < len(markers); i++ {
		for j := i + 1; j < len(markers); j++ {
			if markers[j].pos < markers[i].pos {
				markers[i], markers[j] = markers[j], markers[i]
			}
		}
	}

	for _, mk := range markers {
		switch mk.kind {
		case "disable":
			stack = append(stack, openBlock{ruleID: mk.ruleID, start: mk.end})
		case "enable":
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].ruleID == mk.ruleID || mk.ruleID == "" || stack[i].ruleID == "" {
					blk := stack[i]
					stack = append(stack[:i], stack[i+1:]...)
					end := mk.pos
					if end > blk.start {
						out = append(out, Suppression{RuleID: blk.ruleID, Start: blk.start, End: end})
					}
					break
				}
			}
		}
	}
	// unclosed disables run to EOF
	for _, blk := range stack {
		if blk.start < len(content) {
			out = append(out, Suppression{RuleID: blk.ruleID, Start: blk.start, End: len(content)})
		}
	}

	return out
}

// IsSuppressed reports whether [start,end) for ruleID is covered by any suppression.
func IsSuppressed(suppressions []Suppression, ruleID string, start, end int) bool {
	for _, s := range suppressions {
		if s.RuleID != "" && s.RuleID != ruleID && !strings.EqualFold(s.RuleID, ruleID) {
			continue
		}
		// match overlaps suppression range
		if start < s.End && end > s.Start {
			return true
		}
	}
	return false
}

func lineBounds(content string, pos int) (start, end int) {
	start = pos
	for start > 0 && content[start-1] != '\n' {
		start--
	}
	end = pos
	for end < len(content) && content[end] != '\n' {
		end++
	}
	if end < len(content) {
		end++ // include newline
	}
	return start, end
}

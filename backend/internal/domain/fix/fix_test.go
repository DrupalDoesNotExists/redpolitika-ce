package fix_test

import (
	"testing"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/fix"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func TestTitleCaseRussian(t *testing.T) {
	n, err := fix.Build("title_case", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := n.Fix("привет мир", fix.Context{})
	want := cases.Title(language.Und).String("привет мир")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWhenConditionalFix(t *testing.T) {
	cond, err := detect.Build("case", map[string]interface{}{"mode": "all_lower"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	then, err := fix.Build("regex_replace", map[string]interface{}{
		"pattern": "ё", "replacement": "е",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	when := &fix.WhenNode{Condition: cond, Then: then}

	if got := when.Fix("ёлка", fix.Context{}); got != "елка" {
		t.Fatalf("lowercase: got %q", got)
	}
	if got := when.Fix("Ёлка", fix.Context{}); got != "Ёлка" {
		t.Fatalf("capitalized must stay: got %q", got)
	}
}

func TestRegexReplaceDetectGroups(t *testing.T) {
	n, err := fix.Build("regex_replace", map[string]interface{}{
		"pattern":     ".*",
		"replacement": "$1 $2",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := fix.Context{
		Text:   "у нас есть",
		Start:  0,
		End:    10,
		Groups: []string{"у нас есть", "у нас", "есть"},
	}
	got := n.Fix("у нас есть", ctx)
	if got != "у нас есть" {
		// $1 $2 with space → "у нас есть"
		t.Fatalf("got %q want %q", got, "у нас есть")
	}
	ctx.Groups = []string{"у нас есть", "у нас", "есть"}
	n2 := &fix.RegexReplaceNode{Replacement: "$2, $1"}
	if got := n2.Fix("у нас есть", ctx); got != "есть, у нас" {
		t.Fatalf("got %q want %q", got, "есть, у нас")
	}
}

func TestFixContextBeforeAfter(t *testing.T) {
	ctx := fix.Context{Text: "абв:Где", Start: 7, End: 10} // "Где" — wrong, use bytes
	// "абв:" = 7 bytes (3*2+1), "Где" = 6 bytes
	text := "абв:Где"
	start := len("абв:")
	end := len(text)
	ctx = fix.Context{Text: text, Start: start, End: end}
	if got := ctx.Before(1); got != ":" {
		t.Fatalf("Before: got %q", got)
	}
	if got := ctx.After(1); got != "" {
		t.Fatalf("After at end: got %q", got)
	}
	// context-aware: lowercase only when preceded by colon
	lower, _ := fix.Build("lowercase", nil, nil)
	match := text[start:end]
	if ctx.Before(1) == ":" {
		got := lower.Fix(match, ctx)
		if got != "где" {
			t.Fatalf("context lowercase: got %q", got)
		}
	} else {
		t.Fatal("expected colon before match")
	}
}

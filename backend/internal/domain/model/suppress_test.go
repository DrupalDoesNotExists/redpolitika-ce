package model_test

import (
	"testing"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
)

func TestParseSuppressionsBlock(t *testing.T) {
	text := "до <!-- rp:disable clutter/very --> очень важно <!-- rp:enable --> после очень важно"
	sup := model.ParseSuppressions(text)
	if len(sup) != 1 {
		t.Fatalf("expected 1 suppression, got %+v", sup)
	}
	if !model.IsSuppressed(sup, "clutter/very", sup[0].Start+1, sup[0].Start+2) {
		t.Fatal("inside range must be suppressed")
	}
	// "после очень" should not be suppressed
	idx := len("до <!-- rp:disable clutter/very --> очень важно <!-- rp:enable --> ")
	if model.IsSuppressed(sup, "clutter/very", idx, idx+5) {
		t.Fatal("after enable must not be suppressed")
	}
}

func TestParseSuppressionsDisableLine(t *testing.T) {
	text := "строка один\nочень важно <!-- rp:disable-line redundancy/very -->\nстрока три"
	sup := model.ParseSuppressions(text)
	if len(sup) != 1 {
		t.Fatalf("expected 1 line suppression, got %+v", sup)
	}
	// mid of line 2
	mid := len("строка один\nочень")
	if !model.IsSuppressed(sup, "redundancy/very", mid, mid+2) {
		t.Fatal("line with disable-line must be suppressed")
	}
}

func TestRuleDetectHonoursSuppress(t *testing.T) {
	inner, err := detect.Build("wordlist", map[string]interface{}{
		"list": []string{"очень"}, "case_sensitive": false,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := model.NewRule(
		"redundancy/very", 5, "cleanliness",
		"wordlist", inner, nil,
		"уберите очень", "Очень", "", model.Examples{}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	plain := model.NewText("это очень важно")
	if n := len(rule.Detect(plain)); n != 1 {
		t.Fatalf("expected 1 flag, got %d", n)
	}

	suppressed := model.NewText("<!-- rp:disable redundancy/very -->это очень важно")
	if n := len(rule.Detect(suppressed)); n != 0 {
		t.Fatalf("expected 0 flags under suppress, got %d", n)
	}
}

// paragraph_end may match a single letter that also appears earlier in the
// paragraph; occurrence must index that letter in the full paragraph text
// (A23), otherwise the frontend highlights the first copy.
func TestRuleDetectOccurrenceCountsPriorTextCopies(t *testing.T) {
	inner, err := detect.Build("regex", map[string]interface{}{
		"pattern": "[^.!?]", "case_sensitive": false,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	node, err := detect.Build("paragraph_end", nil, []detect.Node{inner})
	if err != nil {
		t.Fatal(err)
	}
	rule, err := model.NewRule(
		"readability/sentence-final-punctuation", 3, "readability",
		"", node, nil,
		"Добавьте точку.", "Пунктуация", "", model.Examples{}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	text := "привет без котлет"
	flags := rule.Detect(model.NewText(text))
	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}
	f := flags[0]
	if f.MatchText().Value() != "т" {
		t.Fatalf("match_text: got %q", f.MatchText().Value())
	}
	// т in привет (0), т in котлет mid (1), final т (2)
	if f.Occurrence().Value() != 2 {
		t.Fatalf("occurrence: got %d, want 2 (not first т in привет)", f.Occurrence().Value())
	}
	span := f.Span()
	got := text[span.Start():span.End()]
	if got != "т" || span.Start() != len(text)-len("т") {
		t.Fatalf("span [%d:%d] %q, want final т", span.Start(), span.End(), got)
	}
}

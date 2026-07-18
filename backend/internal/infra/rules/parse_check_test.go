package rules

import (
	"testing"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
)

func TestParseSentenceStartWithChild(t *testing.T) {
	data := []byte(`
rules:
  - id: "capitalization/sentence-start-lower"
    severity: 4
    category: "readability"
    name: "Строчная в начале предложения"
    detect:
      sentence_start:
        regex:
          pattern: "[а-яё]"
          case_sensitive: true
`)
	parsed, err := ParseYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(parsed))
	}
	dn := parsed[0].DetectNode()
	if dn == nil {
		t.Fatal("detectNode is nil — child was dropped")
	}
	// Type assert to ensure Inner is wired
	if _, ok := dn.(*detect.SentenceStartNode); !ok {
		t.Fatalf("expected *SentenceStartNode, got %T", dn)
	}
	text := "Хорошо. плохо. Ок."
	got := dn.Detect(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %+v", got)
	}
	if text[got[0].Start:got[0].End] != "п" {
		t.Fatalf("unexpected match %q", text[got[0].Start:got[0].End])
	}
}

func TestParseInlineYAML(t *testing.T) {
	data := []byte(`
rules:
  - id: "capitalization/after-colon"
    severity: 4
    category: "readability"
    name: "Заглавная после двоеточия"
    detect:
      after:
        pattern:
          contains: { value: ": ", case_sensitive: false }
        max_chars: 80
        child:
          regex:
            pattern: "^[А-ЯЁ]"
            case_sensitive: true
  - id: "readability/long-sentence"
    severity: 2
    category: "readability"
    detect:
      length:
        min: 150
        child:
          regex: "[^.!?\\n]*[.!?]"
`)
	parsed, err := ParseYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(parsed))
	}

	var afterColon, longSentence bool
	for _, r := range parsed {
		switch r.ID().Value() {
		case "capitalization/after-colon":
			afterColon = true
			dn := r.DetectNode()
			if dn == nil {
				t.Fatal("after-colon: detectNode is nil")
			}
			if len(dn.Detect("Итог: заказ выполнен.")) != 0 {
				t.Fatal("after-colon: lowercase after colon must not match")
			}
			if len(dn.Detect("Итог: Заказ выполнен.")) == 0 {
				t.Fatal("after-colon: uppercase after colon must match")
			}
		case "readability/long-sentence":
			longSentence = true
			dn := r.DetectNode()
			if dn == nil {
				t.Fatal("long-sentence: detectNode is nil")
			}
			long := "Рособрнадзор и Минпросвещения опубликовали приказы, где утвердили особенности выдачи аттестатов за 9 и 11 класс в 2020 году, а также уточнили порядок апелляций."
			if len(dn.Detect(long)) == 0 {
				t.Fatal("long-sentence: expected match on long text")
			}
		}
	}
	if !afterColon || !longSentence {
		t.Fatal("expected both after-colon and long-sentence rules")
	}
}

func TestParseFixLevelSuggestion(t *testing.T) {
	data := []byte(`
rules:
  - id: "style/ellipsis"
    severity: 3
    category: "cleanliness"
    name: "Многоточие"
    detect:
      regex: "\\.\\.\\."
    fix:
      replace:
        with: "…"
      suggestion: "Замените три точки на символ многоточия"
`)
	parsed, err := ParseYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(parsed))
	}
	got := parsed[0].Suggestion().Value()
	want := "Замените три точки на символ многоточия"
	if got != want {
		t.Fatalf("suggestion: got %q, want %q", got, want)
	}
}

func TestDetectRegisteredMethods(t *testing.T) {
	methods := detect.Registered()
	if len(methods) == 0 {
		t.Error("no detect methods registered")
	}
}

// Capital needs case_sensitive; missing period must use paragraph_end, not sentence_end.
func TestCapitalAndPeriodDetectSemantics(t *testing.T) {
	data := []byte(`
rules:
  - id: "readability/capital-sentence-start"
    severity: 4
    category: "readability"
    detect:
      sentence_start:
        regex:
          pattern: "[а-яё]"
          case_sensitive: true
  - id: "readability/sentence-final-punctuation"
    severity: 3
    category: "readability"
    detect:
      paragraph_end:
        regex:
          pattern: "[а-яёА-ЯЁa-zA-Z0-9»\")\\]]"
          case_sensitive: true
`)
	parsed, err := ParseYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	var capital, period detect.Node
	for _, r := range parsed {
		switch r.ID().String() {
		case "readability/capital-sentence-start":
			capital = r.DetectNode()
		case "readability/sentence-final-punctuation":
			period = r.DetectNode()
		}
	}
	if capital == nil || period == nil {
		t.Fatal("detect nodes nil")
	}

	ok := "Дело было вечером. Делать было нечего."
	if got := capital.Detect(ok); len(got) != 0 {
		t.Fatalf("capital on OK text: %+v", got)
	}
	if got := period.Detect(ok); len(got) != 0 {
		t.Fatalf("period on OK text: %+v", got)
	}

	badCap := "дело было вечером. делать было нечего."
	if got := capital.Detect(badCap); len(got) != 2 {
		t.Fatalf("capital on lower starts: got %+v", got)
	}
	if got := period.Detect("Дело было вечером"); len(got) != 1 {
		t.Fatalf("period missing punct: got %+v", got)
	}
	if got := period.Detect("Дело было вечером."); len(got) != 0 {
		t.Fatalf("period with punct: %+v", got)
	}
}

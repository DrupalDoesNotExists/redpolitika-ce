package detect_test

import (
	"testing"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
)

func mustBuild(t *testing.T, method string, args map[string]interface{}, children []detect.Node) detect.Node {
	t.Helper()
	n, err := detect.Build(method, args, children)
	if err != nil {
		t.Fatalf("build %s: %v", method, err)
	}
	return n
}

func TestAfterColonSemantics(t *testing.T) {
	pattern := mustBuild(t, "contains", map[string]interface{}{"value": ": ", "case_sensitive": true}, nil)
	inner := mustBuild(t, "regex", map[string]interface{}{"pattern": "^[А-ЯЁ]", "case_sensitive": true}, nil)
	node := mustBuild(t, "after", map[string]interface{}{"max": 5}, []detect.Node{inner, pattern})

	bad := node.Detect("Итог: Заказ выполнен.")
	if len(bad) == 0 {
		t.Fatal("expected match on uppercase after colon")
	}
	good := node.Detect("Итог: заказ выполнен.")
	if len(good) != 0 {
		t.Fatalf("expected no match on lowercase after colon, got %+v", good)
	}
}

func TestBeforeSemantics(t *testing.T) {
	pattern := mustBuild(t, "contains", map[string]interface{}{"value": " важно", "case_sensitive": false}, nil)
	inner := mustBuild(t, "wordlist", map[string]interface{}{"list": []string{"очень"}, "case_sensitive": false}, nil)
	node := mustBuild(t, "before", map[string]interface{}{"max": 20}, []detect.Node{inner, pattern})

	got := node.Detect("Это очень важно для нас.")
	if len(got) == 0 {
		t.Fatal("expected 'очень' before ' важно'")
	}
}

func TestPositionFirstParagraph(t *testing.T) {
	inner := mustBuild(t, "wordlist", map[string]interface{}{"list": []string{"введение"}, "case_sensitive": false}, nil)
	node := mustBuild(t, "position", map[string]interface{}{"type": "first_paragraph"}, []detect.Node{inner})

	text := "Это введение к статье.\n\nА это уже второй абзац с введение."
	got := node.Detect(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 match in first paragraph, got %+v", got)
	}
}

func TestLengthOnChildMatch(t *testing.T) {
	inner := mustBuild(t, "regex", map[string]interface{}{"pattern": `[^.!?\n]+[.!?]`, "case_sensitive": true}, nil)
	node := mustBuild(t, "length", map[string]interface{}{"min": 50}, []detect.Node{inner})

	long := "Рособрнадзор и Минпросвещения опубликовали приказы про аттестаты."
	if len(node.Detect(long)) == 0 {
		t.Fatal("expected long sentence match")
	}
	short := "Коротко."
	if len(node.Detect(short)) != 0 {
		t.Fatal("short sentence must not match")
	}
	// long single word should not match via word-scan when child is set
	word := "Суперкалифрагилистикэкспиалидошесноесловобезточек"
	if len(node.Detect(word)) != 0 {
		t.Fatal("long word without sentence child match must not flag")
	}
}

func TestCaseOnChild(t *testing.T) {
	inner := mustBuild(t, "regex", map[string]interface{}{"pattern": `москва|МОСКВА`, "case_sensitive": true}, nil)
	node := mustBuild(t, "case", map[string]interface{}{"mode": "all_caps"}, []detect.Node{inner})

	got := node.Detect("город МОСКВА и москва рядом")
	if len(got) != 1 {
		t.Fatalf("expected only ALL_CAPS match, got %+v", got)
	}
	if got[0].Start < 0 || textSlice("город МОСКВА и москва рядом", got[0]) != "МОСКВА" {
		t.Fatalf("unexpected match %q", textSlice("город МОСКВА и москва рядом", got[0]))
	}
}

func textSlice(s string, m detect.MatchRange) string {
	return s[m.Start:m.End]
}

func TestThresholdPerWords(t *testing.T) {
	inner := mustBuild(t, "wordlist", map[string]interface{}{"list": []string{"вроде"}, "case_sensitive": false}, nil)
	node := mustBuild(t, "threshold", map[string]interface{}{"count": 3, "per": "words", "window": 100}, []detect.Node{inner})

	// 3× вроде in a short text
	text := "вроде да вроде нет и вроде снова"
	got := node.Detect(text)
	if len(got) < 3 {
		t.Fatalf("expected >=3 matches, got %+v", got)
	}

	sparse := "вроде один раз только"
	if len(node.Detect(sparse)) != 0 {
		t.Fatal("single occurrence must not pass threshold")
	}
}

func TestNearSameSentence(t *testing.T) {
	p1 := mustBuild(t, "wordlist", map[string]interface{}{"list": []string{"X"}, "case_sensitive": true}, nil)
	p2 := mustBuild(t, "wordlist", map[string]interface{}{"list": []string{"Y"}, "case_sensitive": true}, nil)
	node := mustBuild(t, "near", map[string]interface{}{"window": "sentence"}, []detect.Node{p1, p2})

	got := node.Detect("Here is X and also Y together. Alone X here.")
	if len(got) != 1 {
		t.Fatalf("expected 1 X near Y in first sentence, got %+v", got)
	}
}

func TestNearCharsWindow(t *testing.T) {
	p1 := mustBuild(t, "contains", map[string]interface{}{"value": "foo", "case_sensitive": true}, nil)
	p2 := mustBuild(t, "contains", map[string]interface{}{"value": "bar", "case_sensitive": true}, nil)
	node := mustBuild(t, "near", map[string]interface{}{"window": 10}, []detect.Node{p1, p2})

	close := node.Detect("foo .... bar")
	if len(close) == 0 {
		t.Fatal("expected near match within 10 chars")
	}
	far := node.Detect("foo" + ".......... .........." + "bar")
	if len(far) != 0 {
		t.Fatalf("expected no match when far, got %+v", far)
	}
}

func TestExcludeWhitelist(t *testing.T) {
	inner := mustBuild(t, "regex", map[string]interface{}{
		"pattern": `[а-яА-ЯёЁ]*ё[а-яА-ЯёЁ]*`, "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "exclude", map[string]interface{}{
		"list": []string{"ёлка", "ёж"}, "case_sensitive": false,
	}, []detect.Node{inner})

	got := node.Detect("ёлка и ещё пчёлы")
	// ёлка excluded; ещё and пчёлы should match
	if len(got) < 2 {
		t.Fatalf("expected matches outside whitelist, got %+v", got)
	}
	for _, m := range got {
		s := "ёлка и ещё пчёлы"[m.Start:m.End]
		if s == "ёлка" {
			t.Fatal("whitelist word must be excluded")
		}
	}
}

func TestRegexCaptureGroups(t *testing.T) {
	node := mustBuild(t, "regex", map[string]interface{}{
		"pattern": `(у нас) (есть)`, "case_sensitive": true,
	}, nil)
	got := node.Detect("у нас есть план")
	if len(got) != 1 || len(got[0].Groups) < 3 {
		t.Fatalf("expected groups, got %+v", got)
	}
	if got[0].Groups[1] != "у нас" || got[0].Groups[2] != "есть" {
		t.Fatalf("unexpected groups: %+v", got[0].Groups)
	}
}

func TestSentenceStartWithChild(t *testing.T) {
	// No ^ — sentence_start itself filters to sentence-start positions.
	inner := mustBuild(t, "regex", map[string]interface{}{
		"pattern": `[а-яё]`, "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "sentence_start", nil, []detect.Node{inner})

	text := "Нормально. плохое начало. И снова нормально."
	got := node.Detect(text)
	if len(got) != 1 {
		t.Fatalf("expected 1 lowercase sentence start, got %+v", got)
	}
	if textSlice(text, got[0]) != "п" {
		t.Fatalf("unexpected match %q", textSlice(text, got[0]))
	}

	// Without child — zero-length positions at each sentence start
	bare := mustBuild(t, "sentence_start", nil, nil)
	bareGot := bare.Detect(text)
	if len(bareGot) != 3 {
		t.Fatalf("expected 3 sentence starts, got %+v", bareGot)
	}
}

func TestSentenceEndWithChild(t *testing.T) {
	// No $ — sentence_end filters to sentence-end positions.
	inner := mustBuild(t, "regex", map[string]interface{}{
		"pattern": `же`, "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "sentence_end", nil, []detect.Node{inner})

	text := "Сделай же. Не надо же. Хватит."
	got := node.Detect(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 sentence-final 'же', got %+v", got)
	}
	for _, m := range got {
		if textSlice(text, m) != "же" {
			t.Fatalf("unexpected match %q", textSlice(text, m))
		}
	}
}

func TestParagraphStartWithChild(t *testing.T) {
	inner := mustBuild(t, "wordlist", map[string]interface{}{
		"list": []string{"Во-первых"}, "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "paragraph_start", nil, []detect.Node{inner})

	text := "Во-первых, важно.\n\nА здесь Во-первых не в начале.\n\nВо-первых снова."
	got := node.Detect(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 paragraph-initial matches, got %+v", got)
	}
	for _, m := range got {
		if textSlice(text, m) != "Во-первых" {
			t.Fatalf("unexpected match %q", textSlice(text, m))
		}
	}
}

func TestParagraphEndWithChild(t *testing.T) {
	// No $ — paragraph_end filters to paragraph-end positions.
	inner := mustBuild(t, "regex", map[string]interface{}{
		"pattern": `конец`, "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "paragraph_end", nil, []detect.Node{inner})

	text := "Первый абзац конец\n\nСередина без\n\nТретий тоже конец"
	got := node.Detect(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 paragraph-final 'конец', got %+v", got)
	}
}

func TestWordBoundaryWithChild(t *testing.T) {
	inner := mustBuild(t, "contains", map[string]interface{}{
		"value": "word", "case_sensitive": true,
	}, nil)
	node := mustBuild(t, "word_boundary", nil, []detect.Node{inner})

	text := "a word and sword and word."
	got := node.Detect(text)
	if len(got) != 2 {
		t.Fatalf("expected 2 whole-word matches, got %+v", got)
	}
	for _, m := range got {
		if textSlice(text, m) != "word" {
			t.Fatalf("unexpected match %q", textSlice(text, m))
		}
	}
}

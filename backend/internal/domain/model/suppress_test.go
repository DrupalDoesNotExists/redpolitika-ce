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

package service_test

import (
	"testing"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/service"
)

func TestScoreIgnoresAccepted(t *testing.T) {
	calc := service.NewScoreCalculator()

	pending, err := model.NewFlag(1, "r1", "x", "", nil, 5, "cleanliness", 0, "m", mustSpan(t, 0, 1), 0, "", "", model.Examples{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := model.NewFlag(2, "r1", "y", "", nil, 5, "cleanliness", 0, "m", mustSpan(t, 2, 3), 0, "", "", model.Examples{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := accepted.Accept(); err != nil {
		t.Fatal(err)
	}

	grouped := map[model.Category][]service.ScoredFlag{
		mustCat(t, "cleanliness"): {
			{Severity: mustSev(t, 5), Flag: pending},
			{Severity: mustSev(t, 5), Flag: accepted},
		},
	}
	clean, _ := calc.ComputeFromMap(grouped, model.WordCountFromInt(100))
	// only pending counts: 10 - 5 = 5
	if clean.Value() != 5 {
		t.Fatalf("expected 5.0 (accepted ignored), got %v", clean.Value())
	}
}

func mustSpan(t *testing.T, a, b int) model.Span {
	t.Helper()
	s, err := model.NewSpan(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustCat(t *testing.T, v string) model.Category {
	t.Helper()
	c, err := model.CategoryFromString(v)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustSev(t *testing.T, v int) model.Severity {
	t.Helper()
	s, err := model.SeverityFromInt(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

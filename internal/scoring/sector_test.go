package scoring

import "testing"

func TestBuildSectorStrength_AveragesAndComputesBreadth(t *testing.T) {
	s := BuildSectorStrength("semis", []float64{10, -5, 20})
	if s.ConstituentCount != 3 {
		t.Fatalf("ConstituentCount = %d, want 3", s.ConstituentCount)
	}
	wantAvg := (10.0 - 5.0 + 20.0) / 3.0
	if s.StrengthScore != wantAvg {
		t.Errorf("StrengthScore = %v, want %v", s.StrengthScore, wantAvg)
	}
	wantBreadth := 2.0 / 3.0 * 100
	if diff := s.Breadth - wantBreadth; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("Breadth = %v, want %v", s.Breadth, wantBreadth)
	}
}

func TestBuildSectorStrength_EmptyInputIsZeroValueNotPanic(t *testing.T) {
	s := BuildSectorStrength("semis", nil)
	if s.ConstituentCount != 0 || s.StrengthScore != 0 || s.Breadth != 0 {
		t.Errorf("expected zero-value result for empty input, got %+v", s)
	}
}

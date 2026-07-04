package scoring

import "testing"

func TestComposite_WeightsSignalsAndBlendsQualityFactors(t *testing.T) {
	signals := []Signal{
		{Name: "momentum", Value: 80, Weight: 2},
		{Name: "sentiment", Value: 20, Weight: 1},
	}
	// weighted signal score = (80*2 + 20*1) / 3 = 60
	o := Composite("NVDA", "semis", signals, 80, 60, 70)

	wantSignalScore := 60.0
	wantComposite := wantSignalScore * ((80.0/100)*0.4 + (60.0/100)*0.3 + (70.0/100)*0.3)
	if diff := o.CompositeScore - wantComposite; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("composite = %v, want %v", o.CompositeScore, wantComposite)
	}
	if o.Confidence != "high" {
		t.Fatalf("confidence = %q, want high (evidence>=70 and >=3 total weight)", o.Confidence)
	}
}

func TestComposite_NoSignalsYieldsZeroScoreNotPanic(t *testing.T) {
	o := Composite("XYZ", "unknown", nil, 0, 0, 0)
	if o.CompositeScore != 0 {
		t.Fatalf("expected 0 composite with no signals, got %v", o.CompositeScore)
	}
	if o.Confidence != "low" {
		t.Fatalf("expected low confidence with no evidence, got %q", o.Confidence)
	}
}

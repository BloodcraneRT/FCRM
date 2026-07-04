package simulation

import "testing"

func TestRun_ExpectedValueMath(t *testing.T) {
	in := Input{
		Symbol:             "AVAV",
		Action:             ActionBuyNow,
		Horizon:            HorizonMonths,
		ProbabilitySuccess: 0.6,
		UpsideCase:         40,
		DownsideCase:       -15,
	}
	r := Run(in)
	want := 0.6*40 + 0.4*(-15) // 24 - 6 = 18
	if r.ExpectedReturn != want {
		t.Fatalf("expected return = %v, want %v", r.ExpectedReturn, want)
	}
}

func TestRun_CertainFailureMatchesDownside(t *testing.T) {
	r := Run(Input{ProbabilitySuccess: 0, UpsideCase: 100, DownsideCase: -50})
	if r.ExpectedReturn != -50 {
		t.Fatalf("expected -50 with 0%% success probability, got %v", r.ExpectedReturn)
	}
}

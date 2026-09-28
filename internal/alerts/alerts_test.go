package alerts

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/scoring"
)

func TestEvaluate_FiresOnlyRulesAboveThreshold(t *testing.T) {
	th := DefaultThresholds()

	scored := []scoring.ScoredSymbol{
		{
			Symbol:   "AVAV",
			Vertical: "defense",
			Inputs: scoring.SignalInputs{
				HasPriceData:    true,
				PriceMoveDayPct: 12, // above critical (10)
				FilingCount7d:   2,  // above min (1)
			},
			Opportunity: scoring.Opportunity{Symbol: "AVAV", Vertical: "defense", CompositeScore: 10, Explanation: "quiet"},
		},
		{
			Symbol:   "QUIET",
			Vertical: "misc",
			Inputs: scoring.SignalInputs{
				HasPriceData:    true,
				PriceMoveDayPct: 1, // below warning
				FilingCount7d:   0, // below min
			},
			Opportunity: scoring.Opportunity{Symbol: "QUIET", Vertical: "misc", CompositeScore: 5, Explanation: "nothing"},
		},
	}

	alerts := Evaluate(scored, th)

	var gotAVAV, gotQuiet int
	for _, a := range alerts {
		switch a.Symbol {
		case "AVAV":
			gotAVAV++
		case "QUIET":
			gotQuiet++
		}
	}
	if gotAVAV != 2 { // price (critical) + filing
		t.Errorf("AVAV fired %d alerts, want 2 (price critical + filing)", gotAVAV)
	}
	if gotQuiet != 0 {
		t.Errorf("QUIET fired %d alerts, want 0 (nothing crosses any threshold)", gotQuiet)
	}

	for _, a := range alerts {
		if a.Symbol == "AVAV" && a.AlertType == "price" && a.Severity != "critical" {
			t.Errorf("expected AVAV price alert to be critical (12%% >= 10%% threshold), got %s", a.Severity)
		}
	}
}

func TestEvaluate_ScoreChangeFiresOnMagnitudeNotSign(t *testing.T) {
	th := DefaultThresholds()
	scored := []scoring.ScoredSymbol{
		{Symbol: "NEG", Opportunity: scoring.Opportunity{Symbol: "NEG", CompositeScore: -60, Explanation: "bear case"}},
	}
	alerts := Evaluate(scored, th)
	if len(alerts) != 1 || alerts[0].AlertType != "score_change" {
		t.Fatalf("expected one score_change alert for a strongly negative score, got %+v", alerts)
	}
}

func TestPersist_SendsExpectedSQL(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery, _ = url.QueryUnescape(r.URL.Query().Get("query"))
		fmt.Fprint(w, `{"dataset":[]}`)
	}))
	defer srv.Close()

	db := questdb.NewClient(srv.URL)
	err := Persist(context.Background(), db, Alert{
		Symbol: "NVDA", AlertType: "price", Severity: "warning",
		Message: "test message", TriggeredValue: 7.5, Threshold: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(gotQuery, "alert_events") || !strings.Contains(gotQuery, "'NVDA'") || !strings.Contains(gotQuery, "false") {
		t.Errorf("unexpected SQL: %s", gotQuery)
	}
}

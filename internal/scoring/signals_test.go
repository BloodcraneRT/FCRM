package scoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func TestFetchSignalInputs_ReadsEachTableIndependently(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, _ := url.QueryUnescape(r.URL.Query().Get("query"))
		switch {
		case strings.Contains(q, "github_trends"):
			fmt.Fprint(w, `{"dataset":[[12.5]]}`)
		case strings.Contains(q, "filing_events"):
			fmt.Fprint(w, `{"dataset":[[2]]}`)
		case strings.Contains(q, "market_bars"):
			fmt.Fprint(w, `{"dataset":[[100.0,105.0]]}`)
		case strings.Contains(q, "source_health"):
			fmt.Fprint(w, `{"dataset":[["ok",3],["down",1]]}`)
		default:
			t.Fatalf("unexpected query: %s", q)
		}
	}))
	defer srv.Close()

	db := questdb.NewClient(srv.URL)
	in := FetchSignalInputs(context.Background(), db, "NVDA", "semis")

	if in.GithubMomentum1d != 12.5 {
		t.Errorf("GithubMomentum1d = %v, want 12.5", in.GithubMomentum1d)
	}
	if in.FilingCount7d != 2 {
		t.Errorf("FilingCount7d = %v, want 2", in.FilingCount7d)
	}
	if !in.HasPriceData || in.PriceMoveDayPct != 5.0 {
		t.Errorf("PriceMoveDayPct = %v (hasData=%v), want 5.0 (true)", in.PriceMoveDayPct, in.HasPriceData)
	}
	if in.SourceHealthPct != 75.0 {
		t.Errorf("SourceHealthPct = %v, want 75.0", in.SourceHealthPct)
	}
}

func TestFetchSignalInputs_DegradesGracefullyOnQueryError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"error":"table does not exist"}`)
	}))
	defer srv.Close()

	db := questdb.NewClient(srv.URL)
	in := FetchSignalInputs(context.Background(), db, "NVDA", "semis")

	if in != (SignalInputs{}) {
		t.Errorf("expected zero-value SignalInputs when every query errors, got %+v", in)
	}
}

func TestBuildOpportunity_ExplanationTracesEveryInput(t *testing.T) {
	o := BuildOpportunity("NVDA", "semis", SignalInputs{
		GithubMomentum1d: 10,
		FilingCount7d:    1,
		PriceMoveDayPct:  2,
		HasPriceData:     true,
		SourceHealthPct:  100,
	})

	if o.Confidence != "high" {
		t.Errorf("confidence = %q, want high (evidence_quality=100)", o.Confidence)
	}
	for _, want := range []string{"github_momentum", "filing_activity", "price_momentum"} {
		if !strings.Contains(o.Explanation, want) {
			t.Errorf("explanation missing %q: %s", want, o.Explanation)
		}
	}
}

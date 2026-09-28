// SignalInputs/FetchSignalInputs/BuildOpportunity close the loop that
// Composite/Persist left open: nothing in the codebase actually called them
// against real ingested data, so opportunity_scores stayed empty and the
// `opportunities` CLI command always showed nothing. This file reads the
// raw tables connectors already populate (github_trends, filing_events,
// market_bars, source_health) and turns them into an explainable
// Opportunity per watchlist symbol, with no invented numbers: every
// input traces back to a row a connector actually wrote.
package scoring

import (
	"context"
	"fmt"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// SignalInputs are the raw, already-ingested numbers BuildOpportunity turns
// into weighted Signals. Kept separate from the QuestDB queries that
// produce them so the scoring math is unit-testable without a database.
type SignalInputs struct {
	GithubMomentum1d float64 // avg stars_delta_1d across the vertical's tracked repos, last 24h
	FilingCount7d    float64 // count of filing_events for this symbol, last 7 days
	PriceMoveDayPct  float64 // % (close-open)/open from the latest market_bars quote row
	HasPriceData     bool    // whether any market_bars row exists for this symbol at all
	SourceHealthPct  float64 // 0-100, % of connector runs reporting "ok" in the last 24h
}

// FetchSignalInputs queries QuestDB for whatever raw data has actually been
// ingested for a symbol/vertical. Any single query failing (e.g. a table
// with no rows yet) degrades that input to its zero value rather than
// failing the whole fetch — a cold start should produce a low-confidence
// score, not an error.
func FetchSignalInputs(ctx context.Context, db *questdb.Client, symbol, vertical string) SignalInputs {
	var in SignalInputs

	if r, err := db.Query(ctx, fmt.Sprintf(
		"SELECT avg(stars_delta_1d) FROM github_trends WHERE vertical = %s AND ts > dateadd('d', -1, now())",
		lit(vertical),
	)); err == nil {
		in.GithubMomentum1d = firstFloat(r)
	}

	if r, err := db.Query(ctx, fmt.Sprintf(
		"SELECT count() FROM filing_events WHERE symbol = %s AND ts > dateadd('d', -7, now())",
		lit(symbol),
	)); err == nil {
		in.FilingCount7d = firstFloat(r)
	}

	if r, err := db.Query(ctx, fmt.Sprintf(
		"SELECT open, close FROM market_bars WHERE symbol = %s LATEST ON ts PARTITION BY symbol",
		lit(symbol),
	)); err == nil && len(r.Dataset) > 0 && len(r.Dataset[0]) == 2 {
		open := toFloat(r.Dataset[0][0])
		close := toFloat(r.Dataset[0][1])
		in.HasPriceData = true
		if open != 0 {
			in.PriceMoveDayPct = (close - open) / open * 100
		}
	}

	if r, err := db.Query(ctx, "SELECT status, count() FROM source_health WHERE ts > dateadd('d', -1, now()) GROUP BY status"); err == nil {
		var ok, total float64
		for _, row := range r.Dataset {
			if len(row) != 2 {
				continue
			}
			n := toFloat(row[1])
			total += n
			if s, _ := row[0].(string); s == "ok" {
				ok += n
			}
		}
		if total > 0 {
			in.SourceHealthPct = ok / total * 100
		}
	}

	return in
}

// BuildOpportunity turns SignalInputs into an explainable Opportunity using
// the same Composite() every manual/CLI path uses — auto-generated scores
// and hand-invoked ones go through identical, inspectable math.
func BuildOpportunity(symbol, vertical string, in SignalInputs) Opportunity {
	signals := []Signal{
		{
			Name:        "github_momentum",
			Value:       clamp(in.GithubMomentum1d*2, -100, 100),
			Weight:      1.0,
			Explanation: fmt.Sprintf("avg 1d GitHub star delta across %s-tagged repos = %.1f", vertical, in.GithubMomentum1d),
		},
		{
			Name:        "filing_activity",
			Value:       clamp(in.FilingCount7d*20, 0, 100),
			Weight:      1.0,
			Explanation: fmt.Sprintf("%d SEC filing(s) in the last 7 days", int(in.FilingCount7d)),
		},
		{
			Name:        "price_momentum",
			Value:       clamp(in.PriceMoveDayPct*10, -100, 100),
			Weight:      1.5,
			Explanation: fmt.Sprintf("latest quote day move = %.2f%%", in.PriceMoveDayPct),
		},
	}

	evidenceQuality := in.SourceHealthPct
	timingScore := clamp(in.FilingCount7d*25, 0, 100)
	practicalityScore := 50.0
	if in.HasPriceData {
		practicalityScore = 80.0
	}

	return Composite(symbol, vertical, signals, evidenceQuality, timingScore, practicalityScore)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func firstFloat(r *questdb.QueryResult) float64 {
	if len(r.Dataset) == 0 || len(r.Dataset[0]) == 0 {
		return 0
	}
	return toFloat(r.Dataset[0][0])
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

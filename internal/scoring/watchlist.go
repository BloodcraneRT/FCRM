package scoring

import (
	"context"
	"log"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// WatchlistItem is the minimal shape ScoreWatchlist needs from a tracked
// symbol. Defined here rather than imported from internal/watchlists so
// scoring never depends on the SQLite metadata package — callers adapt
// their own item type to this one.
type WatchlistItem struct {
	Symbol   string
	Vertical string
}

// ScoredSymbol bundles a symbol's raw inputs alongside the Opportunity they
// produced. Callers that need the precise, unclamped numbers (e.g.
// internal/alerts, which fires on an exact price move or filing count
// rather than a lossy 0-100 scaled Signal.Value) use Inputs directly
// instead of re-deriving them from Opportunity.Signals.
type ScoredSymbol struct {
	Symbol      string
	Vertical    string
	Inputs      SignalInputs
	Opportunity Opportunity
}

// ScoreWatchlist is what turns ingestion into opportunity/sector rows
// automatically: for every tracked symbol it fetches whatever raw signals
// connectors have actually landed in QuestDB, scores it, persists both the
// opportunity_scores row and its underlying signal_scores rows, then rolls
// per-vertical composite scores up into a sector_strength row. Call it once
// per ingest cycle (worker loop or `terminal pull`) after connectors run —
// scoring older/stale data is harmless since every row is timestamped.
//
// A failure scoring or persisting one symbol is logged and skipped rather
// than aborting the run; a bad symbol shouldn't block the rest of the
// watchlist from getting scored.
func ScoreWatchlist(ctx context.Context, db *questdb.Client, items []WatchlistItem) []ScoredSymbol {
	byVertical := map[string][]float64{}
	var scored []ScoredSymbol

	for _, it := range items {
		if it.Symbol == "" {
			continue
		}
		vertical := it.Vertical
		if vertical == "" {
			vertical = "unclassified"
		}

		inputs := FetchSignalInputs(ctx, db, it.Symbol, vertical)
		opp := BuildOpportunity(it.Symbol, vertical, inputs)
		if err := Persist(ctx, db, opp); err != nil {
			log.Printf("scoring: persist opportunity for %s: %v", it.Symbol, err)
			continue
		}

		scored = append(scored, ScoredSymbol{Symbol: it.Symbol, Vertical: vertical, Inputs: inputs, Opportunity: opp})
		byVertical[vertical] = append(byVertical[vertical], opp.CompositeScore)
	}

	for vertical, scores := range byVertical {
		strength := BuildSectorStrength(vertical, scores)
		if err := PersistSectorStrength(ctx, db, strength); err != nil {
			log.Printf("scoring: persist sector strength for %s: %v", vertical, err)
		}
	}

	return scored
}

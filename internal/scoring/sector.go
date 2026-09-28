package scoring

import (
	"context"
	"fmt"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// SectorStrength is a per-vertical rollup of the opportunity scores just
// computed for its symbols this run — deliberately derived from the same
// numbers opportunity_scores just got, not a separate ungrounded model, so
// "sector strength" and "opportunity ranking" never tell contradictory
// stories about the same data.
type SectorStrength struct {
	Vertical         string
	StrengthScore    float64 // avg composite score across the vertical's symbols this run, -100..100
	Breadth          float64 // % of symbols with a positive composite score
	ConstituentCount int
}

// BuildSectorStrength aggregates one run's worth of opportunity composite
// scores for a vertical. Momentum-over-time (1d/5d/1m) needs history this
// function doesn't have access to — left as 0/NaN here and a documented gap
// (see docs/architecture.md) until sector_strength has enough rows to
// compute deltas against past runs.
func BuildSectorStrength(vertical string, compositeScores []float64) SectorStrength {
	if len(compositeScores) == 0 {
		return SectorStrength{Vertical: vertical}
	}

	var sum float64
	var positive int
	for _, s := range compositeScores {
		sum += s
		if s > 0 {
			positive++
		}
	}

	return SectorStrength{
		Vertical:         vertical,
		StrengthScore:    sum / float64(len(compositeScores)),
		Breadth:          float64(positive) / float64(len(compositeScores)) * 100,
		ConstituentCount: len(compositeScores),
	}
}

func PersistSectorStrength(ctx context.Context, db *questdb.Client, s SectorStrength) error {
	sql := fmt.Sprintf(
		"INSERT INTO sector_strength (ts, vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m, constituent_count) "+
			"VALUES (now(), %s, %f, %f, NaN, NaN, NaN, %d)",
		lit(s.Vertical), s.StrengthScore, s.Breadth, s.ConstituentCount,
	)
	return db.Exec(ctx, sql)
}

// Package alerts turns scored watchlist symbols into alert_events rows.
// The alert_events table has existed in the schema since the platform was
// bootstrapped but nothing ever wrote to it — this closes that gap using
// the same signals scoring just computed, on explicit, inspectable
// thresholds (no anomaly-detection model, per the platform's "no black
// boxes" principle).
package alerts

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/scoring"
)

// Thresholds controls when a rule fires. All are plain numbers a user can
// reason about and override — there is no learned/implicit threshold.
type Thresholds struct {
	PriceMoveWarningPct  float64 // |day move %| at/above this -> warning
	PriceMoveCriticalPct float64 // |day move %| at/above this -> critical
	ScoreMagnitude       float64 // |composite_score| at/above this -> score_change alert
	MinFilingsForAlert   float64 // filing_count_7d at/above this -> filing alert
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		PriceMoveWarningPct:  5,
		PriceMoveCriticalPct: 10,
		ScoreMagnitude:       50,
		MinFilingsForAlert:   1,
	}
}

type Alert struct {
	Symbol         string
	AlertType      string // price | volume | news | filing | score_change
	Severity       string // info | warning | critical
	Message        string
	TriggeredValue float64
	Threshold      float64
}

// Evaluate checks every scored symbol against Thresholds and returns the
// alerts that fire. Pure function, no I/O — testable without QuestDB.
func Evaluate(scored []scoring.ScoredSymbol, th Thresholds) []Alert {
	var out []Alert
	for _, s := range scored {
		in := s.Inputs

		if in.HasPriceData {
			abs := math.Abs(in.PriceMoveDayPct)
			if abs >= th.PriceMoveWarningPct {
				severity := "warning"
				if abs >= th.PriceMoveCriticalPct {
					severity = "critical"
				}
				out = append(out, Alert{
					Symbol:         s.Symbol,
					AlertType:      "price",
					Severity:       severity,
					Message:        fmt.Sprintf("%s moved %.2f%% (latest quote vs. open), at/above the %.1f%% threshold", s.Symbol, in.PriceMoveDayPct, th.PriceMoveWarningPct),
					TriggeredValue: in.PriceMoveDayPct,
					Threshold:      th.PriceMoveWarningPct,
				})
			}
		}

		if in.FilingCount7d >= th.MinFilingsForAlert {
			out = append(out, Alert{
				Symbol:         s.Symbol,
				AlertType:      "filing",
				Severity:       "info",
				Message:        fmt.Sprintf("%s has %d SEC filing(s) in the last 7 days", s.Symbol, int(in.FilingCount7d)),
				TriggeredValue: in.FilingCount7d,
				Threshold:      th.MinFilingsForAlert,
			})
		}

		if abs := math.Abs(s.Opportunity.CompositeScore); abs >= th.ScoreMagnitude {
			out = append(out, Alert{
				Symbol:         s.Symbol,
				AlertType:      "score_change",
				Severity:       "warning",
				Message:        fmt.Sprintf("%s (%s) composite opportunity score %.1f crossed the %.1f threshold: %s", s.Symbol, s.Vertical, s.Opportunity.CompositeScore, th.ScoreMagnitude, s.Opportunity.Explanation),
				TriggeredValue: s.Opportunity.CompositeScore,
				Threshold:      th.ScoreMagnitude,
			})
		}
	}
	return out
}

// Persist writes an alert to alert_events, unacknowledged by default.
func Persist(ctx context.Context, db *questdb.Client, a Alert) error {
	sql := fmt.Sprintf(
		"INSERT INTO alert_events (ts, symbol, alert_type, severity, message, triggered_value, threshold, acknowledged) "+
			"VALUES (now(), %s, %s, %s, %s, %f, %f, false)",
		lit(a.Symbol), lit(a.AlertType), lit(a.Severity), lit(a.Message), a.TriggeredValue, a.Threshold,
	)
	return db.Exec(ctx, sql)
}

// EvaluateAndPersist is the entry point worker/CLI code calls: evaluate
// every rule against this cycle's scored symbols, persist whatever fires,
// and log (rather than abort) on a persist failure so one bad row doesn't
// drop the rest.
func EvaluateAndPersist(ctx context.Context, db *questdb.Client, scored []scoring.ScoredSymbol, th Thresholds) []Alert {
	fired := Evaluate(scored, th)
	for _, a := range fired {
		if err := Persist(ctx, db, a); err != nil {
			log.Printf("alerts: persist %s/%s for %s: %v", a.AlertType, a.Severity, a.Symbol, err)
		}
	}
	return fired
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

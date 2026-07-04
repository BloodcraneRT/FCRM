// Package scoring turns raw signals into an explainable opportunity score.
// The explicit design goal is "no black boxes": every score is a weighted
// sum of named, inspectable components, and every result carries a plain-
// English explanation string alongside the number.
package scoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Signal is one named, weighted input into a composite opportunity score.
// Value should be normalized to roughly -100..100 (or 0..100 for
// non-directional signals like evidence quality) before scoring.
type Signal struct {
	Name        string
	Value       float64
	Weight      float64 // relative importance, weights need not sum to 1 — normalized internally
	Explanation string
}

type Opportunity struct {
	Symbol            string
	Vertical          string
	CompositeScore    float64
	EvidenceQuality   float64
	TimingScore       float64
	PracticalityScore float64
	Confidence        string
	ThesisID          string
	Explanation       string
	Signals           []Signal
}

// Composite computes a weighted-average composite score from signals and
// builds a human-readable explanation of how it was derived — the score is
// always traceable back to its inputs, never a hidden model output.
func Composite(symbol, vertical string, signals []Signal, evidenceQuality, timingScore, practicalityScore float64) Opportunity {
	var weightedSum, totalWeight float64
	var parts []string
	for _, s := range signals {
		weightedSum += s.Value * s.Weight
		totalWeight += s.Weight
		parts = append(parts, fmt.Sprintf("%s=%.1f (w=%.2f)", s.Name, s.Value, s.Weight))
	}

	var signalScore float64
	if totalWeight > 0 {
		signalScore = weightedSum / totalWeight
	}

	// Composite blends the raw signal score with evidence/timing/practicality
	// as multipliers scaled 0-1, so a strong signal with weak evidence or bad
	// timing gets pulled down rather than hidden.
	composite := signalScore * ((evidenceQuality/100.0)*0.4 + (timingScore/100.0)*0.3 + (practicalityScore/100.0)*0.3)

	confidence := "low"
	switch {
	case evidenceQuality >= 70 && totalWeight >= 3:
		confidence = "high"
	case evidenceQuality >= 40:
		confidence = "medium"
	}

	explanation := fmt.Sprintf(
		"signal_score=%.1f from [%s]; evidence_quality=%.0f, timing=%.0f, practicality=%.0f -> composite=%.1f",
		signalScore, strings.Join(parts, ", "), evidenceQuality, timingScore, practicalityScore, composite,
	)

	return Opportunity{
		Symbol:            symbol,
		Vertical:          vertical,
		CompositeScore:    composite,
		EvidenceQuality:   evidenceQuality,
		TimingScore:       timingScore,
		PracticalityScore: practicalityScore,
		Confidence:        confidence,
		Explanation:       explanation,
		Signals:           signals,
	}
}

// Persist writes the opportunity score and each underlying signal to
// QuestDB so the full history — and the reasoning behind every past
// score — is queryable later, not just the latest snapshot.
func Persist(ctx context.Context, db *questdb.Client, o Opportunity) error {
	for _, s := range o.Signals {
		sql := fmt.Sprintf(
			"INSERT INTO signal_scores (ts, symbol, signal_name, value, weight, explanation, source) VALUES (now(), %s, %s, %f, %f, %s, 'scoring_engine')",
			lit(o.Symbol), lit(s.Name), s.Value, s.Weight, lit(s.Explanation),
		)
		if err := db.Exec(ctx, sql); err != nil {
			return fmt.Errorf("scoring: persist signal %s: %w", s.Name, err)
		}
	}

	sql := fmt.Sprintf(
		"INSERT INTO opportunity_scores (ts, symbol, vertical, composite_score, evidence_quality, timing_score, practicality_score, confidence, thesis_id, explanation) "+
			"VALUES (now(), %s, %s, %f, %f, %f, %f, %s, %s, %s)",
		lit(o.Symbol), lit(o.Vertical), o.CompositeScore, o.EvidenceQuality, o.TimingScore, o.PracticalityScore,
		lit(o.Confidence), lit(o.ThesisID), lit(o.Explanation),
	)
	if err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("scoring: persist opportunity: %w", err)
	}
	return nil
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

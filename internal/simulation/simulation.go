// Package simulation models the possible actions a trader could take on an
// opportunity and estimates expected value for each, so "what should I do"
// becomes a comparison of explicit numbers instead of a gut call.
package simulation

import (
	"context"
	"fmt"
	"strings"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Action is the fixed vocabulary of things a user can decide to do about an
// opportunity. Kept as a closed set so simulation results are comparable
// across symbols and over time.
type Action string

const (
	ActionBuyNow             Action = "buy_now"
	ActionWait               Action = "wait"
	ActionWatchConfirmation  Action = "watch_confirmation"
	ActionHedge              Action = "hedge"
	ActionSizeSmaller        Action = "size_smaller"
	ActionAvoid              Action = "avoid"
	ActionAddWatchlist       Action = "add_watchlist"
	ActionResearchDeeper     Action = "research_deeper"
	ActionSetAlert           Action = "set_alert"
	ActionCompareAlternative Action = "compare_alternative"
)

type Horizon string

const (
	HorizonDays   Horizon = "days"
	HorizonWeeks  Horizon = "weeks"
	HorizonMonths Horizon = "months"
	HorizonYears  Horizon = "years"
)

// Input is the trader's estimate of a scenario: how likely is the bull case,
// and what's the payoff in each direction. These should come from the
// thesis note, not be invented by the platform.
type Input struct {
	Symbol             string
	Action             Action
	Horizon            Horizon
	ProbabilitySuccess float64 // 0..1
	UpsideCase         float64 // % return if thesis plays out
	DownsideCase       float64 // % return (negative) if thesis fails
	Assumptions        string
}

type Result struct {
	Input
	ExpectedReturn float64 // probability-weighted % return
	Notes          string
}

// Run computes expected value from the trader-supplied scenario. It does
// not try to guess probabilities or payoffs itself — garbage in, garbage
// out is preferable to a false sense of precision from a hidden model.
func Run(in Input) Result {
	expected := in.ProbabilitySuccess*in.UpsideCase + (1-in.ProbabilitySuccess)*in.DownsideCase

	notes := fmt.Sprintf(
		"E[return] = %.1f%%*%.1f%% + %.1f%%*%.1f%% = %.2f%% over %s horizon for action=%s",
		in.ProbabilitySuccess*100, in.UpsideCase, (1-in.ProbabilitySuccess)*100, in.DownsideCase, expected, in.Horizon, in.Action,
	)

	return Result{Input: in, ExpectedReturn: expected, Notes: notes}
}

func Persist(ctx context.Context, db *questdb.Client, r Result) error {
	sql := fmt.Sprintf(
		"INSERT INTO simulation_runs (ts, symbol, action, horizon, expected_return, downside_case, upside_case, probability_success, assumptions, result_notes) "+
			"VALUES (now(), %s, %s, %s, %f, %f, %f, %f, %s, %s)",
		lit(r.Symbol), lit(string(r.Action)), lit(string(r.Horizon)), r.ExpectedReturn, r.DownsideCase, r.UpsideCase, r.ProbabilitySuccess,
		lit(r.Assumptions), lit(r.Notes),
	)
	return db.Exec(ctx, sql)
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

package main

import (
	"context"

	"github.com/spf13/cobra"
)

func (a *app) opportunitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "opportunities",
		Short: "Show latest opportunity ranking",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := a.db.Query(context.Background(), `
				SELECT symbol, vertical, composite_score, evidence_quality, timing_score, confidence, explanation
				FROM opportunity_scores
				LATEST ON ts PARTITION BY symbol
				ORDER BY composite_score DESC`)
			if err != nil {
				return err
			}
			printResult(r)
			return nil
		},
	}
}

func (a *app) catalystsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "catalysts",
		Short: "Show upcoming catalyst timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := a.db.Query(context.Background(), `
				SELECT ts, symbol, vertical, category, title, confidence
				FROM catalyst_events
				WHERE ts > dateadd('d', -1, now())
				ORDER BY ts ASC
				LIMIT 50`)
			if err != nil {
				return err
			}
			printResult(r)
			return nil
		},
	}
}

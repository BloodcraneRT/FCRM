package main

import (
	"context"

	"github.com/spf13/cobra"
)

func (a *app) sectorStrengthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sector-strength",
		Short: "Show latest sector/theme strength scores",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := a.db.Query(context.Background(), `
				SELECT vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m
				FROM sector_strength
				LATEST ON ts PARTITION BY vertical
				ORDER BY strength_score DESC`)
			if err != nil {
				return err
			}
			printResult(r)
			return nil
		},
	}
}

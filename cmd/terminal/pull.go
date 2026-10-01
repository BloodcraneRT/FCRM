package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/BloodcraneRT/fin-terminal/internal/connectors"
	"github.com/BloodcraneRT/fin-terminal/internal/ingest"
)

func (a *app) pullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Run all data connectors once and ingest into QuestDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			items, err := store.List(ctx, "default")
			if err != nil {
				return err
			}

			var symbols []string
			for _, it := range items {
				symbols = append(symbols, it.Symbol)
			}
			connectors := connectors.Build(ctx, a.cfg, symbols)

			results := ingest.RunAll(ctx, a.db, connectors)
			fmt.Printf("%-14s %-10s %-8s %s\n", "SOURCE", "ROWS", "LATENCY", "ERROR")
			for _, r := range results {
				errStr := ""
				if r.Err != nil {
					errStr = r.Err.Error()
				}
				fmt.Printf("%-14s %-10d %-8s %s\n", r.Source, r.Rows, r.Latency.Round(1e6), errStr)
			}
			return nil
		},
	}
}

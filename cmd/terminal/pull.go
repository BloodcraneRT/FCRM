package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/BloodcraneRT/fin-terminal/internal/datasources/finnhub"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/fred"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/githubtrends"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/secedgar"
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
			var secWatch []secedgar.Watch
			for _, it := range items {
				symbols = append(symbols, it.Symbol)
				if cik, ok := secedgar.CommonCIKs[it.Symbol]; ok {
					secWatch = append(secWatch, secedgar.Watch{Symbol: it.Symbol, CIK: cik})
				}
			}

			userAgent := os.Getenv("SEC_EDGAR_USER_AGENT")
			connectors := []ingest.Connector{
				secedgar.New(userAgent, secWatch),
				githubtrends.New(a.cfg.GitHubToken, seedRepos()),
				finnhub.New(a.cfg.FinnhubAPIKey, symbols),
				fred.New(a.cfg.FREDAPIKey, nil),
			}

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

// seedRepos are tracked as dev-adoption proxies across verticals this
// platform cares about. Extend via config as the watchlist of technologies
// grows — see docs/data-sources.md.
func seedRepos() []githubtrends.Repo {
	return []githubtrends.Repo{
		{OwnerSlashName: "questdb/questdb", Vertical: "data-infra"},
		{OwnerSlashName: "OpenBB-finance/OpenBB", Vertical: "fintech"},
		{OwnerSlashName: "ggerganov/llama.cpp", Vertical: "ai-infra"},
		{OwnerSlashName: "vllm-project/vllm", Vertical: "ai-infra"},
	}
}

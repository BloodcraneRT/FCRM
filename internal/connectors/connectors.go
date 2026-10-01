// Package connectors is the single place that decides which data
// connectors run, so the worker daemon and `terminal pull` never drift.
// Add a new source by registering it in Build.
package connectors

import (
	"context"
	"log"
	"os"

	"github.com/BloodcraneRT/fin-terminal/internal/config"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/finnhub"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/fred"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/githubtrends"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/secedgar"
	"github.com/BloodcraneRT/fin-terminal/internal/ingest"
)

// Repos tracked as developer-adoption proxies. Extend as verticals grow.
var Repos = []githubtrends.Repo{
	{OwnerSlashName: "questdb/questdb", Vertical: "data-infra"},
	{OwnerSlashName: "OpenBB-finance/OpenBB", Vertical: "fintech"},
	{OwnerSlashName: "ggerganov/llama.cpp", Vertical: "ai-infra"},
	{OwnerSlashName: "vllm-project/vllm", Vertical: "ai-infra"},
}

// Build returns every connector for the given watchlist symbols. CIKs are
// resolved live from SEC, falling back to the offline CommonCIKs map.
func Build(ctx context.Context, cfg config.Config, symbols []string) []ingest.Connector {
	ua := os.Getenv("SEC_EDGAR_USER_AGENT")

	ciks := map[string]string{}
	if ua != "" && len(symbols) > 0 {
		if m, err := secedgar.ResolveCIKs(ctx, ua, symbols); err != nil {
			log.Println("connectors: live CIK resolution failed, using fallback map:", err)
		} else {
			ciks = m
		}
	}
	var watch []secedgar.Watch
	for _, s := range symbols {
		cik, ok := ciks[s]
		if !ok {
			cik, ok = secedgar.CommonCIKs[s]
		}
		if ok {
			watch = append(watch, secedgar.Watch{Symbol: s, CIK: cik})
		}
	}

	return []ingest.Connector{
		secedgar.New(ua, watch),
		githubtrends.New(cfg.GitHubToken, Repos),
		finnhub.New(cfg.FinnhubAPIKey, symbols),
		fred.New(nil),
	}
}

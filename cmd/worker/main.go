// Command worker is the ingestion daemon: on a fixed interval it runs every
// registered connector, writes results to QuestDB, and records source
// health. This is what you run in Docker Compose for continuous ingestion;
// the CLI's `terminal pull` runs the same connectors on demand.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/config"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/finnhub"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/githubtrends"
	"github.com/BloodcraneRT/fin-terminal/internal/datasources/secedgar"
	"github.com/BloodcraneRT/fin-terminal/internal/ingest"
	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/watchlists"
)

func main() {
	cfg := config.Load()
	db := questdb.NewClient(cfg.QuestDBHTTPAddr)

	interval := 15 * time.Minute
	if v := os.Getenv("INGEST_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("fin-terminal worker starting: questdb=%s interval=%s", cfg.QuestDBHTTPAddr, interval)

	runOnce(ctx, cfg, db)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down")
			return
		case <-ticker.C:
			runOnce(ctx, cfg, db)
		}
	}
}

func runOnce(ctx context.Context, cfg config.Config, db *questdb.Client) {
	store, err := watchlists.Open(cfg.SQLitePath)
	if err != nil {
		log.Println("worker: cannot open watchlist store:", err)
		return
	}
	defer store.Close()

	items, err := store.List(ctx, "default")
	if err != nil {
		log.Println("worker: cannot list watchlist:", err)
		return
	}

	var symbols []string
	var secWatch []secedgar.Watch
	for _, it := range items {
		symbols = append(symbols, it.Symbol)
		if cik, ok := secedgar.CommonCIKs[it.Symbol]; ok {
			secWatch = append(secWatch, secedgar.Watch{Symbol: it.Symbol, CIK: cik})
		}
	}

	connectors := []ingest.Connector{
		secedgar.New(os.Getenv("SEC_EDGAR_USER_AGENT"), secWatch),
		githubtrends.New(cfg.GitHubToken, []githubtrends.Repo{
			{OwnerSlashName: "questdb/questdb", Vertical: "data-infra"},
			{OwnerSlashName: "OpenBB-finance/OpenBB", Vertical: "fintech"},
		}),
		finnhub.New(cfg.FinnhubAPIKey, symbols),
	}

	results := ingest.RunAll(ctx, db, connectors)
	for _, r := range results {
		if r.Err != nil {
			log.Printf("connector %-12s rows=%-4d latency=%-10s error=%v", r.Source, r.Rows, r.Latency, r.Err)
		} else {
			log.Printf("connector %-12s rows=%-4d latency=%-10s ok", r.Source, r.Rows, r.Latency)
		}
	}
}

# Runbook

## Local dev (no Docker)

```bash
# 1. Run QuestDB (needs Docker OR download the binary from questdb.io)
docker run -p 9000:9000 -p 9009:9009 -p 8812:8812 questdb/questdb:8.2.1

# 2. Apply schema
go run ./cmd/migrate

# 3. Add tickers to your watchlist
go run ./cmd/terminal watchlist add NVDA --vertical semis --note "AI compute demand"
go run ./cmd/terminal watchlist add AVAV --vertical defense --note "counter-drone contract cycle"

# 4. Set connector credentials (SEC EDGAR needs a descriptive User-Agent; Finnhub needs a free key)
export SEC_EDGAR_USER_AGENT="fin-terminal you@example.com"
export FINNHUB_API_KEY="..."        # free at finnhub.io
export GITHUB_TOKEN="..."           # optional, raises GitHub API rate limit

# 5. Pull data once
go run ./cmd/terminal pull

# 6. Check what's queryable
go run ./cmd/terminal status
go run ./cmd/terminal opportunities
go run ./cmd/terminal catalysts

# 7. Run the API (for the dashboard)
go run ./cmd/api

# 8. Run the worker (continuous ingestion, 15-min default interval)
go run ./cmd/worker
```

## Full stack via Docker Compose

```bash
cd docker
docker compose up --build
```

This starts QuestDB, runs migrations once, then starts the worker and API.
Set `GITHUB_TOKEN` / `FINNHUB_API_KEY` / `SEC_EDGAR_USER_AGENT` in your shell
or a `.env` file next to `docker-compose.yml` before running.

## Daily workflow

```bash
go run ./cmd/terminal pull                 # refresh data
go run ./cmd/terminal opportunities         # see current ranking
go run ./cmd/terminal thesis new NVDA "AI compute demand"   # write a thesis
go run ./cmd/terminal simulate NVDA --action watch_confirmation --horizon months --prob 0.6 --upside 30 --downside -10
go run ./cmd/terminal brief                 # export research/briefs/YYYY-MM-DD.md
```

## Tests

```bash
go test ./...
```

## Adding a new data connector

1. Create `internal/datasources/<name>/`.
2. Implement `Run(ctx context.Context, db *questdb.Client) (int, error)` and `Name() string` (satisfies `ingest.Connector`).
3. Register it in `cmd/terminal/pull.go` and `cmd/worker/main.go`.
4. Document it in `docs/data-sources.md`.

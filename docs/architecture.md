# Architecture

## Principles

- **Go-first.** Ingestion, API, CLI, scoring, and simulation are all Go. Python is not part of the runtime.
- **QuestDB is the only time-series store.** Prices, bars, events, signals, scores, and simulation history all live there. SQLite holds only local app metadata (watchlists, thesis index, settings) — never market data.
- **No black-box signals.** Every score in `signal_scores` and `opportunity_scores` carries an `explanation` string that shows exactly which inputs produced it.
- **Raw vs. enriched separation.** Ingestion connectors write raw observations (`market_bars`, `news_events`, `filing_events`, `github_trends`, ...). Scoring/simulation read those and write derived tables (`signal_scores`, `opportunity_scores`, `simulation_runs`). Nothing mutates raw tables after ingest.
- **Connectors are cheap to add.** Implement `ingest.Connector` (`Name() string`, `Run(ctx, *questdb.Client) (int, error)`), register it in `cmd/worker` and `cmd/terminal/pull.go`. That's it.
- **Source registry via `source_health`.** Every connector run writes a health row (ok/degraded/down, latency, rows ingested, error) — the dashboard can show data-source reliability over time, not just the latest pull.

## Processes

| Binary | Purpose |
|---|---|
| `cmd/migrate` | Applies `db/questdb/migrations/*.sql` in order, idempotently (`CREATE TABLE IF NOT EXISTS`). |
| `cmd/worker` | Long-running daemon; runs all connectors on `INGEST_INTERVAL` (default 15m). |
| `cmd/api` | HTTP/JSON API (see `docs/openapi.yaml`) backing the dashboard. Stdlib `net/http`, no framework. |
| `cmd/terminal` | CLI: watchlist management, on-demand `pull`, sector strength, opportunity ranking, catalyst timeline, thesis notes, simulation, daily brief export. |

## Data flow

```
[connectors: sec_edgar, github_trends, finnhub, ...]
        │  raw rows
        ▼
   QuestDB (market_bars, news_events, filing_events, github_trends, catalyst_events, source_health)
        │  read
        ▼
   internal/scoring (ScoreWatchlist, every ingest cycle) ──► signal_scores, opportunity_scores, sector_strength
        │
        ▼
   internal/simulation (Run, manual/CLI only — see below) ──► simulation_runs  (expected value per action, per trader-supplied scenario)
        │
        ▼
   internal/alerts (EvaluateAndPersist, every ingest cycle) ──► alert_events  (price/filing/score-change threshold crossings)
        │
        ▼
   cmd/api  ──►  web/dashboard
   cmd/terminal ──► CLI views, research/briefs/*.md
```

## QuestDB schema

See `db/questdb/migrations/001_core_schema.sql` for full DDL. Every table uses a designated `ts` timestamp column, `PARTITION BY DAY` (high-frequency) or `MONTH` (low-frequency), and WAL. `SYMBOL` columns (ticker, source, vertical, etc.) are dictionary-encoded for fast `GROUP BY` / `LATEST ON` queries — e.g. "give me the latest opportunity score per symbol" is:

```sql
SELECT * FROM opportunity_scores LATEST ON ts PARTITION BY symbol ORDER BY composite_score DESC;
```

## Local metadata (SQLite)

`internal/watchlists` owns a small SQLite DB (`modernc.org/sqlite`, pure Go, no cgo) with three tables: `watchlist_items`, `settings`, `thesis_index`. This is intentionally not versioned/time-series — "what am I tracking" not "what happened."

## Scoring vs. simulation: why only one is automatic

`internal/scoring.ScoreWatchlist` runs every ingest cycle (worker loop and `terminal pull` alike) because its inputs — GitHub star momentum, SEC filing counts, latest quote move, connector health — are all things the platform actually measured. `internal/simulation.Run` deliberately stays manual/CLI-only: its inputs (probability of success, upside/downside case) are a trader's judgment call, not something to be invented by the platform. Auto-generating those would violate the "no black boxes, no invented numbers" principle above just as badly as a hidden ML model would — see `internal/simulation/simulation.go`'s own doc comment ("does not try to guess probabilities or payoffs itself").

## Known gaps (see also docs/data-sources.md and the run report's Gaps section)

- ILP (line-protocol) ingestion path (`internal/questdb/ilp.go`) exists but connectors currently write via `/exec` SQL inserts for simplicity — fine at current volume, worth switching to ILP (or the official `questdb/go-questdb-client`) once ingesting intraday bars/ticks at real volume.
- `secedgar.CommonCIKs` is a small hand-seeded ticker→CIK map, verified once (2026-09-27) against SEC's own `company_tickers.json`; re-verify before extending it further.
- No auth on `cmd/api` — fine for local/self-hosted use, add a token/reverse-proxy before exposing beyond localhost.
- `scoring.BuildSectorStrength` only has this run's cross-sectional composite scores to work with, so `momentum_1d`/`momentum_5d`/`momentum_1m` are written as `NaN` — computing them needs a query against `sector_strength`'s own history (yesterday's/last week's/last month's `strength_score` per vertical), not yet implemented.
- `scoring.ScoreWatchlist`'s signals (GitHub momentum, filing count, single-quote day move) are intentionally simple and will read as near-zero/low-confidence until real connectors (a Finnhub key, a populated GitHub repo list per vertical) are actually running — that's correct behavior (low evidence → low confidence), not a bug, but worth knowing before trusting an early score.

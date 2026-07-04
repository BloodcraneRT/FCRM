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
   internal/scoring (Composite) ──► signal_scores, opportunity_scores  (with explanation strings)
        │
        ▼
   internal/simulation (Run) ──► simulation_runs  (expected value per action, per trader-supplied scenario)
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

## Known gaps (see also docs/data-sources.md and the run report's Gaps section)

- ILP (line-protocol) ingestion path (`internal/questdb/ilp.go`) exists but connectors currently write via `/exec` SQL inserts for simplicity — fine at current volume, worth switching to ILP (or the official `questdb/go-questdb-client`) once ingesting intraday bars/ticks at real volume.
- `secedgar.CommonCIKs` is a small hand-seeded ticker→CIK map; verify/extend before relying on it for symbols beyond the largest mega-caps.
- No auth on `cmd/api` — fine for local/self-hosted use, add a token/reverse-proxy before exposing beyond localhost.

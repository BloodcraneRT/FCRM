# fin-terminal

A self-hosted financial intelligence terminal: Go ingestion connectors, QuestDB
for all time-series data (prices, events, signals, scores, simulations),
SQLite for local watchlist/thesis metadata, a Go HTTP API, a lightweight
dashboard, and a CLI for daily research workflow.

Not a stock-tip generator. It's a repeatable system for tracking verticals
and catalysts, scoring opportunities with explainable signals, simulating
possible actions, and improving the platform every day.

## Quick start

```bash
docker run -p 9000:9000 -p 9009:9009 -p 8812:8812 questdb/questdb:8.2.1 &
go run ./cmd/migrate

go run ./cmd/terminal watchlist add NVDA --vertical semis --note "AI compute demand"
export SEC_EDGAR_USER_AGENT="fin-terminal you@example.com"
go run ./cmd/terminal pull          # ingest + auto-score + auto-alert in one step
go run ./cmd/terminal opportunities
go run ./cmd/terminal alerts

go run ./cmd/api    # dashboard + JSON API at http://localhost:8081
```

See `docs/runbook.md` for the full setup (Docker Compose, env vars, daily
workflow) and `docs/architecture.md` for how the pieces fit together.

## Layout

```
cmd/            terminal (CLI), api, worker, migrate
internal/       config, questdb, ingest, datasources, scoring, simulation, watchlists, api
db/questdb/     schema migrations
db/sample-data/ optional seed data
web/dashboard/  static dashboard (served by cmd/api)
research/       thesis notes, daily briefs, source notes
docs/           architecture, data sources, runbook, OpenAPI spec
docker/         Dockerfiles + docker-compose.yml
```

## Status

Early. See the latest research brief under `research/briefs/` and
`docs/architecture.md`'s "Known gaps" section for what's mocked, missing an
API key, or otherwise not production-grade yet.

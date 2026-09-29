# Finance Intelligence Terminal Run — 2026-09-28

**Scope:** incremental build run (per the prior run's advice). No new web/GitHub
research fan-out this run, so no new opportunity calls; the 2026-07-04 brief's
opportunity map and source ranking still stand and may be stale.

## What changed
- **FRED macro connector** (`internal/datasources/fred`): 11 series (10Y, 2Y,
  10Y-2Y, fed funds, CPI, unemployment, HY OAS, USD index, WTI, Fed balance
  sheet, VIX). Missing values (".") skipped, inserts chunked at 100 rows,
  API key scrubbed from transport errors.
- **`macro_series` table** (migration 002), `DEDUP UPSERT KEYS(ts, series_id)` so
  overlapping pulls are idempotent.
- Wired into `terminal pull` and the worker; `GET /api/v1/macro` (latest per
  series) added to the API and OpenAPI spec; `FRED_API_KEY` passed to compose.
- Tests: parsing/skip-missing/SQL shape, missing key, key non-leak (httptest).

## Verified
`go build`, `go vet`, `gofmt`, `go test ./...` all clean.

## Gaps / risks
- Still not run against a live QuestDB (no Docker here); the `DEDUP UPSERT KEYS`
  DDL and the ISO-timestamp inserts are unverified against a real server.
- `FRED_API_KEY` not set anywhere; connector reports "down" until it is.
- Macro data isn't yet consumed by scoring or the dashboard.
- CIK map, alert producer, worker→scoring wiring: unchanged from prior run.

## Next best steps
1. Derive regime signals (curve inversion, HY OAS widening, VIX spike) from
   `macro_series` into `signal_scores`; show a macro panel on the dashboard.
2. Wire scoring/simulation into the worker loop.
3. Verify `secedgar.CommonCIKs`; build the `alert_events` producer.
4. Next research-only run: refresh opportunity map and source ranking.

## Routine improvement
Alternate run types: odd runs = research + brief only; even runs = build one
item from "Next best steps". Check first whether Docker is available to run
QuestDB for a real integration test.

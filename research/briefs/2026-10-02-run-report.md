# Run 2026-10-02 (incremental build run)

Followed the previous run's advice: one top-ranked item, diff-sized.

## Changes
- **Data**: new `macro_series` table (migration 002, DEDUP UPSERT on ts+series_id).
- **Connector**: `internal/datasources/fred` — 8 default macro series, idempotent upserts, API key redacted from errors, httptest-based tests.
- **Wiring**: registered in `cmd/worker` and `terminal pull`.
- **Docs**: data-sources.md updated.

## Gaps
- No live web/GitHub research fan-out this run (sandbox run kept to the build item); opportunity map from 2026-07-04 is unchanged and stale.
- Not tested against a real QuestDB or real FRED key; DEDUP syntax needs verification on the deployed QuestDB version.
- Migration runner picks up 002 via glob (covered by existing migrate test).

## Next
1. Verify against live QuestDB (Docker) and FRED key.
2. Finnhub earnings-calendar connector -> `earnings_events`.
3. RSS news connector -> `news_events`.
4. Surface macro_series in API/dashboard.
5. Write scoring/simulation outputs from worker loop; alert_events producer.

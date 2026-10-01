# Finance Intelligence Terminal — Run Report, 2026-09-30

Incremental build run: picked the top "next best steps" from the 2026-07-04
report (FRED macro connector, verify CIKs) rather than re-doing the whole
platform.

## What changed

- **New `macro_series` table** (`db/questdb/migrations/002_macro_series.sql`),
  `DEDUP UPSERT KEYS(ts, series_id)` so overlapping re-pulls are idempotent.
- **New keyless FRED connector** (`internal/datasources/fred`): DGS10, DGS2,
  T10Y2Y, DFF, VIXCLS, BAMLH0A0HYM2 (HY OAS), CPIAUCSL, UNRATE, DCOILWTICO,
  DTWEXBGS via the public `fredgraph.csv` endpoint. Verified live against all
  10 series today; unit-tested CSV parser (handles "." missing markers).
- **Bug fix: two wrong CIKs.** `AVAV` and `KTOS` in `secedgar.CommonCIKs`
  were wrong (checked against SEC `company_tickers.json`): AVAV is
  0001368622, KTOS is 0001069258. The other 12 were correct.
- **Live CIK resolution** (`secedgar.ResolveCIKs`): any watchlist ticker now
  resolves from SEC; the static map is only an offline fallback.
- **Bug fix: duplicate filings.** EDGAR connector re-inserted the newest 20
  filings on every run; it now skips accession numbers already stored.
- **`internal/connectors`**: one registry used by both worker and
  `terminal pull`, which had duplicated (and diverging) wiring.

## Macro snapshot (FRED, latest obs)

10Y 5.26%, 2Y 4.89%, 10Y-2Y +0.41 (steepening), fed funds 3.88%, VIX 16.0,
HY OAS 3.08, WTI $96.16, UNRATE 4.1% (Aug). Read: long-end yields high while
the policy rate is lower and credit spreads are tight; oil near $96 is a
tailwind for energy and an input-cost risk for consumers. Not investment
advice; these are raw prints, not yet scored by the platform.

## Research notes

Searches this run surfaced: hyperscaler 2026 capex estimates of ~$660–690B
(~$450B AI-directed) per secondary sources, power semiconductors (SiC/GaN)
forecast at $71.4B by 2031 (TechInsights), and defense backlog/small-cap
AI-ISR themes. These are consistent with the AI-power-bottleneck and
counter-drone theses in the 2026-07-04 brief but are secondary-source
estimates; no thesis was re-scored. Open-source finds: OpenTerminalUI
(github.com/Hitheshkaranth/OpenTerminalUI) worth skimming for UI ideas;
QuestDB's 9.x line adds features worth checking against our pinned 8.2.1.

## Gaps / risks

- Still no live QuestDB in this sandbox; the new INSERT SQL and DEDUP DDL
  are not run against a real instance. Run `go run ./cmd/migrate` locally.
- Finnhub key still unset, so `market_bars` stays empty. Next run: add a
  keyless EOD price connector (Stooq CSV) so prices don't depend on a key.
- `SEC_EDGAR_USER_AGENT` must be set or filings are skipped.
- FRED CPI/UNRATE are monthly; 120-day lookback yields only ~3 points.

## Next best steps

1. Keyless EOD prices (Stooq) → `market_bars`; then `sector_strength` from ETFs.
2. Wire scoring/simulation into the worker loop; add `alert_events` producer.
3. Macro panel in the dashboard + `terminal macro` command.
4. Earnings calendar connector; RSS news connector.
5. Pin QuestDB to a current release after testing migrations.

## Prompt/routine improvements

- Tell the routine to read the latest brief's "Next best steps" first and ship
  one or two items per run.
- State the sandbox facts up front: no nested Docker, outbound HTTPS works
  for sec.gov and fred.stlouisfed.org, so prefer keyless sources.
- Ask for a sandbox-runnable integration check (e.g. QuestDB binary) so SQL
  gets exercised, not just unit-tested.

# Run 2026-09-29

## Summary
Built two keyless connectors that fill the two biggest data gaps: `market_bars` (Yahoo daily bars for watchlist + benchmark ETFs/indexes) and `news_events` (RSS headlines, ticker-tagged). Both were smoke-tested against the live endpoints (15 bars, 40 headlines parsed; QuestDB stubbed). Migration `003` adds dedup upserts.

## State of the repo
PR #1 (FRED, auto-scoring, alerts, CIK fix) and PR #2 (FRED, duplicate of #1's connector; both add migration `002`) are open, unmerged drafts. **They conflict with each other**; merge one and close the other. This run deliberately avoided FRED/scoring/alerts to not add a third overlap. Note the scoring in PR #1 only becomes useful once `market_bars` has data, which this PR provides keylessly.

## Research (thin this run)
Web search returned mostly newsletters and aggregator pages, so no market thesis is asserted. Unverified themes only: AI infra/semis and energy-for-data-centers were reported as 2026 YTD leaders (XLK ~+33%, XLE ~+21% in H1, per a secondary source). Treat as a prompt for the sector-strength view, not evidence. OSS to look at: OpenTerminalUI (self-hosted terminal), QuantDinger, OpenBB, Fincept Terminal; none are Go.

## Gaps
- QuestDB never run live; DEDUP DDL and multi-row INSERT unverified.
- Yahoo endpoint is unofficial; Stooq (also reachable) is a candidate fallback.
- RSS ticker match is regex-based; 2-letter tickers only match as `$MP` or `(NYSE: MP)`.
- News sentiment is NULL; no scoring consumes news yet.

## Next
1. Resolve PR #1 vs #2, then wire news counts/importance into scoring.
2. Run docker compose live and verify migrations 001-003.
3. Earnings calendar connector (Finnhub free) into `earnings_events`.
4. Stooq fallback for bars; DeFiLlama for crypto infra.

## Routine improvement
Start each run by listing open PRs and choosing work that does not overlap them; ask for merges rather than stacking parallel branches.

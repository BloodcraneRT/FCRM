# Finance Intelligence Terminal — Run Report, 2026-09-27

This is the second recorded run (see `2026-07-04-run-report.md` for the
bootstrap run). Per that run's own "prompt improvements" note, this run is
incremental: it worked the top unfinished items off the July run's "Next
best steps" list before doing a fresh research pass, rather than
re-researching and re-bootstrapping everything from scratch.

## What changed in the platform this run

1. **Fixed a real data-correctness bug**: `secedgar.CommonCIKs` had wrong
   CIKs for AVAV (pointed at a stale/incorrect CIK) and KTOS (pointed at a
   different registrant). Verified all 14 entries against SEC's own
   `company_tickers.json`; both are now correct, the rest already matched.
   This would have silently pulled the wrong companies' SEC filings into
   `filing_events` for two of the platform's highest-conviction watchlist
   names.
2. **Built the FRED macro connector** (`internal/datasources/fred`) — the
   #1 ranked unbuilt data source from the July run. Pulls 10 free official
   series (10Y/2Y yields, 10Y-2Y spread, fed funds rate, unemployment,
   payrolls, CPI, industrial production, WTI crude, VIX) into a new
   `macro_series` QuestDB table (`db/questdb/migrations/002_macro_series.sql`).
   Unit-tested against a mocked FRED API; needs a free `FRED_API_KEY` to
   run live.
3. **Closed the biggest gap found while reviewing the codebase**:
   `opportunity_scores` and `sector_strength` had zero producers —
   `internal/scoring.Composite`/`Persist` existed but nothing ever called
   them, so `terminal opportunities` and `terminal sector-strength` always
   returned empty. Added `internal/scoring.ScoreWatchlist`, which reads
   whatever's actually been ingested (GitHub star momentum, SEC filing
   count, latest quote move, connector health) for every watchlist symbol,
   builds an explainable `Opportunity` via the existing `Composite()` math,
   persists it, and rolls per-vertical scores into `sector_strength`. Wired
   into both the worker loop and `terminal pull`, so scores now populate
   automatically every ingest cycle — no manual CLI invocation required.
   `internal/simulation` deliberately stays manual-only; see
   `docs/architecture.md` for why auto-simulating would violate the
   platform's "no invented numbers" principle.
4. **Built the `alert_events` producer** (`internal/alerts`) — another
   schema table that existed with zero producers. Three explicit,
   inspectable threshold rules (day price move, new filing activity,
   composite score magnitude) evaluated every ingest cycle against that
   cycle's scored symbols. Added `terminal alerts` CLI command,
   `GET /api/v1/alerts`, and an Alerts section in the dashboard.
5. Docs updated to match: `docs/data-sources.md`, `docs/architecture.md`
   (including a new "why only scoring is automatic, not simulation"
   section and expanded Known Gaps), `docs/openapi.yaml`.
6. All new code has unit tests (mocked HTTP for both FRED and QuestDB);
   `go build ./...`, `go vet ./...`, `gofmt -l .` (clean), and
   `go test ./...` all pass. No live QuestDB in this sandbox still — same
   constraint as the July run.

## Research pass: verticals scanned, September 2026 state

- **AI infra / power grid**: still the highest-conviction structural
  thesis. Texas (ERCOT, ~474 GW interconnect queue, ~90% data centers)
  froze new data-center grid connections and permits in August/September;
  multiple counties adopted moratoria. Morgan Stanley projects a ~49 GW
  power shortfall against ~74 GW of 2028 demand. Equinix's under-construction
  capacity is 96% pre-leased at 1.4% vacancy. The bottleneck thesis from
  July has only gotten more confirmed, not less — this is no longer a
  contrarian call, it's consensus, which changes the risk/reward (see
  Opportunities below).
- **Semiconductors / memory**: HBM is now the harder constraint than logic.
  SK Hynix and Micron report 2026 HBM production fully sold out; Global X
  and CSIS both frame memory (not GPUs) as the 2026 AI semiconductor
  bottleneck. SK Hynix is building a $3.9B Indiana packaging plant; Micron
  is planning a US advanced-packaging facility. Price spikes (~50% by
  some estimates) are a margin tailwind for memory makers and a margin
  headwind for everyone downstream who consumes memory.
- **Defense / counter-drone**: AVAV and KTOS are both down significantly
  YTD despite record backlogs — a real evidence-vs-price divergence.
  AVAV landed a $465M Army laser-weapon contract plus a $50M international
  LOCUST order and record $480.5M quarterly revenue. Pentagon's FY2027
  budget request includes ~$75B for unmanned/counter-drone systems. The
  July thesis (backlog growth not yet priced in) looks intact; the stock
  weakness this quarter is itself worth investigating as a potential
  entry rather than a thesis failure — flagged as a research-deeper item,
  not a conclusion.
- **Rare earths**: MP Materials Q2 2026 revenue $108.5M, +89% YoY, driven
  by NdPr sales; stock still down 17% over the trailing year and below
  consensus target — a real disconnect between the operating ramp and the
  share price. Both MP and USAR sold off in early September on US-China
  "thaw" headlines, confirming the July note that policy de-escalation is
  the main tail risk to the entire vertical, not a hypothetical.
- **Macro backdrop — this is the biggest surprise vs. the July run**: the
  FOMC's September 2026 meeting *hiked* to 3.75-4.0% on persistent core
  PCE inflation above 3% all year, reversing the market's earlier
  rate-cut expectations (JPMorgan now sees another hike in December, not
  cuts). The 2s10s curve is +26 to +33 bps — positively sloped, no
  recession signal. This matters for every rate-sensitive vertical
  (CRE, regional banks, growth/duration-sensitive names) covered below
  and is exactly the kind of regime signal the new `macro_series`/FRED
  connector exists to track automatically going forward instead of via a
  one-off web search each run.
- **CRE / regional banks**: still deteriorating, not stabilizing. Office
  loan delinquency hit 12.34% (highest since tracking began in 2000);
  some banks carry CRE concentration over 300% of capital. Banks are
  extending/modifying rather than recognizing losses — analysts explicitly
  warn this "postpones the inevitable," which combined with a hiking (not
  cutting) Fed removes the main relief valve (falling rates lowering cap
  rates) the bull case for a 2026 stabilization depended on.
- **Insurance / reinsurance**: pricing pressure continuing as expected —
  Guy Carpenter's property-cat rate-on-line index fell 14% at April 2026
  renewals (vs. -12% in January), confirming the July "steepest rate
  decline since 2014" read. 2026 hurricane season projected below-normal
  on a building El Niño. Global reinsurance capital is at a record >$700B.
  This is a slow, well-telegraphed soft-market grind, not a catalyst-driven
  trade — fits `sector_strength` tracking better than a single-name bet.

## GitHub / open-source finds

- **TradingAgents** (`TauricResearch/TradingAgents`) — a multi-agent LLM
  trading framework (fundamental/sentiment/technical analyst agents,
  bull/bear researcher debate, a trading agent, a risk-management sign-off
  layer) went from ~9,300 stars in March 2026 to ~99,456 by August 24,
  overtaking `ai-hedge-fund` as the most-starred finance-AI repo on
  GitHub. Worth tracking as a `github_trends` row (vertical: `ai-infra`
  or a new `quant-agents` tag) — its star velocity is itself a genuine
  adoption signal for "LLM-agents-as-trading-research" as a category, even
  setting aside whether any specific strategy it produces is any good.
- **Fincept Terminal** (`Fincept-Corporation/FinceptTerminal`) — a native
  C++20/Qt6 desktop finance terminal (DCF, portfolio optimization,
  VaR/Sharpe, 100+ data connectors, dozens of AI trader/analyst agents).
  Directly comparable in ambition to this project's stated goal, but
  Python/C++-centric rather than Go-first — worth periodically diffing
  its connector list against `docs/data-sources.md` for sources we're
  missing, not worth adopting its stack.
- **OpenBB** — already tracked in `seedRepos()`; still the standard
  "connect once, consume everywhere" open data layer (Python/Excel/MCP
  surfaces). No action needed beyond continuing to watch its
  `github_trends` row.
- **QuantConnect Lean** — mature, still the reference open-source
  backtesting/live-trading engine (C#/Python). Not adopted (Go-first
  constraint) but its documented event-driven backtest architecture is a
  reasonable reference if/when `internal/simulation` grows a backtesting
  mode beyond single-scenario expected-value math.

## Gaps / risks

- Same core gap as July: **no live QuestDB in this sandbox** — everything
  new this run (`fred`, `scoring.ScoreWatchlist`, `alerts`) is unit-tested
  against mocked HTTP, not run against a real QuestDB instance. Run
  `docker compose up`, `go run ./cmd/migrate`, then `go run ./cmd/worker`
  or `terminal pull` locally to validate the full path before trusting it.
- `FRED_API_KEY` and `FINNHUB_API_KEY` are still unset — `macro_series`
  and `market_bars` will stay empty (and therefore `scoring.ScoreWatchlist`
  will keep producing low-confidence, near-zero scores) until real keys
  are configured. This is correct degrade-gracefully behavior, not a bug,
  but it means the auto-scoring pipeline built this run can't be
  meaningfully validated end-to-end without keys either.
- `sector_strength.momentum_1d/5d/1m` are written as `NaN` — computing
  real momentum needs a query against `sector_strength`'s own history,
  not yet built (see `docs/architecture.md` Known Gaps).
- The September-2026 research pass above is a single web-search pass, same
  caveat as July: treat it as a starting point for `terminal thesis new`,
  not a conclusion. Nothing in it has been run through the platform's own
  (now-automatic) scoring against live data yet.

## Next best steps

1. Get free `FRED_API_KEY` and `FINNHUB_API_KEY`, run the full stack
   (`docker compose up` → `migrate` → `worker`/`pull`) to validate
   `macro_series`, `market_bars`, and the new auto-scoring/alerts pipeline
   end-to-end against live data — the single highest-value next step since
   three features now depend on it working correctly.
2. Compute real `sector_strength` momentum (1d/5d/1m) from its own history
   now that the table has a producer writing to it regularly.
3. Add a `github_trends` entry for `TauricResearch/TradingAgents` (tagged
   `ai-infra` or a new `quant-agents` vertical) to track the open-source
   AI-trading-agent category's adoption velocity as a signal in its own
   right.
4. Add the USPTO PatentsView connector (still #1 on the ranked-next-sources
   list, free/no-key, unblocked by nothing).
5. Write `research/thesis/` notes for the two highest-conviction items
   from this run's pass — AVAV/KTOS backlog-vs-price divergence and the
   MP Materials operating-ramp-vs-share-price divergence — and run them
   through `terminal simulate` once real inputs (not just the research
   pass) support a probability/upside/downside estimate.

## Prompt/routine improvements for next run

- The July run's advice to "pick the top unfinished item first" worked
  well — keep doing that. This run additionally suggests: once
  `FRED_API_KEY`/`FINNHUB_API_KEY` exist, the very next run's first action
  should be running the full pipeline live and fixing whatever breaks,
  before any further research or feature work — three features (fred,
  scoring, alerts) are now stacked on data that's never actually flowed
  through them.
- Verifying `secedgar.CommonCIKs` against SEC's public
  `company_tickers.json` (no key required) was cheap and caught a real
  bug; worth a periodic (e.g. monthly) automated re-check rather than a
  one-time fix, since tickers occasionally change CIK association on
  corporate actions.
- Consider having each run end by running `go test ./...` and pasting a
  one-line pass/fail summary into this file automatically, so a future
  reader doesn't have to trust the prose above.

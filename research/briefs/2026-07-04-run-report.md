# Finance Intelligence Terminal — Run Report, 2026-07-04

Full run output lives in the chat session; this file is the durable copy for
the repo. See `docs/architecture.md`, `docs/data-sources.md`, and
`docs/runbook.md` for the living reference versions of sections 7-9 below.

## Best opportunities found (ranked, see caveats)

1. **Counter-drone defense (AVAV, KTOS)** — highest evidence quality: AVAV's
   $500M Army Titan RF contract (July 2, 2026) and 65% backlog growth are
   confirmed, not speculative. Main risk: multiple expansion already priced
   in a 35%-in-3-days move; AeroVironment Investor Day July 8, 2026 is the
   near-term confirm/deny catalyst.
2. **AI infra power bottleneck (BE, GEV, VST)** — market still framing this
   as a chip story; grid interconnect queues (24-36 months) and transformer
   lead times (~160 weeks) suggest power/generation equipment is the
   binding constraint and may be underpriced relative to how structural it
   is. Time horizon: months-years, needs continued capex-guidance
   confirmation (late July hyperscaler earnings).
3. **Rare earths (MP, USAR)** — thesis largely already re-rated (USAR
   +107% YTD); real remaining edge is in the H2 2026 magnet-revenue ramp at
   MP's Independence facility, not the scarcity premium itself, which is a
   policy-reversal tail risk (US-China de-escalation would remove it fast).
4. **Insurance/reinsurance cycle turn (RNR, ACGL, CB)** — soft-market
   transition (steepest rate decline since 2014) looks underappreciated by
   equities still priced for peak-cycle margins; hurricane season
   (Aug-Oct 2026) is the swing catalyst in either direction.
5. **CRE/regional bank distress** — genuine dislocation (12.34% office CMBS
   delinquency, record distressed-sale share) but the bear case (structural,
   not cyclical, demand decline) is real; only interesting for
   well-capitalized opportunistic entries, not a blanket long.

**Caveat:** all of the above came from a single research pass on 2026-07-04;
treat as a starting point for thesis notes (`terminal thesis new <symbol>`),
not a conclusion. Nothing here has been run through the platform's own
scoring/simulation yet — that requires live QuestDB + connector data.

## Verticals scanned this run

AI infra/data centers, semiconductors, energy/power grid, telecom,
cybersecurity, defense, fintech, biotech, real estate, commodities, crypto
infra, small caps, distressed assets, industrial automation/supply chain,
insurance, consumer trends, software infrastructure. Full per-vertical
bull/bear/catalyst/risk breakdown is in the session transcript; distilled
into `catalyst_events` seed rows at `db/sample-data/seed_catalysts.sql`.

## What changed in the platform this run

- Bootstrapped the full Go repo (`cmd/{terminal,api,worker,migrate}`,
  `internal/{config,questdb,ingest,datasources,scoring,simulation,
  watchlists,api}`).
- QuestDB schema: 13 tables per the spec (`market_bars` … `source_health`).
- Three working ingestion connectors: SEC EDGAR filings (no key needed),
  GitHub repo trends (uses existing `GITHUB_TOKEN`), Finnhub quotes (needs
  a free key — not yet configured, see Gaps).
- Explainable scoring engine (`internal/scoring`) — every score carries the
  weighted inputs that produced it as a string, not just a number.
- Simulation engine (`internal/simulation`) modeling the 10 requested
  actions (buy_now, wait, watch_confirmation, hedge, size_smaller, avoid,
  add_watchlist, research_deeper, set_alert, compare_alternative) as
  explicit expected-value math over trader-supplied probability/upside/
  downside inputs.
- CLI (`cmd/terminal`, cobra): `watchlist add/remove/show`, `pull`,
  `sector-strength`, `opportunities`, `catalysts`, `thesis new`,
  `simulate`, `brief`, `status`, `dashboard`.
- HTTP API (`cmd/api`) + static dashboard (`web/dashboard`) — plain
  net/http and vanilla JS/htmx-style fetch, no build step.
- Docker Compose stack (QuestDB + migrate + worker + api).
- Unit tests for the QuestDB HTTP client, ILP line-protocol writer,
  scoring math, simulation math, and the migration SQL splitter — all
  passing. Full docker-compose stack could not be smoke-tested live in
  this sandbox (no privileged Docker-in-Docker); see Gaps.

## Gaps / risks (see also docs/architecture.md → Known gaps)

- **No live QuestDB in this environment** (nested Docker unavailable) —
  schema and connector SQL are unit-tested and manually reviewed but not
  yet run against a real QuestDB instance. Run `docker compose up` locally
  and re-verify before trusting ingested data.
- **No API keys configured** — Finnhub (`FINNHUB_API_KEY`) needed for
  market_bars to populate at all; FRED/USPTO/DeFiLlama connectors are
  designed but not yet built.
- **`secedgar.CommonCIKs` is a small hand-seeded map** — verify each CIK
  before relying on it; only mega-caps (AAPL/MSFT/NVDA/GOOGL/AMZN/META/TSLA)
  are confidently correct, the rest (AVAV/KTOS/MP/CRCL/BE/GEV/VST) should be
  double-checked against SEC's company search before production use.
- **Seed catalyst/sector data in `db/sample-data/` is illustrative**,
  sourced from one research pass — not a live feed, will go stale
  immediately.
- **No auth on the API** — fine for localhost, needs a token or reverse
  proxy before exposing beyond that.

## Next best steps

1. Get a free Finnhub key, set `FINNHUB_API_KEY`, run `terminal pull`
   against a real QuestDB to validate the full path end-to-end.
2. Build the FRED connector (macro backbone, free key, generous limits) —
   highest ranked unbuilt source.
3. Wire `internal/scoring` and `internal/simulation` into the worker loop
   so `opportunity_scores`/`simulation_runs` populate automatically instead
   of only via manual CLI calls.
4. Verify/replace the guessed CIKs in `secedgar.CommonCIKs`.
5. Add an `alert_events` producer (price/volume/news thresholds) — schema
   exists, nothing writes to it yet.

## Prompt/routine improvements for next run

- Split the mega-prompt into phases across runs: (1) research-only runs
  that append to `research/briefs/`, (2) build runs that pick one
  connector/feature off the "Next best steps" list rather than trying to
  rebuild the whole platform every time. This run intentionally did a
  broad first pass; subsequent runs should be incremental and diff-sized.
- Next run should start by reading the previous run's "Next best steps"
  section (this one) and picking the top unfinished item first, before any
  new research fan-out.
- Add a check at the start of each run for whether Docker-in-Docker is
  available in the current sandbox; if so, actually spin up QuestDB and
  run the full integration path instead of unit tests alone.

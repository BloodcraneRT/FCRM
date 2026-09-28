# Data Sources

Ranked list from the 2026 research pass. "Implemented" = a connector exists in `internal/datasources/`. Everything else is a documented next step.

## Implemented

| Source | Provides | Update freq | API key | Go fit | QuestDB landing |
|---|---|---|---|---|---|
| **SEC EDGAR** (`internal/datasources/secedgar`) | Company filings (10-K/10-Q/8-K/4/13F) via `data.sec.gov/submissions` | Real-time as filed | None (User-Agent header only) | Clean JSON, ideal | `filing_events` |
| **GitHub REST API** (`internal/datasources/githubtrends`) | Repo stars/forks as a dev-adoption proxy | On demand / 15min poll | Optional (`GITHUB_TOKEN`, raises 60/hr → 5,000/hr) | Clean JSON | `github_trends` |
| **Finnhub** (`internal/datasources/finnhub`) | Real-time-ish equity quotes | 60 req/min free tier | Required (`FINNHUB_API_KEY`, free at finnhub.io) | Clean JSON | `market_bars` |
| **FRED** (`internal/datasources/fred`) | Official macro backbone: 10Y/2Y yields, yield curve spread, fed funds rate, unemployment, payrolls, CPI, industrial production, WTI crude, VIX | Daily/monthly per series | Required (`FRED_API_KEY`, free at fred.stlouisfed.org/docs/api/api_key.html) | Clean JSON, ~120 req/min | `macro_series` |

## Ranked next connectors to build

1. **USPTO PatentsView** — free, no key, innovation-velocity proxy per company/vertical → `catalyst_events` or a new `patent_filings` table.
2. **DeFiLlama** — free, effectively unmetered, TVL/yield data for the crypto-infra vertical.
3. **GDELT** — free, 15-min updates, global news/event/sentiment corpus at BigQuery/flat-file scale → `news_events` with sentiment.
4. **Capitol Trace / Finnhub congressional trades** — STOCK Act disclosures, real edge signal on insider-adjacent conviction.
5. **BLS / BEA APIs** — free, official labor/economic data complementing FRED.
6. **Twelve Data** — best free-tier real-time market data breadth (8 calls/min, 800/day) if Finnhub's coverage proves insufficient.

## Explicitly avoided / deprioritized

- **Stooq** — no real API, CAPTCHA-blocks bulk automation since 2020. Not viable for a production Go pipeline.
- **Yahoo Finance (yfinance-style)** — no official API since 2017, actively rate-limited/blocked, breaks silently. Backfill-only at best.
- **NewsAPI.org free tier** — license explicitly bars production/commercial use. Prototyping only.
- **Reddit API** — paid since 2023 for meaningful volume; budget for it if social sentiment becomes a priority signal.
- **Google Trends** — no official API, fragile scraping. Nice-to-have, not a dependency.

## Paid, institutional-grade (context only)

Polygon.io/Massive (real-time equities/options/tick), Databento (tick-level), Unusual Whales (options flow), Benzinga (real-time news wire), Bloomberg/Refinitiv terminals.

## Environment variables

| Var | Used by | Required? |
|---|---|---|
| `GITHUB_TOKEN` | github_trends connector | No (raises rate limit) |
| `FINNHUB_API_KEY` | finnhub connector | Yes, for market_bars to populate |
| `SEC_EDGAR_USER_AGENT` | secedgar connector | Yes (SEC requires a descriptive UA, e.g. `"fin-terminal you@example.com"`) |
| `FRED_API_KEY` | fred connector | Yes, for macro_series to populate |
| `QUESTDB_HTTP_ADDR` | all services | No (defaults to `http://localhost:9000`) |

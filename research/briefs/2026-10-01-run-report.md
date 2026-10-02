# Run 2026-10-01 — keyless connectors + macro backbone

## Executive summary
Built the top item from the last run's "Next best steps": a macro connector, plus two keyless connectors
(Stooq prices, Google News RSS) so `terminal pull` produces data with zero API keys. Research pass confirms
the dominant theme is still AI power/electrical infrastructure, with leadership rotating from chips toward power equipment.

## Opportunity map (research hypotheses, not recommendations)
| Vertical | Names to track | Bull | Bear | Confirm / disprove | Horizon | Confidence |
|---|---|---|---|---|---|---|
| Power & electrical for AI | VRT, GEV, ETN, NVT, CEG | Reported backlog growth (VRT cited ~$15B backlog, +109% y/y), grid equipment shortages, US DC demand ~76 GW 2026 → ~134 GW 2030 (S&P Global) | Crowded, VRT already +87% YTD per press; capex digestion risk | Confirm: backlog/book-to-bill in 10-Q/8-K, hyperscaler capex guides, utility interconnect queues. Disprove: capex cuts, order pushouts | 6–24 mo | Medium |
| Regulated utilities w/ DC load | AEP, WEC, EVRG, PNW | Rate-base growth from large-load tariffs | Rate-case/regulatory pushback, rate sensitivity | 10y yield (FRED DGS10) and large-load contract disclosures | 12–36 mo | Low-Med |
| Nuclear/IPP | CEG, VST | Contracted baseload for hyperscalers | Policy and valuation | PPA announcements via 8-K | 12+ mo | Medium |

Sources: [Yahoo Finance](https://finance.yahoo.com/news/2-artificial-intelligence-ai-stocks-080500321.html), [KuCoin blog](https://www.kucoin.com/blog/ai-power-stocks-the-hidden-infrastructure-companies-fueling-the-artificial-intelligence-boom), [iTiger](https://www.itiger.com/news/2592917834). Secondary press — verify figures against filings before relying on them.

## Built this run
- `internal/datasources/{stooq,fred,rssnews}` with unit tests; `internal/questdb/lit.go` shared literal/batch helpers.
- Migration `002_macro_and_dedup.sql`: `macro_series` table; DEDUP on bars/news/macro.
- Connectors wired into `terminal pull` and the worker; always-on index/sector-ETF set (`stooq.MarketSymbols`).

## Gaps / risks
- No live QuestDB available in this sandbox, so migration 002 (esp. DEDUP syntax) and the inserts are unit-tested only. Run `go run ./cmd/migrate` locally first.
- Stooq connection was reset from this sandbox; may be blocked or flaky. FRED and Google News responded.
- News sentiment is a keyword heuristic only.
- Hypotheses above are research leads from press articles, not investment advice.

## Next best steps
1. Run full stack locally; verify migration 002 and a real `pull`.
2. Earnings calendar connector (Finnhub `/calendar/earnings`) → `earnings_events`.
3. Wire scoring/simulation into the worker; add an `alert_events` producer.
4. Dashboard panels for macro and market overview; verify CIK map.

## Routine improvement
Pick one item from "Next best steps" per run, keep diffs small, and test for live QuestDB (java/docker exist in the sandbox but the QuestDB download was 403) before building more.

-- fin-terminal core QuestDB schema
-- Apply with: cat db/questdb/migrations/*.sql | psql questdb, or via internal/questdb.Client.Exec
-- Design notes:
--   * Every table has a designated timestamp column (`ts`) for partitioning + SAMPLE BY.
--   * PARTITION BY DAY for high-frequency tables (bars, ticks, events); MONTH for low-frequency ones.
--   * `symbol` columns use QuestDB SYMBOL type (dictionary-encoded) for fast GROUP BY / LATEST ON.
--   * WAL enabled (default in modern QuestDB) so ingestion never blocks reads.

-- ============================================================
-- Market data
-- ============================================================

CREATE TABLE IF NOT EXISTS market_bars (
    ts TIMESTAMP,                -- bar close time
    symbol SYMBOL CAPACITY 4096 CACHE,
    asset_class SYMBOL,          -- equity | etf | index | crypto | future
    interval SYMBOL,             -- 1m | 5m | 1h | 1d
    open DOUBLE,
    high DOUBLE,
    low DOUBLE,
    close DOUBLE,
    volume LONG,
    vwap DOUBLE,
    source SYMBOL                -- stooq | alphavantage | finnhub | manual ...
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS market_ticks (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    price DOUBLE,
    size LONG,
    side SYMBOL,                 -- buy | sell | unknown
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL;

-- ============================================================
-- Events (news, filings, earnings, catalysts)
-- ============================================================

CREATE TABLE IF NOT EXISTS news_events (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    headline STRING,
    url STRING,
    source SYMBOL,
    sentiment DOUBLE,             -- -1..1, NaN if not scored
    importance SYMBOL             -- low | medium | high
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS filing_events (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    cik SYMBOL,
    form_type SYMBOL,             -- 10-K | 10-Q | 8-K | 4 | 13F ...
    accession_no STRING,
    url STRING,
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS earnings_events (
    ts TIMESTAMP,                 -- scheduled report time
    symbol SYMBOL CAPACITY 4096 CACHE,
    fiscal_period SYMBOL,         -- Q1FY26 etc
    eps_estimate DOUBLE,
    eps_actual DOUBLE,
    revenue_estimate DOUBLE,
    revenue_actual DOUBLE,
    when_type SYMBOL,             -- BMO | AMC | unknown
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY MONTH WAL;

CREATE TABLE IF NOT EXISTS catalyst_events (
    ts TIMESTAMP,                 -- event/catalyst time (can be future-dated)
    symbol SYMBOL CAPACITY 4096 CACHE,
    vertical SYMBOL,              -- ai-infra | semis | energy | ...
    category SYMBOL,              -- regulatory | product | macro | legal | conference ...
    title STRING,
    description STRING,
    confidence SYMBOL,            -- confirmed | rumored | speculative
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY MONTH WAL;

-- ============================================================
-- Trend / alt-data time series
-- ============================================================

CREATE TABLE IF NOT EXISTS github_trends (
    ts TIMESTAMP,
    repo STRING,                  -- owner/name
    vertical SYMBOL,
    stars LONG,
    stars_delta_1d LONG,
    forks LONG,
    commits_30d LONG,
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS web_trends (
    ts TIMESTAMP,
    term STRING,
    vertical SYMBOL,
    score DOUBLE,                 -- normalized interest 0-100
    source SYMBOL                 -- google-trends | reddit | stocktwits | ...
) TIMESTAMP(ts) PARTITION BY DAY WAL;

-- ============================================================
-- Signals / scoring / simulation (the platform's own output, versioned over time)
-- ============================================================

CREATE TABLE IF NOT EXISTS signal_scores (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    signal_name SYMBOL,           -- momentum | volume_anomaly | sentiment | catalyst_proximity ...
    value DOUBLE,
    weight DOUBLE,
    explanation STRING,           -- human-readable why, never a black box
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS opportunity_scores (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    vertical SYMBOL,
    composite_score DOUBLE,       -- 0-100
    evidence_quality DOUBLE,      -- 0-100
    timing_score DOUBLE,          -- 0-100
    practicality_score DOUBLE,    -- 0-100
    confidence SYMBOL,            -- low | medium | high
    thesis_id STRING,             -- FK-by-convention to research/thesis/<id>.md
    explanation STRING
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS simulation_runs (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    action SYMBOL,                -- buy_now | wait | watch_confirmation | hedge | size_smaller | avoid | add_watchlist | research_deeper | set_alert | compare_alternative
    horizon SYMBOL,                -- days | weeks | months | years
    expected_return DOUBLE,
    downside_case DOUBLE,
    upside_case DOUBLE,
    probability_success DOUBLE,
    assumptions STRING,
    result_notes STRING
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS alert_events (
    ts TIMESTAMP,
    symbol SYMBOL CAPACITY 4096 CACHE,
    alert_type SYMBOL,             -- price | volume | news | filing | score_change
    severity SYMBOL,                -- info | warning | critical
    message STRING,
    triggered_value DOUBLE,
    threshold DOUBLE,
    acknowledged BOOLEAN
) TIMESTAMP(ts) PARTITION BY DAY WAL;

CREATE TABLE IF NOT EXISTS sector_strength (
    ts TIMESTAMP,
    vertical SYMBOL,
    strength_score DOUBLE,          -- -100..100 relative strength
    breadth DOUBLE,                 -- % of constituents advancing / showing positive signal
    momentum_1d DOUBLE,
    momentum_5d DOUBLE,
    momentum_1m DOUBLE,
    constituent_count INT
) TIMESTAMP(ts) PARTITION BY DAY WAL;

-- ============================================================
-- Platform self-monitoring
-- ============================================================

CREATE TABLE IF NOT EXISTS source_health (
    ts TIMESTAMP,
    source SYMBOL,                 -- stooq | sec_edgar | github_api | ...
    status SYMBOL,                 -- ok | degraded | down
    latency_ms LONG,
    rows_ingested LONG,
    error_message STRING
) TIMESTAMP(ts) PARTITION BY DAY WAL;

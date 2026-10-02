-- Macro time series (FRED etc.) plus idempotent re-ingestion via DEDUP.

CREATE TABLE IF NOT EXISTS macro_series (
    ts TIMESTAMP,
    series_id SYMBOL CAPACITY 256 CACHE,   -- DGS10 | T10Y2Y | VIXCLS ...
    value DOUBLE,
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY MONTH WAL;

ALTER TABLE macro_series DEDUP ENABLE UPSERT KEYS(ts, series_id);
ALTER TABLE market_bars DEDUP ENABLE UPSERT KEYS(ts, symbol, interval, source);
ALTER TABLE news_events DEDUP ENABLE UPSERT KEYS(ts, symbol, source);

-- Macro / rates / volatility time-series (FRED and similar).
-- DEDUP makes re-ingesting overlapping windows idempotent.
CREATE TABLE IF NOT EXISTS macro_series (
    ts TIMESTAMP,                  -- observation date
    series_id SYMBOL CAPACITY 512 CACHE,   -- FRED id, e.g. DGS10, VIXCLS
    category SYMBOL,               -- rates | volatility | credit | inflation | labor | commodities | fx
    value DOUBLE,
    source SYMBOL
) TIMESTAMP(ts) PARTITION BY MONTH WAL
DEDUP UPSERT KEYS(ts, series_id);

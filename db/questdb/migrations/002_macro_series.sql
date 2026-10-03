-- Macro time-series (FRED and similar). One row per observation date.
CREATE TABLE IF NOT EXISTS macro_series (
    ts TIMESTAMP,                  -- observation date (UTC midnight)
    series_id SYMBOL,              -- FRED id, e.g. DGS10, CPIAUCSL
    label SYMBOL,                  -- human label, e.g. "10Y Treasury yield"
    value DOUBLE,
    source SYMBOL                  -- fred | bls | ...
) TIMESTAMP(ts) PARTITION BY YEAR WAL
DEDUP UPSERT KEYS(ts, series_id);

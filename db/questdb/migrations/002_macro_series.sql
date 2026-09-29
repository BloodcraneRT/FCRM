-- Macro backbone: one row per (series, observation date). DEDUP makes
-- re-pulling an overlapping window idempotent.
CREATE TABLE IF NOT EXISTS macro_series (
    ts TIMESTAMP,                 -- observation date (UTC midnight)
    series_id SYMBOL,             -- FRED id, e.g. DGS10
    category SYMBOL,              -- rates | curve | inflation | labor | credit | fx | commodity | liquidity | vol
    title STRING,
    value DOUBLE,
    source SYMBOL                 -- fred
) TIMESTAMP(ts) PARTITION BY MONTH WAL DEDUP UPSERT KEYS(ts, series_id);

-- Macro backbone (FRED connector). Kept separate from 001 so re-running
-- migrate on an existing install only adds this table, never touches the
-- core schema already applied.

CREATE TABLE IF NOT EXISTS macro_series (
    ts TIMESTAMP,
    series_id SYMBOL,              -- FRED series id, e.g. DGS10 | UNRATE | CPIAUCSL
    vertical SYMBOL,                -- rates | labor | inflation | industrial | commodities | volatility
    value DOUBLE,
    source SYMBOL                   -- fred
) TIMESTAMP(ts) PARTITION BY MONTH WAL;

-- Idempotent re-pulls: daily bars and headlines are re-fetched every ingest
-- cycle (overlapping ranges / feeds), so upsert on natural keys instead of
-- appending duplicates. UNVERIFIED against a live QuestDB in CI-less sandbox.
ALTER TABLE market_bars DEDUP ENABLE UPSERT KEYS(ts, symbol, interval, source);
ALTER TABLE news_events DEDUP ENABLE UPSERT KEYS(ts, symbol, headline);

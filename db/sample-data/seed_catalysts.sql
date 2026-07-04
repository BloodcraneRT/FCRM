-- Optional seed data so the dashboard/CLI have something to show before
-- live connectors are configured. Apply with:
--   go run ./cmd/migrate db/sample-data
-- Sourced from the 2026-07-04 vertical research pass (see research/briefs/).
-- These are illustrative starting points, NOT investment advice — verify
-- every figure before acting on it.

INSERT INTO catalyst_events (ts, symbol, vertical, category, title, description, confidence, source) VALUES
  ('2026-07-08T13:00:00.000000Z', 'AVAV', 'defense', 'conference', 'AeroVironment Investor Day', 'FY2027 guidance following $500M Army counter-drone (Titan RF) contract win and 65% backlog growth to $1.2B.', 'confirmed', 'manual_seed');

INSERT INTO catalyst_events (ts, symbol, vertical, category, title, description, confidence, source) VALUES
  ('2026-07-15T00:00:00.000000Z', 'MP', 'commodities', 'product', 'Independence facility magnet shipments', 'MP Materials targeting H2 2026 start of magnet sales from its Independence facility amid China rare-earth export controls.', 'rumored', 'manual_seed');

INSERT INTO catalyst_events (ts, symbol, vertical, category, title, description, confidence, source) VALUES
  ('2026-07-20T00:00:00.000000Z', 'CRCL', 'fintech', 'regulatory', 'Open USD consortium go-live watch', 'Stripe/Coinbase/Visa-backed Open USD stablecoin network (140+ partners) launching on Base and Solana; margin-compression risk for Circle.', 'rumored', 'manual_seed');

INSERT INTO catalyst_events (ts, symbol, vertical, category, title, description, confidence, source) VALUES
  ('2026-08-01T00:00:00.000000Z', 'RNR', 'insurance', 'macro', 'Atlantic hurricane season peak window', 'Aug-Oct 2026 peak season; any major loss event would abruptly re-harden reinsurance pricing after 15-25% April 2026 rate declines.', 'speculative', 'manual_seed');

INSERT INTO sector_strength (ts, vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m, constituent_count) VALUES
  (now(), 'defense', 62.0, 0.7, 1.2, 4.5, 18.0, 12);

INSERT INTO sector_strength (ts, vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m, constituent_count) VALUES
  (now(), 'ai-infra-power', 55.0, 0.6, 0.4, 2.1, 11.0, 20);

INSERT INTO sector_strength (ts, vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m, constituent_count) VALUES
  (now(), 'reinsurance', -18.0, 0.3, -0.5, -1.2, -6.0, 9);

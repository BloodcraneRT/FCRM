// Package finnhub pulls quotes into market_bars. Finnhub's free tier
// (60 req/min) covers real-time US equity quotes and basic company news —
// good enough for a daily/intraday polling worker without paying for a
// tick-level feed. Requires a free API key from finnhub.io.
package finnhub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

type Connector struct {
	APIKey  string
	Symbols []string
	http    *http.Client
}

func New(apiKey string, symbols []string) *Connector {
	return &Connector{APIKey: apiKey, Symbols: symbols, http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Connector) Name() string { return "finnhub" }

type quoteResponse struct {
	Current   float64 `json:"c"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Open      float64 `json:"o"`
	PrevClose float64 `json:"pc"`
	Timestamp int64   `json:"t"`
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	if c.APIKey == "" {
		return 0, fmt.Errorf("finnhub: no API key configured (set FINNHUB_API_KEY) — get one free at finnhub.io")
	}

	rows := 0
	var firstErr error
	for _, symbol := range c.Symbols {
		q, err := c.fetchQuote(ctx, symbol)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if q.Current == 0 {
			continue // symbol not found / market closed with no data
		}

		sql := fmt.Sprintf(
			"INSERT INTO market_bars (ts, symbol, asset_class, interval, open, high, low, close, volume, source) "+
				"VALUES (now(), %s, 'equity', 'quote', %f, %f, %f, %f, 0, 'finnhub')",
			sqlLit(symbol), q.Open, q.High, q.Low, q.Current,
		)
		if err := db.Exec(ctx, sql); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rows++
	}
	return rows, firstErr
}

func (c *Connector) fetchQuote(ctx context.Context, symbol string) (*quoteResponse, error) {
	url := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", symbol, c.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub: fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub: %s returned %d", symbol, resp.StatusCode)
	}

	var q quoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("finnhub: decode %s: %w", symbol, err)
	}
	return &q, nil
}

func sqlLit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

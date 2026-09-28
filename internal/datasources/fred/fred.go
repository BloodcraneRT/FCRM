// Package fred pulls macro time series from the St. Louis Fed's FRED API
// into macro_series — the free, official macro backbone (rates, labor,
// inflation, industrial activity) that sector/opportunity scoring can key
// off of without ever guessing at macro regime. Requires a free API key
// from https://fred.stlouisfed.org/docs/api/api_key.html.
package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Series is one FRED series to poll, tagged with the vertical/context it
// feeds so downstream scoring can group macro signals meaningfully instead
// of treating every series as an undifferentiated number.
type Series struct {
	ID       string // FRED series id, e.g. "DGS10"
	Vertical string // macro grouping this series informs, e.g. "rates"
}

// CoreSeries is the default macro backbone: the small set of free FRED
// series that between them cover rates, labor, inflation, industrial
// activity, commodities, and volatility regime.
var CoreSeries = []Series{
	{ID: "DGS10", Vertical: "rates"},            // 10-year Treasury yield
	{ID: "DGS2", Vertical: "rates"},             // 2-year Treasury yield
	{ID: "T10Y2Y", Vertical: "rates"},           // 10Y-2Y yield curve spread
	{ID: "FEDFUNDS", Vertical: "rates"},         // effective federal funds rate
	{ID: "UNRATE", Vertical: "labor"},           // unemployment rate
	{ID: "PAYEMS", Vertical: "labor"},           // nonfarm payrolls
	{ID: "CPIAUCSL", Vertical: "inflation"},     // CPI, all urban consumers
	{ID: "INDPRO", Vertical: "industrial"},      // industrial production index
	{ID: "DCOILWTICO", Vertical: "commodities"}, // WTI crude spot price
	{ID: "VIXCLS", Vertical: "volatility"},      // CBOE volatility index
}

type Connector struct {
	APIKey  string
	Series  []Series
	http    *http.Client
	baseURL string // overridable for tests; defaults to the real FRED API
}

func New(apiKey string, series []Series) *Connector {
	if len(series) == 0 {
		series = CoreSeries
	}
	return &Connector{
		APIKey:  apiKey,
		Series:  series,
		http:    &http.Client{Timeout: 10 * time.Second},
		baseURL: "https://api.stlouisfed.org/fred/series/observations",
	}
}

func (c *Connector) Name() string { return "fred" }

type observationsResponse struct {
	Observations []struct {
		Date  string `json:"date"`
		Value string `json:"value"` // FRED encodes missing values as the literal string "."
	} `json:"observations"`
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	if c.APIKey == "" {
		return 0, fmt.Errorf("fred: no API key configured (set FRED_API_KEY) — get one free at fred.stlouisfed.org/docs/api/api_key.html")
	}

	rows := 0
	var firstErr error
	for _, s := range c.Series {
		value, err := c.fetchLatest(ctx, s.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if value == nil {
			continue // most recent observation not yet published (missing value)
		}

		sql := fmt.Sprintf(
			"INSERT INTO macro_series (ts, series_id, vertical, value, source) VALUES (now(), %s, %s, %f, 'fred')",
			sqlLit(s.ID), sqlLit(s.Vertical), *value,
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

// fetchLatest returns the most recent non-missing observation for a series,
// or nil if FRED has no published value yet (returned as "." in the API).
func (c *Connector) fetchLatest(ctx context.Context, seriesID string) (*float64, error) {
	url := fmt.Sprintf("%s?series_id=%s&api_key=%s&file_type=json&sort_order=desc&limit=1",
		c.baseURL, seriesID, c.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fred: fetch %s: %w", seriesID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fred: %s returned %d", seriesID, resp.StatusCode)
	}

	var out observationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("fred: decode %s: %w", seriesID, err)
	}
	if len(out.Observations) == 0 || out.Observations[0].Value == "." {
		return nil, nil
	}

	v, err := strconv.ParseFloat(out.Observations[0].Value, 64)
	if err != nil {
		return nil, fmt.Errorf("fred: parse value %q for %s: %w", out.Observations[0].Value, seriesID, err)
	}
	return &v, nil
}

func sqlLit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Package fred pulls macro series from the St. Louis Fed's FRED API into
// macro_series. Free key from fred.stlouisfed.org; ~120 req/min limit, far
// above what a handful of series needs. Rows are deduped in QuestDB on
// (ts, series_id), so overlapping pull windows are safe.
package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

const chunkSize = 100

type Series struct {
	ID       string
	Category string
	Title    string
}

// DefaultSeries is the macro regime dashboard: rates, curve, credit stress,
// inflation, labor, dollar, oil, Fed liquidity, and vol.
var DefaultSeries = []Series{
	{"DGS10", "rates", "10-Year Treasury yield"},
	{"DGS2", "rates", "2-Year Treasury yield"},
	{"T10Y2Y", "curve", "10Y-2Y Treasury spread"},
	{"FEDFUNDS", "rates", "Effective fed funds rate"},
	{"CPIAUCSL", "inflation", "CPI, all urban consumers"},
	{"UNRATE", "labor", "Unemployment rate"},
	{"BAMLH0A0HYM2", "credit", "ICE BofA US high-yield OAS"},
	{"DTWEXBGS", "fx", "Broad trade-weighted USD index"},
	{"DCOILWTICO", "commodity", "WTI crude oil, $/bbl"},
	{"WALCL", "liquidity", "Fed balance sheet total assets"},
	{"VIXCLS", "vol", "CBOE VIX close"},
}

type Connector struct {
	APIKey   string
	Series   []Series
	Lookback time.Duration
	BaseURL  string // overridable for tests
	http     *http.Client
}

func New(apiKey string, series []Series) *Connector {
	if len(series) == 0 {
		series = DefaultSeries
	}
	return &Connector{
		APIKey:   apiKey,
		Series:   series,
		Lookback: 400 * 24 * time.Hour, // enough for monthly series YoY
		BaseURL:  "https://api.stlouisfed.org/fred",
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Connector) Name() string { return "fred" }

type observation struct {
	Date  time.Time
	Value float64
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	if c.APIKey == "" {
		return 0, fmt.Errorf("fred: no API key configured (set FRED_API_KEY) — free at fred.stlouisfed.org")
	}
	rows := 0
	var firstErr error
	for _, s := range c.Series {
		obs, err := c.fetch(ctx, s.ID)
		if err == nil && len(obs) > 0 {
			// SQL rides in a GET URL, so keep each statement small.
			for start := 0; start < len(obs) && err == nil; start += chunkSize {
				end := min(start+chunkSize, len(obs))
				if err = db.Exec(ctx, insertSQL(s, obs[start:end])); err == nil {
					rows += end - start
				}
			}
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return rows, firstErr
}

func (c *Connector) fetch(ctx context.Context, id string) ([]observation, error) {
	q := url.Values{}
	q.Set("series_id", id)
	q.Set("api_key", c.APIKey)
	q.Set("file_type", "json")
	q.Set("observation_start", time.Now().Add(-c.Lookback).Format("2006-01-02"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/series/observations?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("fred: %s: build request: %w", id, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// url.Error embeds the URL (and thus the key); drop it.
		if ue, ok := err.(*url.Error); ok {
			err = ue.Err
		}
		return nil, fmt.Errorf("fred: %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fred: %s returned %d", id, resp.StatusCode)
	}
	var body struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fred: decode %s: %w", id, err)
	}
	var out []observation
	for _, o := range body.Observations {
		v, err := strconv.ParseFloat(o.Value, 64)
		if err != nil { // FRED uses "." for missing
			continue
		}
		d, err := time.Parse("2006-01-02", o.Date)
		if err != nil {
			continue
		}
		out = append(out, observation{Date: d, Value: v})
	}
	return out, nil
}

func insertSQL(s Series, obs []observation) string {
	var b strings.Builder
	b.WriteString("INSERT INTO macro_series (ts, series_id, category, title, value, source) VALUES ")
	for i, o := range obs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "('%s', %s, %s, %s, %s, 'fred')",
			o.Date.Format("2006-01-02T15:04:05.000000Z"), lit(s.ID), lit(s.Category), lit(s.Title),
			strconv.FormatFloat(o.Value, 'f', -1, 64))
	}
	return b.String()
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

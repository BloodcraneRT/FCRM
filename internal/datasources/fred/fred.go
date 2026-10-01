// Package fred ingests macro series (rates, curve, volatility, credit
// spreads, inflation, labor, oil, dollar) into macro_series. It uses FRED's
// public CSV endpoint (fredgraph.csv), which needs no API key, so the macro
// backbone works out of the box.
package fred

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Series is one FRED series to track.
type Series struct {
	ID       string
	Category string
}

// DefaultSeries is the starter macro panel: curve, policy rate, vol, credit
// stress, inflation, labor, oil, dollar.
var DefaultSeries = []Series{
	{"DGS10", "rates"}, {"DGS2", "rates"}, {"T10Y2Y", "rates"}, {"DFF", "rates"},
	{"VIXCLS", "volatility"},
	{"BAMLH0A0HYM2", "credit"}, // ICE BofA US high-yield OAS
	{"CPIAUCSL", "inflation"},
	{"UNRATE", "labor"},
	{"DCOILWTICO", "commodities"},
	{"DTWEXBGS", "fx"}, // broad trade-weighted dollar
}

type Observation struct {
	TS    time.Time
	Value float64
}

type Connector struct {
	Series   []Series
	Lookback time.Duration // how far back each run re-reads (DEDUP makes overlap safe)
	BaseURL  string
	http     *http.Client
}

func New(series []Series) *Connector {
	if len(series) == 0 {
		series = DefaultSeries
	}
	return &Connector{
		Series:   series,
		Lookback: 120 * 24 * time.Hour,
		BaseURL:  "https://fred.stlouisfed.org/graph/fredgraph.csv",
		http:     &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Connector) Name() string { return "fred" }

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	total := 0
	var firstErr error
	for _, s := range c.Series {
		obs, err := c.fetch(ctx, s.ID)
		if err == nil && len(obs) > 0 {
			err = insert(ctx, db, s, obs)
			if err == nil {
				total += len(obs)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return total, firstErr
}

func (c *Connector) fetch(ctx context.Context, id string) ([]Observation, error) {
	q := url.Values{"id": {id}, "cosd": {time.Now().Add(-c.Lookback).Format("2006-01-02")}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fred: fetch %s: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fred: %s returned %d", id, resp.StatusCode)
	}
	return ParseCSV(resp.Body)
}

// ParseCSV parses fredgraph.csv (header "observation_date,SERIES" or
// "DATE,SERIES"), skipping FRED's "." missing-value markers.
func ParseCSV(r io.Reader) ([]Observation, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("fred: parse csv: %w", err)
	}
	var out []Observation
	for i, row := range rows {
		if i == 0 || len(row) < 2 {
			continue
		}
		v := strings.TrimSpace(row[1])
		if v == "" || v == "." {
			continue
		}
		ts, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			continue
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		out = append(out, Observation{TS: ts, Value: f})
	}
	return out, nil
}

func insert(ctx context.Context, db *questdb.Client, s Series, obs []Observation) error {
	var b strings.Builder
	b.WriteString("INSERT INTO macro_series (ts, series_id, category, value, source) VALUES ")
	for i, o := range obs {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "('%s', '%s', '%s', %s, 'fred')",
			o.TS.Format("2006-01-02T15:04:05.000000Z"), s.ID, s.Category,
			strconv.FormatFloat(o.Value, 'f', -1, 64))
	}
	return db.Exec(ctx, b.String())
}

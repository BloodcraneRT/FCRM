// Package fred ingests macro series from the St. Louis Fed FRED API
// (free key, https://fred.stlouisfed.org/docs/api/fred/) into macro_series.
// Rows are upserted on (ts, series_id) so repeated pulls are idempotent.
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

// Series is one FRED series to track.
type Series struct {
	ID    string
	Label string
}

// DefaultSeries is a compact macro backbone: rates, curve, inflation,
// labor, credit stress, and the dollar.
var DefaultSeries = []Series{
	{"DGS10", "10Y Treasury yield"},
	{"DGS2", "2Y Treasury yield"},
	{"T10Y2Y", "10Y-2Y spread"},
	{"FEDFUNDS", "Fed funds rate"},
	{"CPIAUCSL", "CPI (all urban)"},
	{"UNRATE", "Unemployment rate"},
	{"BAMLH0A0HYM2", "High-yield OAS"},
	{"DTWEXBGS", "Broad dollar index"},
}

type Connector struct {
	APIKey   string
	Series   []Series
	Lookback int // observations per series per pull
	BaseURL  string
	http     *http.Client
}

func New(apiKey string, series []Series) *Connector {
	if len(series) == 0 {
		series = DefaultSeries
	}
	return &Connector{
		APIKey: apiKey, Series: series, Lookback: 30,
		BaseURL: "https://api.stlouisfed.org",
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Connector) Name() string { return "fred" }

type rawObs struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

type Observation struct {
	Date  time.Time
	Value float64
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	if c.APIKey == "" {
		return 0, fmt.Errorf("fred: FRED_API_KEY not set (free key at fred.stlouisfed.org)")
	}
	rows := 0
	var firstErr error
	for _, s := range c.Series {
		obs, err := c.fetch(ctx, s.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, o := range obs {
			if err := db.Exec(ctx, InsertSQL(s, o)); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			rows++
		}
	}
	return rows, firstErr
}

// InsertSQL builds the macro_series insert for one observation.
func InsertSQL(s Series, o Observation) string {
	return fmt.Sprintf(
		"INSERT INTO macro_series (ts, series_id, label, value, source) VALUES ('%s', %s, %s, %s, 'fred')",
		o.Date.UTC().Format("2006-01-02T15:04:05.000000Z"), lit(s.ID), lit(s.Label),
		strconv.FormatFloat(o.Value, 'f', -1, 64),
	)
}

func (c *Connector) fetch(ctx context.Context, id string) ([]Observation, error) {
	q := url.Values{}
	q.Set("series_id", id)
	q.Set("api_key", c.APIKey)
	q.Set("file_type", "json")
	q.Set("sort_order", "desc")
	q.Set("limit", strconv.Itoa(c.Lookback))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/fred/series/observations?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fred: fetch %s: %w", id, redact(err, c.APIKey))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fred: %s returned %d", id, resp.StatusCode)
	}
	var body struct {
		Observations []rawObs `json:"observations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fred: decode %s: %w", id, err)
	}
	return parse(body.Observations), nil
}

// parse converts raw FRED rows, skipping missing values (FRED uses ".").
func parse(raw []rawObs) []Observation {
	out := make([]Observation, 0, len(raw))
	for _, r := range raw {
		if r.Value == "." || r.Value == "" {
			continue
		}
		v, err := strconv.ParseFloat(r.Value, 64)
		if err != nil {
			continue
		}
		d, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			continue
		}
		out = append(out, Observation{Date: d, Value: v})
	}
	return out
}

// redact strips the API key from transport errors, which embed the URL.
func redact(err error, key string) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), key, "***"))
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Package fred ingests macro series into macro_series. It uses FRED's
// keyless fredgraph.csv endpoint, so it works with zero setup. Series cover
// the rates/credit/liquidity backbone: 10y yield, 2s10s curve, HY spreads,
// fed funds, VIX, DXY proxy, oil, and more.
package fred

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// DefaultSeries is the macro dashboard backbone.
var DefaultSeries = []string{
	"DGS10",        // 10y Treasury yield
	"T10Y2Y",       // 10y-2y curve
	"FEDFUNDS",     // fed funds rate
	"BAMLH0A0HYM2", // HY OAS credit spread
	"VIXCLS",       // VIX
	"DCOILWTICO",   // WTI crude
	"DTWEXBGS",     // broad USD index
	"WALCL",        // Fed balance sheet
}

type Connector struct {
	Series  []string
	Since   time.Time
	BaseURL string
	http    *http.Client
}

func New(series []string) *Connector {
	if len(series) == 0 {
		series = DefaultSeries
	}
	return &Connector{Series: series, Since: time.Now().AddDate(-1, 0, 0), BaseURL: "https://fred.stlouisfed.org", http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Connector) Name() string { return "fred" }

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	total := 0
	var firstErr error
	for _, id := range c.Series {
		rows, err := c.fetch(ctx, id)
		if err == nil && len(rows) > 0 {
			err = questdb.InsertBatched(func(q string) error { return db.Exec(ctx, q) },
				"INSERT INTO macro_series (ts, series_id, value, source)", rows, 100)
			if err == nil {
				total += len(rows)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("fred %s: %w", id, err)
		}
	}
	return total, firstErr
}

func (c *Connector) fetch(ctx context.Context, id string) ([]string, error) {
	url := fmt.Sprintf("%s/graph/fredgraph.csv?id=%s&cosd=%s", c.BaseURL, id, c.Since.Format("2006-01-02"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return parseCSV(resp.Body, id)
}

// parseCSV handles "observation_date,SERIES" rows; FRED marks gaps with ".".
func parseCSV(r io.Reader, id string) ([]string, error) {
	recs, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	var out []string
	for i, rec := range recs {
		if i == 0 || len(rec) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(rec[1]), 64)
		if err != nil || math.IsNaN(v) {
			continue
		}
		d, err := time.Parse("2006-01-02", rec[0])
		if err != nil {
			continue
		}
		out = append(out, fmt.Sprintf("(%s,%s,%g,'fred')", questdb.Ts(d), questdb.Lit(id), v))
	}
	return out, nil
}

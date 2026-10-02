// Package stooq pulls free, keyless daily OHLCV bars from stooq.com into
// market_bars. It's the zero-config price backbone: no API key, CSV over
// HTTP, covers US equities/ETFs (".us"), indexes ("^spx") and more.
// Re-pulls are safe because market_bars has DEDUP UPSERT KEYS (migration 002).
package stooq

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// MarketSymbols are always pulled so the dashboard has index/sector context
// even with an empty watchlist: broad indexes plus sector/theme ETFs.
var MarketSymbols = []string{"^SPX", "^NDX", "SPY", "QQQ", "IWM", "SMH", "XLE", "XLU", "ITA", "CIBR", "IBB", "GLD", "USO"}

type Connector struct {
	Symbols []string // plain tickers, e.g. NVDA, SPY; "^SPX" style for indexes
	Days    int      // lookback window; default 120
	BaseURL string   // overridable for tests
	http    *http.Client
}

func New(symbols []string) *Connector {
	return &Connector{Symbols: symbols, Days: 120, BaseURL: "https://stooq.com", http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Connector) Name() string { return "stooq" }

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	total := 0
	var firstErr error
	for _, s := range c.Symbols {
		rows, err := c.fetch(ctx, s)
		if err == nil && len(rows) > 0 {
			err = questdb.InsertBatched(func(q string) error { return db.Exec(ctx, q) },
				"INSERT INTO market_bars (ts, symbol, asset_class, interval, open, high, low, close, volume, source)", rows, 100)
			if err == nil {
				total += len(rows)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stooq %s: %w", s, err)
		}
	}
	return total, firstErr
}

func stooqSymbol(s string) string {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "^") || strings.Contains(s, ".") {
		return s
	}
	return s + ".us"
}

func assetClass(s string) string {
	if strings.HasPrefix(s, "^") {
		return "index"
	}
	return "equity"
}

func (c *Connector) fetch(ctx context.Context, symbol string) ([]string, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -c.Days)
	url := fmt.Sprintf("%s/q/d/l/?s=%s&i=d&d1=%s&d2=%s", c.BaseURL, stooqSymbol(symbol), start.Format("20060102"), end.Format("20060102"))
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
	return parseCSV(resp.Body, strings.ToUpper(symbol), assetClass(symbol))
}

// parseCSV turns stooq's "Date,Open,High,Low,Close,Volume" CSV into VALUES tuples.
func parseCSV(r io.Reader, symbol, class string) ([]string, error) {
	recs, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	var out []string
	for i, rec := range recs {
		if i == 0 {
			if len(rec) < 5 || !strings.EqualFold(rec[0], "Date") {
				return nil, fmt.Errorf("unexpected response (no data for symbol?): %q", strings.Join(rec, ","))
			}
			continue
		}
		if len(rec) < 5 {
			continue
		}
		d, err := time.Parse("2006-01-02", rec[0])
		if err != nil {
			continue
		}
		o, h, l, cl := pf(rec[1]), pf(rec[2]), pf(rec[3]), pf(rec[4])
		var vol int64
		if len(rec) > 5 {
			vol = int64(pf(rec[5]))
		}
		out = append(out, fmt.Sprintf("(%s,%s,%s,'1d',%g,%g,%g,%g,%d,'stooq')",
			questdb.Ts(d.Add(21*time.Hour)), questdb.Lit(symbol), questdb.Lit(class), o, h, l, cl, vol))
	}
	return out, nil
}

func pf(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

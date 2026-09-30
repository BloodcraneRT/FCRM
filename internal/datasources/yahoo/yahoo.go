// Package yahoo pulls daily OHLCV bars for stocks, ETFs and indexes into
// market_bars using Yahoo Finance's unofficial chart endpoint. It needs no
// API key, which makes it the zero-setup price feed; the trade-off is that
// the endpoint is undocumented and can change or rate-limit without notice.
// source_health will show it degraded/down if that happens, and Finnhub
// remains the keyed fallback.
package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Instrument is one symbol to pull. Yahoo uses ^GSPC-style index symbols;
// AssetClass is stored as-is in market_bars.asset_class.
type Instrument struct {
	Symbol     string
	AssetClass string // equity | etf | index
}

type Connector struct {
	Instruments []Instrument
	Range       string // Yahoo range, e.g. 5d, 1mo, 1y
	BaseURL     string
	http        *http.Client
}

func New(instruments []Instrument) *Connector {
	return &Connector{
		Instruments: instruments,
		Range:       "5d",
		BaseURL:     "https://query1.finance.yahoo.com",
		http:        &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Connector) Name() string { return "yahoo" }

// DefaultMarketInstruments are the index/ETF benchmarks that give the
// dashboard a market overview and sector-relative context even with an
// empty watchlist.
func DefaultMarketInstruments() []Instrument {
	idx := []string{"^GSPC", "^NDX", "^RUT", "^VIX"}
	etf := []string{"SPY", "QQQ", "IWM", "SMH", "XLE", "XLU", "XLF", "XLV", "ITA", "GLD", "USO", "TLT", "HYG"}
	var out []Instrument
	for _, s := range idx {
		out = append(out, Instrument{s, "index"})
	}
	for _, s := range etf {
		out = append(out, Instrument{s, "etf"})
	}
	return out
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// Bar is a parsed daily bar.
type Bar struct {
	TS                     time.Time
	Open, High, Low, Close float64
	Volume                 int64
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	rows := 0
	var firstErr error
	for _, in := range c.Instruments {
		bars, err := c.fetch(ctx, in.Symbol)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if len(bars) == 0 {
			continue
		}
		if err := db.Exec(ctx, insertSQL(in, bars)); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rows += len(bars)
	}
	return rows, firstErr
}

func (c *Connector) fetch(ctx context.Context, symbol string) ([]Bar, error) {
	u := fmt.Sprintf("%s/v8/finance/chart/%s?range=%s&interval=1d", c.BaseURL, url.PathEscape(symbol), url.QueryEscape(c.Range))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (fin-terminal)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo: fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo: %s returned %d", symbol, resp.StatusCode)
	}
	var cr chartResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("yahoo: decode %s: %w", symbol, err)
	}
	return parseBars(symbol, &cr)
}

func parseBars(symbol string, cr *chartResponse) ([]Bar, error) {
	if cr.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo: %s: %s", symbol, cr.Chart.Error.Description)
	}
	if len(cr.Chart.Result) == 0 || len(cr.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo: %s: empty result", symbol)
	}
	r := cr.Chart.Result[0]
	q := r.Indicators.Quote[0]
	var bars []Bar
	for i, ts := range r.Timestamp {
		// Yahoo emits null OHLC for the in-progress or holiday sessions.
		if i >= len(q.Close) || q.Close[i] == nil || i >= len(q.Open) || q.Open[i] == nil ||
			i >= len(q.High) || q.High[i] == nil || i >= len(q.Low) || q.Low[i] == nil {
			continue
		}
		var vol int64
		if i < len(q.Volume) && q.Volume[i] != nil {
			vol = int64(*q.Volume[i])
		}
		// Normalize to the UTC calendar day so re-pulls produce the same
		// (ts, symbol, interval, source) key and dedup upserts cleanly.
		day := time.Unix(ts, 0).UTC().Truncate(24 * time.Hour)
		bars = append(bars, Bar{day, *q.Open[i], *q.High[i], *q.Low[i], *q.Close[i], vol})
	}
	return bars, nil
}

func insertSQL(in Instrument, bars []Bar) string {
	var b strings.Builder
	b.WriteString("INSERT INTO market_bars (ts, symbol, asset_class, interval, open, high, low, close, volume, source) VALUES ")
	for i, bar := range bars {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "('%s', %s, %s, '1d', %g, %g, %g, %g, %d, 'yahoo')",
			bar.TS.Format("2006-01-02T15:04:05.000000Z"), lit(in.Symbol), lit(in.AssetClass),
			bar.Open, bar.High, bar.Low, bar.Close, bar.Volume)
	}
	return b.String()
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

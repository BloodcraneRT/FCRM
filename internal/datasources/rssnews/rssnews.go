// Package rssnews ingests headlines from keyless RSS feeds (Google News
// search RSS per ticker/theme, plus any extra feeds) into news_events.
// Sentiment is a transparent keyword score, not a model: see Score.
package rssnews

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

type Connector struct {
	Symbols []string
	BaseURL string // Google News RSS root; overridable for tests
	http    *http.Client
}

func New(symbols []string) *Connector {
	return &Connector{Symbols: symbols, BaseURL: "https://news.google.com", http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Connector) Name() string { return "rss_news" }

type rss struct {
	Items []struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
		Source  string `xml:"source"`
	} `xml:"channel>item"`
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	total := 0
	var firstErr error
	for _, s := range c.Symbols {
		rows, err := c.fetch(ctx, s)
		if err == nil && len(rows) > 0 {
			err = questdb.InsertBatched(func(q string) error { return db.Exec(ctx, q) },
				"INSERT INTO news_events (ts, symbol, headline, url, source, sentiment, importance)", rows, 25)
			if err == nil {
				total += len(rows)
			}
		}
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("rss %s: %w", s, err)
		}
	}
	return total, firstErr
}

func (c *Connector) fetch(ctx context.Context, symbol string) ([]string, error) {
	q := url.QueryEscape(symbol + " stock when:2d")
	u := fmt.Sprintf("%s/rss/search?q=%s&hl=en-US&gl=US&ceid=US:en", c.BaseURL, q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fin-terminal/0.1")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var doc rss
	if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	var out []string
	for _, it := range doc.Items {
		ts, err := time.Parse(time.RFC1123, it.PubDate)
		if err != nil {
			continue
		}
		src := it.Source
		if src == "" {
			src = "google_news"
		}
		out = append(out, fmt.Sprintf("(%s,%s,%s,%s,%s,%g,'low')",
			questdb.Ts(ts), questdb.Lit(strings.ToUpper(symbol)), questdb.Lit(it.Title), questdb.Lit(it.Link),
			questdb.Lit(src), Score(it.Title)))
	}
	return out, nil
}

var pos = []string{"beats", "surge", "soar", "upgrade", "record", "raises", "wins", "contract", "approval", "rally", "jumps"}
var neg = []string{"miss", "plunge", "downgrade", "probe", "lawsuit", "cuts", "recall", "fraud", "halt", "falls", "slump", "investigation"}

// Score is a deliberately simple, inspectable headline sentiment in [-1,1]:
// (positive hits - negative hits) / total hits. 0 when no keywords match.
func Score(headline string) float64 {
	h := strings.ToLower(headline)
	p, n := 0, 0
	for _, w := range pos {
		if strings.Contains(h, w) {
			p++
		}
	}
	for _, w := range neg {
		if strings.Contains(h, w) {
			n++
		}
	}
	if p+n == 0 {
		return 0
	}
	return float64(p-n) / float64(p+n)
}

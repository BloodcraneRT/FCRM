// Package rssnews ingests public RSS/Atom-free news feeds into news_events.
// Headlines are tagged to watchlist tickers when the title contains a
// cashtag ($NVDA), an exchange tag ((NASDAQ: NVDA)) or, for tickers of 3+
// letters, the bare uppercase ticker. Everything else is stored under the
// MARKET symbol. Importance uses a small explicit keyword list, not a model,
// so it is easy to inspect and tune.
package rssnews

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

type Feed struct {
	Name string // stored in news_events.source
	URL  string
}

func DefaultFeeds() []Feed {
	return []Feed{
		{"marketwatch", "https://feeds.content.dowjones.io/public/rss/mw_topstories"},
		{"marketwatch-marketpulse", "https://feeds.content.dowjones.io/public/rss/mw_marketpulse"},
	}
}

type Connector struct {
	Feeds   []Feed
	Symbols []string
	http    *http.Client
}

func New(feeds []Feed, symbols []string) *Connector {
	return &Connector{Feeds: feeds, Symbols: symbols, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Connector) Name() string { return "rss_news" }

type rss struct {
	Items []struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
	} `xml:"channel>item"`
}

// Item is a parsed headline.
type Item struct {
	TS       time.Time
	Title    string
	Link     string
	Symbol   string
	Impor    string
	FeedName string
}

func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	rows := 0
	var firstErr error
	for _, f := range c.Feeds {
		items, err := c.fetch(ctx, f)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var stmts []string
		for _, it := range items {
			for _, sym := range matchSymbols(it.Title, c.Symbols) {
				it.Symbol = sym
				stmts = append(stmts, valueTuple(it))
			}
		}
		if len(stmts) == 0 {
			continue
		}
		sql := "INSERT INTO news_events (ts, symbol, headline, url, source, sentiment, importance) VALUES " + strings.Join(stmts, ", ")
		if err := db.Exec(ctx, sql); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		rows += len(stmts)
	}
	return rows, firstErr
}

func (c *Connector) fetch(ctx context.Context, f Feed) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fin-terminal (rss reader)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rss_news: fetch %s: %w", f.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss_news: %s returned %d", f.Name, resp.StatusCode)
	}
	var doc rss
	if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("rss_news: decode %s: %w", f.Name, err)
	}
	var out []Item
	for _, it := range doc.Items {
		ts, err := time.Parse(time.RFC1123Z, strings.TrimSpace(it.PubDate))
		if err != nil {
			ts, err = time.Parse(time.RFC1123, strings.TrimSpace(it.PubDate))
		}
		if err != nil {
			continue
		}
		title := strings.TrimSpace(it.Title)
		out = append(out, Item{TS: ts.UTC(), Title: title, Link: strings.TrimSpace(it.Link), Impor: importance(title), FeedName: f.Name})
	}
	return out, nil
}

var highKeywords = []string{
	"bankruptcy", "chapter 11", "sec charges", "investigation", "subpoena", "guidance", "downgrade",
	"upgrade", "acquire", "acquisition", "merger", "halts", "recall", "fda approv", "delist",
	"short report", "fed raises", "fed cuts", "rate hike", "rate cut", "tariff", "sanction",
}

func importance(title string) string {
	l := strings.ToLower(title)
	for _, k := range highKeywords {
		if strings.Contains(l, k) {
			return "high"
		}
	}
	return "low"
}

// matchSymbols returns the watchlist tickers a headline mentions, or
// ["MARKET"] when none match.
func matchSymbols(title string, symbols []string) []string {
	var out []string
	for _, s := range symbols {
		if s == "" {
			continue
		}
		q := regexp.QuoteMeta(s)
		pats := []string{`\$` + q + `\b`, `(?:NASDAQ|NYSE|AMEX|NYSEARCA)\s*:\s*` + q + `\b`}
		if len(s) >= 3 {
			pats = append(pats, `\b`+q+`\b`)
		}
		for _, p := range pats {
			if regexp.MustCompile(p).MatchString(title) {
				out = append(out, s)
				break
			}
		}
	}
	if len(out) == 0 {
		return []string{"MARKET"}
	}
	return out
}

func valueTuple(it Item) string {
	return fmt.Sprintf("('%s', %s, %s, %s, %s, NULL, %s)",
		it.TS.Format("2006-01-02T15:04:05.000000Z"), lit(it.Symbol), lit(it.Title), lit(it.Link), lit(it.FeedName), lit(it.Impor))
}

func lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

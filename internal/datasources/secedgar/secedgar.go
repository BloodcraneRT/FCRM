// Package secedgar ingests recent filings from SEC EDGAR's free submissions
// API (data.sec.gov). No API key required — SEC only requires a descriptive
// User-Agent header identifying the caller. Rate limit: stay under 10 req/s.
package secedgar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

const submissionsURLFmt = "https://data.sec.gov/submissions/CIK%s.json"

// Watch is one company to track filings for.
type Watch struct {
	Symbol string // ticker, used only for tagging rows
	CIK    string // 10-digit zero-padded CIK, e.g. "0000320193" for Apple
}

type Connector struct {
	UserAgent string // e.g. "fin-terminal contact@example.com" — required by SEC
	Watchlist []Watch
	http      *http.Client
}

func New(userAgent string, watchlist []Watch) *Connector {
	return &Connector{
		UserAgent: userAgent,
		Watchlist: watchlist,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Connector) Name() string { return "sec_edgar" }

type submissionsResponse struct {
	Filings struct {
		Recent struct {
			AccessionNumber []string `json:"accessionNumber"`
			Form            []string `json:"form"`
			FilingDate      []string `json:"filingDate"`
			PrimaryDocument []string `json:"primaryDocument"`
		} `json:"recent"`
	} `json:"filings"`
}

// Run pulls the most recent filings for each watched CIK and writes one
// filing_events row per filing (capped at the newest 20 per company per run
// to keep this cheap — EDGAR returns full history in one payload).
func (c *Connector) Run(ctx context.Context, db *questdb.Client) (int, error) {
	if len(c.Watchlist) == 0 {
		return 0, nil
	}
	if c.UserAgent == "" {
		return 0, fmt.Errorf("sec_edgar: SEC requires a descriptive User-Agent (set SEC_EDGAR_USER_AGENT)")
	}

	total := 0
	var firstErr error
	for _, w := range c.Watchlist {
		n, err := c.fetchOne(ctx, db, w)
		total += n
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return total, firstErr
}

func (c *Connector) fetchOne(ctx context.Context, db *questdb.Client, w Watch) (int, error) {
	url := fmt.Sprintf(submissionsURLFmt, w.CIK)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("sec_edgar: fetch %s: %w", w.Symbol, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return 0, fmt.Errorf("sec_edgar: %s returned %d: %s", w.Symbol, resp.StatusCode, string(body))
	}

	var parsed submissionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, fmt.Errorf("sec_edgar: decode %s: %w", w.Symbol, err)
	}

	recent := parsed.Filings.Recent
	limit := len(recent.Form)
	if limit > 20 {
		limit = 20
	}

	// filing_events has no natural dedup key (several 8-Ks can share a day),
	// so skip accession numbers already stored for this symbol.
	seen := map[string]bool{}
	if r, err := db.Query(ctx, fmt.Sprintf("SELECT accession_no FROM filing_events WHERE symbol = %s", questdbSym(w.Symbol))); err == nil {
		for _, row := range r.Dataset {
			if len(row) > 0 {
				if a, ok := row[0].(string); ok {
					seen[a] = true
				}
			}
		}
	}

	rows := 0
	for i := 0; i < limit; i++ {
		if seen[recent.AccessionNumber[i]] {
			continue
		}
		filingDate, err := time.Parse("2006-01-02", recent.FilingDate[i])
		if err != nil {
			continue
		}
		docURL := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%s/%s/%s",
			strings.TrimLeft(w.CIK, "0"),
			strings.ReplaceAll(recent.AccessionNumber[i], "-", ""),
			recent.PrimaryDocument[i],
		)

		sql := fmt.Sprintf(
			"INSERT INTO filing_events (ts, symbol, cik, form_type, accession_no, url, source) VALUES (%s, %s, %s, %s, %s, %s, 'sec_edgar')",
			questdbTimestamp(filingDate),
			questdbSym(w.Symbol),
			questdbSym(w.CIK),
			questdbSym(recent.Form[i]),
			questdbLit(recent.AccessionNumber[i]),
			questdbLit(docURL),
		)
		if err := db.Exec(ctx, sql); err != nil {
			return rows, fmt.Errorf("sec_edgar: insert %s filing: %w", w.Symbol, err)
		}
		rows++
	}
	return rows, nil
}

func questdbTimestamp(t time.Time) string {
	return "'" + t.Format("2006-01-02T15:04:05.000000Z") + "'"
}
func questdbSym(s string) string { return questdbLit(s) }
func questdbLit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

package secedgar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const tickersURL = "https://www.sec.gov/files/company_tickers.json"

// ResolveCIKs maps tickers to zero-padded CIKs using SEC's published
// company_tickers.json, so any watchlist ticker works without hand-seeding.
// Tickers SEC does not know are omitted; callers can fall back to
// CommonCIKs.
func ResolveCIKs(ctx context.Context, userAgent string, symbols []string) (map[string]string, error) {
	return resolveFrom(ctx, tickersURL, userAgent, symbols)
}

func resolveFrom(ctx context.Context, endpoint, userAgent string, symbols []string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("sec_edgar: fetch ticker map: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sec_edgar: ticker map returned %d", resp.StatusCode)
	}
	var raw map[string]struct {
		CIK    int64  `json:"cik_str"`
		Ticker string `json:"ticker"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("sec_edgar: decode ticker map: %w", err)
	}
	byTicker := make(map[string]string, len(raw))
	for _, v := range raw {
		byTicker[strings.ToUpper(v.Ticker)] = fmt.Sprintf("%010d", v.CIK)
	}
	out := make(map[string]string)
	for _, s := range symbols {
		if cik, ok := byTicker[strings.ToUpper(s)]; ok {
			out[s] = cik
		}
	}
	return out, nil
}

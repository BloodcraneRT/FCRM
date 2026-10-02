package rssnews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScore(t *testing.T) {
	if Score("NVDA beats estimates, raises guidance") != 1 || Score("Firm faces probe, shares plunge") != -1 || Score("Quarterly update") != 0 {
		t.Fatal("score wrong")
	}
}

func TestFetchParsesRSS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><item><title>It's a record quarter</title><link>http://x/1</link><pubDate>Wed, 30 Sep 2026 14:00:00 GMT</pubDate><source>Reuters</source></item></channel></rss>`))
	}))
	defer srv.Close()
	c := New(nil)
	c.BaseURL = srv.URL
	rows, err := c.fetch(context.Background(), "nvda")
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if !strings.Contains(rows[0], "'It''s a record quarter'") || !strings.Contains(rows[0], "'NVDA'") {
		t.Errorf("bad row %s", rows[0])
	}
}

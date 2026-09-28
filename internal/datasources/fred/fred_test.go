package fred

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func TestRun_WritesOneRowPerSeriesWithNonMissingValue(t *testing.T) {
	fredSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seriesID := r.URL.Query().Get("series_id")
		if seriesID == "MISSING" {
			fmt.Fprint(w, `{"observations":[{"date":"2026-09-26","value":"."}]}`)
			return
		}
		fmt.Fprint(w, `{"observations":[{"date":"2026-09-26","value":"4.12"}]}`)
	}))
	defer fredSrv.Close()

	var insertedSQL []string
	questSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		insertedSQL = append(insertedSQL, r.URL.Query().Get("query"))
		fmt.Fprint(w, `{"query":"","dataset":[],"count":0}`)
	}))
	defer questSrv.Close()

	c := New("test-key", []Series{
		{ID: "DGS10", Vertical: "rates"},
		{ID: "MISSING", Vertical: "labor"},
	})
	c.baseURL = fredSrv.URL

	db := questdb.NewClient(questSrv.URL)
	rows, err := c.Run(context.Background(), db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != 1 {
		t.Fatalf("got %d rows, want 1 (the MISSING series has no published value)", rows)
	}
	if len(insertedSQL) != 1 || !strings.Contains(insertedSQL[0], "'DGS10'") || !strings.Contains(insertedSQL[0], "'rates'") {
		t.Fatalf("unexpected inserted SQL: %+v", insertedSQL)
	}
}

func TestRun_RequiresAPIKey(t *testing.T) {
	c := New("", CoreSeries)
	db := questdb.NewClient("http://127.0.0.1:1")
	if _, err := c.Run(context.Background(), db); err == nil {
		t.Fatal("expected error when no API key is configured")
	}
}

func TestFetchLatest_SurfacesNon200AsError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New("test-key", nil)
	c.baseURL = srv.URL

	if _, err := c.fetchLatest(context.Background(), "DGS10"); err == nil {
		t.Fatal("expected error on non-200 response")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 request, got %d", calls)
	}
}

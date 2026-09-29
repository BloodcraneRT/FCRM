package fred

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func TestRunParsesSkipsMissingAndInserts(t *testing.T) {
	fredSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("series_id") != "DGS10" || r.URL.Query().Get("api_key") != "k" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		io.WriteString(w, `{"observations":[{"date":"2026-09-24","value":"4.10"},{"date":"2026-09-25","value":"."},{"date":"2026-09-26","value":"4.15"}]}`)
	}))
	defer fredSrv.Close()

	var got []string
	qdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Query().Get("query"))
		io.WriteString(w, `{"ddl":"OK"}`)
	}))
	defer qdb.Close()

	c := New("k", []Series{{"DGS10", "rates", "10-Year Treasury yield"}})
	c.BaseURL = fredSrv.URL
	n, err := c.Run(context.Background(), questdb.NewClient(qdb.URL))
	if err != nil || n != 2 {
		t.Fatalf("rows=%d err=%v", n, err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "'2026-09-24T00:00:00.000000Z', 'DGS10'") ||
		!strings.Contains(got[0], "4.15") || strings.Contains(got[0], "2026-09-25") {
		t.Fatalf("bad SQL: %v", got)
	}
}

func TestMissingKey(t *testing.T) {
	if _, err := New("", nil).Run(context.Background(), nil); err == nil {
		t.Fatal("expected error without key")
	}
}

func TestErrorDoesNotLeakKey(t *testing.T) {
	c := New("secret-key", []Series{{"X", "c", "t"}})
	c.BaseURL = "http://127.0.0.1:1"
	_, err := c.Run(context.Background(), nil)
	if err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("err=%v", err)
	}
}

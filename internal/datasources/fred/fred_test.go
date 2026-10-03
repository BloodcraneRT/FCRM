package fred

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func TestParseSkipsMissingValues(t *testing.T) {
	got := parse([]rawObs{{"2026-09-01", "4.25"}, {"2026-09-02", "."}, {"bad", "1"}})
	if len(got) != 1 || got[0].Value != 4.25 {
		t.Fatalf("unexpected parse result: %+v", got)
	}
}

func TestInsertSQL(t *testing.T) {
	o := parse([]rawObs{{"2026-09-01", "4.25"}})[0]
	sql := InsertSQL(Series{"DGS10", "10Y Treasury yield"}, o)
	want := "VALUES ('2026-09-01T00:00:00.000000Z', 'DGS10', '10Y Treasury yield', 4.25, 'fred')"
	if !strings.Contains(sql, want) {
		t.Fatalf("sql=%s", sql)
	}
}

func TestRunMissingKey(t *testing.T) {
	if _, err := New("", nil).Run(context.Background(), questdb.NewClient("http://127.0.0.1:1")); err == nil {
		t.Fatal("expected error without key")
	}
}

func TestRunIngestsObservations(t *testing.T) {
	fredSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"observations":[{"date":"2026-09-01","value":"4.2"},{"date":"2026-09-02","value":"."}]}`)
	}))
	defer fredSrv.Close()

	var mu sync.Mutex
	var stmts []string
	qdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		stmts = append(stmts, r.URL.Query().Get("query"))
		mu.Unlock()
		io.WriteString(w, `{"ddl":"OK"}`)
	}))
	defer qdb.Close()

	c := New("k", []Series{{"DGS10", "10Y"}})
	c.BaseURL = fredSrv.URL
	n, err := c.Run(context.Background(), questdb.NewClient(qdb.URL))
	if err != nil || n != 1 || len(stmts) != 1 {
		t.Fatalf("n=%d err=%v stmts=%v", n, err, stmts)
	}
}

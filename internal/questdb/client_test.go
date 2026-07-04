package questdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuery_ParsesResultAndSurfacesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		if strings.Contains(q, "FAIL") {
			w.Write([]byte(`{"query":"","error":"table does not exist"}`))
			return
		}
		w.Write([]byte(`{"query":"select 1","columns":[{"name":"x","type":"INT"}],"dataset":[[1]],"count":1}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)

	res, err := c.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Dataset) != 1 || res.Dataset[0][0].(float64) != 1 {
		t.Fatalf("unexpected dataset: %+v", res.Dataset)
	}

	_, err = c.Query(context.Background(), "SELECT * FROM FAIL")
	if err == nil {
		t.Fatal("expected error for questdb-reported failure, got nil")
	}
}

func TestPing_FailsWhenUnreachable(t *testing.T) {
	c := NewClient("http://127.0.0.1:1") // nothing listens here
	if err := c.Ping(context.Background()); err == nil {
		t.Fatal("expected Ping to fail against an unreachable address")
	}
}

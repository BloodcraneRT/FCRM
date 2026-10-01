package secedgar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveFrom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user agent")
		}
		w.Write([]byte(`{"0":{"cik_str":1368622,"ticker":"AVAV","title":"AEROVIRONMENT INC"}}`))
	}))
	defer srv.Close()
	got, err := resolveFrom(context.Background(), srv.URL, "test ua", []string{"avav", "ZZZZ"})
	if err != nil {
		t.Fatal(err)
	}
	if got["avav"] != "0001368622" || len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

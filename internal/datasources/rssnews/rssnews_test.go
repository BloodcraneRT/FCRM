package rssnews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func TestMatchSymbols(t *testing.T) {
	syms := []string{"NVDA", "MP", "BE"}
	cases := map[string][]string{
		"NVDA jumps on guidance":                  {"NVDA"},
		"Shares of $MP surge":                     {"MP"},
		"Bloom Energy (NYSE: BE) wins deal":       {"BE"},
		"Stocks to BE careful about, MP3 players": {"MARKET"},
		"Fed holds rates":                         {"MARKET"},
	}
	for title, want := range cases {
		if got := matchSymbols(title, syms); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v want %v", title, got, want)
		}
	}
}

func TestImportance(t *testing.T) {
	if importance("Company files for Chapter 11") != "high" || importance("Nice day") != "low" {
		t.Fatal("importance mismatch")
	}
}

const feed = `<?xml version="1.0"?><rss version="2.0"><channel>
<item><title>NVDA raises guidance &amp; O'Neil talks</title><link>http://x/1</link><pubDate>Mon, 28 Sep 2026 14:00:00 +0000</pubDate></item>
<item><title>Bad date</title><link>http://x/2</link><pubDate>garbage</pubDate></item>
</channel></rss>`

func TestRunInsertsEscaped(t *testing.T) {
	rssSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(feed)) }))
	defer rssSrv.Close()
	var got string
	qdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("query")
		w.Write([]byte(`{}`))
	}))
	defer qdb.Close()

	c := New([]Feed{{"test", rssSrv.URL}}, []string{"NVDA"})
	n, err := c.Run(context.Background(), questdb.NewClient(qdb.URL))
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if !strings.Contains(got, "O''Neil") || !strings.Contains(got, "'NVDA'") || !strings.Contains(got, "'high'") {
		t.Fatalf("unexpected sql: %s", got)
	}
}

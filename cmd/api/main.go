// Command api runs the fin-terminal HTTP API that backs the web dashboard.
// See docs/architecture.md for the OpenAPI spec location and route list.
package main

import (
	"log"
	"net/http"

	"github.com/BloodcraneRT/fin-terminal/internal/api"
	"github.com/BloodcraneRT/fin-terminal/internal/config"
	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/watchlists"
)

func main() {
	cfg := config.Load()

	db := questdb.NewClient(cfg.QuestDBHTTPAddr)
	wl, err := watchlists.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("api: cannot open watchlist store: %v", err)
	}
	defer wl.Close()

	server := api.New(db, wl)

	root := http.NewServeMux()
	root.Handle("/healthz", server)
	root.Handle("/api/", server)
	root.Handle("/", http.FileServer(http.Dir("web/dashboard")))

	log.Printf("fin-terminal API + dashboard listening on %s (questdb=%s)", cfg.APIAddr, cfg.QuestDBHTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.APIAddr, root))
}

// Package api exposes fin-terminal's data over HTTP/JSON so the web
// dashboard (or any other client) doesn't need to speak QuestDB SQL or
// touch the SQLite metadata store directly. Deliberately plain net/http —
// the route count doesn't justify a framework yet.
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/watchlists"
)

type Server struct {
	db  *questdb.Client
	wl  *watchlists.Store
	mux *http.ServeMux
}

func New(db *questdb.Client, wl *watchlists.Store) *Server {
	s := &Server{db: db, wl: wl, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/watchlist", s.handleWatchlist)
	s.mux.HandleFunc("GET /api/v1/opportunities", s.handleOpportunities)
	s.mux.HandleFunc("GET /api/v1/sector-strength", s.handleSectorStrength)
	s.mux.HandleFunc("GET /api/v1/catalysts", s.handleCatalysts)
	s.mux.HandleFunc("GET /api/v1/source-health", s.handleSourceHealth)
	s.mux.HandleFunc("GET /api/v1/macro", s.handleMacro)
}

// handleMacro returns the latest observation per macro series.
func (s *Server) handleMacro(w http.ResponseWriter, r *http.Request) {
	s.queryToJSON(w, r.Context(), `
		SELECT ts, series_id, category, title, value
		FROM macro_series
		LATEST ON ts PARTITION BY series_id
		ORDER BY category, series_id`)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWatchlist(w http.ResponseWriter, r *http.Request) {
	list := r.URL.Query().Get("list")
	items, err := s.wl.List(r.Context(), list)
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []watchlists.Item{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleOpportunities(w http.ResponseWriter, r *http.Request) {
	s.queryToJSON(w, r.Context(), `
		SELECT symbol, vertical, composite_score, evidence_quality, timing_score, practicality_score, confidence, explanation, ts
		FROM opportunity_scores
		LATEST ON ts PARTITION BY symbol
		ORDER BY composite_score DESC`)
}

func (s *Server) handleSectorStrength(w http.ResponseWriter, r *http.Request) {
	s.queryToJSON(w, r.Context(), `
		SELECT vertical, strength_score, breadth, momentum_1d, momentum_5d, momentum_1m, ts
		FROM sector_strength
		LATEST ON ts PARTITION BY vertical
		ORDER BY strength_score DESC`)
}

func (s *Server) handleCatalysts(w http.ResponseWriter, r *http.Request) {
	s.queryToJSON(w, r.Context(), `
		SELECT ts, symbol, vertical, category, title, description, confidence
		FROM catalyst_events
		WHERE ts > dateadd('d', -1, now())
		ORDER BY ts ASC
		LIMIT 200`)
}

func (s *Server) handleSourceHealth(w http.ResponseWriter, r *http.Request) {
	s.queryToJSON(w, r.Context(), `
		SELECT source, status, latency_ms, rows_ingested, error_message, ts
		FROM source_health
		LATEST ON ts PARTITION BY source
		ORDER BY ts DESC`)
}

func (s *Server) queryToJSON(w http.ResponseWriter, ctx context.Context, sql string) {
	result, err := s.db.Query(ctx, sql)
	if err != nil {
		writeError(w, err)
		return
	}
	rows := make([]map[string]interface{}, 0, len(result.Dataset))
	for _, row := range result.Dataset {
		m := make(map[string]interface{}, len(result.Columns))
		for i, col := range result.Columns {
			if i < len(row) {
				m[col.Name] = row[i]
			}
		}
		rows = append(rows, m)
	}
	writeJSON(w, http.StatusOK, rows)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

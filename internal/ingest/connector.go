// Package ingest defines the connector contract every data source
// implements, plus the runner that executes them and records source_health
// so the platform always knows which feeds are alive.
package ingest

import (
	"context"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// Connector is the contract for any data source connector. Adding a new
// data source means implementing this interface and registering it —
// nothing else in the platform needs to change.
type Connector interface {
	Name() string
	// Run fetches fresh data and writes it into QuestDB. It returns the
	// number of rows ingested for observability/health tracking.
	Run(ctx context.Context, db *questdb.Client) (rowsIngested int, err error)
}

// RunAll executes every connector, recording a source_health row for each
// so failures are visible in the dashboard instead of silently swallowed.
func RunAll(ctx context.Context, db *questdb.Client, connectors []Connector) []Result {
	results := make([]Result, 0, len(connectors))
	for _, c := range connectors {
		start := time.Now()
		rows, err := c.Run(ctx, db)
		latency := time.Since(start)

		res := Result{Source: c.Name(), Rows: rows, Latency: latency, Err: err}
		results = append(results, res)
		recordHealth(ctx, db, res)
	}
	return results
}

type Result struct {
	Source  string
	Rows    int
	Latency time.Duration
	Err     error
}

func recordHealth(ctx context.Context, db *questdb.Client, r Result) {
	status := "ok"
	errMsg := ""
	if r.Err != nil {
		status = "down"
		errMsg = r.Err.Error()
	} else if r.Rows == 0 {
		status = "degraded"
	}

	sql := "INSERT INTO source_health (ts, source, status, latency_ms, rows_ingested, error_message) VALUES (now(), $1, $2, $3, $4, $5)"
	_ = sql // QuestDB /exec doesn't support parameter binding over HTTP GET; build inline instead.
	stmt := questdbInsertHealth(r.Source, status, r.Latency.Milliseconds(), int64(r.Rows), errMsg)
	_ = db.Exec(ctx, stmt)
}

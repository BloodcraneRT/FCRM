package ingest

import (
	"fmt"
	"strings"
)

// QuestDB's HTTP /exec endpoint takes a raw SQL string (no prepared-statement
// binding), so every connector needs a small, consistent way to build safe
// literals. These helpers are intentionally dumb: escape single quotes and
// wrap in quotes. Fine for our own connector-generated values; never build
// these from raw end-user input.

func sqlStr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func questdbInsertHealth(source, status string, latencyMs, rows int64, errMsg string) string {
	return fmt.Sprintf(
		"INSERT INTO source_health (ts, source, status, latency_ms, rows_ingested, error_message) VALUES (now(), %s, %s, %d, %d, %s)",
		sqlStr(source), sqlStr(status), latencyMs, rows, sqlStr(errMsg),
	)
}

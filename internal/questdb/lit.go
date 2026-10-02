package questdb

import (
	"strings"
	"time"
)

// Lit renders s as a single-quoted SQL string literal. QuestDB's /exec
// endpoint has no parameter binding, so connectors use this for values they
// generate themselves. Never feed it raw end-user input.
func Lit(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// Ts renders t as a QuestDB timestamp literal (UTC, microsecond precision).
func Ts(t time.Time) string { return "'" + t.UTC().Format("2006-01-02T15:04:05.000000Z") + "'" }

// InsertBatched runs "INSERT INTO <head> VALUES <rows...>" in chunks so
// large backfills don't build an oversized URL (/exec uses GET).
func InsertBatched(exec func(sql string) error, head string, rows []string, chunk int) error {
	if chunk <= 0 {
		chunk = 50
	}
	for i := 0; i < len(rows); i += chunk {
		end := i + chunk
		if end > len(rows) {
			end = len(rows)
		}
		if err := exec(head + " VALUES " + strings.Join(rows[i:end], ",")); err != nil {
			return err
		}
	}
	return nil
}

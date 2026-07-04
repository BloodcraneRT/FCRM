// Command migrate applies every .sql file in db/questdb/migrations against
// QuestDB, in filename order. QuestDB's /exec endpoint runs one statement
// per call, so this splits each file on ";" — fine as long as migration
// files never embed a literal semicolon inside a string value.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BloodcraneRT/fin-terminal/internal/config"
	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

func main() {
	dir := "db/questdb/migrations"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	cfg := config.Load()
	db := questdb.NewClient(cfg.QuestDBHTTPAddr)

	waitForQuestDB(db, 60*time.Second)

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		log.Fatalf("migrate: glob %s: %v", dir, err)
	}
	sort.Strings(files)

	ctx := context.Background()
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("migrate: read %s: %v", f, err)
		}
		applied := 0
		for _, stmt := range splitStatements(string(content)) {
			if err := db.Exec(ctx, stmt); err != nil {
				log.Fatalf("migrate: %s: statement %d failed: %v\n--- statement ---\n%s", f, applied+1, err, stmt)
			}
			applied++
		}
		fmt.Printf("applied %s (%d statements)\n", f, applied)
	}
	fmt.Println("migrations complete")
}

func splitStatements(sql string) []string {
	var out []string
	for _, raw := range strings.Split(sql, ";") {
		stmt := strings.TrimSpace(stripComments(raw))
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}

func stripComments(block string) string {
	lines := strings.Split(block, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func waitForQuestDB(db *questdb.Client, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		if err := db.Ping(context.Background()); err == nil {
			return
		}
		if time.Now().After(deadline) {
			log.Fatalf("migrate: QuestDB not reachable after %s", timeout)
		}
		time.Sleep(2 * time.Second)
	}
}

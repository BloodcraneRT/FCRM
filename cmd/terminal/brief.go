package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func (a *app) briefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brief",
		Short: "Export a daily brief markdown file (watchlist + opportunities + catalysts)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var b strings.Builder
			now := time.Now().UTC()
			fmt.Fprintf(&b, "# Daily Brief — %s\n\n", now.Format("2006-01-02"))

			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			items, err := store.List(ctx, "default")
			if err != nil {
				return err
			}
			fmt.Fprintf(&b, "## Watchlist (%d)\n\n", len(items))
			for _, it := range items {
				fmt.Fprintf(&b, "- **%s** (%s) — %s\n", it.Symbol, it.Vertical, it.Note)
			}

			fmt.Fprintf(&b, "\n## Opportunity Ranking\n\n")
			if r, err := a.db.Query(ctx, `SELECT symbol, vertical, composite_score, confidence FROM opportunity_scores LATEST ON ts PARTITION BY symbol ORDER BY composite_score DESC`); err == nil {
				for _, row := range r.Dataset {
					fmt.Fprintf(&b, "- %v (%v): score=%v confidence=%v\n", row[0], row[1], row[2], row[3])
				}
			} else {
				fmt.Fprintf(&b, "_(no opportunity scores yet — run `terminal pull` and scoring first)_\n")
			}

			fmt.Fprintf(&b, "\n## Upcoming Catalysts\n\n")
			if r, err := a.db.Query(ctx, `SELECT ts, symbol, title FROM catalyst_events WHERE ts > dateadd('d', -1, now()) ORDER BY ts ASC LIMIT 20`); err == nil {
				for _, row := range r.Dataset {
					fmt.Fprintf(&b, "- %v — %v: %v\n", row[0], row[1], row[2])
				}
			} else {
				fmt.Fprintf(&b, "_(no catalysts recorded yet)_\n")
			}

			path := filepath.Join("research", "briefs", fmt.Sprintf("%s.md", now.Format("2006-01-02")))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
				return err
			}
			fmt.Println("wrote", path)
			return nil
		},
	}
}

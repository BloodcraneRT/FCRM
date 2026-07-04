package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func (a *app) statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check QuestDB connectivity and recent source health",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := a.db.Ping(ctx); err != nil {
				fmt.Println("QuestDB: UNREACHABLE —", err)
			} else {
				fmt.Println("QuestDB: OK (", a.cfg.QuestDBHTTPAddr, ")")
			}

			r, err := a.db.Query(ctx, `SELECT source, status, latency_ms, rows_ingested, ts FROM source_health LATEST ON ts PARTITION BY source ORDER BY ts DESC`)
			if err != nil {
				fmt.Println("(source_health not queryable yet — run migrations first)")
				return nil
			}
			printResult(r)
			return nil
		},
	}
}

func (a *app) dashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Print the dashboard URL (run `docker compose up` first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Dashboard + API: http://localhost:8081  (run `go run ./cmd/api`)")
			fmt.Println("QuestDB UI:      " + a.cfg.QuestDBHTTPAddr)
			return nil
		},
	}
}

// Command terminal is the fin-terminal CLI: the primary way to add tickers,
// pull data, check sector strength and opportunity rankings, write thesis
// notes, and run simulations without touching QuestDB SQL directly.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/BloodcraneRT/fin-terminal/internal/config"
	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
	"github.com/BloodcraneRT/fin-terminal/internal/watchlists"
)

type app struct {
	cfg config.Config
	db  *questdb.Client
	wl  *watchlists.Store
}

func main() {
	cfg := config.Load()
	a := &app{
		cfg: cfg,
		db:  questdb.NewClient(cfg.QuestDBHTTPAddr),
	}

	root := &cobra.Command{
		Use:   "terminal",
		Short: "fin-terminal — a self-hosted financial intelligence terminal",
	}

	root.AddCommand(
		a.watchlistCmd(),
		a.pullCmd(),
		a.sectorStrengthCmd(),
		a.opportunitiesCmd(),
		a.alertsCmd(),
		a.catalystsCmd(),
		a.thesisCmd(),
		a.simulateCmd(),
		a.briefCmd(),
		a.dashboardCmd(),
		a.statusCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func (a *app) openWatchlist() (*watchlists.Store, error) {
	if a.wl != nil {
		return a.wl, nil
	}
	store, err := watchlists.Open(a.cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	a.wl = store
	return store, nil
}

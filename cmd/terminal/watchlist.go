package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/BloodcraneRT/fin-terminal/internal/watchlists"
)

func (a *app) watchlistCmd() *cobra.Command {
	var listName, vertical, note string

	cmd := &cobra.Command{
		Use:   "watchlist",
		Short: "Manage and view watchlists",
	}

	add := &cobra.Command{
		Use:   "add <symbol>",
		Short: "Add a ticker to a watchlist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			err = store.Add(context.Background(), watchlists.Item{
				Symbol:   args[0],
				ListName: listName,
				Vertical: vertical,
				Note:     note,
			})
			if err == nil {
				fmt.Printf("added %s to watchlist %q\n", args[0], defaultList(listName))
			}
			return err
		},
	}
	add.Flags().StringVar(&listName, "list", "default", "watchlist name")
	add.Flags().StringVar(&vertical, "vertical", "", "sector/vertical tag")
	add.Flags().StringVar(&note, "note", "", "why this ticker matters")

	remove := &cobra.Command{
		Use:   "remove <symbol>",
		Short: "Remove a ticker from a watchlist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			if err := store.Remove(context.Background(), args[0], listName); err != nil {
				return err
			}
			fmt.Printf("removed %s from watchlist %q\n", args[0], defaultList(listName))
			return nil
		},
	}
	remove.Flags().StringVar(&listName, "list", "default", "watchlist name")

	show := &cobra.Command{
		Use:   "show",
		Short: "Show a watchlist",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			items, err := store.List(context.Background(), listName)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("(empty)")
				return nil
			}
			fmt.Printf("%-10s %-16s %-8s %s\n", "SYMBOL", "VERTICAL", "ADDED", "NOTE")
			for _, it := range items {
				fmt.Printf("%-10s %-16s %-8s %s\n", it.Symbol, it.Vertical, it.AddedAt.Format("2006-01-02"), it.Note)
			}
			return nil
		},
	}
	show.Flags().StringVar(&listName, "list", "default", "watchlist name")

	cmd.AddCommand(add, remove, show)
	return cmd
}

func defaultList(l string) string {
	if l == "" {
		return "default"
	}
	return l
}

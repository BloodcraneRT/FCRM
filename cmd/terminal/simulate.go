package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/BloodcraneRT/fin-terminal/internal/simulation"
)

func (a *app) simulateCmd() *cobra.Command {
	var action, horizon, assumptions string
	var prob, upside, downside float64

	cmd := &cobra.Command{
		Use:   "simulate <symbol>",
		Short: "Simulate expected value of an action (buy_now, wait, hedge, avoid, ...)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			in := simulation.Input{
				Symbol:             args[0],
				Action:             simulation.Action(action),
				Horizon:            simulation.Horizon(horizon),
				ProbabilitySuccess: prob,
				UpsideCase:         upside,
				DownsideCase:       downside,
				Assumptions:        assumptions,
			}
			result := simulation.Run(in)
			fmt.Println(result.Notes)

			if err := simulation.Persist(context.Background(), a.db, result); err != nil {
				fmt.Println("warning: could not persist to QuestDB:", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&action, "action", string(simulation.ActionResearchDeeper), "buy_now|wait|watch_confirmation|hedge|size_smaller|avoid|add_watchlist|research_deeper|set_alert|compare_alternative")
	cmd.Flags().StringVar(&horizon, "horizon", string(simulation.HorizonWeeks), "days|weeks|months|years")
	cmd.Flags().Float64Var(&prob, "prob", 0.5, "probability thesis plays out (0-1)")
	cmd.Flags().Float64Var(&upside, "upside", 0, "% return if thesis succeeds")
	cmd.Flags().Float64Var(&downside, "downside", 0, "% return if thesis fails (negative)")
	cmd.Flags().StringVar(&assumptions, "assumptions", "", "what this estimate assumes")
	return cmd
}

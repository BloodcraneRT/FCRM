package main

import (
	"fmt"

	"github.com/BloodcraneRT/fin-terminal/internal/questdb"
)

// printResult renders a QuestDB QueryResult as a simple aligned table.
// Deliberately not fancy — this is a terminal tool, not a TUI framework.
func printResult(r *questdb.QueryResult) {
	if len(r.Dataset) == 0 {
		fmt.Println("(no rows)")
		return
	}
	for _, col := range r.Columns {
		fmt.Printf("%-20s", col.Name)
	}
	fmt.Println()
	for _, row := range r.Dataset {
		for _, v := range row {
			fmt.Printf("%-20v", v)
		}
		fmt.Println()
	}
}

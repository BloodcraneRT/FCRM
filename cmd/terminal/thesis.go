package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const thesisTemplate = `# %s — %s

- **Created:** %s
- **Symbol:** %s
- **Vertical:**
- **Time horizon:**
- **Confidence:**

## Bull case


## Bear case


## What the market may be missing


## Confirming data


## Disconfirming data


## Catalysts


## Risks


## Next research step


## Platform feature needed to track this

`

func (a *app) thesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "thesis",
		Short: "Manage research thesis notes",
	}

	newCmd := &cobra.Command{
		Use:   "new <symbol> <title>",
		Short: "Create a new thesis note under research/thesis/",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := args[0]
			title := strings.Join(args[1:], " ")
			now := time.Now().UTC()
			thesisID := fmt.Sprintf("%s-%s", symbol, now.Format("20060102-150405"))
			fileName := fmt.Sprintf("%s.md", thesisID)
			path := filepath.Join("research", "thesis", fileName)

			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			content := fmt.Sprintf(thesisTemplate, symbol, title, now.Format("2006-01-02"), symbol)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}

			store, err := a.openWatchlist()
			if err != nil {
				return err
			}
			if err := store.IndexThesis(thesisID, symbol, title, path); err != nil {
				return err
			}

			fmt.Printf("created %s (thesis_id=%s)\n", path, thesisID)
			return nil
		},
	}

	cmd.AddCommand(newCmd)
	return cmd
}

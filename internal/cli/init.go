package cli

import (
	"fmt"

	"github.com/ledger-ai/ledger/internal/db"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the ledger database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if force {
				if err := db.InitFresh(database); err != nil {
					return fmt.Errorf("init fresh: %w", err)
				}
				if jsonOut {
					outputJSON(map[string]interface{}{"initialized": true, "fresh": true})
				} else {
					outputText("Database reinitialized (fresh) at %s\n", dbPath)
				}
			} else {
				if err := db.Init(database); err != nil {
					return fmt.Errorf("init: %w", err)
				}
				if jsonOut {
					outputJSON(map[string]interface{}{"initialized": true, "fresh": false})
				} else {
					outputText("Database initialized at %s\n", dbPath)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "recreate database (destructive)")
	return cmd
}

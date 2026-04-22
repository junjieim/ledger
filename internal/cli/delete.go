package cli

import (
	"fmt"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if err := repo.DeleteTransaction(database, id); err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"deleted": true, "id": id})
			} else {
				outputText("Deleted transaction %s\n", id)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "transaction ID (required)")
	cmd.MarkFlagRequired("id")
	return cmd
}

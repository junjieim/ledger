package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newRefundCmd() *cobra.Command {
	var (
		id     string
		amount float64
		note   string
	)

	cmd := &cobra.Command{
		Use:   "refund",
		Short: "Record a refund against an existing expense",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("--id required")
			}
			if amount < 0 {
				return fmt.Errorf("--amount must be non-negative; omit to refund remaining")
			}

			t, err := repo.Refund(database, id, amount, note)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(t)
				return nil
			}

			outputText("Refund total %.2f %s on transaction %s\n", t.RefundAmount, t.Currency, t.ID)
			outputText("Net amount: %.2f %s\n", t.NetAmount, t.Currency)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "transaction id to refund")
	cmd.Flags().Float64Var(&amount, "amount", 0, "refund amount; omit or use 0 to refund remaining")
	cmd.Flags().StringVar(&note, "note", "", "optional refund note appended to existing note")

	return cmd
}

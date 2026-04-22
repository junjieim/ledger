package cli

import (
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newBalanceCmd() *cobra.Command {
	var currency string

	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Show balance per currency",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := repo.GetBalance(database, strings.ToUpper(currency))
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(result)
			} else {
				if len(result.Balances) == 0 {
					outputText("No transactions found.\n")
					return nil
				}
				outputText("%-5s  %12s\n", "CUR", "BALANCE")
				outputText("%s\n", strings.Repeat("-", 20))
				for _, b := range result.Balances {
					outputText("%-5s  %12.2f\n", b.Currency, b.Balance)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&currency, "currency", "", "filter to one currency")
	return cmd
}

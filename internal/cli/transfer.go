package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newTransferCmd() *cobra.Command {
	var (
		fromCurrency string
		toCurrency   string
		fromAmount   float64
		toAmount     float64
		date         string
		note         string
	)

	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create a linked currency transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromAmount <= 0 || toAmount <= 0 {
				return fmt.Errorf("--from-amount and --to-amount must be positive")
			}
			if strings.EqualFold(fromCurrency, toCurrency) {
				return fmt.Errorf("--from-currency and --to-currency must be different")
			}

			input := repo.TransferInput{
				FromCurrency: strings.ToUpper(strings.TrimSpace(fromCurrency)),
				ToCurrency:   strings.ToUpper(strings.TrimSpace(toCurrency)),
				FromAmount:   fromAmount,
				ToAmount:     toAmount,
				OccurredAt:   date,
			}
			if note != "" {
				input.Note = &note
			}

			result, err := repo.CreateTransfer(database, input)
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(result)
			} else {
				outputText("Created transfer %s\n", result.TransferGroup)
				outputText("  expense: %s %.2f (%s)\n", result.Expense.Currency, result.Expense.Amount, result.Expense.ID)
				outputText("  income:  %s %.2f (%s)\n", result.Income.Currency, result.Income.Amount, result.Income.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&fromCurrency, "from-currency", "", "source currency")
	cmd.Flags().StringVar(&toCurrency, "to-currency", "", "target currency")
	cmd.Flags().Float64Var(&fromAmount, "from-amount", 0, "source amount")
	cmd.Flags().Float64Var(&toAmount, "to-amount", 0, "target amount")
	cmd.Flags().StringVar(&date, "date", "", "transfer date (ISO8601)")
	cmd.Flags().StringVar(&note, "note", "", "transfer note")

	cmd.MarkFlagRequired("from-currency")
	cmd.MarkFlagRequired("to-currency")
	cmd.MarkFlagRequired("from-amount")
	cmd.MarkFlagRequired("to-amount")

	return cmd
}

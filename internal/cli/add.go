package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var (
		amount      float64
		direction   string
		currency    string
		category    string
		date        string
		description string
		rawInput    string
		tags        []string
		note        string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if amount <= 0 {
				return fmt.Errorf("--amount must be positive")
			}
			dir := strings.ToLower(direction)
			if dir != "income" && dir != "expense" {
				return fmt.Errorf("--direction must be 'income' or 'expense'")
			}

			in := repo.AddTransactionInput{
				Direction:  dir,
				Amount:     amount,
				Currency:   strings.ToUpper(currency),
				Tags:       tags,
				OccurredAt: date,
			}

			if category != "" {
				catID, err := repo.ResolveCategoryID(database, category)
				if err != nil {
					return fmt.Errorf("resolve category: %w", err)
				}
				if catID == nil {
					return fmt.Errorf("category %q not found", category)
				}
				in.CategoryID = catID
			}
			if description != "" {
				in.Description = &description
			}
			if rawInput != "" {
				in.RawInput = &rawInput
			}
			if note != "" {
				in.Note = &note
			}

			t, err := repo.AddTransaction(database, in)
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(t)
			} else {
				outputText("Added %s: %.2f %s", t.Direction, t.Amount, t.Currency)
				if t.Category != "" {
					outputText(" [%s]", t.Category)
				}
				outputText(" on %s (id: %s)\n", t.OccurredAt, t.ID)
			}
			return nil
		},
	}

	cmd.Flags().Float64Var(&amount, "amount", 0, "amount (required)")
	cmd.Flags().StringVar(&direction, "direction", "", "income or expense (required)")
	cmd.Flags().StringVar(&currency, "currency", "CNY", "ISO 4217 currency code")
	cmd.Flags().StringVar(&category, "category", "", "category name")
	cmd.Flags().StringVar(&date, "date", "", "transaction date (ISO8601, default today)")
	cmd.Flags().StringVar(&description, "description", "", "structured description")
	cmd.Flags().StringVar(&rawInput, "raw-input", "", "original natural language input")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tags (repeatable)")
	cmd.Flags().StringVar(&note, "note", "", "additional note")

	cmd.MarkFlagRequired("amount")
	cmd.MarkFlagRequired("direction")

	return cmd
}

package cli

import (
	"fmt"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		id          string
		amount      float64
		direction   string
		currency    string
		category    string
		date        string
		description string
		note        string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a transaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			in := repo.UpdateTransactionInput{ID: id}
			changed := false

			if cmd.Flags().Changed("amount") {
				in.Amount = &amount
				changed = true
			}
			if cmd.Flags().Changed("direction") {
				in.Direction = &direction
				changed = true
			}
			if cmd.Flags().Changed("currency") {
				in.Currency = &currency
				changed = true
			}
			if cmd.Flags().Changed("category") {
				catID, err := repo.ResolveCategoryID(database, category)
				if err != nil {
					return fmt.Errorf("resolve category: %w", err)
				}
				if catID == nil {
					return fmt.Errorf("category %q not found", category)
				}
				in.CategoryID = catID
				changed = true
			}
			if cmd.Flags().Changed("date") {
				in.Date = &date
				changed = true
			}
			if cmd.Flags().Changed("description") {
				in.Description = &description
				changed = true
			}
			if cmd.Flags().Changed("note") {
				in.Note = &note
				changed = true
			}

			if !changed {
				return fmt.Errorf("at least one field must be specified to update")
			}

			t, err := repo.UpdateTransaction(database, in)
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(t)
			} else {
				outputText("Updated transaction %s\n", t.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "transaction ID (required)")
	cmd.Flags().Float64Var(&amount, "amount", 0, "new amount")
	cmd.Flags().StringVar(&direction, "direction", "", "new direction")
	cmd.Flags().StringVar(&currency, "currency", "", "new currency")
	cmd.Flags().StringVar(&category, "category", "", "new category name")
	cmd.Flags().StringVar(&date, "date", "", "new date")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	cmd.Flags().StringVar(&note, "note", "", "new note")

	cmd.MarkFlagRequired("id")
	return cmd
}

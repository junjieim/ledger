package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var (
		from      string
		to        string
		month     string
		direction string
		category  string
		tag       string
		currency  string
		minAmount float64
		maxAmount float64
		limit     int
		offset    int
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --month shorthand
			if month != "" {
				t, err := time.Parse("2006-01", month)
				if err != nil {
					return fmt.Errorf("invalid --month format, use YYYY-MM: %w", err)
				}
				from = t.Format("2006-01-02")
				to = t.AddDate(0, 1, -1).Format("2006-01-02")
			}

			in := repo.QueryInput{
				From:      from,
				To:        to,
				Direction: strings.ToLower(direction),
				Category:  category,
				Tag:       tag,
				Currency:  strings.ToUpper(currency),
				Limit:     limit,
				Offset:    offset,
			}
			if cmd.Flags().Changed("min-amount") {
				in.MinAmount = &minAmount
			}
			if cmd.Flags().Changed("max-amount") {
				in.MaxAmount = &maxAmount
			}

			result, err := repo.QueryTransactions(database, in)
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(result)
			} else {
				outputText("Total: %d\n", result.Total)
				outputText("%-36s  %-8s  %10s  %10s  %-5s  %-8s  %-10s  %s\n",
					"ID", "DIR", "AMOUNT", "NET", "CUR", "CATEGORY", "DATE", "DESCRIPTION")
				outputText("%s\n", strings.Repeat("-", 112))
				for _, t := range result.Items {
					desc := ""
					if t.Description != nil {
						desc = *t.Description
					}
					cat := t.Category
					net := ""
					if t.RefundAmount > 0 {
						net = fmt.Sprintf("%.2f", t.NetAmount)
					}
					outputText("%-36s  %-8s  %10.2f  %10s  %-5s  %-8s  %-10s  %s\n",
						t.ID, t.Direction, t.Amount, net, t.Currency, cat, t.OccurredAt, desc)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "start date")
	cmd.Flags().StringVar(&to, "to", "", "end date")
	cmd.Flags().StringVar(&month, "month", "", "month shorthand (YYYY-MM)")
	cmd.Flags().StringVar(&direction, "direction", "", "income or expense")
	cmd.Flags().StringVar(&category, "category", "", "category name")
	cmd.Flags().StringVar(&tag, "tag", "", "tag filter")
	cmd.Flags().StringVar(&currency, "currency", "", "currency filter")
	cmd.Flags().Float64Var(&minAmount, "min-amount", 0, "minimum amount")
	cmd.Flags().Float64Var(&maxAmount, "max-amount", 0, "maximum amount")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")

	return cmd
}

package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/search"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		keyword string
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search transactions by keyword",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(keyword) == "" {
				return fmt.Errorf("--keyword required")
			}

			result, err := search.Transactions(database, search.Input{
				Keyword: keyword,
				Limit:   limit,
			})
			if err != nil {
				return err
			}

			return outputSearchResult(result)
		},
	}

	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword query")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (0 = unlimited)")

	return cmd
}

func outputSearchResult(result *model.SearchResult) error {
	if jsonOut {
		outputJSON(result)
		return nil
	}

	outputText("%-36s  %7s  %10s  %-5s  %-8s  %-10s  %s\n",
		"ID", "SCORE", "AMOUNT", "CUR", "CATEGORY", "DATE", "DESCRIPTION")
	outputText("%s\n", strings.Repeat("-", 110))
	for _, item := range result.Items {
		outputText("%-36s  %7.4f  %10.2f  %-5s  %-8s  %-10s  %s\n",
			item.ID, item.Score, item.Amount, item.Currency, item.Category, item.OccurredAt, item.Description)
	}
	return nil
}

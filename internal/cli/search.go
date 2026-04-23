package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/embedding"
	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/ledger-ai/ledger/internal/search"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		keyword  string
		semantic string
		mode     string
		limit    int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search transactions by keyword and/or semantics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(keyword) == "" && strings.TrimSpace(semantic) == "" {
				return fmt.Errorf("at least one of --keyword or --semantic is required")
			}

			var embedder *embedding.Client
			effectiveMode := strings.ToLower(strings.TrimSpace(mode))
			if effectiveMode == "" {
				effectiveMode = "hybrid"
			}
			if (effectiveMode == "semantic" || (effectiveMode == "hybrid" && strings.TrimSpace(semantic) != "")) && strings.TrimSpace(semantic) != "" {
				settings, err := repo.EffectiveEmbeddingSettings(database)
				if err != nil {
					return err
				}
				if strings.TrimSpace(settings.APIKey) == "" {
					if effectiveMode == "hybrid" && strings.TrimSpace(keyword) != "" {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: embedding is not configured, so hybrid search is returning keyword results only.\n")
						semantic = ""
						effectiveMode = "keyword"
					} else {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: embedding is not configured, so semantic search returned no vector results.\n")
						return outputSearchResult(&model.SearchResult{Items: []model.SearchItem{}})
					}
				}
			}

			if (effectiveMode == "semantic" || (effectiveMode == "hybrid" && strings.TrimSpace(semantic) != "")) && strings.TrimSpace(semantic) != "" {
				settings, err := repo.EffectiveEmbeddingSettings(database)
				if err != nil {
					return err
				}
				client, err := embedding.NewClient(settings)
				if err != nil {
					return err
				}
				embedder = client
			}

			result, err := search.Transactions(database, search.Input{
				Keyword:  keyword,
				Semantic: semantic,
				Mode:     effectiveMode,
				Limit:    limit,
			}, embedder)
			if err != nil {
				return err
			}

			return outputSearchResult(result)
		},
	}

	cmd.Flags().StringVar(&keyword, "keyword", "", "keyword query")
	cmd.Flags().StringVar(&semantic, "semantic", "", "semantic query")
	cmd.Flags().StringVar(&mode, "mode", "hybrid", "keyword, semantic, or hybrid")
	cmd.Flags().IntVar(&limit, "limit", 10, "max results")

	return cmd
}

func outputSearchResult(result *model.SearchResult) error {
	if jsonOut {
		outputJSON(result)
		return nil
	}

	outputText("%-36s  %7s  %-8s  %10s  %-5s  %-8s  %-10s  %s\n",
		"ID", "SCORE", "MATCH", "AMOUNT", "CUR", "CATEGORY", "DATE", "DESCRIPTION")
	outputText("%s\n", strings.Repeat("-", 120))
	for _, item := range result.Items {
		outputText("%-36s  %7.4f  %-8s  %10.2f  %-5s  %-8s  %-10s  %s\n",
			item.ID, item.Score, item.MatchType, item.Amount, item.Currency, item.Category, item.OccurredAt, item.Description)
	}
	return nil
}

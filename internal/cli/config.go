package cli

import (
	"strings"

	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	var (
		apiKey     string
		modelName  string
		modelURL   string
		dimensions int
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or update embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				cfg *model.EmbeddingConfig
				err error
			)

			changed := false
			input := repo.SaveEmbeddingConfigInput{}
			if cmd.Flags().Changed("api-key") {
				changed = true
				input.APIKey = &apiKey
			}
			if cmd.Flags().Changed("model-name") {
				changed = true
				input.ModelName = &modelName
			}
			if cmd.Flags().Changed("model-url") {
				changed = true
				input.ModelURL = &modelURL
			}
			if cmd.Flags().Changed("dimensions") {
				changed = true
				input.Dimensions = &dimensions
			}

			if changed {
				cfg, err = repo.SaveEmbeddingConfig(database, input)
			} else {
				cfg, err = repo.GetEmbeddingConfig(database)
			}
			if err != nil {
				return err
			}

			masked := *cfg
			masked.APIKey = maskAPIKey(masked.APIKey)
			if jsonOut {
				outputJSON(masked)
				return nil
			}

			outputText("Embedding config\n")
			outputText("  API key: %s\n", masked.APIKey)
			outputText("  Model: %s\n", masked.ModelName)
			outputText("  URL: %s\n", masked.ModelURL)
			outputText("  Dimensions: %d\n", masked.Dimensions)
			outputText("  Updated at: %s\n", masked.UpdatedAt)
			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "embedding API key")
	cmd.Flags().StringVar(&modelName, "model-name", "", "embedding model name")
	cmd.Flags().StringVar(&modelURL, "model-url", "", "embedding model URL")
	cmd.Flags().IntVar(&dimensions, "dimensions", 0, "embedding dimensions")

	return cmd
}

func maskAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(not set)"
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

package cli

import (
	"fmt"

	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newConfigSetCmd(),
		newConfigUpdateCmd(),
		newConfigShowCmd(),
	)

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	var (
		apiKey     string
		modelName  string
		modelURL   string
		dimensions int
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := repo.SaveEmbeddingConfig(database, repo.SaveEmbeddingConfigInput{
				APIKey:     &apiKey,
				ModelName:  &modelName,
				ModelURL:   &modelURL,
				Dimensions: &dimensions,
			})
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(publicEmbeddingConfig(cfg))
				return nil
			}

			outputText("Embedding config updated\n")
			outputText("  Model: %s\n", cfg.ModelName)
			outputText("  URL: %s\n", cfg.ModelURL)
			outputText("  Dimensions: %d\n", cfg.Dimensions)
			outputText("  Updated at: %s\n", cfg.UpdatedAt)
			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "embedding API key")
	cmd.Flags().StringVar(&modelName, "model-name", "", "embedding model name")
	cmd.Flags().StringVar(&modelURL, "model-url", "", "embedding model URL")
	cmd.Flags().IntVar(&dimensions, "dimensions", 0, "embedding dimensions")
	must(cmd.MarkFlagRequired("api-key"))
	must(cmd.MarkFlagRequired("model-name"))
	must(cmd.MarkFlagRequired("model-url"))
	must(cmd.MarkFlagRequired("dimensions"))

	return cmd
}

func newConfigUpdateCmd() *cobra.Command {
	var (
		apiKey     string
		modelName  string
		modelURL   string
		dimensions int
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			input := repo.SaveEmbeddingConfigInput{}
			changed := false
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
			if !changed {
				return fmt.Errorf("at least one config field must be provided")
			}

			cfg, err := repo.SaveEmbeddingConfig(database, input)
			if err != nil {
				return err
			}

			if jsonOut {
				outputJSON(publicEmbeddingConfig(cfg))
				return nil
			}

			outputText("Embedding config updated\n")
			outputText("  Model: %s\n", cfg.ModelName)
			outputText("  URL: %s\n", cfg.ModelURL)
			outputText("  Dimensions: %d\n", cfg.Dimensions)
			outputText("  Updated at: %s\n", cfg.UpdatedAt)
			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "embedding API key")
	cmd.Flags().StringVar(&modelName, "model-name", "", "embedding model name")
	cmd.Flags().StringVar(&modelURL, "model-url", "", "embedding model URL")
	cmd.Flags().IntVar(&dimensions, "dimensions", 0, "embedding dimensions")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show non-secret embedding configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := repo.GetEmbeddingConfig(database)
			if err != nil {
				return err
			}

			public := publicEmbeddingConfig(cfg)
			if jsonOut {
				outputJSON(public)
				return nil
			}

			outputText("Embedding config\n")
			outputText("  Model: %s\n", public["model_name"])
			outputText("  URL: %s\n", public["model_url"])
			outputText("  Dimensions: %d\n", public["dimensions"])
			outputText("  Updated at: %s\n", public["updated_at"])
			return nil
		},
	}

	return cmd
}

func publicEmbeddingConfig(cfg *model.EmbeddingConfig) map[string]interface{} {
	return map[string]interface{}{
		"model_name": cfg.ModelName,
		"model_url":  cfg.ModelURL,
		"dimensions": cfg.Dimensions,
		"updated_at": cfg.UpdatedAt,
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

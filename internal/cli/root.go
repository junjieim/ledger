package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ledger-ai/ledger/internal/db"
	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

var (
	dbPath   string
	jsonOut  bool
	database *sql.DB
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ledger",
		Short: "AI-agent friendly ledger CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			database, err = db.Open(dbPath)
			if err != nil {
				return err
			}
			warnIfEmbeddingConfigMissing(cmd)
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if database != nil {
				database.Close()
				database = nil
			}
		},
	}

	root.PersistentFlags().StringVar(&dbPath, "db", "./data/ledger.db", "database path")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "output JSON format")

	root.AddCommand(
		newInitCmd(),
		newConfigCmd(),
		newAddCmd(),
		newDeleteCmd(),
		newUpdateCmd(),
		newQueryCmd(),
		newSearchCmd(),
		newBalanceCmd(),
		newTransferCmd(),
		newCategoryCmd(),
		newTagCmd(),
		newAuditCmd(),
	)

	return root
}

func warnIfEmbeddingConfigMissing(cmd *cobra.Command) {
	if isConfigCommand(cmd) {
		return
	}
	if database == nil {
		return
	}

	settings, err := repo.EffectiveEmbeddingSettings(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to load embedding config: %v\n", err)
		return
	}
	missing := missingEmbeddingConfigFields(repoEmbeddingSettings{
		APIKey:     settings.APIKey,
		ModelName:  settings.ModelName,
		ModelURL:   settings.ModelURL,
		Dimensions: settings.Dimensions,
	})
	if len(missing) == 0 {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"Warning: embedding configuration is incomplete (missing: %s). Run ledger config set to complete embedding setup.\n",
		strings.Join(missing, ", "),
	)
}

func isConfigCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if cmd.Name() == "config" {
		return true
	}
	if cmd.Parent() != nil && cmd.Parent().Name() == "config" {
		return true
	}
	return false
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func outputText(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format, a...)
}

func missingEmbeddingConfigFields(settings repoEmbeddingSettings) []string {
	var missing []string
	if strings.TrimSpace(settings.APIKey) == "" {
		missing = append(missing, "api_key")
	}
	if strings.TrimSpace(settings.ModelName) == "" {
		missing = append(missing, "model_name")
	}
	if strings.TrimSpace(settings.ModelURL) == "" {
		missing = append(missing, "model_url")
	}
	if settings.Dimensions <= 0 {
		missing = append(missing, "dimensions")
	}
	return missing
}

type repoEmbeddingSettings struct {
	APIKey     string
	ModelName  string
	ModelURL   string
	Dimensions int
}

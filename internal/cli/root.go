package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ledger-ai/ledger/internal/db"
	"github.com/ledger-ai/ledger/internal/embedding"
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
			warnIfEmbeddingAPIKeyMissing(cmd)
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

func warnIfEmbeddingAPIKeyMissing(cmd *cobra.Command) {
	if cmd != nil && cmd.Name() == "config" && cmd.Flags().Changed("api-key") {
		return
	}
	if database == nil {
		if strings.TrimSpace(os.Getenv(embedding.ZhipuAPIKeyEnv)) != "" {
			return
		}
		fmt.Fprintf(os.Stderr, "Warning: embedding API key is not configured. Run ledger config --api-key ... or set %s.\n", embedding.ZhipuAPIKeyEnv)
		return
	}

	settings, err := repo.EffectiveEmbeddingSettings(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to load embedding config: %v\n", err)
		return
	}
	if strings.TrimSpace(settings.APIKey) != "" {
		return
	}

	if cmd != nil && cmd.Name() == "config" {
		fmt.Fprintf(os.Stderr, "Warning: embedding API key is not configured yet. Use this command to save it.\n")
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: embedding API key is not configured. Run ledger config --api-key ... or set %s.\n", embedding.ZhipuAPIKeyEnv)
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func outputText(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format, a...)
}

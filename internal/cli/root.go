package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ledger-ai/ledger/internal/db"
	"github.com/ledger-ai/ledger/internal/embedding"
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
			warnIfZhipuAPIKeyMissing()

			// Skip DB open for init --force (it handles its own)
			if cmd.Name() == "init" {
				return nil
			}
			var err error
			database, err = db.Open(dbPath)
			if err != nil {
				return err
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if database != nil {
				database.Close()
			}
		},
	}

	root.PersistentFlags().StringVar(&dbPath, "db", "./data/ledger.db", "database path")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "output JSON format")

	root.AddCommand(
		newInitCmd(),
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

func warnIfZhipuAPIKeyMissing() {
	if strings.TrimSpace(os.Getenv(embedding.ZhipuAPIKeyEnv)) != "" {
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: %s is not set. Semantic search and embedding sync will not work until it is configured.\n", embedding.ZhipuAPIKeyEnv)
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func outputText(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format, a...)
}

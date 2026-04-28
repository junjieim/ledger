package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ledger-ai/ledger/internal/db"
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

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func outputText(format string, a ...interface{}) {
	fmt.Fprintf(os.Stdout, format, a...)
}

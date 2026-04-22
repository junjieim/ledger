package cli

import (
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	var (
		action string
		from   string
		to     string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				limit = 20
			}
			entries, err := repo.QueryAuditLog(database, action, from, to, limit)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"items": entries})
				return nil
			}
			outputText("%-36s  %-20s  %-12s  %-36s  %s\n", "ID", "ACTION", "TARGET", "TARGET_ID", "CREATED_AT")
			outputText("%s\n", strings.Repeat("-", 130))
			for _, entry := range entries {
				outputText("%-36s  %-20s  %-12s  %-36s  %s\n", entry.ID, entry.Action, entry.TargetType, entry.TargetID, entry.CreatedAt)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&action, "action", "", "filter by action")
	cmd.Flags().StringVar(&from, "from", "", "start time/date filter")
	cmd.Flags().StringVar(&to, "to", "", "end time/date filter")
	cmd.Flags().IntVar(&limit, "limit", 20, "max entries")
	return cmd
}

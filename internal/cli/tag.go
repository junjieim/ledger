package cli

import (
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags",
	}

	cmd.AddCommand(
		newTagListCmd(),
		newTagAddCmd(),
		newTagRemoveCmd(),
	)
	return cmd
}

func newTagListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := repo.ListTags(database)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"items": tags})
				return nil
			}
			outputText("%-36s  %s\n", "ID", "NAME")
			outputText("%s\n", strings.Repeat("-", 60))
			for _, tag := range tags {
				outputText("%-36s  %s\n", tag.ID, tag.Name)
			}
			return nil
		},
	}
}

func newTagAddCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, err := repo.AddTag(database, name)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(tag)
			} else {
				outputText("Added tag %s (%s)\n", tag.Name, tag.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "tag name")
	cmd.MarkFlagRequired("name")
	return cmd
}

func newTagRemoveCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := repo.RemoveTag(database, name); err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"removed": true, "name": name})
			} else {
				outputText("Removed tag %s\n", name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "tag name")
	cmd.MarkFlagRequired("name")
	return cmd
}

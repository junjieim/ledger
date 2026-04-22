package cli

import (
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/repo"
	"github.com/spf13/cobra"
)

func newCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "category",
		Short: "Manage categories",
	}

	cmd.AddCommand(
		newCategoryListCmd(),
		newCategoryAddCmd(),
		newCategoryRemoveCmd(),
	)
	return cmd
}

func newCategoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			categories, err := repo.ListCategories(database)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"items": categories})
				return nil
			}
			outputText("%-36s  %-16s  %-10s  %-36s  %s\n", "ID", "NAME", "DIRECTION", "PARENT_ID", "ICON")
			outputText("%s\n", strings.Repeat("-", 120))
			for _, category := range categories {
				parentID := ""
				icon := ""
				if category.ParentID != nil {
					parentID = *category.ParentID
				}
				if category.Icon != nil {
					icon = *category.Icon
				}
				outputText("%-36s  %-16s  %-10s  %-36s  %s\n", category.ID, category.Name, category.Direction, parentID, icon)
			}
			return nil
		},
	}
}

func newCategoryAddCmd() *cobra.Command {
	var (
		name      string
		direction string
		parent    string
		icon      string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a category",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := strings.ToLower(strings.TrimSpace(direction))
			if dir != "income" && dir != "expense" && dir != "both" {
				return fmt.Errorf("--direction must be income, expense, or both")
			}

			input := repo.AddCategoryInput{
				Name:      name,
				Direction: dir,
			}
			if parent != "" {
				parentID, err := repo.ResolveCategoryID(database, parent)
				if err != nil {
					return err
				}
				if parentID == nil {
					return fmt.Errorf("parent category %q not found", parent)
				}
				input.ParentID = parentID
			}
			if icon != "" {
				input.Icon = &icon
			}

			category, err := repo.AddCategory(database, input)
			if err != nil {
				return err
			}
			if jsonOut {
				outputJSON(category)
			} else {
				outputText("Added category %s (%s)\n", category.Name, category.ID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "category name")
	cmd.Flags().StringVar(&direction, "direction", "", "income, expense, or both")
	cmd.Flags().StringVar(&parent, "parent", "", "parent category name")
	cmd.Flags().StringVar(&icon, "icon", "", "optional icon")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("direction")
	return cmd
}

func newCategoryRemoveCmd() *cobra.Command {
	var (
		name  string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a category",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := repo.RemoveCategory(database, name, force); err != nil {
				return err
			}
			if jsonOut {
				outputJSON(map[string]interface{}{"removed": true, "name": name, "force": force})
			} else {
				outputText("Removed category %s\n", name)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "category name")
	cmd.Flags().BoolVar(&force, "force", false, "detach references and remove anyway")
	cmd.MarkFlagRequired("name")
	return cmd
}

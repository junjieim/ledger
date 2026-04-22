package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ledger-ai/ledger/internal/model"
)

type AddCategoryInput struct {
	Name      string
	Direction string
	ParentID  *string
	Icon      *string
}

func ListCategories(db *sql.DB) ([]model.Category, error) {
	rows, err := db.Query(`
		SELECT id, name, parent_id, direction, icon
		FROM categories
		ORDER BY COALESCE(parent_id, ''), name
	`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.ParentID, &category.Direction, &category.Icon); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func AddCategory(db *sql.DB, in AddCategoryInput) (*model.Category, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("category name is required")
	}

	if existing, err := ResolveCategoryID(db, name); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("category %q already exists", name)
	}

	id := uuid.New().String()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO categories (id, name, parent_id, direction, icon)
		 VALUES (?, ?, ?, ?, ?)`,
		id, name, in.ParentID, in.Direction, in.Icon,
	); err != nil {
		return nil, fmt.Errorf("insert category: %w", err)
	}

	if err := logAudit(tx, "add_category", "category", id, in); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetCategoryByID(db, id)
}

func GetCategoryByID(db *sql.DB, id string) (*model.Category, error) {
	var category model.Category
	err := db.QueryRow(
		`SELECT id, name, parent_id, direction, icon
		 FROM categories
		 WHERE id = ?`,
		id,
	).Scan(&category.ID, &category.Name, &category.ParentID, &category.Direction, &category.Icon)
	if err != nil {
		return nil, err
	}
	return &category, nil
}

func RemoveCategory(db *sql.DB, name string, force bool) error {
	categoryID, err := ResolveCategoryID(db, strings.TrimSpace(name))
	if err != nil {
		return err
	}
	if categoryID == nil {
		return fmt.Errorf("category %q not found", name)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var usageCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transactions WHERE category_id = ?`, *categoryID).Scan(&usageCount); err != nil {
		return fmt.Errorf("check category usage: %w", err)
	}
	if usageCount > 0 && !force {
		return fmt.Errorf("category %q is referenced by %d transactions; use --force to detach and remove it", name, usageCount)
	}

	var childCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM categories WHERE parent_id = ?`, *categoryID).Scan(&childCount); err != nil {
		return fmt.Errorf("check child categories: %w", err)
	}
	if childCount > 0 && !force {
		return fmt.Errorf("category %q has %d child categories; use --force to detach and remove it", name, childCount)
	}

	if force {
		if _, err := tx.Exec(`UPDATE transactions SET category_id = NULL WHERE category_id = ?`, *categoryID); err != nil {
			return fmt.Errorf("detach category from transactions: %w", err)
		}
		if _, err := tx.Exec(`UPDATE categories SET parent_id = NULL WHERE parent_id = ?`, *categoryID); err != nil {
			return fmt.Errorf("detach child categories: %w", err)
		}
	}

	res, err := tx.Exec(`DELETE FROM categories WHERE id = ?`, *categoryID)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return fmt.Errorf("category %q not found", name)
	}

	if err := logAudit(tx, "remove_category", "category", *categoryID, map[string]interface{}{
		"name":  name,
		"force": force,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

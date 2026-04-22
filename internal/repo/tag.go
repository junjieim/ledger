package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ledger-ai/ledger/internal/model"
)

func ListTags(db *sql.DB) ([]model.Tag, error) {
	rows, err := db.Query(`SELECT id, name FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func AddTag(db *sql.DB, name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("tag name is required")
	}

	if existing, err := getTagByName(db, name); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("tag %q already exists", name)
	}

	id := uuid.New().String()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO tags (id, name) VALUES (?, ?)`, id, name); err != nil {
		return nil, fmt.Errorf("insert tag: %w", err)
	}
	if err := logAudit(tx, "add_tag", "tag", id, map[string]string{"name": name}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &model.Tag{ID: id, Name: name}, nil
}

func RemoveTag(db *sql.DB, name string) error {
	tag, err := getTagByName(db, strings.TrimSpace(name))
	if err != nil {
		return err
	}
	if tag == nil {
		return fmt.Errorf("tag %q not found", name)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, tag.ID); err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if err := logAudit(tx, "remove_tag", "tag", tag.ID, map[string]string{"name": tag.Name}); err != nil {
		return err
	}
	return tx.Commit()
}

func getTagByName(db *sql.DB, name string) (*model.Tag, error) {
	var tag model.Tag
	err := db.QueryRow(`SELECT id, name FROM tags WHERE name = ?`, name).Scan(&tag.ID, &tag.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup tag %q: %w", name, err)
	}
	return &tag, nil
}

package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ledger-ai/ledger/internal/embedding"
	"github.com/ledger-ai/ledger/internal/model"
)

func EnsureEmbeddingConfigTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS embedding_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			api_key TEXT,
			model_name TEXT NOT NULL,
			model_url TEXT NOT NULL,
			dimensions INTEGER NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure embedding_config table: %w", err)
	}

	defaults := embedding.DefaultSettings()
	_, err = db.Exec(`
		INSERT OR IGNORE INTO embedding_config (id, api_key, model_name, model_url, dimensions)
		VALUES (1, '', ?, ?, ?)
	`, defaults.ModelName, defaults.ModelURL, defaults.Dimensions)
	if err != nil {
		return fmt.Errorf("seed embedding_config table: %w", err)
	}

	return nil
}

func GetEmbeddingConfig(db *sql.DB) (*model.EmbeddingConfig, error) {
	if err := EnsureEmbeddingConfigTable(db); err != nil {
		return nil, err
	}

	cfg := &model.EmbeddingConfig{}
	err := db.QueryRow(`
		SELECT COALESCE(api_key, ''), model_name, model_url, dimensions, updated_at
		FROM embedding_config
		WHERE id = 1
	`).Scan(&cfg.APIKey, &cfg.ModelName, &cfg.ModelURL, &cfg.Dimensions, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("load embedding config: %w", err)
	}
	return cfg, nil
}

type SaveEmbeddingConfigInput struct {
	APIKey     *string
	ModelName  *string
	ModelURL   *string
	Dimensions *int
}

func SaveEmbeddingConfig(db *sql.DB, in SaveEmbeddingConfigInput) (*model.EmbeddingConfig, error) {
	cfg, err := GetEmbeddingConfig(db)
	if err != nil {
		return nil, err
	}

	if in.APIKey != nil {
		cfg.APIKey = strings.TrimSpace(*in.APIKey)
	}
	if in.ModelName != nil {
		cfg.ModelName = strings.TrimSpace(*in.ModelName)
	}
	if in.ModelURL != nil {
		cfg.ModelURL = strings.TrimSpace(*in.ModelURL)
	}
	if in.Dimensions != nil {
		cfg.Dimensions = *in.Dimensions
	}

	settings := embedding.Settings{
		APIKey:     cfg.APIKey,
		ModelName:  cfg.ModelName,
		ModelURL:   cfg.ModelURL,
		Dimensions: cfg.Dimensions,
	}.WithDefaults()
	if err := settings.Validate(false); err != nil {
		return nil, err
	}

	cfg.APIKey = settings.APIKey
	cfg.ModelName = settings.ModelName
	cfg.ModelURL = settings.ModelURL
	cfg.Dimensions = settings.Dimensions
	cfg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if _, err := db.Exec(`
		INSERT INTO embedding_config (id, api_key, model_name, model_url, dimensions, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			api_key = excluded.api_key,
			model_name = excluded.model_name,
			model_url = excluded.model_url,
			dimensions = excluded.dimensions,
			updated_at = excluded.updated_at
	`, cfg.APIKey, cfg.ModelName, cfg.ModelURL, cfg.Dimensions, cfg.UpdatedAt); err != nil {
		return nil, fmt.Errorf("save embedding config: %w", err)
	}

	return cfg, nil
}

func EffectiveEmbeddingSettings(db *sql.DB) (embedding.Settings, error) {
	cfg, err := GetEmbeddingConfig(db)
	if err != nil {
		return embedding.Settings{}, err
	}

	return embedding.Settings{
		APIKey:     cfg.APIKey,
		ModelName:  cfg.ModelName,
		ModelURL:   cfg.ModelURL,
		Dimensions: cfg.Dimensions,
	}.WithDefaults(), nil
}

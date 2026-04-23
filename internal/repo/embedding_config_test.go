package repo

import "testing"

func TestEmbeddingConfigLifecycle(t *testing.T) {
	db := newTestDB(t)

	if err := EnsureEmbeddingConfigTable(db); err != nil {
		t.Fatalf("ensure embedding config table: %v", err)
	}

	initial, err := GetEmbeddingConfig(db)
	if err != nil {
		t.Fatalf("get embedding config: %v", err)
	}
	if initial.ModelName == "" || initial.ModelURL == "" || initial.Dimensions <= 0 {
		t.Fatalf("expected seeded/default config, got %+v", initial)
	}
	if initial.APIKey != "" {
		t.Fatalf("expected empty api key by default, got %q", initial.APIKey)
	}

	apiKey := "dummy-key-123456"
	modelName := "embedding-3"
	modelURL := "https://example.com/embed"
	dimensions := 4
	saved, err := SaveEmbeddingConfig(db, SaveEmbeddingConfigInput{
		APIKey:     &apiKey,
		ModelName:  &modelName,
		ModelURL:   &modelURL,
		Dimensions: &dimensions,
	})
	if err != nil {
		t.Fatalf("save embedding config: %v", err)
	}
	if saved.APIKey != apiKey || saved.ModelURL != modelURL || saved.Dimensions != dimensions {
		t.Fatalf("unexpected saved config: %+v", saved)
	}

	effective, err := EffectiveEmbeddingSettings(db)
	if err != nil {
		t.Fatalf("effective embedding settings: %v", err)
	}
	if effective.APIKey != apiKey || effective.ModelName != modelName || effective.ModelURL != modelURL || effective.Dimensions != dimensions {
		t.Fatalf("unexpected effective settings: %+v", effective)
	}
}

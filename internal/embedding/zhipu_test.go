package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSettingsDefaultsAndValidation(t *testing.T) {
	settings := Settings{}
	withDefaults := settings.WithDefaults()
	if withDefaults.ModelName == "" || withDefaults.ModelURL == "" || withDefaults.Dimensions <= 0 {
		t.Fatalf("expected defaults to be applied, got %+v", withDefaults)
	}

	if err := settings.Validate(true); err == nil {
		t.Fatal("expected missing api key validation error")
	}

	settings.APIKey = "dummy"
	if err := settings.Validate(true); err != nil {
		t.Fatalf("expected settings with api key to validate: %v", err)
	}

	sig1 := ConfigSignature(Settings{ModelName: "embedding-3", ModelURL: "https://example.com", Dimensions: 4})
	sig2 := ConfigSignature(Settings{ModelName: "embedding-3", ModelURL: "https://example.com", Dimensions: 6})
	if sig1 == sig2 {
		t.Fatalf("expected config signature to change when dimensions change: %q", sig1)
	}
}

func TestNewClientFromEnvAndEmbedTexts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := map[string]any{
			"data": []map[string]any{
				{
					"index":     0,
					"embedding": []float64{1, 2, 3, 4},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv(ZhipuAPIKeyEnv, "env-key")
	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("new client from env: %v", err)
	}

	client.endpoint = server.URL
	client.dimensions = 4
	client.model = "embedding-3"
	client.settings.ModelURL = server.URL
	client.settings.Dimensions = 4

	vectors, err := client.EmbedTexts(context.Background(), []string{"午餐"})
	if err != nil {
		t.Fatalf("embed texts: %v", err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 4 {
		t.Fatalf("unexpected vectors: %#v", vectors)
	}

	if _, ok := os.LookupEnv(ZhipuAPIKeyEnv); !ok {
		t.Fatalf("expected env %s to stay set during test", ZhipuAPIKeyEnv)
	}
}

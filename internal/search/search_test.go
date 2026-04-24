package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	ledgerdb "github.com/ledger-ai/ledger/internal/db"
	"github.com/ledger-ai/ledger/internal/embedding"
	"github.com/ledger-ai/ledger/internal/repo"
)

func newSearchTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	db, err := ledgerdb.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := ledgerdb.InitFresh(db); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return db
}

func TestTransactionsKeywordAndHybridSearch(t *testing.T) {
	db := newSearchTestDB(t)
	categoryID, err := repo.ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}
	desc := "和同事吃火锅"
	raw := "今晚和同事一起吃火锅"
	if _, err := repo.AddTransaction(db, repo.AddTransactionInput{
		Direction:   "expense",
		Amount:      88,
		Currency:    "CNY",
		CategoryID:  categoryID,
		Description: &desc,
		RawInput:    &raw,
		OccurredAt:  "2026-04-23",
	}); err != nil {
		t.Fatalf("add transaction: %v", err)
	}

	keywordResult, err := Transactions(db, Input{
		Keyword: "火锅",
		Mode:    "keyword",
		Limit:   5,
	}, nil)
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(keywordResult.Items) != 1 || keywordResult.Items[0].MatchType != "keyword" {
		t.Fatalf("unexpected keyword result: %#v", keywordResult.Items)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	client, err := embedding.NewClient(embedding.Settings{
		APIKey:     "dummy",
		ModelName:  "embedding-3",
		ModelURL:   server.URL,
		Dimensions: 4,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	hybridResult, err := Transactions(db, Input{
		Keyword:  "火锅",
		Semantic: "和同事聚餐",
		Mode:     "hybrid",
		Limit:    5,
	}, client)
	if err != nil {
		t.Fatalf("hybrid search: %v", err)
	}
	if len(hybridResult.Items) != 1 || hybridResult.Items[0].MatchType != "hybrid" {
		t.Fatalf("unexpected hybrid result: %#v", hybridResult.Items)
	}
}

func TestKeywordSearchReturnsAllByDefault(t *testing.T) {
	db := newSearchTestDB(t)
	categoryID, err := repo.ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}

	// Insert 3 transactions all containing "火锅"
	for _, desc := range []string{"和同事吃火锅", "家人吃火锅", "朋友吃火锅"} {
		d := desc
		if _, err := repo.AddTransaction(db, repo.AddTransactionInput{
			Direction:   "expense",
			Amount:      100,
			Currency:    "CNY",
			CategoryID:  categoryID,
			Description: &d,
			OccurredAt:  "2026-04-23",
		}); err != nil {
			t.Fatalf("add transaction: %v", err)
		}
	}

	result, err := Transactions(db, Input{
		Keyword: "火锅",
		Mode:    "keyword",
		Limit:   0, // unlimited
	}, nil)
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Items))
	}
}

func TestSemanticSearchThresholdFiltering(t *testing.T) {
	db := newSearchTestDB(t)
	categoryID, err := repo.ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}

	// Insert 2 transactions
	for _, desc := range []string{"吃火锅", "交房租"} {
		d := desc
		if _, err := repo.AddTransaction(db, repo.AddTransactionInput{
			Direction:   "expense",
			Amount:      100,
			Currency:    "CNY",
			CategoryID:  categoryID,
			Description: &d,
			OccurredAt:  "2026-04-23",
		}); err != nil {
			t.Fatalf("add transaction: %v", err)
		}
	}

	// Mock embedding server: return [1,0,0,0] for any request
	// The first call embeds both docs, second call embeds the query
	// Both docs and query get same vector → cosine similarity = 1.0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		input := body["input"].([]any)
		data := make([]map[string]any, len(input))
		for i := range input {
			data[i] = map[string]any{
				"index":     i,
				"embedding": []float64{1, 0, 0, 0},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()

	client, err := embedding.NewClient(embedding.Settings{
		APIKey:     "dummy",
		ModelName:  "embedding-3",
		ModelURL:   server.URL,
		Dimensions: 4,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	// All vectors are identical → cosine = 1.0, threshold 0.5 should return all
	result, err := Transactions(db, Input{
		Semantic:  "吃饭",
		Mode:      "semantic",
		Limit:     0,
		Threshold: 0.5,
	}, client)
	if err != nil {
		t.Fatalf("semantic search: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 results with threshold 0.5, got %d", len(result.Items))
	}

	// Threshold 1.1 → nothing passes
	result, err = Transactions(db, Input{
		Semantic:  "吃饭",
		Mode:      "semantic",
		Limit:     0,
		Threshold: 1.1,
	}, client)
	if err != nil {
		t.Fatalf("semantic search high threshold: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected 0 results with threshold 1.1, got %d", len(result.Items))
	}
}

func TestHybridSearchNoDefaultTruncation(t *testing.T) {
	db := newSearchTestDB(t)
	categoryID, err := repo.ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}

	// Insert 15 transactions with "火锅" — more than the old default limit of 10
	for i := 0; i < 15; i++ {
		d := fmt.Sprintf("火锅聚餐第%d次", i+1)
		if _, err := repo.AddTransaction(db, repo.AddTransactionInput{
			Direction:   "expense",
			Amount:      float64(50 + i),
			Currency:    "CNY",
			CategoryID:  categoryID,
			Description: &d,
			OccurredAt:  "2026-04-23",
		}); err != nil {
			t.Fatalf("add transaction %d: %v", i, err)
		}
	}

	// keyword-only hybrid with limit=0 should return all 15
	result, err := Transactions(db, Input{
		Keyword: "火锅",
		Mode:    "hybrid",
		Limit:   0,
	}, nil)
	if err != nil {
		t.Fatalf("hybrid search: %v", err)
	}
	if len(result.Items) != 15 {
		t.Fatalf("expected 15 results with no limit, got %d", len(result.Items))
	}
}

func TestResolveModeValidation(t *testing.T) {
	if _, err := resolveMode(Input{Mode: "keyword"}); err == nil {
		t.Fatal("expected keyword mode without keyword to fail")
	}
	if _, err := resolveMode(Input{Mode: "semantic"}); err == nil {
		t.Fatal("expected semantic mode without semantic query to fail")
	}
	if _, err := resolveMode(Input{Mode: "invalid", Keyword: "火锅"}); err == nil {
		t.Fatal("expected invalid mode to fail")
	}
}

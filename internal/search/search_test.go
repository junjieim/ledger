package search

import (
	"database/sql"
	"encoding/json"
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

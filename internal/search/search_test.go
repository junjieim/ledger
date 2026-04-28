package search

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	ledgerdb "github.com/ledger-ai/ledger/internal/db"
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

func addSearchTransaction(t *testing.T, db *sql.DB, desc string) string {
	t.Helper()

	categoryID, err := repo.ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}
	tx, err := repo.AddTransaction(db, repo.AddTransactionInput{
		Direction:   "expense",
		Amount:      88,
		Currency:    "CNY",
		CategoryID:  categoryID,
		Description: &desc,
		OccurredAt:  "2026-04-23",
	})
	if err != nil {
		t.Fatalf("add transaction: %v", err)
	}
	return tx.ID
}

func TestTransactionsKeyword(t *testing.T) {
	db := newSearchTestDB(t)
	addSearchTransaction(t, db, "和同事吃火锅")
	addSearchTransaction(t, db, "地铁通勤")

	result, err := Transactions(db, Input{
		Keyword: "火锅",
		Limit:   5,
	})
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 keyword result, got %#v", result.Items)
	}
	if result.Items[0].Description != "和同事吃火锅" {
		t.Fatalf("unexpected keyword result: %#v", result.Items[0])
	}
}

func TestKeywordSearchReturnsAllByDefault(t *testing.T) {
	db := newSearchTestDB(t)

	for _, desc := range []string{"和同事吃火锅", "家人吃火锅", "朋友吃火锅"} {
		addSearchTransaction(t, db, desc)
	}

	result, err := Transactions(db, Input{
		Keyword: "火锅",
		Limit:   0,
	})
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Items))
	}
}

func TestKeywordSearchNoDefaultTruncation(t *testing.T) {
	db := newSearchTestDB(t)

	for i := 0; i < 15; i++ {
		addSearchTransaction(t, db, fmt.Sprintf("火锅聚餐第%d次", i+1))
	}

	result, err := Transactions(db, Input{
		Keyword: "火锅",
		Limit:   0,
	})
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if len(result.Items) != 15 {
		t.Fatalf("expected 15 results with no limit, got %d", len(result.Items))
	}
}

func TestKeywordSearchEmptyQueryReturnsNoMatches(t *testing.T) {
	db := newSearchTestDB(t)
	addSearchTransaction(t, db, "和同事吃火锅")

	result, err := Transactions(db, Input{Keyword: ""})
	if err != nil {
		t.Fatalf("empty keyword search: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no results for empty keyword, got %#v", result.Items)
	}
}

func TestKeywordSearchHandlesDeletedTransactions(t *testing.T) {
	db := newSearchTestDB(t)
	deletedID := addSearchTransaction(t, db, "和同事吃火锅")
	keptID := addSearchTransaction(t, db, "朋友吃火锅")

	if _, err := Transactions(db, Input{Keyword: "火锅"}); err != nil {
		t.Fatalf("initial keyword search: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM transactions WHERE id = ?`, deletedID); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}

	result, err := Transactions(db, Input{Keyword: "火锅"})
	if err != nil {
		t.Fatalf("keyword search after delete: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 remaining result, got %#v", result.Items)
	}
	if result.Items[0].ID != keptID {
		t.Fatalf("expected kept transaction %s, got %#v", keptID, result.Items[0])
	}
}

package db

import (
	"path/filepath"
	"testing"
)

func TestInitFreshResetsSchemaAndSeedsDefaults(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := InitFresh(db); err != nil {
		t.Fatalf("init fresh: %v", err)
	}

	var categoryCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&categoryCount); err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if categoryCount == 0 {
		t.Fatal("expected seeded categories")
	}

	if _, err := db.Exec(`INSERT INTO transactions (id, direction, amount, currency, occurred_at) VALUES ('tx-1', 'expense', 10, 'CNY', '2026-04-23')`); err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	if err := InitFresh(db); err != nil {
		t.Fatalf("re-init fresh: %v", err)
	}

	var transactionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&transactionCount); err != nil {
		t.Fatalf("count transactions after reset: %v", err)
	}
	if transactionCount != 0 {
		t.Fatalf("expected transactions to be reset, got %d", transactionCount)
	}
}

func TestRefundAmountColumnExistsAndIsConstrained(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := InitFresh(db); err != nil {
		t.Fatalf("init fresh: %v", err)
	}

	rows, err := db.Query(`PRAGMA table_info(transactions)`)
	if err != nil {
		t.Fatalf("table info: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		if name == "refund_amount" {
			found = true
			if notNull != 1 {
				t.Fatalf("expected refund_amount to be NOT NULL")
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}
	if !found {
		t.Fatal("expected refund_amount column")
	}

	if _, err := db.Exec(`INSERT INTO transactions (id, direction, amount, refund_amount, currency, occurred_at) VALUES ('tx-neg', 'expense', 10, -1, 'CNY', '2026-04-23')`); err == nil {
		t.Fatal("expected negative refund_amount to fail")
	}
	if _, err := db.Exec(`INSERT INTO transactions (id, direction, amount, refund_amount, currency, occurred_at) VALUES ('tx-over', 'expense', 10, 11, 'CNY', '2026-04-23')`); err == nil {
		t.Fatal("expected refund_amount greater than amount to fail")
	}
}

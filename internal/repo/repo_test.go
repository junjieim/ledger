package repo

import (
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	ledgerdb "github.com/ledger-ai/ledger/internal/db"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	db, err := ledgerdb.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := ledgerdb.InitFresh(db); err != nil {
		t.Fatalf("init test db: %v", err)
	}
	return db
}

func TestTransactionLifecycleAndBalance(t *testing.T) {
	db := newTestDB(t)

	categoryID, err := ResolveCategoryID(db, "餐饮")
	if err != nil {
		t.Fatalf("resolve category: %v", err)
	}
	if categoryID == nil {
		t.Fatal("expected seeded category 餐饮")
	}

	description := "午餐牛肉面"
	rawInput := "中午吃了一碗牛肉面花了 28.5"
	note := "公司附近"
	added, err := AddTransaction(db, AddTransactionInput{
		Direction:   "expense",
		Amount:      28.5,
		Currency:    "CNY",
		CategoryID:  categoryID,
		Description: &description,
		RawInput:    &rawInput,
		Note:        &note,
		Tags:        []string{"工作日", "午餐"},
		OccurredAt:  "2026-04-23",
	})
	if err != nil {
		t.Fatalf("add transaction: %v", err)
	}
	if added.Category != "餐饮" {
		t.Fatalf("expected category 餐饮, got %q", added.Category)
	}
	if len(added.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(added.Tags))
	}

	queried, err := QueryTransactions(db, QueryInput{
		Category:  "餐饮",
		Tag:       "工作日",
		Direction: "expense",
		Currency:  "CNY",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("query transactions: %v", err)
	}
	if queried.Total != 1 || len(queried.Items) != 1 {
		t.Fatalf("expected 1 queried transaction, got total=%d len=%d", queried.Total, len(queried.Items))
	}

	newAmount := 30.0
	newDescription := "午餐牛肉面加饮料"
	newDate := "2026-04-24"
	transportID, err := ResolveCategoryID(db, "交通")
	if err != nil {
		t.Fatalf("resolve category 交通: %v", err)
	}
	updated, err := UpdateTransaction(db, UpdateTransactionInput{
		ID:          added.ID,
		Amount:      &newAmount,
		CategoryID:  transportID,
		Description: &newDescription,
		Date:        &newDate,
		Tags:        &[]string{"通勤", "工作日"},
	})
	if err != nil {
		t.Fatalf("update transaction: %v", err)
	}
	if updated.Amount != newAmount || updated.OccurredAt != newDate {
		t.Fatalf("unexpected updated transaction: amount=%v date=%s", updated.Amount, updated.OccurredAt)
	}
	if updated.Description == nil || *updated.Description != newDescription {
		t.Fatalf("unexpected updated description: %#v", updated.Description)
	}
	if updated.Category != "交通" {
		t.Fatalf("expected updated category 交通, got %q", updated.Category)
	}
	slices.Sort(updated.Tags)
	if len(updated.Tags) != 2 || updated.Tags[0] != "工作日" || updated.Tags[1] != "通勤" {
		t.Fatalf("unexpected replaced tags: %#v", updated.Tags)
	}

	updated, err = UpdateTransaction(db, UpdateTransactionInput{
		ID:         added.ID,
		AddTags:    []string{"高铁"},
		RemoveTags: []string{"通勤"},
	})
	if err != nil {
		t.Fatalf("update transaction tags incrementally: %v", err)
	}
	slices.Sort(updated.Tags)
	if len(updated.Tags) != 2 || updated.Tags[0] != "工作日" || updated.Tags[1] != "高铁" {
		t.Fatalf("unexpected incrementally updated tags: %#v", updated.Tags)
	}

	updated, err = UpdateTransaction(db, UpdateTransactionInput{
		ID:        added.ID,
		ClearTags: true,
	})
	if err != nil {
		t.Fatalf("clear transaction tags: %v", err)
	}
	if len(updated.Tags) != 0 {
		t.Fatalf("expected tags to be cleared, got %#v", updated.Tags)
	}

	balance, err := GetBalance(db, "CNY")
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if len(balance.Balances) != 1 || balance.Balances[0].Balance != -30.0 {
		t.Fatalf("unexpected balance result: %#v", balance.Balances)
	}

	entries, err := QueryAuditLog(db, "", "", "", 20)
	if err != nil {
		t.Fatalf("query audit log: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 audit entries, got %d", len(entries))
	}

	if err := DeleteTransaction(db, added.ID); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}
	afterDelete, err := QueryTransactions(db, QueryInput{Limit: 10})
	if err != nil {
		t.Fatalf("query after delete: %v", err)
	}
	if afterDelete.Total != 0 {
		t.Fatalf("expected 0 transactions after delete, got %d", afterDelete.Total)
	}
}

func TestTransferCategoryAndTagLifecycle(t *testing.T) {
	db := newTestDB(t)

	category, err := AddCategory(db, AddCategoryInput{
		Name:      "差旅",
		Direction: "expense",
	})
	if err != nil {
		t.Fatalf("add category: %v", err)
	}

	if _, err := AddTag(db, "报销"); err != nil {
		t.Fatalf("add tag: %v", err)
	}

	description := "机票"
	added, err := AddTransaction(db, AddTransactionInput{
		Direction:   "expense",
		Amount:      1200,
		Currency:    "CNY",
		CategoryID:  &category.ID,
		Description: &description,
		Tags:        []string{"报销"},
		OccurredAt:  "2026-04-23",
	})
	if err != nil {
		t.Fatalf("add categorized transaction: %v", err)
	}

	if err := RemoveCategory(db, "差旅", false); err == nil {
		t.Fatal("expected remove category without force to fail")
	}
	if err := RemoveCategory(db, "差旅", true); err != nil {
		t.Fatalf("force remove category: %v", err)
	}

	reloaded, err := GetTransaction(db, added.ID)
	if err != nil {
		t.Fatalf("reload transaction: %v", err)
	}
	if reloaded.CategoryID != nil {
		t.Fatalf("expected category to be detached, got %#v", reloaded.CategoryID)
	}

	transfer, err := CreateTransfer(db, TransferInput{
		FromCurrency: "USD",
		ToCurrency:   "CNY",
		FromAmount:   100,
		ToAmount:     720,
		OccurredAt:   "2026-04-23",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if transfer.TransferGroup == "" || transfer.Expense == nil || transfer.Income == nil {
		t.Fatalf("unexpected transfer result: %#v", transfer)
	}

	balances, err := GetBalance(db, "")
	if err != nil {
		t.Fatalf("get balances: %v", err)
	}
	if len(balances.Balances) < 2 {
		t.Fatalf("expected multi-currency balances, got %#v", balances.Balances)
	}

	if err := DeleteTransaction(db, transfer.Expense.ID); err != nil {
		t.Fatalf("delete transfer leg: %v", err)
	}
	all, err := QueryTransactions(db, QueryInput{Limit: 20})
	if err != nil {
		t.Fatalf("query transactions: %v", err)
	}
	for _, item := range all.Items {
		if item.TransferGroup != nil && *item.TransferGroup == transfer.TransferGroup {
			t.Fatalf("expected transfer pair to be deleted, still found %+v", item)
		}
	}

	if err := RemoveTag(db, "报销"); err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	tags, err := ListTags(db)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected 0 tags after removal, got %d", len(tags))
	}
}

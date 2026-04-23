package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledger-ai/ledger/internal/model"
)

type AddTransactionInput struct {
	Direction   string
	Amount      float64
	Currency    string
	CategoryID  *string
	Description *string
	RawInput    *string
	Note        *string
	Tags        []string
	OccurredAt  string
}

type TransferInput struct {
	FromCurrency string
	ToCurrency   string
	FromAmount   float64
	ToAmount     float64
	OccurredAt   string
	Note         *string
}

func AddTransaction(db *sql.DB, in AddTransactionInput) (*model.Transaction, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	if in.OccurredAt == "" {
		in.OccurredAt = time.Now().Format("2006-01-02")
	}
	if in.Currency == "" {
		in.Currency = "CNY"
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO transactions (id, direction, amount, currency, category_id, description, raw_input, note, occurred_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, in.Direction, in.Amount, in.Currency, in.CategoryID, in.Description, in.RawInput, in.Note, in.OccurredAt, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	// Add tags
	for _, tagName := range in.Tags {
		tagID, err := ensureTag(tx, tagName)
		if err != nil {
			return nil, fmt.Errorf("ensure tag %q: %w", tagName, err)
		}
		_, err = tx.Exec("INSERT INTO transaction_tags (transaction_id, tag_id) VALUES (?, ?)", id, tagID)
		if err != nil {
			return nil, fmt.Errorf("link tag: %w", err)
		}
	}

	// Audit log
	if err := logAudit(tx, "add_transaction", "transaction", id, in); err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return GetTransaction(db, id)
}

func CreateTransfer(db *sql.DB, in TransferInput) (*model.TransferResult, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if in.OccurredAt == "" {
		in.OccurredAt = time.Now().Format("2006-01-02")
	}

	groupID := "tf-" + uuid.New().String()
	expenseID := uuid.New().String()
	incomeID := uuid.New().String()

	expenseDescription := fmt.Sprintf("Transfer to %s", in.ToCurrency)
	incomeDescription := fmt.Sprintf("Transfer from %s", in.FromCurrency)

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO transactions (id, direction, amount, currency, transfer_group, description, note, occurred_at, created_at, updated_at)
		 VALUES (?, 'expense', ?, ?, ?, ?, ?, ?, ?, ?)`,
		expenseID, in.FromAmount, in.FromCurrency, groupID, expenseDescription, in.Note, in.OccurredAt, now, now,
	); err != nil {
		return nil, fmt.Errorf("insert transfer expense: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO transactions (id, direction, amount, currency, transfer_group, description, note, occurred_at, created_at, updated_at)
		 VALUES (?, 'income', ?, ?, ?, ?, ?, ?, ?, ?)`,
		incomeID, in.ToAmount, in.ToCurrency, groupID, incomeDescription, in.Note, in.OccurredAt, now, now,
	); err != nil {
		return nil, fmt.Errorf("insert transfer income: %w", err)
	}

	if err := logAudit(tx, "create_transfer", "transaction", groupID, in); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	expense, err := GetTransaction(db, expenseID)
	if err != nil {
		return nil, err
	}
	income, err := GetTransaction(db, incomeID)
	if err != nil {
		return nil, err
	}

	return &model.TransferResult{
		TransferGroup: groupID,
		Expense:       expense,
		Income:        income,
	}, nil
}

func ensureTag(tx *sql.Tx, name string) (string, error) {
	var id string
	err := tx.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	id = uuid.New().String()
	_, err = tx.Exec("INSERT INTO tags (id, name) VALUES (?, ?)", id, name)
	return id, err
}

func GetTransaction(db *sql.DB, id string) (*model.Transaction, error) {
	t := &model.Transaction{}
	var catName sql.NullString
	var createdAt model.SQLiteTime
	var updatedAt model.SQLiteTime
	err := db.QueryRow(
		`SELECT t.id, t.direction, t.amount, t.currency, t.transfer_group,
		        t.category_id, c.name, t.description, t.raw_input, t.note,
		        t.occurred_at, t.created_at, t.updated_at
		 FROM transactions t
		 LEFT JOIN categories c ON t.category_id = c.id
		 WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.Direction, &t.Amount, &t.Currency, &t.TransferGroup,
		&t.CategoryID, &catName, &t.Description, &t.RawInput, &t.Note,
		&t.OccurredAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = createdAt.Time
	t.UpdatedAt = updatedAt.Time
	if catName.Valid {
		t.Category = catName.String
	}

	// Load tags
	rows, err := db.Query(
		`SELECT tg.name FROM tags tg
		 JOIN transaction_tags tt ON tt.tag_id = tg.id
		 WHERE tt.transaction_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		t.Tags = append(t.Tags, name)
	}
	return t, rows.Err()
}

func DeleteTransaction(db *sql.DB, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if this is part of a transfer — delete the pair
	var transferGroup sql.NullString
	err = tx.QueryRow("SELECT transfer_group FROM transactions WHERE id = ?", id).Scan(&transferGroup)
	if err != nil {
		return fmt.Errorf("find transaction: %w", err)
	}

	if transferGroup.Valid {
		_, err = tx.Exec("DELETE FROM transaction_tags WHERE transaction_id IN (SELECT id FROM transactions WHERE transfer_group = ?)", transferGroup.String)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM transactions WHERE transfer_group = ?", transferGroup.String)
	} else {
		_, err = tx.Exec("DELETE FROM transaction_tags WHERE transaction_id = ?", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM transactions WHERE id = ?", id)
	}
	if err != nil {
		return err
	}

	if err := logAudit(tx, "delete_transaction", "transaction", id, nil); err != nil {
		return err
	}
	return tx.Commit()
}

type UpdateTransactionInput struct {
	ID          string
	Amount      *float64
	Direction   *string
	Currency    *string
	CategoryID  *string
	Date        *string
	Description *string
	Note        *string
}

func UpdateTransaction(db *sql.DB, in UpdateTransactionInput) (*model.Transaction, error) {
	sets := []string{}
	args := []interface{}{}

	if in.Amount != nil {
		sets = append(sets, "amount = ?")
		args = append(args, *in.Amount)
	}
	if in.Direction != nil {
		sets = append(sets, "direction = ?")
		args = append(args, *in.Direction)
	}
	if in.Currency != nil {
		sets = append(sets, "currency = ?")
		args = append(args, *in.Currency)
	}
	if in.CategoryID != nil {
		sets = append(sets, "category_id = ?")
		args = append(args, *in.CategoryID)
	}
	if in.Date != nil {
		sets = append(sets, "occurred_at = ?")
		args = append(args, *in.Date)
	}
	if in.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *in.Description)
	}
	if in.Note != nil {
		sets = append(sets, "note = ?")
		args = append(args, *in.Note)
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, in.ID)

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("UPDATE transactions SET %s WHERE id = ?", strings.Join(sets, ", "))
	res, err := tx.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("transaction not found: %s", in.ID)
	}

	if err := logAudit(tx, "update_transaction", "transaction", in.ID, in); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return GetTransaction(db, in.ID)
}

type QueryInput struct {
	From      string
	To        string
	Direction string
	Category  string
	Tag       string
	Currency  string
	MinAmount *float64
	MaxAmount *float64
	Limit     int
	Offset    int
}

func QueryTransactions(db *sql.DB, in QueryInput) (*model.QueryResult, error) {
	if in.Limit <= 0 {
		in.Limit = 50
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if in.From != "" {
		where = append(where, "t.occurred_at >= ?")
		args = append(args, in.From)
	}
	if in.To != "" {
		where = append(where, "t.occurred_at <= ?")
		args = append(args, in.To)
	}
	if in.Direction != "" {
		where = append(where, "t.direction = ?")
		args = append(args, in.Direction)
	}
	if in.Currency != "" {
		where = append(where, "t.currency = ?")
		args = append(args, in.Currency)
	}
	if in.Category != "" {
		where = append(where, "c.name = ?")
		args = append(args, in.Category)
	}
	if in.Tag != "" {
		where = append(where, "t.id IN (SELECT tt.transaction_id FROM transaction_tags tt JOIN tags tg ON tt.tag_id = tg.id WHERE tg.name = ?)")
		args = append(args, in.Tag)
	}
	if in.MinAmount != nil {
		where = append(where, "t.amount >= ?")
		args = append(args, *in.MinAmount)
	}
	if in.MaxAmount != nil {
		where = append(where, "t.amount <= ?")
		args = append(args, *in.MaxAmount)
	}

	whereClause := strings.Join(where, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM transactions t LEFT JOIN categories c ON t.category_id = c.id WHERE %s",
		whereClause,
	)
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch items
	selectQuery := fmt.Sprintf(
		`SELECT t.id, t.direction, t.amount, t.currency, t.transfer_group,
		        t.category_id, c.name, t.description, t.raw_input, t.note,
		        t.occurred_at, t.created_at, t.updated_at
		 FROM transactions t
		 LEFT JOIN categories c ON t.category_id = c.id
		 WHERE %s
		 ORDER BY t.occurred_at DESC, t.created_at DESC
		 LIMIT ? OFFSET ?`,
		whereClause,
	)
	args = append(args, in.Limit, in.Offset)
	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Transaction
	for rows.Next() {
		t := &model.Transaction{}
		var catName sql.NullString
		var createdAt model.SQLiteTime
		var updatedAt model.SQLiteTime
		if err := rows.Scan(&t.ID, &t.Direction, &t.Amount, &t.Currency, &t.TransferGroup,
			&t.CategoryID, &catName, &t.Description, &t.RawInput, &t.Note,
			&t.OccurredAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		t.CreatedAt = createdAt.Time
		t.UpdatedAt = updatedAt.Time
		if catName.Valid {
			t.Category = catName.String
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each item
	for _, item := range items {
		tagRows, err := db.Query(
			"SELECT tg.name FROM tags tg JOIN transaction_tags tt ON tt.tag_id = tg.id WHERE tt.transaction_id = ?",
			item.ID,
		)
		if err != nil {
			return nil, err
		}
		for tagRows.Next() {
			var name string
			if err := tagRows.Scan(&name); err != nil {
				tagRows.Close()
				return nil, err
			}
			item.Tags = append(item.Tags, name)
		}
		tagRows.Close()
	}

	return &model.QueryResult{Total: total, Items: items}, nil
}

func GetBalance(db *sql.DB, currency string) (*model.BalanceResult, error) {
	query := `SELECT currency, SUM(CASE direction WHEN 'income' THEN amount ELSE -amount END) AS balance
	          FROM transactions`
	args := []interface{}{}
	if currency != "" {
		query += " WHERE currency = ?"
		args = append(args, currency)
	}
	query += " GROUP BY currency ORDER BY currency"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []model.Balance
	for rows.Next() {
		var b model.Balance
		if err := rows.Scan(&b.Currency, &b.Balance); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return &model.BalanceResult{Balances: balances}, rows.Err()
}

// ResolveCategoryID finds a category ID by name. Returns nil if not found.
func ResolveCategoryID(db *sql.DB, name string) (*string, error) {
	var id string
	err := db.QueryRow("SELECT id FROM categories WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

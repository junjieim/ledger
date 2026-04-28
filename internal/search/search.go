package search

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/tokenizer"
)

type Input struct {
	Keyword string
	Limit   int
}

type indexedDocument struct {
	Transaction *model.Transaction
	SearchText  string
	Hash        string
}

type scoredMatch struct {
	ID    string
	Score float64
}

func Transactions(db *sql.DB, in Input) (*model.SearchResult, error) {
	keyword := strings.TrimSpace(in.Keyword)
	if keyword == "" {
		return &model.SearchResult{}, nil
	}

	docs, docMap, err := loadDocuments(db)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return &model.SearchResult{}, nil
	}

	if err := syncKeywordIndex(db, docs); err != nil {
		return nil, err
	}

	matches, err := keywordSearch(db, keyword, in.Limit)
	if err != nil {
		return nil, err
	}

	items := make([]model.SearchItem, 0, len(matches))
	for _, match := range matches {
		doc, ok := docMap[match.ID]
		if !ok {
			continue
		}
		item := model.SearchItem{
			ID:            doc.ID,
			Score:         match.Score,
			Direction:     doc.Direction,
			Amount:        doc.Amount,
			Currency:      doc.Currency,
			Category:      doc.Category,
			OccurredAt:    doc.OccurredAt,
			Tags:          doc.Tags,
			TransferGroup: doc.TransferGroup,
		}
		if doc.Description != nil {
			item.Description = *doc.Description
		}
		items = append(items, item)
	}

	return &model.SearchResult{Items: items}, nil
}

func loadDocuments(db *sql.DB) ([]indexedDocument, map[string]*model.Transaction, error) {
	rows, err := db.Query(`
		SELECT
			t.id, t.direction, t.amount, t.currency, t.transfer_group,
			t.category_id, c.name, t.description, t.raw_input, t.note,
			t.occurred_at, t.created_at, t.updated_at,
			COALESCE(GROUP_CONCAT(tg.name, char(31)), '')
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		LEFT JOIN transaction_tags tt ON tt.transaction_id = t.id
		LEFT JOIN tags tg ON tg.id = tt.tag_id
		GROUP BY
			t.id, t.direction, t.amount, t.currency, t.transfer_group,
			t.category_id, c.name, t.description, t.raw_input, t.note,
			t.occurred_at, t.created_at, t.updated_at
		ORDER BY t.occurred_at DESC, t.created_at DESC
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("load search documents: %w", err)
	}
	defer rows.Close()

	var docs []indexedDocument
	docMap := make(map[string]*model.Transaction)
	for rows.Next() {
		tx := &model.Transaction{}
		var category sql.NullString
		var tagsJoined string
		var createdAt model.SQLiteTime
		var updatedAt model.SQLiteTime
		if err := rows.Scan(
			&tx.ID, &tx.Direction, &tx.Amount, &tx.Currency, &tx.TransferGroup,
			&tx.CategoryID, &category, &tx.Description, &tx.RawInput, &tx.Note,
			&tx.OccurredAt, &createdAt, &updatedAt, &tagsJoined,
		); err != nil {
			return nil, nil, fmt.Errorf("scan search document: %w", err)
		}
		tx.CreatedAt = createdAt.Time
		tx.UpdatedAt = updatedAt.Time
		if category.Valid {
			tx.Category = category.String
		}
		if tagsJoined != "" {
			tx.Tags = strings.Split(tagsJoined, string(rune(31)))
		}

		searchText := buildSearchText(tx)
		docs = append(docs, indexedDocument{
			Transaction: tx,
			SearchText:  searchText,
			Hash:        hashText(searchText),
		})
		docMap[tx.ID] = tx
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate search documents: %w", err)
	}
	return docs, docMap, nil
}

func syncKeywordIndex(db *sql.DB, docs []indexedDocument) error {
	if _, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS transactions_search USING fts5(
		transaction_id UNINDEXED,
		content
	)`); err != nil {
		return fmt.Errorf("create transactions_search: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS keyword_index_hashes (
		transaction_id TEXT PRIMARY KEY,
		source_hash TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create keyword_index_hashes: %w", err)
	}

	existing := map[string]string{}
	rows, err := db.Query(`SELECT transaction_id, source_hash FROM keyword_index_hashes`)
	if err != nil {
		return fmt.Errorf("load keyword index hashes: %w", err)
	}
	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			rows.Close()
			return fmt.Errorf("scan keyword index hash: %w", err)
		}
		existing[id] = hash
	}
	rows.Close()

	currentIDs := make(map[string]struct{}, len(docs))
	var pending []indexedDocument
	for _, doc := range docs {
		currentIDs[doc.Transaction.ID] = struct{}{}
		if existing[doc.Transaction.ID] != doc.Hash {
			pending = append(pending, doc)
		}
	}

	var staleIDs []string
	for id := range existing {
		if _, ok := currentIDs[id]; !ok {
			staleIDs = append(staleIDs, id)
		}
	}

	if len(pending) == 0 && len(staleIDs) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin keyword index sync: %w", err)
	}
	defer tx.Rollback()

	for _, id := range staleIDs {
		if _, err := tx.Exec(`DELETE FROM transactions_search WHERE transaction_id = ?`, id); err != nil {
			return fmt.Errorf("delete stale keyword index %s: %w", id, err)
		}
		if _, err := tx.Exec(`DELETE FROM keyword_index_hashes WHERE transaction_id = ?`, id); err != nil {
			return fmt.Errorf("delete stale keyword hash %s: %w", id, err)
		}
	}

	for _, doc := range pending {
		if _, ok := existing[doc.Transaction.ID]; ok {
			if _, err := tx.Exec(`DELETE FROM transactions_search WHERE transaction_id = ?`, doc.Transaction.ID); err != nil {
				return fmt.Errorf("delete old keyword index %s: %w", doc.Transaction.ID, err)
			}
		}
		content, err := tokenizer.TokenizeDocument(doc.SearchText)
		if err != nil {
			return fmt.Errorf("tokenize document %s: %w", doc.Transaction.ID, err)
		}
		if _, err := tx.Exec(`INSERT INTO transactions_search (transaction_id, content) VALUES (?, ?)`, doc.Transaction.ID, content); err != nil {
			return fmt.Errorf("insert keyword index %s: %w", doc.Transaction.ID, err)
		}
		if _, err := tx.Exec(`INSERT INTO keyword_index_hashes (transaction_id, source_hash) VALUES (?, ?) ON CONFLICT(transaction_id) DO UPDATE SET source_hash = excluded.source_hash`,
			doc.Transaction.ID, doc.Hash); err != nil {
			return fmt.Errorf("upsert keyword hash %s: %w", doc.Transaction.ID, err)
		}
	}

	return tx.Commit()
}

func keywordSearch(db *sql.DB, keyword string, limit int) ([]scoredMatch, error) {
	query, err := tokenizer.TokenizeQuery(keyword)
	if err != nil {
		return nil, fmt.Errorf("tokenize keyword query: %w", err)
	}
	if query == "" {
		return nil, nil
	}

	var rows *sql.Rows
	if limit > 0 {
		rows, err = db.Query(`
			SELECT transaction_id, bm25(transactions_search) AS rank
			FROM transactions_search
			WHERE transactions_search MATCH ?
			ORDER BY rank
			LIMIT ?
		`, query, limit)
	} else {
		rows, err = db.Query(`
			SELECT transaction_id, bm25(transactions_search) AS rank
			FROM transactions_search
			WHERE transactions_search MATCH ?
			ORDER BY rank
		`, query)
	}
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}
	defer rows.Close()

	var matches []scoredMatch
	position := 0
	for rows.Next() {
		var id string
		var rank float64
		if err := rows.Scan(&id, &rank); err != nil {
			return nil, fmt.Errorf("scan keyword match: %w", err)
		}
		matches = append(matches, scoredMatch{
			ID:    id,
			Score: rankScore(position),
		})
		position++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate keyword matches: %w", err)
	}
	return matches, nil
}

func rankScore(position int) float64 {
	return 1.0 / float64(position+1)
}

func buildSearchText(tx *model.Transaction) string {
	parts := make([]string, 0, 5+len(tx.Tags))
	if tx.Category != "" {
		parts = append(parts, tx.Category)
	}
	parts = append(parts, tx.Tags...)
	if tx.Description != nil && strings.TrimSpace(*tx.Description) != "" {
		parts = append(parts, strings.TrimSpace(*tx.Description))
	}
	if tx.RawInput != nil && strings.TrimSpace(*tx.RawInput) != "" {
		parts = append(parts, strings.TrimSpace(*tx.RawInput))
	}
	if tx.Note != nil && strings.TrimSpace(*tx.Note) != "" {
		parts = append(parts, strings.TrimSpace(*tx.Note))
	}
	if len(parts) == 0 {
		parts = append(parts, tx.Direction, tx.Currency)
	}
	return strings.Join(parts, "\n")
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

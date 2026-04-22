package search

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ledger-ai/ledger/internal/embedding"
	"github.com/ledger-ai/ledger/internal/model"
	"github.com/ledger-ai/ledger/internal/tokenizer"
)

const rrfK = 60.0

type Input struct {
	Keyword  string
	Semantic string
	Mode     string
	Limit    int
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

func Transactions(db *sql.DB, in Input, embedder *embedding.Client) (*model.SearchResult, error) {
	mode, err := resolveMode(in)
	if err != nil {
		return nil, err
	}
	if in.Limit <= 0 {
		in.Limit = 10
	}

	docs, docMap, err := loadDocuments(db)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return &model.SearchResult{}, nil
	}

	if mode == "keyword" || (mode == "hybrid" && strings.TrimSpace(in.Keyword) != "") {
		if err := rebuildKeywordIndex(db, docs); err != nil {
			return nil, err
		}
	}
	if mode == "semantic" || (mode == "hybrid" && strings.TrimSpace(in.Semantic) != "") {
		if embedder == nil {
			return nil, fmt.Errorf("semantic search requires %s", embedding.ZhipuAPIKeyEnv)
		}
		if err := syncEmbeddings(context.Background(), db, docs, embedder); err != nil {
			return nil, err
		}
	}

	var keywordMatches []scoredMatch
	if mode == "keyword" || (mode == "hybrid" && strings.TrimSpace(in.Keyword) != "") {
		keywordMatches, err = keywordSearch(db, strings.TrimSpace(in.Keyword), in.Limit)
		if err != nil {
			return nil, err
		}
	}

	var semanticMatches []scoredMatch
	if mode == "semantic" || (mode == "hybrid" && strings.TrimSpace(in.Semantic) != "") {
		semanticMatches, err = semanticSearch(context.Background(), db, strings.TrimSpace(in.Semantic), in.Limit, embedder)
		if err != nil {
			return nil, err
		}
	}

	finalMode := mode
	matches := keywordMatches
	switch mode {
	case "semantic":
		matches = semanticMatches
	case "hybrid":
		if strings.TrimSpace(in.Keyword) == "" {
			finalMode = "semantic"
			matches = semanticMatches
		} else if strings.TrimSpace(in.Semantic) == "" {
			finalMode = "keyword"
			matches = keywordMatches
		} else {
			matches = hybridize(keywordMatches, semanticMatches, in.Limit)
		}
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
			MatchType:     finalMode,
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

func resolveMode(in Input) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = "hybrid"
	}
	switch mode {
	case "keyword":
		if strings.TrimSpace(in.Keyword) == "" {
			return "", fmt.Errorf("--keyword is required when --mode=keyword")
		}
	case "semantic":
		if strings.TrimSpace(in.Semantic) == "" {
			return "", fmt.Errorf("--semantic is required when --mode=semantic")
		}
	case "hybrid":
		if strings.TrimSpace(in.Keyword) == "" && strings.TrimSpace(in.Semantic) == "" {
			return "", fmt.Errorf("at least one of --keyword or --semantic is required")
		}
	default:
		return "", fmt.Errorf("invalid --mode %q, must be keyword, semantic, or hybrid", in.Mode)
	}
	return mode, nil
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
		if err := rows.Scan(
			&tx.ID, &tx.Direction, &tx.Amount, &tx.Currency, &tx.TransferGroup,
			&tx.CategoryID, &category, &tx.Description, &tx.RawInput, &tx.Note,
			&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tagsJoined,
		); err != nil {
			return nil, nil, fmt.Errorf("scan search document: %w", err)
		}
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

func rebuildKeywordIndex(db *sql.DB, docs []indexedDocument) error {
	if _, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS transactions_search USING fts5(
		transaction_id UNINDEXED,
		content
	)`); err != nil {
		return fmt.Errorf("create transactions_search: %w", err)
	}
	if _, err := db.Exec(`DELETE FROM transactions_search`); err != nil {
		return fmt.Errorf("clear transactions_search: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin keyword index rebuild: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO transactions_search (transaction_id, content) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare keyword index insert: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		content, err := tokenizer.TokenizeDocument(doc.SearchText)
		if err != nil {
			return fmt.Errorf("tokenize document %s: %w", doc.Transaction.ID, err)
		}
		if _, err := stmt.Exec(doc.Transaction.ID, content); err != nil {
			return fmt.Errorf("insert keyword index for %s: %w", doc.Transaction.ID, err)
		}
	}
	return tx.Commit()
}

func keywordSearch(db *sql.DB, keyword string, limit int) ([]scoredMatch, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, nil
	}
	query, err := tokenizer.TokenizeQuery(keyword)
	if err != nil {
		return nil, fmt.Errorf("tokenize keyword query: %w", err)
	}
	if query == "" {
		return nil, nil
	}

	rows, err := db.Query(`
		SELECT transaction_id, bm25(transactions_search) AS rank
		FROM transactions_search
		WHERE transactions_search MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
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

func syncEmbeddings(ctx context.Context, db *sql.DB, docs []indexedDocument, embedder *embedding.Client) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS transaction_embeddings (
		transaction_id TEXT PRIMARY KEY,
		source_hash TEXT NOT NULL,
		embedding_json TEXT NOT NULL,
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
	)`); err != nil {
		return fmt.Errorf("create transaction_embeddings: %w", err)
	}

	existing := map[string]string{}
	rows, err := db.Query(`SELECT transaction_id, source_hash FROM transaction_embeddings`)
	if err != nil {
		return fmt.Errorf("load existing embeddings: %w", err)
	}
	for rows.Next() {
		var id string
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing embedding: %w", err)
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
	for id := range existing {
		if _, ok := currentIDs[id]; ok {
			continue
		}
		if _, err := db.Exec(`DELETE FROM transaction_embeddings WHERE transaction_id = ?`, id); err != nil {
			return fmt.Errorf("delete stale embedding %s: %w", id, err)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	texts := make([]string, 0, len(pending))
	for _, doc := range pending {
		texts = append(texts, doc.SearchText)
	}
	vectors, err := embedder.EmbedTexts(ctx, texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(pending) {
		return fmt.Errorf("embedding client returned %d vectors for %d documents", len(vectors), len(pending))
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin embedding sync: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO transaction_embeddings (transaction_id, source_hash, embedding_json, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(transaction_id) DO UPDATE SET
			source_hash = excluded.source_hash,
			embedding_json = excluded.embedding_json,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("prepare embedding upsert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for i, doc := range pending {
		vectorJSON, err := vectorToJSON(vectors[i])
		if err != nil {
			return fmt.Errorf("encode embedding for %s: %w", doc.Transaction.ID, err)
		}
		if _, err := stmt.Exec(doc.Transaction.ID, doc.Hash, vectorJSON, now); err != nil {
			return fmt.Errorf("upsert embedding for %s: %w", doc.Transaction.ID, err)
		}
	}

	return tx.Commit()
}

func semanticSearch(ctx context.Context, db *sql.DB, query string, limit int, embedder *embedding.Client) ([]scoredMatch, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	vectors, err := embedder.EmbedTexts(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("embedding client returned %d vectors for semantic query", len(vectors))
	}
	rows, err := db.Query(`
		SELECT transaction_id, embedding_json
		FROM transaction_embeddings
	`)
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}
	defer rows.Close()

	var matches []scoredMatch
	for rows.Next() {
		var id string
		var embeddingJSON string
		if err := rows.Scan(&id, &embeddingJSON); err != nil {
			return nil, fmt.Errorf("scan semantic match: %w", err)
		}
		var stored []float32
		if err := json.Unmarshal([]byte(embeddingJSON), &stored); err != nil {
			return nil, fmt.Errorf("decode stored embedding for %s: %w", id, err)
		}
		score := cosineSimilarity(vectors[0], stored)
		matches = append(matches, scoredMatch{
			ID:    id,
			Score: score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic matches: %w", err)
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func hybridize(keywordMatches, semanticMatches []scoredMatch, limit int) []scoredMatch {
	combined := map[string]float64{}
	for i, match := range keywordMatches {
		combined[match.ID] += 1.0 / (rrfK + float64(i+1))
	}
	for i, match := range semanticMatches {
		combined[match.ID] += 1.0 / (rrfK + float64(i+1))
	}

	results := make([]scoredMatch, 0, len(combined))
	for id, score := range combined {
		results = append(results, scoredMatch{ID: id, Score: score})
	}

	sortMatches(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func sortMatches(matches []scoredMatch) {
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
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

func vectorToJSON(vector []float32) (string, error) {
	data, err := json.Marshal(vector)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot float64
	var normA float64
	var normB float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		normA += af * af
		normB += bf * bf
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

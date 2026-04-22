package repo

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/ledger-ai/ledger/internal/model"
)

func logAudit(tx *sql.Tx, action, targetType, targetID string, detail interface{}) error {
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		detailJSON = []byte("{}")
	}
	_, err = tx.Exec(
		`INSERT INTO audit_log (id, action, target_type, target_id, detail)
		 VALUES (?, ?, ?, ?, ?)`,
		uuid.New().String(), action, targetType, targetID, string(detailJSON),
	)
	return err
}

func QueryAuditLog(db *sql.DB, action string, from, to string, limit int) ([]model.AuditEntry, error) {
	query := "SELECT id, action, target_type, target_id, COALESCE(agent_id,''), COALESCE(input_summary,''), COALESCE(detail,''), created_at FROM audit_log WHERE 1=1"
	args := []interface{}{}

	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}
	if from != "" {
		query += " AND created_at >= ?"
		args = append(args, from)
	}
	if to != "" {
		query += " AND created_at <= ?"
		args = append(args, to)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.TargetType, &e.TargetID, &e.AgentID, &e.InputSummary, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

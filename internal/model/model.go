package model

import "time"

type Transaction struct {
	ID            string    `json:"id"`
	Direction     string    `json:"direction"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	TransferGroup *string   `json:"transfer_group,omitempty"`
	CategoryID    *string   `json:"category_id,omitempty"`
	Category      string    `json:"category,omitempty"`
	Description   *string   `json:"description,omitempty"`
	RawInput      *string   `json:"raw_input,omitempty"`
	Note          *string   `json:"note,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
	OccurredAt    string    `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Category struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id,omitempty"`
	Direction string  `json:"direction"`
	Icon      *string `json:"icon,omitempty"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuditEntry struct {
	ID           string `json:"id"`
	Action       string `json:"action"`
	TargetType   string `json:"target_type"`
	TargetID     string `json:"target_id"`
	AgentID      string `json:"agent_id,omitempty"`
	InputSummary string `json:"input_summary,omitempty"`
	Detail       string `json:"detail,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type Balance struct {
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
}

type QueryResult struct {
	Total int            `json:"total"`
	Items []*Transaction `json:"items"`
}

type BalanceResult struct {
	Balances []Balance `json:"balances"`
}

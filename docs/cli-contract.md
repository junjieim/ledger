# CLI Interface Contract V2

All commands accept structured parameters only. Natural language understanding is the external AI agent's responsibility.

Notes:
- Refunds use `transactions.refund_amount` on the original expense row. Databases can be dropped and recreated; no legacy compatibility is maintained.

## Global Options

```
--db PATH       Database path (default ~/.ledger/ledger.db)
--json          Output JSON format (default human-readable table)
```

## Commands

### `ledger init`

Initialize database (create tables + seed default categories).

```
ledger init
ledger init --force       # Recreate database (destructive)
```

---

### `ledger add`

Add a transaction.

**Required:**
- `--amount FLOAT` — Amount (e.g. 15.5)
- `--direction STRING` — `income` | `expense`

**Optional:**
- `--currency STRING` — ISO 4217 (default `CNY`)
- `--category STRING` — Category name, must match existing
- `--date STRING` — ISO8601 date (default today)
- `--description STRING` — Structured summary (agent-generated)
- `--raw-input STRING` — Original natural language input (for search indexing)
- `--tag STRING` — Tag name, repeatable
- `--note STRING` — Additional note

**Output (--json):**
```json
{
  "id": "uuid",
  "direction": "expense",
  "amount": 15.0,
  "refund_amount": 0,
  "net_amount": 15.0,
  "currency": "CNY",
  "category": "餐饮",
  "description": "午餐，牛肉面",
  "tags": ["工作日"],
  "occurred_at": "2026-04-22"
}
```

**Exit codes:** 0 success | 1 param error | 2 database error

---

### `ledger delete`

Delete a transaction. If part of a transfer, automatically deletes the linked counterpart.

**Required:**
- `--id STRING` — Transaction UUID

**Output:** `{ "deleted": true, "id": "uuid" }`

---

### `ledger update`

Update a transaction.

**Required:**
- `--id STRING` — Transaction UUID

**Optional (at least one):**
- `--amount INT`
- `--direction STRING`
- `--currency STRING`
- `--category STRING`
- `--date STRING`
- `--description STRING`
- `--note STRING`
- `--tag STRING` — Replace tags with the provided full set, repeatable
- `--add-tag STRING` — Add tags without replacing existing ones, repeatable
- `--remove-tag STRING` — Remove tags by name, repeatable
- `--clear-tags` — Remove all tags

**Output:** Updated full transaction record (same format as `add`)

---

### `ledger refund`

Record a full or partial refund against an existing expense. This updates the original expense row instead of creating a separate refund row.

**Required:**
- `--id STRING` — Original expense transaction UUID

**Optional:**
- `--amount FLOAT` — Refund amount to add. Omit or pass `0` to refund the remaining amount.
- `--note STRING` — Refund note. Appended to the transaction note as `[退款 N CCY] <note>`.

Rules:
- Only `expense` transactions can be refunded.
- Transfer legs cannot be refunded.
- Cumulative refunds cannot exceed the original amount.
- Refunds affect balance through net expense: `amount - refund_amount`.

**Output (--json):**
```json
{
  "id": "uuid",
  "direction": "expense",
  "amount": 100.0,
  "refund_amount": 25.0,
  "net_amount": 75.0,
  "currency": "CNY",
  "category": "购物",
  "note": "[退款 25 CNY] 退货",
  "occurred_at": "2026-04-22"
}
```

---

### `ledger query`

Query transactions by filters. No params returns latest 50.

**All optional:**
- `--from DATE` — Start date (inclusive)
- `--to DATE` — End date (inclusive)
- `--month STRING` — Shorthand (e.g. `2026-04` → `--from 2026-04-01 --to 2026-04-30`)
- `--direction STRING` — `income` | `expense`
- `--category STRING` — Category name
- `--tag STRING` — Tag filter
- `--currency STRING` — Currency filter
- `--min-amount FLOAT` — Minimum amount
- `--max-amount FLOAT` — Maximum amount
- `--limit INT` — Max results (default 50)
- `--offset INT` — Pagination offset

**Output:**
```json
{
  "total": 42,
  "items": [
    {
      "id": "uuid",
      "direction": "expense",
      "amount": 15.0,
      "refund_amount": 0,
      "net_amount": 15.0,
      "currency": "CNY",
      "category": "餐饮",
      "description": "午餐，牛肉面",
      "tags": ["工作日"],
      "transfer_group": null,
      "occurred_at": "2026-04-22"
    }
  ]
}
```

---

### `ledger search`

Keyword search.

**Required:**
- `--keyword STRING` — Keyword search (FTS5)

**Optional:**
- `--limit INT` — Max results (`0` = unlimited)

Notes:
- `ledger config` has been removed.
- `--semantic`, `--mode`, and `--threshold` have been removed.
- Search JSON output no longer contains `match_type`.

**Output:**
```json
{
  "items": [
    {
      "id": "uuid",
      "score": 0.92,
      "direction": "expense",
      "amount": 200.0,
      "currency": "CNY",
      "category": "餐饮",
      "description": "和同事吃火锅",
      "occurred_at": "2026-04-18"
    }
  ]
}
```

---

### `ledger balance`

Balance per currency. Expense rows use net amount (`amount - refund_amount`), so refunds reduce spending without being counted as ordinary income.

**Optional:**
- `--currency STRING` — Filter to one currency (default all)

**Output:**
```json
{
  "balances": [
    { "currency": "CNY", "balance": 15234.0 },
    { "currency": "USD", "balance": 500.0 }
  ]
}
```

---

### `ledger transfer`

Currency exchange. Creates two linked transactions (one expense, one income).

**Required:**
- `--from-currency STRING` — Source currency
- `--to-currency STRING` — Target currency
- `--from-amount FLOAT` — Source amount
- `--to-amount FLOAT` — Target amount

**Optional:**
- `--date STRING` — Date (default today)
- `--note STRING`

**Output:**
```json
{
  "transfer_group": "tf-uuid",
  "expense": { "id": "uuid", "amount": 100.0, "currency": "USD" },
  "income":  { "id": "uuid", "amount": 720.0, "currency": "CNY" }
}
```

---

### `ledger category list|add|remove`

**`list`** — List all categories (tree structure)

**`add`:**
- Required: `--name STRING`, `--direction income|expense|both`
- Optional: `--parent STRING`, `--icon STRING`

**`remove`:**
- Required: `--name STRING`
- Optional: `--force` (force delete even if transactions reference it)

---

### `ledger tag list|add|remove`

**`list`** — List all tags

**`add`:** Required: `--name STRING`

**`remove`:** Required: `--name STRING`

---

### `ledger audit`

View audit log.

**All optional:**
- `--action STRING` — Filter by action type
- `--from DATE` — Start time
- `--to DATE` — End time
- `--limit INT` — Max entries (default 20)

**Output:**
```json
{
  "items": [
    {
      "id": "uuid",
      "action": "add_transaction",
      "target_type": "transaction",
      "target_id": "uuid",
      "agent_id": "agent-xxx",
      "input_summary": "记一笔午餐",
      "detail": {},
      "created_at": "2026-04-22T12:00:00Z"
    }
  ]
}
```

---

## Phase Mapping

| Phase | Commands |
|-------|----------|
| Phase 1 | init, add, delete, update, query, balance |
| Phase 2 | search |
| Phase 3 | transfer, category, tag, audit |
| Phase 4 | SKILL.md, example, Makefile, cross-compile |

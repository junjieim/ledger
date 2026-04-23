# Example: Update

User intent:
`把刚才那笔午餐改成 35 元，备注改成加了饮料。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db update \
  --id <TRANSACTION_ID> \
  --amount 35 \
  --note "加了饮料" \
  --json
```

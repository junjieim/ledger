# Example: Add Expense

User intent:
`帮我记一笔，今天午饭花了 32 块，吃的是猪脚饭。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db add \
  --amount 32 \
  --direction expense \
  --category 餐饮 \
  --description "午饭猪脚饭" \
  --raw-input "今天午饭花了 32 块，吃的是猪脚饭" \
  --json
```

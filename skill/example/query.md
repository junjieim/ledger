# Example: Query

User intent:
`查一下我这个月的餐饮支出。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db query \
  --month 2026-04 \
  --category 餐饮 \
  --direction expense \
  --json
```

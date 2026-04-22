# Example: Transfer

User intent:
`记一下我把 100 美元换成了 720 人民币。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db transfer \
  --from-currency USD \
  --to-currency CNY \
  --from-amount 100 \
  --to-amount 720 \
  --note "换汇" \
  --json
```

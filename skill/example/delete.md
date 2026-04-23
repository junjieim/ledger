# Example: Delete

User intent:
`把刚才误记的那笔删掉。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db delete \
  --id <TRANSACTION_ID> \
  --json
```

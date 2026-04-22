# Example: Search

User intent:
`找一下之前和同事聚餐的记录。`

Suggested command:
```bash
script/ledger --db ./data/ledger.db search \
  --keyword 聚餐 \
  --semantic "和同事一起吃饭" \
  --mode hybrid \
  --json
```

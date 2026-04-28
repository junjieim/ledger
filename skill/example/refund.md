# Example: Refund

User intent:
`昨天淘宝买的那笔 100 块退了 25 回来。`

Suggested commands:
```bash
# First, find the original transaction id.
script/ledger --db ./data/ledger.db query \
  --category 购物 \
  --limit 5 \
  --json

# Then refund 25 against it.
script/ledger --db ./data/ledger.db refund \
  --id <ORIG_TRANSACTION_ID> \
  --amount 25 \
  --note "退货" \
  --json
```

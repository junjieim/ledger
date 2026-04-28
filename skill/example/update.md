# Example: Update

User intent:
`把刚才那笔午餐改成 35 元，备注改成加了饮料。`

Suggested command:
```bash
script/ledger update \
  --id <TRANSACTION_ID> \
  --amount 35 \
  --note "加了饮料" \
  --json
```

User intent:
`把那笔高铁票改成交通分类，再补上标签：广州、高铁、两人。`

Suggested command:
```bash
script/ledger update \
  --id <TRANSACTION_ID> \
  --category 交通 \
  --tag 广州 \
  --tag 高铁 \
  --tag 两人 \
  --json
```

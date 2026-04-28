# Example: Add Expense

User intent:
`我在新疆吃了牛肉饭，花了 150，好贵。`

Suggested command:
```bash
script/ledger add \
  --amount 150 \
  --direction expense \
  --currency CNY \
  --category 餐饮 \
  --description "牛肉饭" \
  --raw-input "我在新疆吃了牛肉饭，花了 150，好贵。" \
  --tag 新疆 \
  --tag 好贵 \
  --json
```

Why:
- `餐饮` is the objective category because the factual event is eating a meal.
- `新疆` is contextual metadata, so it belongs in a tag.
- `好贵` is subjective feeling, so it also belongs in a tag.

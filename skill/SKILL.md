---
name: "ledger"
description: "Use this skill to operate the Ledger CLI for structured personal bookkeeping, transfer tracking, category/tag management, audit inspection, and search."
---

# Ledger Skill

## When To Use
- Record income, expenses, and currency transfers through shell commands.
- Query balances, transactions, categories, tags, or audit history.
- Search ledger records by keyword or semantic intent.
- Package bookkeeping actions as deterministic tool calls instead of free-form text.

## Preconditions
- The binary is available at `script/ledger` inside the packaged skill directory.
- The working database path is usually `./data/ledger.db`.
- Run `ledger init` once before first use.
- Semantic search requires `ZHIPU_API_KEY` in the environment.

## Operating Rules
- Do natural-language understanding outside the CLI.
- Convert the user request into structured command arguments before execution.
- Prefer `--json` whenever another agent or program will parse the result.
- Do not guess destructive operations. Confirm before `delete`, `category remove`, or `tag remove`.
- For transfers, always use `ledger transfer`; do not manually create the two legs with `ledger add`.

## Core Workflow
1. Ensure the database exists: `script/ledger --db ./data/ledger.db init`
2. Convert the request into one CLI command.
3. Execute the command with `--json` when downstream parsing matters.
4. Summarize the result for the user in natural language.

## Command Reference

### Add A Transaction
```bash
script/ledger --db ./data/ledger.db add \
  --amount 28.5 \
  --direction expense \
  --category 餐饮 \
  --description "午餐牛肉面" \
  --raw-input "中午吃了一碗牛肉面花了 28.5" \
  --tag 工作日 \
  --note "公司附近" \
  --json
```

### Query Transactions
```bash
script/ledger --db ./data/ledger.db query \
  --month 2026-04 \
  --category 餐饮 \
  --limit 20 \
  --json
```

### Search
Keyword only:
```bash
script/ledger --db ./data/ledger.db search \
  --keyword 火锅 \
  --json
```

Hybrid:
```bash
script/ledger --db ./data/ledger.db search \
  --keyword 聚餐 \
  --semantic "和同事吃饭" \
  --mode hybrid \
  --json
```

### Transfer
```bash
script/ledger --db ./data/ledger.db transfer \
  --from-currency USD \
  --to-currency CNY \
  --from-amount 100 \
  --to-amount 720 \
  --note "换汇" \
  --json
```

### Category And Tag Management
```bash
script/ledger --db ./data/ledger.db category add --name 差旅 --direction expense --json
script/ledger --db ./data/ledger.db tag add --name 报销 --json
```

### Audit
```bash
script/ledger --db ./data/ledger.db audit --limit 20 --json
```

## Mapping Hints
- "记一笔" / "record" => `add`
- "查一下" / "show me" => `query`
- "搜一下" / "find similar" => `search`
- "换汇" / "transfer between currencies" => `transfer`
- "分类" => `category`
- "标签" => `tag`
- "审计" / "历史操作" => `audit`

## Safety Notes
- `search --semantic` and hybrid semantic mode will fail fast if `ZHIPU_API_KEY` is missing.
- `category remove --force` detaches referenced transactions and child categories first.
- `delete` removes both legs of a transfer automatically when the target transaction belongs to a transfer group.

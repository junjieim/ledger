---
name: "ledger"
description: "Use this skill to operate the Ledger CLI for structured personal bookkeeping, transfer tracking, category/tag management, audit inspection, and search."
---

# Ledger Skill

## Source And Updates
- GitHub repository: `https://github.com/junjieim/ledger`
- Use the repository's releases page to find the latest packaged skill for this machine's OS and CPU architecture.
- When updating an existing local Ledger skill, preserve the local `data/` directory. It contains the user's Ledger database.
- Do not replace the whole installed skill directory with a destructive copy such as `rsync --delete` from an archive root.
- Safe update workflow:
  1. Download and extract the matching release package.
  2. Copy only `SKILL.md`, `example/`, and `script/ledger` into the installed skill directory.
  3. Keep the installed `data/` directory and existing `data/ledger.db*` files unchanged.
  4. Run `script/ledger --db ./data/ledger.db init` only if the database does not already exist.

## When To Use
- Record income, expenses, and currency transfers through shell commands.
- Record full or partial refunds against existing expenses.
- Query balances, transactions, categories, tags, or audit history.
- Search ledger records by keyword.
- Package bookkeeping actions as deterministic tool calls instead of free-form text.

## Preconditions
- The binary is available at `script/ledger` inside the packaged skill directory.
- The working database path is usually `./data/ledger.db`.
- Run `ledger init` once before first use.

## Operating Rules
- Do natural-language understanding outside the CLI.
- Convert the user request into structured command arguments before execution.
- For `add`, fill every argument that can be inferred confidently instead of relying on defaults when possible:
  `--amount`, `--direction`, `--currency`, `--category`, `--date`, `--description`, `--raw-input`, and useful `--tag` / `--note`.
- Write entries for later retrieval, not just for immediate storage. Prefer structured, specific, searchable values.
- Preserve the user's original bill text in `--raw-input` whenever possible.
- Keep `--description` short and factual so similar transactions are easy to scan and search later.
- `分类` (`--category`) is the objective transaction type: what happened in fact, such as `餐饮`, `购物`, `交通`.
- `标签` (`--tag`) is for subjective, contextual, or attached attributes: place, people, platform, occasion, feeling, campaign, reimbursement status, and similar metadata.
- Do not overload category with mood, place, or other side information. Put those into tags or note instead.
- Prefer `--json` whenever another agent or program will parse the result.
- Do not guess destructive operations. Confirm before `delete`, `category remove`, or `tag remove`.
- For transfers, always use `ledger transfer`; do not manually create the two legs with `ledger add`.
- For refunds, always use `ledger refund`; do not manually add an income row.

## Core Workflow
1. Ensure the database exists: `script/ledger --db ./data/ledger.db init`
2. Before `add`, check existing categories and tags first when classification or tagging is not obvious:
   `script/ledger --db ./data/ledger.db category list --json`
   `script/ledger --db ./data/ledger.db tag list --json`
3. Convert the request into one CLI command.
4. Execute the command with `--json` when downstream parsing matters.
5. Summarize the result for the user in natural language.

## Add Guidance
- Prefer reusing existing categories and tags before creating or inferring new ones.
- Query the current category/tag lists first if there is any ambiguity, so the ledger stays consistent over time.
- Start from the objective fact and choose one category for that fact.
- Then extract searchable attributes into tags.
- If the user expresses a feeling, judgment, or side context, prefer tags or note instead of category.
- If the date or currency is known from the message or conversation context, pass it explicitly.

Example interpretation:
- User input: `我在新疆吃了牛肉饭，花了 150，好贵`
- Objective fact: eating a meal => category `餐饮`
- Attribute / context: `新疆` => tag
- Subjective feeling: `好贵` => tag
- Good `add` command:

```bash
script/ledger --db ./data/ledger.db add \
  --amount 150 \
  --direction expense \
  --currency CNY \
  --category 餐饮 \
  --description "牛肉饭" \
  --raw-input "我在新疆吃了牛肉饭，花了 150，好贵" \
  --tag 新疆 \
  --tag 好贵 \
  --json
```

## Command Reference

### Add A Transaction
Prefer the fullest deterministic command you can infer.
When useful, inspect existing categories/tags before writing:

```bash
script/ledger --db ./data/ledger.db category list --json
script/ledger --db ./data/ledger.db tag list --json
```

```bash
script/ledger --db ./data/ledger.db add \
  --amount 28.5 \
  --direction expense \
  --currency CNY \
  --category 餐饮 \
  --date 2026-04-23 \
  --description "午餐牛肉面" \
  --raw-input "中午吃了一碗牛肉面花了 28.5" \
  --tag 午餐 \
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

### Update Existing Transactions
```bash
script/ledger --db ./data/ledger.db update \
  --id <TRANSACTION_ID> \
  --category 交通 \
  --add-tag 高铁 \
  --add-tag 广州 \
  --remove-tag 临时标签 \
  --json
```

### Refund An Existing Expense
Omit `--amount` to refund the remaining balance. Refunds accumulate on the original expense and cannot exceed the original amount.

```bash
script/ledger --db ./data/ledger.db refund \
  --id <ORIG_TRANSACTION_ID> \
  --amount 25 \
  --note "店家漏单退回" \
  --json
```

### Search
```bash
script/ledger --db ./data/ledger.db search \
  --keyword 火锅 \
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
- "退款" / "退一笔" / "refund" => `refund`
- "分类" => `category`
- "标签" => `tag`
- "审计" / "历史操作" => `audit`

## Safety Notes
- Richer, more structured `add` commands improve later query and search quality.
- `refund` accumulates on the original expense; refunding more than the original amount is rejected.
- `category remove --force` detaches referenced transactions and child categories first.
- `delete` removes both legs of a transfer automatically when the target transaction belongs to a transfer group.

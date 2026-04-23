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
- Semantic search requires embedding to be configured through `ledger config set`.
- If embedding configuration is incomplete, the CLI emits a warning on every non-config command run so the agent or user can fill the missing embedding fields.

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

### Configure Embeddings
```bash
script/ledger --db ./data/ledger.db config set \
  --api-key "<your-embedding-api-key>" \
  --model-name embedding-3 \
  --model-url https://open.bigmodel.cn/api/paas/v4/embeddings \
  --dimensions 2048 \
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
- Richer, more structured `add` commands improve later query and search quality.
- `search --semantic` returns an empty result and warning if embedding has not been configured through `ledger config set`.
- Hybrid search with both `--keyword` and `--semantic` degrades to keyword-only when embedding is not configured, and emits a warning explaining that semantic results are omitted.
- The CLI also emits a non-blocking warning on each non-config command run when embedding is not configured.
- `category remove --force` detaches referenced transactions and child categories first.
- `delete` removes both legs of a transfer automatically when the target transaction belongs to a transfer group.

# Architecture & Tech Stack

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go (pure Go, no CGO) | Single binary distribution |
| SQLite | modernc.org/sqlite | Pure-Go SQLite driver, no CGO dependency |
| CLI Framework | cobra | Industry standard |
| Chinese Tokenization | gse | Pure Go, github.com/go-ego/gse |

## Source Code Structure

```
ledger/
├── cmd/ledger/main.go           # CLI entry point
├── internal/
│   ├── db/                      # Connection management, schema (go:embed)
│   ├── model/                   # Data structures
│   ├── repo/                    # CRUD + audit logging
│   ├── search/                  # FTS5 keyword search
│   ├── tokenizer/               # gse tokenization
│   └── cli/                     # Cobra subcommands
├── skill/
│   ├── SKILL.md                 # Skill prompt for AI agents
│   └── example/                 # Usage examples
├── docs/                        # Project documentation
├── go.mod
├── Makefile
└── README.md
```

## Skill Output Structure (Build Artifact)

```
ledger/
├── SKILL.md                     # AI reads this to understand how to use the CLI
├── example/                     # Usage examples
└── script/ledger                # Compiled single binary
```

## Layered Design

```
External AI Agent (natural language → structured params)
    ↓
SKILL.md guides Agent on which command to call
    ↓
cmd/ledger + internal/cli — CLI parsing, param validation
    ↓
internal/repo — Business logic, CRUD, automatic audit logging
    ↓
internal/db — SQLite connection + transactions
    ↓
internal/search + tokenizer — Search pipeline
```

## Current Search Implementation

- Keyword search uses a dedicated FTS5 table rebuilt from gse-tokenized transaction text.
- The keyword index is synchronized from current transaction text and removes stale entries for deleted transactions.
- Semantic, hybrid, and embedding-backed search paths were rolled back; current search is keyword-only.

## Refund Handling

- Refunds use a single-column net model for family bookkeeping: `transactions.refund_amount` stores the cumulative refunded amount on the original expense row.
- No separate refund transaction row is created. Transaction JSON exposes both `refund_amount` and derived `net_amount = amount - refund_amount`.
- Balance calculations subtract expense net amount, so refunded expenses reduce spending without inflating ordinary income.

## Key Design Decisions

1. **CLI, not MCP** — Simple, cross-platform, any agent that can execute shell commands can use it
2. **Self-built ledger engine** — Full control over schema, no external dependencies on accounting software
3. **No import/export** — Deferred to future phases
4. **Structured input only** — CLI accepts structured params; NLU is the external agent's responsibility
5. **go:embed for schema** — Auto-initialize DB on first run
6. **Repo layer handles audit** — All write operations automatically log to audit_log
7. **--json output** — Default human-readable, --json for agent parsing
8. **DB path** — Default `~/.ledger/ledger.db`, overridable via `--db`

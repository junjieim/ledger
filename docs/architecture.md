# Architecture & Tech Stack

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go (pure Go, no CGO) | Single binary distribution |
| SQLite | ncruces/go-sqlite3 | WASM-based, no CGO dependency |
| Vector Search | sqlite-vec WASM binding | asg017/sqlite-vec-go-bindings/ncruces |
| CLI Framework | cobra | Industry standard |
| Chinese Tokenization | gse | Pure Go, github.com/go-ego/gse |
| Embedding | Zhipu embedding-3 | 2048 dimensions, HTTP API |

## Source Code Structure

```
ledger/
├── cmd/ledger/main.go           # CLI entry point
├── internal/
│   ├── db/                      # Connection management, schema (go:embed)
│   ├── model/                   # Data structures
│   ├── repo/                    # CRUD + audit logging
│   ├── search/                  # FTS5 + vec hybrid search
│   ├── embedding/               # Zhipu API client
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
├── script/ledger                # Compiled single binary
└── data/ledger.db               # Created at runtime
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
internal/search + embedding + tokenizer — Search pipeline
```

## Key Design Decisions

1. **CLI, not MCP** — Simple, cross-platform, any agent that can execute shell commands can use it
2. **Self-built ledger engine** — Full control over schema, no external dependencies on accounting software
3. **No import/export** — Deferred to future phases
4. **Structured input only** — CLI accepts structured params; NLU is the external agent's responsibility
5. **go:embed for schema** — Auto-initialize DB on first run
6. **Repo layer handles audit** — All write operations automatically log to audit_log
7. **--json output** — Default human-readable, --json for agent parsing
8. **DB path** — Default `./data/ledger.db`, overridable via `--db` or env var

# Architecture & Tech Stack

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go (pure Go, no CGO) | Single binary distribution |
| SQLite | modernc.org/sqlite | Pure-Go SQLite driver, no CGO dependency |
| Vector Search | Persisted embeddings + Go cosine similarity | Stable Phase 2 baseline; sqlite-vec integration deferred after runtime compatibility issues |
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
│   ├── search/                  # FTS5 + semantic hybrid search
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

## Current Search Implementation

- Keyword search uses a dedicated FTS5 table rebuilt from gse-tokenized transaction text.
- Semantic search stores embedding runtime settings in a DB-backed `embedding_config` table, persists vectors in SQLite as JSON, and computes cosine similarity in Go.
- The embedding cache is keyed by both document hash and embedding config signature (model name, URL, dimensions), so dimension or model changes trigger a clean re-embed path.
- Hybrid search uses reciprocal-rank fusion over keyword and semantic result lists.
- `sqlite-vec` is intentionally deferred for now because the current in-DB vector-search integrations we evaluated are not yet stable enough for this project.

## Key Design Decisions

1. **CLI, not MCP** — Simple, cross-platform, any agent that can execute shell commands can use it
2. **Self-built ledger engine** — Full control over schema, no external dependencies on accounting software
3. **No import/export** — Deferred to future phases
4. **Structured input only** — CLI accepts structured params; NLU is the external agent's responsibility
5. **go:embed for schema** — Auto-initialize DB on first run
6. **Repo layer handles audit** — All write operations automatically log to audit_log
7. **--json output** — Default human-readable, --json for agent parsing
8. **DB path** — Default `./data/ledger.db`, overridable via `--db` or env var

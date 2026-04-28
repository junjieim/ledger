# Development Plan

## Phase Overview

| Phase | Goal | Deliverable |
|-------|------|-------------|
| **Phase 1** | Skeleton + basic CRUD | Working CLI: init, add, delete, update, query, balance |
| **Phase 2** | Search capability | FTS5 keyword search with gse pre-tokenization |
| **Phase 3** | Complete features | transfer, category mgmt, tag mgmt, audit log |
| **Phase 4** | Skill packaging | SKILL.md, examples, Makefile, cross-compile |

## Phase 1: Skeleton + Basic CRUD

**Goal:** `ledger add --amount 1500 --direction expense` → writes to DB → `ledger query` retrieves it

Tasks:
1. Initialize Go project (go.mod, cobra skeleton)
2. `internal/db` — SQLite connection, go:embed schema, auto-init
3. `internal/model` — Data structures (Transaction, Category, Tag, AuditEntry)
4. `internal/repo/transaction.go` — Transaction CRUD
5. `internal/repo/audit.go` — Auto audit logging on all write ops
6. `internal/cli` — Subcommands: init, add, delete, update, query, balance
7. Tests

**Deliverable:** Compilable binary that can record, query, and summarize transactions.

## Phase 2: Search

**Goal:** `ledger search --keyword "火锅"` returns relevant records

Tasks:
1. `internal/tokenizer/gse.go` — gse Chinese tokenization
2. `internal/search` — FTS5 keyword search (pre-tokenized input)
3. `internal/cli/search.go` — search command
4. Tests

**Deliverable:** Chinese keyword search working end-to-end.

Implementation note:
- Current Phase 2 uses FTS5 with gse-pre-tokenized transaction text and query text.
- Semantic / hybrid / embedding search was rolled back on 2026-04-27.

## Phase 3: Complete Features

**Goal:** Full feature set including currency transfer, category/tag management, audit viewing

Tasks:
1. `internal/repo/transaction.go` — Transfer logic (two linked records in one transaction)
2. `internal/cli/transfer.go` — transfer command
3. `internal/repo/category.go` — Category CRUD
4. `internal/cli/category.go` — category list/add/remove
5. `internal/repo/tag.go` — Tag CRUD
6. `internal/cli/tag.go` — tag list/add/remove
7. `internal/cli/audit.go` — audit command
8. Tests

**Deliverable:** Feature-complete CLI.

## Phase 4: Skill Packaging

**Goal:** Distributable skill directory

Tasks:
1. Write `SKILL.md` — Prompt teaching AI how to use the CLI
2. Write `example/` — Natural language → command mapping examples
3. `Makefile` — Build binary + assemble skill directory
4. Cross-compile targets: darwin-arm64, darwin-amd64, linux-amd64
5. End-to-end test: AI agent uses SKILL.md to operate ledger autonomously

**Deliverable:** Ready-to-distribute skill package.

## Process

- Each phase delivers a usable increment
- Review at end of each phase before proceeding
- No phase mixing — complete one before starting the next

## Backlog / To Do

- Investigate a clean path back to stable in-DB vector search on top of the current SQLite driver stack.
- Add unit tests and baseline regression coverage for the Ledger CLI and repo layers.

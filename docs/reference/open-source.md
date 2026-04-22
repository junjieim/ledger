# Ledger Project Reference: Open-Source Foundations

## Scope

This document evaluates lower-level open-source bookkeeping / accounting
projects as possible foundations or references for an agent-facing ledger
system.

The key question is not just “is it good accounting software?” but:

- does it help with an agent-first ledger product?
- does it preserve schema flexibility?
- does it fit semantic retrieval and audit-focused tooling?

## Recommendation Summary

For this project, the best long-term direction is:

- build a self-owned ledger engine
- use mature open-source projects as semantic references, compatibility targets,
  and inspiration
- avoid overcommitting to a heavy prebuilt accounting product as the true core

If a fast MVP foundation is needed, `Firefly III` is the most practical
short-term wrap target from this set.

## Category 1: Ledger-Kernel-Style References

These are the most useful if the goal is to learn from mature accounting
semantics without giving up control of the final schema.

### Beancount

- Repo: <https://github.com/beancount/beancount>
- Type: plain-text double-entry accounting system
- Why it matters:
  - mature accounting semantics
  - Python-friendly ecosystem
  - useful as a correctness and import/export reference
- Why not use as the main database:
  - not designed as a structured relational backend for semantic retrieval,
    audit-log-first agent tooling, or embedded search layers

### Fava

- Repo: <https://github.com/beancount/fava>
- Docs: <https://beancount.github.io/fava/>
- Type: web interface for Beancount
- Why it matters:
  - useful reference for ledger browsing, querying, and report presentation
- Best use in this project:
  - UI / exploration reference, not core backend

### hledger

- Repo: <https://github.com/simonmichael/hledger>
- Site: <https://hledger.org/>
- Type: plain-text accounting toolkit with strong CLI / TUI / web tooling
- Why it matters:
  - mature reporting and bookkeeping workflows
  - useful semantic reference
- Best use in this project:
  - import/export and accounting behavior reference, not central storage

### Ledger CLI

- Repo: <https://github.com/ledger/ledger>
- Site: <https://ledger-cli.org/>
- Type: long-standing plain-text double-entry accounting system
- Why it matters:
  - mature and respected accounting semantics
- Best use in this project:
  - reference semantics and compatibility direction, not the primary engine

## Category 2: Systems That Can Be Wrapped for a Faster MVP

These are not ideal long-term cores for the current architecture goals, but they
can shorten time-to-first-demo.

### Firefly III

- Repo: <https://github.com/firefly-iii/firefly-iii>
- Docs: <https://docs.firefly-iii.org/>
- API: <https://docs.firefly-iii.org/how-to/firefly-iii/features/api/>
- Webhooks: <https://docs.firefly-iii.org/how-to/firefly-iii/features/webhooks/>
- Search: <https://docs.firefly-iii.org/how-to/firefly-iii/features/search/>
- Why it matters:
  - self-hostable web app
  - has APIs, webhooks, and search
  - relatively practical to wrap with an agent layer
- Best use in this project:
  - short-term MVP or interaction prototype
- Main downside:
  - schema and internal model are not purpose-built for the target system’s
    audit / semantic-retrieval goals

### ERPNext

- Repo: <https://github.com/frappe/erpnext>
- Type: full ERP with accounting modules
- Why it matters:
  - broad, mature open-source business system
  - very complete finance feature set
- Main downside:
  - likely too heavy for the current project stage
  - less suitable if the goal is a focused, agent-first, schema-controlled
    ledger product

### Akaunting

- Repo: <https://github.com/akaunting/akaunting>
- Type: online accounting system
- Why it matters:
  - lighter than a full ERP in some respects
  - more product-ready than kernel tools
- Main downside:
  - still more “complete accounting app” than “clean programmable ledger core”
  - license model requires attention for productization planning

## Category 3: Mature but Poor Fit as a Core

### GnuCash

- Repo: <https://github.com/Gnucash/gnucash>
- Type: mature desktop accounting software
- Why it matters:
  - highly established accounting application
- Why it is not a strong fit:
  - desktop-first product shape
  - relatively awkward fit for agent-first API / tool / search / embedding work

## What to Take Forward

### Best long-term pattern

- self-built ledger engine
- own schema
- own audit model
- own retrieval layer
- selective compatibility with mature accounting ecosystems later

### Best reference value from open source

Use Beancount / hledger / Ledger CLI for:

- accounting semantics
- migration compatibility direction
- regression-test reference cases

Use Firefly III for:

- API ideas
- webhook ideas
- search interaction patterns
- fast MVP wrapping if ever needed

## Recommendation for This Project

### Long-term

Build the core ledger yourselves.

### Near-term

Do not start with a full wrap around an existing accounting product unless the
goal changes from “agent-first ledger engine” to “fast demo on top of an
existing app.”

### If the team wants a fallback prototype path

`Firefly III` is the most practical candidate to wrap temporarily.

## Notes

- This document evaluates foundations, not market competitors.
- Product-level competitor references are documented in
  `docs/reference/competitors.md`.

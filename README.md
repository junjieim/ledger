# Ledger

AI Native personal and family bookkeeping.

Ledger is built for a different workflow: you talk to an AI agent, and the
agent uses Ledger as a deterministic local tool. Humans describe bills,
refunds, transfers, and questions in natural language; Ledger stores the
structured records, search index, and audit history.

## Why

- **Agent-first**: designed for tool calls from coding agents, not manual UI workflows.
- **Natural language outside, structured data inside**: the agent interprets intent; Ledger records explicit amounts, categories, tags, dates, and notes.
- **Local-first**: a single Go binary plus SQLite database. No hosted service is required.
- **Auditable**: every write is recorded in operation history, so agent actions stay reviewable.
- **Practical family finance**: expenses, income, transfers, categories, tags, keyword search, and refund netting.
- **Skill-ready**: release packages include the CLI and an agent skill with usage examples.

## Use It Through Your Agent

Copy this to your coding agent to install Ledger:

```text
Install Ledger from https://github.com/junjieim/ledger/releases/latest.
Find the release asset that matches my operating system and CPU architecture,
install the Ledger CLI and agent skill locally, initialize the local Ledger
database, and tell me where the database and skill were installed.
```

After installation, use normal language:

```text
Record an expense: I spent 150 CNY on beef rice for lunch today. It felt expensive.

Show me how much I spent on food this month.

Record a 25 CNY refund against yesterday's 100 CNY Taobao purchase.
```

Your agent should use Ledger commands behind the scenes and summarize the
result for you. Amounts use major currency units, so `150` means `¥150`, not
cents.

## For Agent Developers

Ledger release packages contain:

- `script/ledger`: the CLI binary
- `SKILL.md`: the agent operating guide
- `example/`: command examples
- `data/`: the runtime database directory

## More

- Architecture: `docs/architecture.md`
- CLI contract: `docs/cli-contract.md`
- Schema: `docs/schema-v1.sql`

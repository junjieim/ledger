-- Ledger Schema V1 (embedded, used by go:embed)
-- PRAGMAs are set in Go code, not here.

CREATE TABLE IF NOT EXISTS categories (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  parent_id   TEXT,
  direction   TEXT CHECK(direction IN ('income', 'expense', 'both')),
  icon        TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (parent_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);

CREATE TABLE IF NOT EXISTS transactions (
  id              TEXT PRIMARY KEY,
  direction       TEXT NOT NULL CHECK(direction IN ('income', 'expense')),
  amount          REAL NOT NULL CHECK(amount > 0),
  refund_amount   REAL NOT NULL DEFAULT 0 CHECK(refund_amount >= 0 AND refund_amount <= amount),
  currency        TEXT NOT NULL DEFAULT 'CNY' CHECK(currency GLOB '[A-Z][A-Z][A-Z]'),
  transfer_group  TEXT,
  category_id     TEXT,
  description     TEXT,
  raw_input       TEXT,
  note            TEXT,
  occurred_at     TEXT NOT NULL,
  created_at      TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_transactions_direction      ON transactions(direction);
CREATE INDEX IF NOT EXISTS idx_transactions_currency       ON transactions(currency);
CREATE INDEX IF NOT EXISTS idx_transactions_category       ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_occurred_at    ON transactions(occurred_at);
CREATE INDEX IF NOT EXISTS idx_transactions_transfer_group ON transactions(transfer_group);

CREATE TABLE IF NOT EXISTS tags (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS transaction_tags (
  transaction_id TEXT NOT NULL,
  tag_id         TEXT NOT NULL,
  PRIMARY KEY (transaction_id, tag_id),
  FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id)         REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS audit_log (
  id            TEXT PRIMARY KEY,
  action        TEXT NOT NULL,
  target_type   TEXT,
  target_id     TEXT,
  agent_id      TEXT,
  input_summary TEXT,
  detail        TEXT,
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_audit_log_target  ON audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_agent   ON audit_log(agent_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at);

-- Seed default categories (ignore if already exist)
INSERT OR IGNORE INTO categories (id, name, direction) VALUES
  ('cat-food',       '餐饮', 'expense'),
  ('cat-transport',  '交通', 'expense'),
  ('cat-shopping',   '购物', 'expense'),
  ('cat-housing',    '住房', 'expense'),
  ('cat-entertain',  '娱乐', 'expense'),
  ('cat-health',     '医疗', 'expense'),
  ('cat-parenting',  '育儿', 'expense'),
  ('cat-salary',     '工资', 'income'),
  ('cat-investment', '投资收益', 'income'),
  ('cat-freelance',  '兼职', 'income'),
  ('cat-gift',       '礼金', 'both'),
  ('cat-social',     '人情', 'both'),
  ('cat-other',      '其他', 'both');

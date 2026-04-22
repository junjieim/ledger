-- ==========================================
-- Ledger Schema V1 (Finalized 2026-04-22)
-- SQLite + sqlite-vec + FTS5 (gse pre-tokenized)
-- ==========================================

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- -------------------------------------------
-- 1. 分类（支持层级）
-- -------------------------------------------
CREATE TABLE categories (
  id          TEXT PRIMARY KEY,            -- UUID
  name        TEXT NOT NULL,
  parent_id   TEXT,                        -- 上级分类，NULL 为顶级
  direction   TEXT CHECK(direction IN ('income', 'expense', 'both')),
  icon        TEXT,                        -- 可选，emoji 或 icon name
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (parent_id) REFERENCES categories(id)
);

CREATE INDEX idx_categories_parent ON categories(parent_id);

-- -------------------------------------------
-- 2. 交易记录
-- -------------------------------------------
CREATE TABLE transactions (
  id              TEXT PRIMARY KEY,           -- UUID
  direction       TEXT NOT NULL CHECK(direction IN ('income', 'expense')),
  amount          REAL NOT NULL CHECK(amount > 0),     -- 实际金额（如 15.5）
  currency        TEXT NOT NULL DEFAULT 'CNY'
                  CHECK(currency GLOB '[A-Z][A-Z][A-Z]'),  -- ISO 4217
  transfer_group  TEXT,              -- 非 NULL 时表示这是 transfer 的一半
  category_id     TEXT,
  description     TEXT,              -- 结构化描述
  raw_input       TEXT,              -- 用户原始自然语言输入
  note            TEXT,              -- 补充备注
  occurred_at     TEXT NOT NULL,     -- 交易发生时间 ISO8601
  created_at      TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE INDEX idx_transactions_direction      ON transactions(direction);
CREATE INDEX idx_transactions_currency       ON transactions(currency);
CREATE INDEX idx_transactions_category       ON transactions(category_id);
CREATE INDEX idx_transactions_occurred_at    ON transactions(occurred_at);
CREATE INDEX idx_transactions_transfer_group ON transactions(transfer_group);

-- -------------------------------------------
-- 3. 标签（灵活打标）
-- -------------------------------------------
CREATE TABLE tags (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE transaction_tags (
  transaction_id TEXT NOT NULL,
  tag_id         TEXT NOT NULL,
  PRIMARY KEY (transaction_id, tag_id),
  FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id)         REFERENCES tags(id) ON DELETE CASCADE
);

-- -------------------------------------------
-- 4. 审计日志
-- -------------------------------------------
CREATE TABLE audit_log (
  id            TEXT PRIMARY KEY,
  action        TEXT NOT NULL,              -- 'add_transaction', 'delete_transaction', etc.
  target_type   TEXT,                       -- 'transaction', 'category', 'tag'
  target_id     TEXT,
  agent_id      TEXT,                       -- 执行操作的 agent
  input_summary TEXT,                       -- 用户原始输入摘要
  detail        TEXT,                       -- JSON，操作前后的 diff 或完整参数
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_audit_log_target  ON audit_log(target_type, target_id);
CREATE INDEX idx_audit_log_agent   ON audit_log(agent_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);

-- -------------------------------------------
-- 5. 全文搜索（FTS5，应用层 gse 预分词）
-- -------------------------------------------
CREATE VIRTUAL TABLE transactions_fts USING fts5(
  description,
  raw_input,
  note,
  content = 'transactions',
  content_rowid = 'rowid'
);

-- FTS 同步触发器
CREATE TRIGGER trg_transactions_ai AFTER INSERT ON transactions BEGIN
  INSERT INTO transactions_fts(rowid, description, raw_input, note)
  VALUES (new.rowid, new.description, new.raw_input, new.note);
END;

CREATE TRIGGER trg_transactions_ad AFTER DELETE ON transactions BEGIN
  INSERT INTO transactions_fts(transactions_fts, rowid, description, raw_input, note)
  VALUES ('delete', old.rowid, old.description, old.raw_input, old.note);
END;

CREATE TRIGGER trg_transactions_au AFTER UPDATE ON transactions BEGIN
  INSERT INTO transactions_fts(transactions_fts, rowid, description, raw_input, note)
  VALUES ('delete', old.rowid, old.description, old.raw_input, old.note);
  INSERT INTO transactions_fts(rowid, description, raw_input, note)
  VALUES (new.rowid, new.description, new.raw_input, new.note);
END;

-- -------------------------------------------
-- 6. 向量搜索（基于 description 字段）（sqlite-vec, 智谱 embedding-3, 2048维）
-- -------------------------------------------
CREATE VIRTUAL TABLE transactions_vec USING vec0(
  transaction_id TEXT PRIMARY KEY,
  embedding      float[2048]
);

-- -------------------------------------------
-- 7. 预置分类
-- -------------------------------------------
INSERT INTO categories (id, name, direction) VALUES
  ('cat-food',       '餐饮', 'expense'),
  ('cat-transport',  '交通', 'expense'),
  ('cat-shopping',   '购物', 'expense'),
  ('cat-housing',    '住房', 'expense'),
  ('cat-entertain',  '娱乐', 'expense'),
  ('cat-health',     '医疗', 'expense'),
  ('cat-parenting',  '育儿', 'expense'),
  ('cat-social',     '人情', 'expense'),
  ('cat-salary',     '工资', 'income'),
  ('cat-investment', '投资收益', 'income'),
  ('cat-freelance',  '兼职', 'income'),
  ('cat-gift',       '礼金', 'both'),
  ('cat-other',      '其他', 'both');

-- -------------------------------------------
-- 常用查询：按币种统计余额
-- -------------------------------------------
-- SELECT currency,
--        SUM(CASE direction WHEN 'income' THEN amount ELSE -amount END) AS balance
-- FROM transactions
-- GROUP BY currency;

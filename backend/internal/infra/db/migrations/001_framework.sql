-- 001_framework.sql
-- CE framework: DB infra for plugins (A6/A12/A13).
-- No CE domain tables — sessions are in-memory (A33), rules on disk.
-- Plugin migrations are delegated via gRPC (A13); this provides the base schema.

CREATE TABLE IF NOT EXISTS _schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

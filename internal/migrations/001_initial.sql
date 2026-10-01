-- +goose Up
-- Create initial tables for microdashboard API

CREATE TABLE IF NOT EXISTS devices (
    device_id TEXT PRIMARY KEY,
    api_key_hash TEXT NOT NULL,
    dashboard_id TEXT,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS dashboards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    json_definition TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    refresh_interval INTEGER NOT NULL DEFAULT 15000,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS latency_monitors (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'GET',
    timeout INTEGER NOT NULL DEFAULT 5000,
    last_elapsed_ms INTEGER,
    last_status INTEGER,
    last_checked INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT,
    target_id TEXT,
    elapsed_ms INTEGER,
    status INTEGER,
    body_bytes INTEGER,
    checked_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS latency_monitors;
DROP TABLE IF EXISTS dashboards;
DROP TABLE IF EXISTS devices;
# Database Schema

## Overview

SQLite database with 5 tables:
- `devices` - IoT device registry
- `dashboards` - Dashboard definitions (JSON)
- `latency_monitors` - HTTP latency check targets
- `metrics` - Time-series metric data
- `alert_config` - Alert configuration (future)

## Schema

### devices

```sql
CREATE TABLE devices (
    device_id TEXT PRIMARY KEY,        -- Unique device identifier
    api_key_hash TEXT NOT NULL,        -- Device API key (stored as-is)
    dashboard_id TEXT,                 -- Assigned dashboard ID (FK to dashboards.id)
    created_at INTEGER NOT NULL        -- Unix timestamp
);
```

**Indexes**: Primary key on `device_id`

**Example**:
```sql
INSERT INTO devices (device_id, api_key_hash, dashboard_id, created_at)
VALUES ('esp-001', 'secret-key-123', 'dash-main', 1790888060);
```

---

### dashboards

```sql
CREATE TABLE dashboards (
    id TEXT PRIMARY KEY,               -- Dashboard UUID/ID
    name TEXT NOT NULL,                -- Human-readable name
    json_definition TEXT NOT NULL,     -- Full dashboard JSON (widgets, layout)
    width INTEGER NOT NULL,            -- Screen width in pixels
    height INTEGER NOT NULL,           -- Screen height in pixels
    refresh_interval INTEGER NOT NULL DEFAULT 15000,  -- Widget refresh ms
    created_at INTEGER NOT NULL        -- Unix timestamp
);
```

**Indexes**: Primary key on `id`

**json_definition structure**:
```json
{
  "id": "dash1",
  "name": "Server Status",
  "width": 128,
  "height": 64,
  "refresh_interval": 15000,
  "widgets": [
    {
      "id": "w1",
      "type": "gauge",
      "x": 0, "y": 0, "width": 32, "height": 16,
      "expression": "${latency:api-server}",
      "unit": "ms",
      "options": {"min": 0, "max": 1000}
    }
  ]
}
```

---

### latency_monitors

```sql
CREATE TABLE latency_monitors (
    id TEXT PRIMARY KEY,               -- Monitor identifier
    url TEXT NOT NULL,                 -- Target URL
    method TEXT NOT NULL DEFAULT 'GET', -- HTTP method
    timeout INTEGER NOT NULL DEFAULT 5000,  -- Timeout in ms
    last_elapsed_ms INTEGER,           -- Last check latency (ms)
    last_status INTEGER,               -- Last HTTP status code
    last_checked INTEGER NOT NULL      -- Unix timestamp of last check
);
```

**Indexes**: Primary key on `id`

**Example**:
```sql
INSERT INTO latency_monitors (id, url, method, timeout, last_checked)
VALUES ('api-health', 'https://api.example.com/health', 'GET', 5000, 1790888060);
```

---

### metrics

```sql
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Auto-incrementing ID
    device_id TEXT,                        -- Device that generated metric (optional)
    target_id TEXT,                        -- Monitor ID that was checked
    elapsed_ms INTEGER,                    -- Latency in milliseconds
    status INTEGER,                        -- HTTP status code
    body_bytes INTEGER,                    -- Response body size in bytes
    checked_at INTEGER NOT NULL            -- Unix timestamp
);
```

**Indexes**: Primary key on `id`

**Note**: `device_id` is empty for monitor-generated metrics, populated for device-pushed metrics.

---

### alert_config (Future)

```sql
CREATE TABLE alert_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_url TEXT,                    -- Webhook URL for alerts
    mqtt_topic TEXT DEFAULT 'md/alerts', -- MQTT topic for alerts
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

---

## Relationships

```
devices.dashboard_id  ──────► dashboards.id
latency_monitors.id    ──────► metrics.target_id (many)
devices.device_id      ──────► metrics.device_id (many)
```

**Note**: No foreign key constraints enforced (SQLite default). Application enforces referential integrity.

---

## Migrations

Migrations run automatically on startup via `store.Migrate()`.

### Migration History

| Version | Description |
|---------|-------------|
| 001 | Initial schema (devices, dashboards, latency_monitors, metrics) |

### Adding Migrations

1. Add new SQL file to `internal/migrations/`
2. Update `store.Migrate()` to include new migration
3. Use `COALESCE` for nullable columns in queries

## Query Patterns

### List Dashboards with Widget Count
```sql
SELECT id, name, json_definition, width, height, refresh_interval, created_at
FROM dashboards
ORDER BY created_at DESC
```

### Get Latest Metric for Device
```sql
SELECT * FROM metrics
WHERE device_id = ?
ORDER BY checked_at DESC
LIMIT 1
```

### Get Monitor with Latest Status
```sql
SELECT id, url, method, timeout,
       COALESCE(last_elapsed_ms, 0) as last_elapsed_ms,
       COALESCE(last_status, 0) as last_status,
       last_checked
FROM latency_monitors
ORDER BY last_checked DESC
```

### Cleanup Old Metrics (Retention)
```sql
DELETE FROM metrics
WHERE checked_at < strftime('%s', 'now', '-30 days')
```

---

## Backup Strategy

### Full Backup
```bash
sqlite3 microdashboard.db ".backup backup_$(date +%Y%m%d).db"
```

### Incremental (WAL Mode)
```bash
# Enable WAL for better concurrency
sqlite3 microdashboard.db "PRAGMA journal_mode=WAL;"

# Backup without locking
sqlite3 microdashboard.db ".backup backup.db"
```

---

## Performance Tuning

### Indexes (Current)
- Primary keys only

### Recommended Additional Indexes
```sql
-- For metrics queries by device
CREATE INDEX idx_metrics_device_checked ON metrics(device_id, checked_at DESC);

-- For metrics by target
CREATE INDEX idx_metrics_target_checked ON metrics(target_id, checked_at DESC);

-- For monitors by last_checked
CREATE INDEX idx_monitors_last_checked ON latency_monitors(last_checked DESC);
```

---

## Data Retention

### Metrics
- Default: 30 days
- Configurable via cleanup job

### Devices/Dashboards/Monitors
- Retained indefinitely
- Soft delete via `deleted_at` column (future)

---

## Migration to PostgreSQL

For production scaling, consider migrating to PostgreSQL/TimescaleDB:

1. Export schema: `sqlite3 microdashboard.db .schema > schema.sql`
2. Adjust types: `INTEGER` → `BIGINT`, `TEXT` → `VARCHAR`
3. Add foreign keys
4. Use TimescaleDB hypertable for `metrics`
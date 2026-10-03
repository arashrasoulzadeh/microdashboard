# Architecture Decision Records

## ADR-001: Go as Primary Language

**Date**: 2026-10-01
**Status**: Accepted

### Context
Need a language for the backend API server that:
- Compiles to single binary for easy deployment
- Good concurrency for background monitoring
- Low memory footprint for container deployment
- Good SQLite support
- Fast HTTP performance

### Decision
Use **Go 1.22+** as the primary language.

### Consequences
- ✅ Single binary deployment
- ✅ Excellent concurrency with goroutines
- ✅ Low memory (~10-20MB RSS)
- ✅ Native SQLite with `mattn/go-sqlite3`
- ✅ Fast HTTP with `julienschmidt/httprouter`
- ❌ CGO required for SQLite (cross-compilation complexity)

---

## ADR-002: SQLite for Data Storage

**Date**: 2026-10-01
**Status**: Accepted

### Context
Need a database for:
- Device registry
- Dashboard definitions
- Latency monitor configs
- Metric time-series data

### Decision
Use **SQLite** (via `mattn/go-sqlite3`) as the primary database.

### Consequences
- ✅ Zero-configuration, single file
- ✅ No separate database server needed
- ✅ ACID compliant
- ✅ Good for <100GB data
- ✅ Easy backup (copy file)
- ❌ Limited concurrent writes (fine for our use case)
- ❌ No horizontal scaling (single-node only)

---

## ADR-003: HTMX for Management UI

**Date**: 2026-10-02
**Status**: Accepted

### Context
Need a management UI for:
- Dashboard creation/editing
- Device management
- Monitor configuration
- Alert settings

### Decision
Use **HTMX** with server-side rendering instead of SPA framework.

### Consequences
- ✅ No build step (no npm, webpack, etc.)
- ✅ Works with Go templates
- ✅ Small footprint (~10KB gzipped)
- ✅ Progressive enhancement
- ✅ Works without JavaScript (mostly)
- ❌ Less interactive than React/Vue
- ❌ Server must render HTML fragments

---

## ADR-004: SQLite JSON for Dashboard Definitions

**Date**: 2026-10-02
**Status**: Accepted

### Context
Dashboards have complex nested structure (widgets array with varying properties).

### Decision
Store dashboard definitions as **JSON text** in SQLite.

### Consequences
- ✅ Flexible schema (no migrations for widget changes)
- ✅ Easy to query with SQLite JSON functions
- ✅ Versionable (store full history if needed)
- ❌ No foreign key constraints on widget IDs
- ❌ Application-level validation required

---

## ADR-005: Per-Device Static API Keys

**Date**: 2026-10-01
**Status**: Accepted

### Context
IoT devices need to authenticate with the API.

### Decision
Use **per-device static API keys** stored in database.

### Consequences
- ✅ Simple for IoT devices (no OAuth flow)
- ✅ Keys can be rotated via admin API
- ✅ Keys shown only once at creation
- ✅ Keys stored as plain text (fast lookup)
- ⚠️ Keys visible in database (mitigate with encryption at rest if needed)

---

## ADR-006: Background Monitor with 30s Interval

**Date**: 2026-10-01
**Status**: Accepted

### Context
Need to periodically check latency targets.

### Decision
Run background goroutine with **30-second interval**.

### Consequences
- ✅ Near-real-time latency data
- ✅ Low resource usage (single HTTP client)
- ✅ Automatic retry on failure
- ⚠️ 30s may be too frequent for many targets (configurable per target in future)

---

## ADR-007: Expression-Based Widget Data Binding

**Date**: 2026-10-02
**Status**: Accepted

### Context
Widgets need to display data from various sources (latency monitors, device metrics).

### Decision
Use **expression syntax** `${source:target}` in widget definitions.

### Consequences
- ✅ Flexible data binding
- ✅ Template replacement for text widgets
- ✅ Easy to extend with new sources
- ✅ Server-side evaluation (secure)
- ❌ Limited to predefined sources (latency, metric)

---

## ADR-008: Arduino SDK for ESP8266/ESP32

**Date**: 2026-10-03
**Status**: Accepted

### Context
IoT devices (ESP8266/ESP32) need to fetch and render dashboards.

### Decision
Provide **Arduino library** (`MicroDashboard`) with:
- HTTP/HTTPS client
- Dashboard fetch and render
- Widget rendering stubs for displays

### Consequences
- ✅ Works on ESP8266 (80KB RAM) and ESP32
- ✅ HTTPS with optional insecure mode
- ✅ Display-agnostic (Adafruit_GFX, U8g2, TFT_eSPI)
- ⚠️ TLS on ESP8266 uses significant RAM (~30KB)

---

## ADR-009: HTTP/HTTPS with Let's Encrypt

**Date**: 2026-10-03
**Status**: Accepted

### Context
Need TLS for production deployments.

### Decision
Support **Let's Encrypt** for automatic TLS certificates.

### Consequences
- ✅ Free, trusted certificates
- ✅ Auto-renewal
- ✅ Works with public domains
- ⚠️ Requires public domain with DNS
- ⚠️ Self-signed for local development

---

## ADR-010: Test Structure - External Packages

**Date**: 2026-10-03
**Status**: Accepted

### Context
Need to test internal packages with good coverage.

### Decision
Move tests to `tests/unit/` as **external test packages** (package `xxx_test`).

### Consequences
- ✅ Tests only use public API
- ✅ Better encapsulation testing
- ✅ Can use `go test -coverpkg=./internal/...`
- ❌ Cannot test unexported functions directly
- ✅ Forces good public API design
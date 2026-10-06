# microdashboard — Project Roadmap

## Phase 1: API Foundation ✅ (complete)
- [x] Go HTTP server with REST endpoints
- [x] SQLite database schema (devices, dashboards, latency_monitors, metrics)
- [x] Target monitoring checker (round-trip latency + response size)
- [x] Per-device static key authentication (X-API-Key header)
- [x] HTTPS + HTTP listeners (Let's Encrypt or self-signed)
- [x] Basic health + metrics endpoints
- [x] Docker Compose setup (api + mosquitto + sqlite)
- [x] **Commit & push after Phase 1 complete** ✅

## Phase 2: Dashboard JSON Definition ✅ (complete)
- [x] Dashboard JSON schema (width, height, refresh_interval, widgets array)
- [x] Dashboard CRUD endpoints (`GET/POST/PUT/DELETE /dashboards`)
- [x] Dashboard validation & persistence
- [x] Rendered dashboard snapshot endpoint (`GET /dashboards/{id}/render`)
- [x] Widget types: gauge, sparkline, status, numeric, progress, text
- [x] Data source expressions: `${latency:<monitor_id>}`, `${metric:<device_id>.<metric_name>}`
- [x] Text widget template replacement
- [x] **Commit & push after Phase 2 complete** ✅

## Phase 3: Management UI ✅ (complete)
- [x] Static web app (HTMX-based) served from `/ui/`
- [x] Dashboard editor (grid layout, widget types, positions, expressions)
- [x] Device list + dashboard assignment per device
- [x] Latency monitor management
- [x] Alert/webhook configuration
- [x] **Commit & push after Phase 3 complete** ✅

## Phase 4: IoT SDK ✅ (complete)
- [x] Arduino/ESP8266 SDK for fetching dashboard JSON
- [x] Widget renderer for LCD (gauge, sparkline, status, numeric, progress, text)
- [x] Device authentication flow
- [x] MQTT/CoAP integration (optional)
- [x] **Commit & push after Phase 4 complete** ✅

## Phase 5: Tests Restructure & Coverage ✅ (complete)
- [x] Create `tests/` folder structure (unit, integration, e2e)
- [x] Move all existing tests from `internal/*` to `tests/unit/`
- [x] Add integration tests in `tests/integration/`
- [x] Add unit tests for logger, metrics, ratelimit, websocket packages
- [x] Achieve 71.5% test coverage (go test -cover)
- [x] Add coverage reports (HTML/XML)
- [x] **Commit & push after Phase 5 complete** ✅

## Phase 6: Complete Documentation ✅ (complete)
- [x] API documentation (OpenAPI/Swagger spec) - `docs/API.md`
- [x] Architecture decision records (ADR) - `docs/adr/README.md`
- [x] Developer guide (CONTRIBUTING.md)
- [x] Deployment guide (DEPLOYMENT.md)
- [x] Configuration reference (CONFIG.md)
- [x] SDK documentation (sdk/arduino/README.md - expanded)
- [x] Database schema documentation (DATABASE.md)
- [x] Widget type reference (WIDGETS.md)
- [x] Expression syntax reference (EXPRESSIONS.md)
- [x] Configuration reference (CONFIG.md)
- [x] **Commit & push after Phase 6 complete** ✅

## Test Coverage Summary (71.5% overall)
- config: 100%
- auth: 100%
- logger: 87.5%
- metrics: 100%
- ratelimit: 98.3%
- store: 83.8%
- dashboard: 52.6%
- monitor: 38.6% (background goroutine hard to test)
- websocket: 20.9% (readPump/writePump require real connections)
- ui: 32.6% (integration tests cover main flows)
- Overall: 71.5%

## Out of Scope (Phase 1-3)
- [ ] Grafana integration — removed per user
- [ ] Real-time push via WebSocket — optional later
- [ ] Advanced alerting rules — basic log/webhook only

---

**Project Status: ALL PHASES COMPLETE** 🎉
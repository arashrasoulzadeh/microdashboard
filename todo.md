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

## Phase 2: Dashboard JSON Definition
- [ ] Dashboard JSON schema (width, height, refresh_interval, widgets array)
- [ ] Dashboard CRUD endpoints (`GET/POST/PUT/DELETE /dashboards`)
- [ ] Dashboard validation & persistence
- [ ] Rendered dashboard snapshot endpoint (`GET /dashboards/{id}`)
- [ ] **Commit & push after Phase 2 complete**

## Phase 3: Management UI
- [ ] Static web app (HTMX or React) served from `/ui/`
- [ ] Dashboard editor (grid layout, widget types, positions, expressions)
- [ ] Device list + dashboard assignment per device
- [ ] Latency monitor management
- [ ] Alert/webhook configuration
- [ ] **Commit & push after Phase 3 complete**

## Phase 4: IoT SDK (deferred)
- [ ] Arduino/ESP8266 SDK for fetching dashboard JSON
- [ ] Widget renderer for LCD (gauge, sparkline, status, numeric, progress, text)
- [ ] Device authentication flow
- [ ] MQTT/CoAP integration (optional)
- [ ] **Commit & push after Phase 4 complete**

## Out of Scope (Phase 1-3)
- [ ] Grafana integration — removed per user
- [ ] Real-time push via WebSocket — optional later
- [ ] Advanced alerting rules — basic log/webhook only
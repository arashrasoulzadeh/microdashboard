# Contributing Guide

## Getting Started

### Prerequisites
- Go 1.22+
- SQLite3 (for development)
- Docker (for containerized development)

### Setup

```bash
git clone https://github.com/arashrasoulzadeh/microdashboard.git
cd microdashboard

# Install dependencies
go mod download

# Run tests
go test ./tests/...

# Build
go build -o microdashboard ./cmd/api/

# Run
./microdashboard
```

### Development with Docker

```bash
# Build and run with hot reload
docker-compose up --build

# With live code changes
docker-compose up --build -d
```

## Project Structure

```
microdashboard/
├── cmd/api/main.go          # HTTP server entry point
├── internal/
│   ├── config/              # Configuration loading
│   ├── store/               # SQLite storage layer
│   ├── auth/                # Authentication middleware
│   ├── monitor/             # Background latency checker
│   ├── dashboard/           # Dashboard rendering & expressions
│   └── ui/                  # HTMX management UI
├── tests/
│   ├── unit/                # External test packages
│   └── integration/         # HTTP integration tests
├── sdk/arduino/             # Arduino/ESP8266 SDK
└── docs/                    # Documentation
```

## Development Workflow

### 1. Create Feature Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Write code in `internal/`
- Add tests in `tests/unit/` and `tests/integration/`
- Update documentation in `docs/`

### 3. Run Tests
```bash
# All tests
go test ./tests/... -coverpkg=./internal/... -coverprofile=coverage.out

# Specific package
go test ./tests/unit/store_test/... -v

# Coverage report
go tool cover -html=coverage.out -o coverage.html
```

### 4. Code Quality
```bash
# Format
go fmt ./...

# Vet
go vet ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

### 5. Commit
```bash
git add .
git commit -m "feat: brief description

- Detailed change 1
- Detailed change 2"
```

### 6. Push and PR
```bash
git push origin feature/your-feature-name
# Create PR on GitHub
```

## Code Style

### Go Conventions
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` formatting
- Exported functions/types: PascalCase
- Private: camelCase
- Error handling: explicit, no panic

### Package Structure
- Each package in `internal/` has single responsibility
- Tests in `tests/unit/<package>_test/`
- External test packages (import internal packages)

### Naming Conventions
- Files: `snake_case.go`
- Types: `PascalCase`
- Functions: `PascalCase` (exported), `camelCase` (private)
- Constants: `PascalCase`
- Variables: `camelCase`

### Error Handling
```go
// Good
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Avoid
if err != nil {
    panic(err)
}
```

## Testing

### Unit Tests
Location: `tests/unit/<package>_test/`

```go
package store_test

import (
    "testing"
    "microdashboard/internal/store"
)

func TestStore_UpsertDevice(t *testing.T) {
    s := setupTestDB(t)
    // ...
}
```

**Rules**:
- External test packages (`_test` suffix)
- Use `t.Cleanup()` for resources
- Test public API only
- Mock external dependencies

### Integration Tests
Location: `tests/integration/`

```go
func TestUI_ApiDashboards(t *testing.T) {
    server, _ := setupTestServer(t)
    // Test HTTP endpoints
}
```

### Coverage Target
- Overall: >70%
- Critical packages (store, auth): >80%
- New code: >90%

## Adding Features

### New API Endpoint
1. Add handler in appropriate package (`internal/ui/`, `internal/dashboard/`, etc.)
2. Register route in `cmd/api/main.go`
3. Add auth middleware if needed
4. Add tests in `tests/integration/`
5. Update `docs/API.md`

### New Widget Type
1. Add to `dashboard.validWidgetTypes` in `internal/dashboard/dashboard.go`
2. Add rendering in `dashboard.Render()`
3. Add `evaluate*` function if new data source
4. Add tests in `tests/unit/dashboard_test/`
5. Update `docs/WIDGETS.md`
6. Update Arduino SDK if needed

### New Data Source for Expressions
1. Add case in `evaluateExpression()` in `internal/dashboard/dashboard.go`
2. Add `evaluate<Source>()` function
3. Add tests
3. Update `docs/EXPRESSIONS.md`

## Debugging

### Local Server
```bash
go run ./cmd/api/
# Server at http://localhost:8080
```

### View Logs
```bash
# Structured logging
curl http://localhost:8080/health
```

### Database Inspection
```bash
sqlite3 ./data/microdashboard.db
sqlite> .tables
sqlite> SELECT * FROM devices;
```

### pprof (if enabled)
```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Release Process

### Versioning
- Semantic Versioning (MAJOR.MINOR.PATCH)
- Tag releases: `git tag v1.2.3`

### Release Checklist
- [ ] All tests pass
- [ ] Coverage > 70%
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Docker image builds
- [ ] Tag and push

## Code Review Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
- [ ] Error handling complete
- [ ] Logging appropriate
- [ ] Performance considered
- [ ] Security reviewed

## Questions?

Open an issue or discussion on GitHub.
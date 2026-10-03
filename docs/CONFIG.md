# Configuration Reference

## Environment Variables

All configuration is done via environment variables.

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | HTTP server port |
| `LISTEN_IP` | `0.0.0.0` | Bind address (use `127.0.0.1` for local only) |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `./data/microdashboard.db` | SQLite database file path |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_TOKEN` | `admin-change-me` | Admin token for management endpoints |

**Important**: Change this in production!
```bash
export ADMIN_TOKEN=$(openssl rand -hex 32)
```

### TLS / Let's Encrypt

| Variable | Default | Description |
|----------|---------|-------------|
| `LETSENCRYPT_EMAIL` | (empty) | Email for Let's Encrypt registration |
| `PUBLIC_DOMAIN` | `localhost` | Public domain for certificate |

If both are set, Let's Encrypt is enabled automatically.

### Monitoring

| Variable | Default | Description |
|----------|---------|-------------|
| `TARGET_POLL_INTERVAL` | `30s` | Interval between latency checks |

Format: Go duration string (e.g., `30s`, `1m`, `5m`)

## Configuration Examples

### Development
```bash
HTTP_PORT=8080
DB_PATH=./data/microdashboard.db
ADMIN_TOKEN=dev-token
```

### Production with Let's Encrypt
```bash
HTTP_PORT=80
LISTEN_IP=0.0.0.0
DB_PATH=/var/lib/microdashboard/microdashboard.db
ADMIN_TOKEN=your-secure-random-token-here
LETSENCRYPT_EMAIL=admin@yourdomain.com
PUBLIC_DOMAIN=dashboard.yourdomain.com
```

### Production without TLS (Behind Proxy)
```bash
HTTP_PORT=8080
LISTEN_IP=127.0.0.1
DB_PATH=/var/lib/microdashboard/microdashboard.db
ADMIN_TOKEN=your-secure-random-token-here
# No LETSENCRYPT_EMAIL or PUBLIC_DOMAIN
```

### Docker Compose Override
```yaml
# docker-compose.override.yml
version: "3.9"
services:
  api:
    environment:
      - HTTP_PORT=8080
      - ADMIN_TOKEN=${ADMIN_TOKEN}
      - LETSENCRYPT_EMAIL=${LETSENCRYPT_EMAIL}
      - PUBLIC_DOMAIN=${PUBLIC_DOMAIN}
    volumes:
      - ./data:/etc/microdashboard/data
```

## Configuration Loading

Configuration is loaded once at startup from environment variables.

The `config.Load()` function reads all variables and returns a `*Config` struct.

```go
cfg := config.Load()
// cfg.HTTPPort, cfg.DBPath, etc.
```

## Validation

- `HTTP_PORT`: Must be valid port number (1-65535)
- `DB_PATH`: Directory must be writable
- `ADMIN_TOKEN`: Non-empty (defaults to `admin-change-me`)
- `TARGET_POLL_INTERVAL`: Valid Go duration (e.g., `30s`, `1m`, `5m`)

## Runtime Configuration Changes

Some settings can be changed at runtime via admin API:
- Latency monitor targets (add/remove/update)
- Device dashboard assignments
- Alert webhook URLs

Database changes take effect immediately without restart.
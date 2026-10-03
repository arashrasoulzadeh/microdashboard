# Deployment Guide

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/arashrasoulzadeh/microdashboard.git
cd microdashboard
docker-compose up -d
```

Access:
- API: http://localhost:80
- UI: http://localhost:80/ui/?token=admin-change-me

### Manual Build

```bash
go build -o microdashboard ./cmd/api/
./microdashboard
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | HTTP port |
| `LISTEN_IP` | `0.0.0.0` | Bind address |
| `DB_PATH` | `./data/microdashboard.db` | SQLite database path |
| `ADMIN_TOKEN` | `admin-change-me` | Admin token for management |
| `LETSENCRYPT_EMAIL` | (empty) | Email for Let's Encrypt |
| `PUBLIC_DOMAIN` | `localhost` | Domain for Let's Encrypt |
| `TARGET_POLL_INTERVAL` | `30s` | Latency check interval |

### Example .env

```env
HTTP_PORT=8080
DB_PATH=./data/microdashboard.db
ADMIN_TOKEN=your-secure-random-token
LETSENCRYPT_EMAIL=admin@example.com
PUBLIC_DOMAIN=dashboard.example.com
```

## Production Deployment

### With Let's Encrypt (Recommended)

1. Set `PUBLIC_DOMAIN` to your domain
2. Set `LETSENCRYPT_EMAIL` to your email
3. Ensure ports 80 and 443 are accessible
4. Run with docker-compose

```yaml
# docker-compose.yml (production)
services:
  api:
    build: .
    ports:
      - "80:80"
      - "443:443"
    environment:
      - ADMIN_TOKEN=your-secure-token
      - LETSENCRYPT_EMAIL=admin@example.com
      - PUBLIC_DOMAIN=dashboard.example.com
    volumes:
      - ./data:/etc/microdashboard/data
      - ./certs:/etc/letsencrypt
    restart: unless-stopped

  mosquitto:
    image: eclipse-mosquitto:2
    ports:
      - "1883:1883"
    restart: unless-stopped
```

### Without Let's Encrypt (Self-Signed)

```bash
# Generate self-signed cert
openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365
```

Mount certs and update server to use them.

### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name dashboard.example.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Database

### Location
Default: `./data/microdashboard.db`

### Backup
```bash
# Online backup (no downtime)
sqlite3 ./data/microdashboard.db ".backup backup_$(date +%Y%m%d).db"

# Or copy file (if no writes)
cp ./data/microdashboard.db backup_$(date +%Y%m%d).db
```

### Restore
```bash
sqlite3 ./data/microdashboard.db ".restore backup_20260101.db"
```

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Metrics
```bash
curl -H "X-API-Key: your-device-key" http://localhost:8080/metrics
```

### Logs
```bash
docker-compose logs -f api
```

## Scaling Considerations

### Current Limitations
- Single-node only (SQLite)
- Single monitor goroutine
- No horizontal scaling

### For Higher Load
1. Migrate to PostgreSQL/TimescaleDB
2. Add Redis for caching
3. Use separate monitor workers
4. Add load balancer

## Security

### Change Default Admin Token
```bash
export ADMIN_TOKEN=$(openssl rand -hex 32)
```

### Firewall
- Allow 80/443 for API/UI
- Allow 1883 for MQTT (if used)
- Block direct database access

### Device Keys
- Generate unique keys per device
- Rotate periodically via admin API
- Never commit keys to version control

## Troubleshooting

### Database Locked
```bash
# Check for other processes
lsof ./data/microdashboard.db
```

### Certificate Issues
```bash
# Check certbot logs
docker-compose logs certbot
```

### High Memory
```bash
# Check Go heap
curl http://localhost:8080/debug/pprof/heap
```

## Upgrading

```bash
git pull
docker-compose build
docker-compose up -d
```

Database migrations run automatically on startup.

## Rollback

```bash
git checkout <previous-tag>
docker-compose build
docker-compose up -d
```

Database is backward compatible (migrations are additive).
# microdashboard - API Specification

OpenAPI 3.0 specification for the microdashboard REST API.

## Base URL
```
http://localhost:8080
```

## Authentication

### Device API Key
- **Type**: Header
- **Name**: `X-API-Key`
- **Description**: Per-device static key for device endpoints

### Admin Token
- **Type**: Query Parameter
- **Name**: `token`
- **Description**: Admin token for management endpoints (default: `admin-change-me`)

---

## Endpoints

### Health Check
**GET** `/health`

Check server health status.

#### Responses
| Code | Description |
|------|-------------|
| 200 | OK |

```json
{
  "status": "ok"
}
```

---

### Device Endpoints

#### Get Device Profile
**GET** `/dev/{device_id}`

Get device profile including assigned dashboard.

**Headers**: `X-API-Key: <device_api_key>`

#### Responses
| Code | Description |
|------|-------------|
| 200 | Device profile |
| 401 | Invalid or missing API key |
| 404 | Device not found |

```json
{
  "DeviceID": "test-esp",
  "APIKeyHash": "secret123",
  "DashboardID": "dash1",
  "CreatedAt": 1790888060
}
```

#### Register Device
**POST** `/devices`

Register a new device (admin only).

**Query Parameters**: `token=<admin_token>`

**Body**:
```json
{
  "device_id": "test-esp",
  "api_key": "secret123",
  "dashboard_id": "dash1"  // optional
}
```

#### Responses
| Code | Description |
|------|-------------|
| 201 | Device created |
| 400 | Invalid request |
| 403 | Invalid admin token |

```json
{"status": "created"}
```

---

### Dashboard Endpoints

#### List Dashboards
**GET** `/dashboards` or **GET** `/api/dashboards`

List all dashboards (admin only).

**Query Parameters**: `token=<admin_token>`

#### Responses
```json
[
  {
    "id": "dash1",
    "name": "Test Dashboard",
    "width": 128,
    "height": 64,
    "refresh_interval": 15000,
    "widget_count": 2,
    "created_at": 1790889965
  }
]
```

#### Create Dashboard
**POST** `/dashboards` or **POST** `/api/dashboards`

Create a new dashboard (admin only).

**Query Parameters**: `token=<admin_token>`

**Body**:
```json
{
  "id": "dash1",
  "name": "Test Dashboard",
  "width": 128,
  "height": 64,
  "refresh_interval": 15000,
  "widgets": [
    {
      "id": "w1",
      "type": "gauge",
      "x": 0,
      "y": 0,
      "width": 32,
      "height": 16,
      "expression": "${latency:monitor1}",
      "unit": "ms"
    }
  ]
}
```

#### Responses
| Code | Description |
|------|-------------|
| 201 | Dashboard created |
| 400 | Validation error |
| 403 | Invalid admin token |

```json
{"status": "created"}
```

#### Get Dashboard
**GET** `/dashboards/{id}` or **GET** `/api/dashboards/{id}`

Get dashboard definition (admin only).

**Query Parameters**: `token=<admin_token>`

#### Update Dashboard
**PUT** `/dashboards/{id}` or **PUT** `/api/dashboards/{id}`

Update dashboard (admin only).

**Query Parameters**: `token=<admin_token>`

**Body**: Same as create

#### Delete Dashboard
**DELETE** `/dashboards/{id}` or **DELETE** `/api/dashboards/{id}`

Delete dashboard (admin only).

**Query Parameters**: `token=<admin_token>`

---

### Dashboard Render Endpoint

#### Get Rendered Dashboard
**GET** `/dashboards/{id}/render` or **GET** `/api/dashboards/{id}/render`

Get dashboard with computed widget values.

**Headers**: `X-API-Key: <device_api_key>` or **Query**: `token=<admin_token>`

#### Responses
```json
{
  "id": "dash1",
  "name": "Test Dashboard",
  "width": 128,
  "height": 64,
  "widgets": [
    {
      "id": "w1",
      "type": "gauge",
      "x": 0,
      "y": 0,
      "width": 32,
      "height": 16,
      "value": 123.45,
      "text": "123ms",
      "unit": "ms",
      "status": "ok"
    }
  ],
  "rendered_at": 1790898338119
}
```

**Widget Status Values**: `ok`, `warn`, `crit`, `pending`

---

### Latency Monitor Endpoints

#### List Monitors
**GET** `/monitors` or **GET** `/api/monitors`

List all latency monitors (admin only).

**Query Parameters**: `token=<admin_token>`

#### Responses
```json
[
  {
    "id": "monitor1",
    "url": "http://httpbin.org/get",
    "method": "GET",
    "timeout": 5000,
    "last_elapsed_ms": 123,
    "last_status": 200,
    "last_checked": 1790898338
  }
]
```

#### Create Monitor
**POST** `/monitors` or **POST** `/api/monitors`

Create a new latency monitor (admin only).

**Query Parameters**: `token=<admin_token>`

**Body**:
```json
{
  "id": "monitor1",
  "url": "http://httpbin.org/get",
  "method": "GET",
  "timeout": 5000
}
```

#### Delete Monitor
**DELETE** `/monitors/{id}` or **DELETE** `/api/monitors/{id}`

Delete monitor (admin only).

**Query Parameters**: `token=<admin_token>`

---

### Metrics Endpoint

#### Get Metrics
**GET** `/metrics`

Get recent metric history (device key).

**Headers**: `X-API-Key: <device_api_key>`

**Query Parameters**: `limit=100` (default: 100)

#### Responses
```json
[
  {
    "ID": 1,
    "DeviceID": "",
    "TargetID": "monitor1",
    "ElapsedMs": 123,
    "Status": 200,
    "BodyBytes": 272,
    "CheckedAt": 1790898338
  }
]
```

---

### Alert Configuration

#### Get Alert Config
**GET** `/api/alerts`

Get alert configuration (admin only).

**Query Parameters**: `token=<admin_token>`

#### Update Alert Config
**POST** `/api/alerts`

Update alert configuration (admin only).

**Query Parameters**: `token=<admin_token>`

**Body**:
```json
{
  "webhook_url": "https://example.com/webhook",
  "mqtt_topic": "md/alerts"
}
```

---

## Dashboard JSON Schema

```json
{
  "id": "string (required)",
  "name": "string (required)",
  "width": "integer (default: 128)",
  "height": "integer (default: 64)",
  "refresh_interval": "integer milliseconds (default: 15000)",
  "widgets": [
    {
      "id": "string (required)",
      "type": "gauge|sparkline|status|numeric|progress|text (required)",
      "x": "integer (required)",
      "y": "integer (required)",
      "width": "integer (required)",
      "height": "integer (required)",
      "expression": "string (required, must contain ${...})",
      "unit": "string (optional)",
      "options": {
        "min": "number (optional)",
        "max": "number (optional)",
        "threshold_warn": "number (optional)",
        "threshold_crit": "number (optional)",
        "precision": "integer (optional)",
        "decimals": "integer (optional)",
        "label": "string (optional)"
      }
    }
  ]
}
```

---

## Expression Syntax

Expressions use `${...}` placeholders that are evaluated at render time.

### Latency Expression
```
${latency:<monitor_id>}
```
Returns the latest latency in milliseconds for the specified monitor.

Example: `${latency:api-server}`

### Metric Expression
```
${metric:<device_id>.<metric_name>}
```
Returns the latest metric value from a device.

Example: `${metric:sensor1.temperature}`

### Template Expressions (Text Widgets)
Text widgets support multiple placeholders in a single expression:
```
"API: ${latency:api1} | DB: ${latency:db1} | Temp: ${metric:sensor1.temp}°C"
```

Each placeholder is replaced with its computed value.

---

## Widget Types

| Type | Description | Use Case |
|------|-------------|----------|
| `gauge` | Circular gauge with needle | Latency, temperature, percentage |
| `sparkline` | Mini line chart | Trends over time (requires history) |
| `status` | Colored indicator (OK/WARN/CRIT) | Health checks |
| `numeric` | Large number display | Exact values |
| `progress` | Horizontal progress bar | Percentages, capacity |
| `text` | Formatted text with placeholders | Labels, combined status |

---

## Widget Options

| Option | Type | Description |
|--------|------|-------------|
| `min` | number | Minimum value for gauge/progress |
| `max` | number | Maximum value for gauge/progress |
| `threshold_warn` | number | Warning threshold (status = warn) |
| `threshold_crit` | number | Critical threshold (status = crit) |
| `precision` | integer | Decimal precision |
| `decimals` | integer | Decimal places for display |
| `label` | string | Optional label text |

---

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": "Error message"
}
```

| HTTP Code | Description |
|-----------|-------------|
| 400 | Bad Request - Invalid JSON or validation error |
| 401 | Unauthorized - Missing or invalid API key |
| 403 | Forbidden - Invalid admin token |
| 404 | Not Found |
| 500 | Internal Server Error |

---

## Rate Limiting

No rate limiting currently implemented. Consider adding for production.

---

## Versioning

Current API version: **v1**

No versioning in URL path. Breaking changes will be communicated via release notes.

---

## Changelog

### v1.0.0 (2026-10-03)
- Initial API release
- Dashboard CRUD and rendering
- Latency monitoring
- Device management
- HTMX management UI
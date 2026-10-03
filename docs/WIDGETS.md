# Widget Type Reference

## Overview

Widgets are the building blocks of dashboards. Each widget has a type, position, size, and data expression.

## Widget Properties (Common)

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | string | Yes | Unique widget identifier |
| `type` | string | Yes | Widget type (see below) |
| `x` | integer | Yes | X position (pixels from left) |
| `y` | integer | Yes | Y position (pixels from top) |
| `width` | integer | Yes | Widget width (pixels) |
| `height` | integer | Yes | Widget height (pixels) |
| `expression` | string | Yes | Data expression (see Expression Reference) |
| `unit` | string | No | Unit label (e.g., "ms", "°C", "%") |
| `options` | object | No | Type-specific options |

---

## Widget Types

### 1. Gauge (`gauge`)

Circular gauge with needle indicator.

**Best for**: Latency, temperature, CPU%, memory%

```json
{
  "id": "w1",
  "type": "gauge",
  "x": 0, "y": 0, "width": 64, "height": 64,
  "expression": "${latency:api-server}",
  "unit": "ms",
  "options": {
    "min": 0,
    "max": 1000,
    "threshold_warn": 500,
    "threshold_crit": 800
  }
}
```

**Options**:
- `min` (number): Minimum value (default: 0)
- `max` (number): Maximum value (default: 100)
- `threshold_warn`: Warning threshold (needle color yellow)
- `threshold_crit`: Critical threshold (needle color red)

**Rendered Output**:
- `value`: Numeric value (float)
- `text`: Formatted string (e.g., "123ms")
- `status`: "ok" | "warn" | "crit"

---

### 2. Numeric (`numeric`)

Large numeric display with optional unit.

**Best for**: Exact values, counters, precise measurements

```json
{
  "id": "w2",
  "type": "numeric",
  "x": 70, "y": 10, "width": 50, "height": 20,
  "expression": "${latency:api-server}",
  "unit": "ms",
  "options": {
    "decimals": 0
  }
}
```

**Options**:
- `decimals`: Decimal places (default: 0)
- `precision`: Significant digits

**Rendered Output**:
- `value`: Numeric value
- `text`: Formatted with unit (e.g., "123 ms")

---

### 3. Status (`status`)

Colored indicator: OK (green), Warning (yellow), Critical (red).

**Best for**: Health checks, service status, threshold alerts

```json
{
  "id": "w3",
  "type": "status",
  "x": 100, "y": 0, "width": 20, "height": 20,
  "expression": "${latency:api-server}",
  "options": {
    "threshold_warn": 200,
    "threshold_crit": 500
  }
}
```

**Options**:
- `threshold_warn`: Value ≥ this = warning (yellow)
- `threshold_crit`: Value ≥ this = critical (red)
- Values below `threshold_warn` = OK (green)

**Rendered Output**:
- `value`: Numeric value
- `text`: Formatted value
- `status`: "ok" | "warn" | "crit"

---

### 4. Progress (`progress`)

Horizontal progress bar.

**Best for**: Percentages, capacity, completion

```json
{
  "id": "w4",
  "type": "progress",
  "x": 0, "y": 50, "width": 128, "height": 10,
  "expression": "${metric:server1.cpu_percent}",
  "unit": "%",
  "options": {
    "min": 0,
    "max": 100
  }
}
```

**Options**:
- `min`: Minimum value (default: 0)
- `max`: Maximum value (default: 100)

**Rendered Output**:
- `value`: Numeric percentage (0-100)
- `text`: Formatted (e.g., "75%")
- `status`: "ok" | "warn" | "crit" (based on thresholds)

---

### 5. Text (`text`)

Formatted text with template replacement.

**Best for**: Labels, combined status, multi-value display

```json
{
  "id": "w5",
  "type": "text",
  "x": 0, "y": 60, "width": 128, "height": 10,
  "expression": "API: ${latency:api1} | DB: ${latency:db1} | Temp: ${metric:sensor1.temp}°C"
}
```

**Features**:
- Multiple `${...}` placeholders in single expression
- All placeholders replaced with computed values
- Static text preserved

**Rendered Output**:
- `value`: Last computed numeric value
- `text`: Fully rendered string (e.g., "API: 123ms | DB: 45ms | Temp: 23°C")
- `status`: Worst status among all placeholders

---

### 6. Sparkline (`sparkline`)

Mini line chart showing trend.

**Best for**: Trends over time, historical view

```json
{
  "id": "w6",
  "type": "sparkline",
  "x": 0, "y": 40, "width": 128, "height": 20,
  "expression": "${latency:api-server}",
  "unit": "ms"
}
```

**Note**: Requires historical data (future enhancement). Currently renders as numeric.

---

## Positioning

- **Coordinate system**: Top-left origin (0,0)
- **Units**: Pixels
- **Bounds**: Must fit within dashboard `width` × `height`
- **Overlap**: Widgets can overlap (last rendered wins)

## Size Guidelines

| Screen Size | Recommended Widget Sizes |
|-------------|-------------------------|
| 128×64 (OLED) | 32×16 to 64×64 |
| 160×80 | 40×20 to 80×40 |
| 320×240 (TFT) | 64×32 to 160×120 |

## Widget Options Reference

| Option | Types | Description |
|--------|-------|-------------|
| `min` | gauge, progress | Minimum scale value |
| `max` | gauge, progress | Maximum scale value |
| `threshold_warn` | gauge, status, progress | Warning threshold |
| `threshold_crit` | gauge, status, progress | Critical threshold |
| `decimals` | numeric | Decimal places to show |
| `precision` | numeric | Significant digits |
| `label` | all | Optional label text |

---

## Status Logic

| Widget | OK | Warning | Critical |
|--------|-----|---------|----------|
| `gauge` | value < warn | value ≥ warn | value ≥ crit |
| `status` | value < warn | value ≥ warn | value ≥ crit |
| `progress` | value < warn | value ≥ warn | value ≥ crit |
| `numeric` | Always OK | N/A | N/A |
| `text` | Best of all placeholders | | |

---

## Example: Complete Dashboard

```json
{
  "id": "server-dashboard",
  "name": "Server Monitor",
  "width": 128,
  "height": 64,
  "refresh_interval": 10000,
  "widgets": [
    {
      "id": "api-latency",
      "type": "gauge",
      "x": 0, "y": 0, "width": 64, "height": 64,
      "expression": "${latency:api-server}",
      "unit": "ms",
      "options": {"min": 0, "max": 1000, "threshold_warn": 300, "threshold_crit": 600}
    },
    {
      "id": "db-latency",
      "type": "gauge",
      "x": 64, "y": 0, "width": 64, "height": 64,
      "expression": "${latency:db-server}",
      "unit": "ms",
      "options": {"min": 0, "max": 500, "threshold_warn": 100, "threshold_crit": 300}
    },
    {
      "id": "cpu-usage",
      "type": "progress",
      "x": 0, "y": 54, "width": 128, "height": 10,
      "expression": "${metric:server1.cpu_percent}",
      "unit": "%",
      "options": {"min": 0, "max": 100, "threshold_warn": 70, "threshold_crit": 90}
    }
  ]
}
```
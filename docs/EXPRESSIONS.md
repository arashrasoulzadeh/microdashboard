# Expression Syntax Reference

## Overview

Expressions bind widget data to data sources using a simple `${source:target}` syntax.

## Syntax

```
${source:target}
```

- **source**: Data source type (`latency`, `metric`)
- **target**: Source-specific identifier

---

## Data Sources

### 1. Latency Monitor

Returns the latest latency measurement for a configured monitor.

**Syntax**: `${latency:<monitor_id>}`

**Example**:
```
${latency:api-server}
${latency:database-health}
${latency:cdn-edge}
```

**Returns**:
- `value`: Latest latency in milliseconds (float)
- `text`: Formatted as "123ms"
- `status`: "ok" | "warn" | "crit" (based on monitor's last HTTP status)

**Requirements**:
- Monitor must exist in `latency_monitors` table
- Monitor must have been checked at least once

---

### 2. Device Metric

Returns the latest metric value from a device.

**Syntax**: `${metric:<device_id>.<metric_name>}`

**Example**:
```
${metric:sensor1.temperature}
${metric:server1.cpu_percent}
${metric:esp-001.humidity}
${metric:gateway.battery_voltage}
```

**Returns**:
- `value`: Latest metric value (float)
- `text`: Formatted with unit (e.g., "23.5°C")
- `status`: "ok" (always OK for metrics)

**Requirements**:
- Device must exist in `devices` table
- Device must have pushed metrics (via `/metrics` or MQTT)
- Metric name must match a field in the metric payload

---

## Template Expressions (Text Widgets)

Text widgets support **multiple placeholders** in a single expression.

### Syntax
```
"Static text ${latency:monitor1} and ${metric:device.metric}"
```

### Behavior
- Each `${...}` placeholder is evaluated independently
- Placeholders replaced with their computed `text` value
- Static text preserved as-is
- Multiple occurrences of same placeholder all replaced

### Example
```
Expression: "API: ${latency:api1} | DB: ${latency:db1} | Temp: ${metric:sensor1.temp}°C"
Rendered:   "API: 123ms | DB: 45ms | Temp: 23.5°C"
```

### Multiple Same Placeholders
```
Expression: "Start: ${latency:api} ... End: ${latency:api}"
Rendered:   "Start: 123ms ... End: 123ms"
```
All occurrences replaced with same value.

---

## Expression Evaluation

### Evaluation Order
1. Parse expression for `${...}` placeholders
2. For each placeholder:
   - Parse source and target
   - Query data source
   - Compute value, text, status
3. Replace placeholders in text (for text widgets)
4. Determine overall widget status (worst of all)

### Caching
- Expressions evaluated on each render request
- No server-side caching (real-time data)
- Dashboard `refresh_interval` controls client poll frequency

---

## Error Handling

### Missing Monitor
```
Expression: ${latency:missing-monitor}
Result: value=null, text="ERR: monitor not found: missing-monitor", status="crit"
```

### Missing Device/Metric
```
Expression: ${metric:unknown.temp}
Result: value=null, text="ERR: device not found: unknown", status="crit"
```

### Invalid Expression Format
```
Expression: ${latency}
Result: value=null, text="ERR: invalid expression format: latency", status="crit"
```

### Empty Expression
```
Expression: "Static text"
Result: value=null, text="Static text", status="ok"
```

---

## Status Propagation

### Single Placeholder Widgets (gauge, numeric, status, progress)
- Status = source status

### Text Widgets (Multiple Placeholders)
- Status = worst status among all placeholders
- Priority: `crit` > `warn` > `ok`

### Example
```
Expression: "${latency:api} | ${metric:device.temp}"
- api status: "crit" (500 error)
- temp status: "ok"
Result status: "crit"
```

---

## Best Practices

### 1. Use Descriptive Monitor IDs
```
✅ ${latency:api-gateway}
✅ ${latency:database-primary}
❌ ${latency:mon1}
```

### 2. Use Consistent Metric Names
```
✅ ${metric:sensor1.temperature}
✅ ${metric:sensor1.humidity}
❌ ${metric:sensor1.temp_c}
```

### 3. Set Appropriate Thresholds
```json
{
  "options": {
    "threshold_warn": 200,
    "threshold_crit": 500
  }
}
```

### 4. Test Expressions Before Deploying
```bash
# Test via API
curl -H "X-API-Key: device-key" \
  "http://localhost:8080/dashboards/dash1/render"
```

---

## Future Extensions

Planned expression sources:
- `${config:<key>}` - Static configuration values
- `${math:<expression>}` - Mathematical operations
- `${time:<format>}` - Current time/date
- `${device:<device_id>.<property>}` - Device metadata

---

## Migration from v0

| Old Syntax | New Syntax |
|------------|------------|
| `latency(monitor1)` | `${latency:monitor1}` |
| `metric(device1, temp)` | `${metric:device1.temperature}` |

Expressions are now more flexible and consistent.
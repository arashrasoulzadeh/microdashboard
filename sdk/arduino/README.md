# MicroDashboard Arduino SDK

Arduino library for ESP8266/ESP32 to connect to MicroDashboard API, fetch dashboard definitions, and render widgets on LCD/OLED displays.

## Features

- **Fetch dashboard definitions** with widget layouts and expressions
- **Fetch rendered dashboards** with computed values from server
- **Support for 6 widget types**: Gauge, Sparkline, Status, Numeric, Progress, Text
- **Expression evaluation** on server side: `${latency:monitor1}`, `${metric:device1.temp}`
- **HTTPS support** with optional certificate verification skip
- **Low memory footprint** - designed for ESP8266 (80KB RAM)
- **Display agnostic** - works with Adafruit_GFX, U8g2, TFT_eSPI, etc.

## Installation

### Arduino IDE
1. Download the library ZIP from releases
2. Sketch → Include Library → Add .ZIP Library

### PlatformIO
```ini
lib_deps = 
  https://github.com/microdashboard/microdashboard.git#sdk/arduino
```

### Manual
Copy `sdk/arduino` to your Arduino libraries folder as `MicroDashboard`.

## Quick Start

```cpp
#include <WiFi.h>
#include <MicroDashboard.h>

const char* WIFI_SSID = "your-ssid";
const char* WIFI_PASS = "your-password";
const char* SERVER = "http://192.168.1.100:8080";
const char* API_KEY = "your-device-api-key";
const char* DASHBOARD_ID = "dash1";

MicroDashboard md(SERVER, API_KEY);

void setup() {
  Serial.begin(115200);
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  while (WiFi.status() != WL_CONNECTED) delay(500);
  
  md.begin();
  
  DashboardConfig dash;
  if (md.fetchDashboard(DASHBOARD_ID, dash)) {
    // Dashboard loaded - dash.widgets contains layout
  }
}

void loop() {
  RenderedDashboard rendered;
  if (md.fetchRenderedDashboard(DASHBOARD_ID, rendered)) {
    // rendered.widgets contains computed values
    for (int i = 0; i < rendered.widgetCount; i++) {
      md.renderWidget(rendered.widgets[i], 
                      rendered.widgets[i].x, rendered.widgets[i].y,
                      rendered.widgets[i].width, rendered.widgets[i].height);
    }
  }
  delay(5000);
}
```

## Expression Syntax

Widgets use expressions to pull data from the server:

| Expression | Description |
|------------|-------------|
| `${latency:monitor1}` | Latency from monitor "monitor1" |
| `${metric:device1.temp}` | Temperature metric from device "device1" |

Multiple expressions in one widget (for text):
```
"API: ${latency:api1} | DB: ${latency:db1}"
```

## Widget Types

| Type | Description | Best For |
|------|-------------|----------|
| `gauge` | Circular gauge with needle | Latency, temperature, % |
| `sparkline` | Mini line chart | Trends over time |
| `status` | Colored indicator (OK/WARN/CRIT) | Health checks |
| `numeric` | Large number display | Exact values |
| `progress` | Horizontal progress bar | Percentages, capacity |
| `text` | Formatted text with placeholders | Labels, combined status |

## Dashboard JSON Format

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
      "x": 0, "y": 0, "width": 64, "height": 64,
      "expression": "${latency:api_server}",
      "unit": "ms",
      "options": { "min": 0, "max": 1000 }
    },
    {
      "id": "w2",
      "type": "text",
      "x": 70, "y": 10, "width": 50, "height": 20,
      "expression": "API: ${latency:api_server}"
    }
  ]
}
```

## Display Libraries

The SDK provides rendering stubs. Implement these for your display:

```cpp
// For Adafruit_GFX (SSD1306, ST7735, ILI9341, etc.)
void MicroDashboard::renderWidgetGauge(RenderedWidget& w, int16_t cx, int16_t cy, int16_t r) {
  // Draw gauge using display.drawLine, drawCircle, etc.
}

// For U8g2
void MicroDashboard::renderWidgetGauge(RenderedWidget& w, int16_t cx, int16_t cy, int16_t r) {
  // Use u8g2.drawCircle, u8g2.drawLine, etc.
}
```

See `examples/BasicDashboard` for a complete SSD1306 example.

## HTTPS / TLS

For production with valid certificates:
```cpp
MicroDashboard md("https://your-domain.com", API_KEY, true);
md.begin();
// Uses system CA certs automatically
```

For self-signed certs (development):
```cpp
MicroDashboard md("https://192.168.1.100:8443", API_KEY, true);
md.setInsecure(true);  // Skip cert verification
md.begin();
```

## Memory Usage (ESP8266)

| Component | RAM |
|-----------|-----|
| Library base | ~2 KB |
| Dashboard (10 widgets) | ~3 KB |
| Rendered dashboard | ~2 KB |
| JSON parsing (ArduinoJson) | ~4 KB |
| **Total** | **~11 KB** |

Leaves ~69 KB for WiFi stack and application.

## API Reference

### MicroDashboard(serverUrl, apiKey, useHTTPS)
Constructor. `serverUrl` can be `http://host:port` or `https://host:port`.

### begin()
Initialize HTTP client. Call after WiFi connected.

### fetchDashboard(id, dashboard)
Fetch dashboard definition (layout only).

### fetchRenderedDashboard(id, rendered)
Fetch dashboard with computed values (ready to display).

### freeDashboard(dashboard) / freeRenderedDashboard(rendered)
Free allocated memory.

### renderWidget(widget, x, y, w, h)
Dispatch to type-specific renderer.

### setInsecure(true)
Skip TLS certificate verification (for self-signed certs).

### setTimeout(ms)
Set HTTP request timeout (default 10000ms).

## License

MIT License - see LICENSE file.
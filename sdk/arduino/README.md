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
1. Download the library ZIP from [releases](https://github.com/microdashboard/microdashboard/releases)
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

// WiFi credentials
const char* WIFI_SSID = "your-ssid";
const char* WIFI_PASS = "your-password";

// MicroDashboard server
const char* SERVER_URL = "http://192.168.1.100:8080";  // or https://your-domain.com
const char* API_KEY = "your-device-api-key";  // From device registration
const char* DASHBOARD_ID = "dash1";

// Use HTTPS? Set to true for production with valid certs
const bool USE_HTTPS = false;

MicroDashboard md(SERVER_URL, API_KEY, USE_HTTPS);
DashboardConfig dashboard;
RenderedDashboard rendered;

void setup() {
  Serial.begin(115200);
  
  // Connect WiFi
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("\nWiFi connected!");
  
  // Initialize MicroDashboard
  md.setTimeout(10000);
  if (USE_HTTPS) {
    md.setInsecure(true);  // For self-signed certs
  }
  
  if (!md.begin()) {
    Serial.println("Failed to init MicroDashboard");
    return;
  }
  
  // Fetch dashboard definition
  Serial.print("Fetching dashboard ");
  Serial.println(DASHBOARD_ID);
  if (md.fetchDashboard(DASHBOARD_ID, dashboard)) {
    Serial.printf("Dashboard: %s (%dx%d, %d widgets)\n", 
                  dashboard.name.c_str(), dashboard.width, dashboard.height, dashboard.widgetCount);
    for (int i = 0; i < dashboard.widgetCount; i++) {
      WidgetConfig& w = dashboard.widgets[i];
      Serial.printf("  Widget %d: %s type=%d pos=(%d,%d) size=%dx%d expr=%s\n",
                    i, w.id.c_str(), w.type, w.x, w.y, w.width, w.height, w.expression.c_str());
    }
  } else {
    Serial.println("Failed to fetch dashboard: " + md.getLastError());
  }
}

void loop() {
  // Fetch rendered dashboard periodically
  if (md.fetchRenderedDashboard(DASHBOARD_ID, rendered)) {
    Serial.printf("Rendered dashboard at %lu, %d widgets\n", rendered.renderedAt, rendered.widgetCount);
    for (int i = 0; i < rendered.widgetCount; i++) {
      RenderedWidget& w = rendered.widgets[i];
      Serial.printf("  %s: value=%.1f text='%s' status=%d\n", 
                    w.id.c_str(), w.value, w.text.c_str(), w.status);
    }
  } else {
    Serial.println("Fetch failed: " + md.getLastError());
  }
  
  delay(5000);
}
```

## API Reference

### MicroDashboard(serverUrl, apiKey, useHTTPS)
Constructor. `serverUrl` can be `http://host:port` or `https://host:port`.

### begin()
Initialize HTTP client. Call after WiFi connected. Returns `true` on success.

### fetchDashboard(id, dashboard)
Fetch dashboard definition (layout only, no computed values).

### fetchRenderedDashboard(id, rendered)
Fetch dashboard with computed widget values (ready to display).

### freeDashboard(dashboard) / freeRenderedDashboard(rendered)
Free allocated memory. Call when done with dashboard.

### renderWidget(widget, x, y, w, h)
Dispatch to type-specific renderer. Implement for your display.

### Type-Specific Renderers
```cpp
renderWidgetGauge(widget, cx, cy, radius);
renderWidgetNumeric(widget, x, y);
renderWidgetStatus(widget, x, y, w, h);
renderWidgetProgress(widget, x, y, w, h);
renderWidgetText(widget, x, y);
renderWidgetSparkline(widget, x, y, w, h);
```

### setInsecure(true)
Skip TLS certificate verification (for self-signed certs).

### setTimeout(ms)
Set HTTP request timeout (default 10000ms).

### isConnected()
Check if HTTP client is connected.

### getLastError()
Get last error message.

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

### Production (Valid Certificates)
```cpp
MicroDashboard md("https://your-domain.com", API_KEY, true);
md.begin();
// Uses system CA certs automatically
```

### Development (Self-Signed Certs)
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

## Examples

### BasicDashboard
Fetch and render dashboard on SSD1306 OLED (128×64).
See `examples/BasicDashboard/BasicDashboard.ino`

## Platform Support

| Platform | Status |
|----------|--------|
| ESP8266 (Arduino core) | ✅ Tested |
| ESP32 (Arduino core) | ✅ Tested |
| ESP32-S2/S3 | ✅ Should work |
| Arduino Uno/Nano | ❌ Insufficient RAM |

## Dependencies

- ArduinoJson (v6.21+)
- WiFi (built-in)
- BearSSL/SSL (built-in for ESP8266/ESP32)

## License

MIT License - see LICENSE file.
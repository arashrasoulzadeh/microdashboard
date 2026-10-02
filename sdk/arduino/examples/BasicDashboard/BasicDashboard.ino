/*
  MicroDashboard - Basic Dashboard Example
  
  This example shows how to connect an ESP8266/ESP32 to MicroDashboard,
  fetch a dashboard definition, and render widgets on a display.
  
  Hardware:
  - ESP8266 (NodeMCU, Wemos D1 Mini) or ESP32
  - Display: SSD1306 OLED (128x64) or ILI9341 TFT (320x240)
  
  Libraries needed:
  - MicroDashboard (this library)
  - ArduinoJson (v6+)
  - Adafruit_GFX + Adafruit_SSD1306 (for OLED) or TFT_eSPI (for TFT)
*/

#include <Arduino.h>
#include <WiFi.h>
#include <MicroDashboard.h>

// ===== CONFIGURATION =====
const char* WIFI_SSID = "YOUR_WIFI_SSID";
const char* WIFI_PASSWORD = "YOUR_WIFI_PASSWORD";

// MicroDashboard server
const char* SERVER_URL = "http://192.168.1.100:8080";  // or https://your-domain.com
const char* API_KEY = "your-device-api-key";  // From device registration
const char* DASHBOARD_ID = "dash1";

// Use HTTPS? Set to true for production with valid certs
const bool USE_HTTPS = false;

// Display config (adjust for your display)
#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
#define OLED_RESET -1
#define OLED_I2C_ADDR 0x3C

// ===== GLOBALS =====
MicroDashboard md(SERVER_URL, API_KEY, USE_HTTPS);
DashboardConfig dashboard;
RenderedDashboard rendered;
unsigned long lastFetch = 0;
unsigned long lastRender = 0;
bool dashboardLoaded = false;

// Display (example for SSD1306)
#if defined(ARDUINO_ARCH_ESP8266) || defined(ARDUINO_ARCH_ESP32)
#include <Wire.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>
Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);
#endif

void setup() {
  Serial.begin(115200);
  delay(100);
  Serial.println("\n=== MicroDashboard Basic Example ===");
  
  // Initialize display
  Wire.begin(4, 5);  // SDA, SCL for ESP8266 (D2, D1)
  if (!display.begin(SSD1306_SWITCHCAPVCC, OLED_I2C_ADDR)) {
    Serial.println(F("SSD1306 allocation failed"));
    for (;;) delay(1000);
  }
  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(WHITE);
  display.setCursor(0, 0);
  display.println("Connecting WiFi...");
  display.display();
  
  // Connect WiFi
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  Serial.print("Connecting to WiFi");
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
    display.print(".");
    display.display();
  }
  Serial.println();
  Serial.print("Connected! IP: ");
  Serial.println(WiFi.localIP());
  
  display.clearDisplay();
  display.setCursor(0, 0);
  display.println("WiFi OK");
  display.println(WiFi.localIP());
  display.display();
  delay(1000);
  
  // Initialize MicroDashboard
  md.setTimeout(10000);
  if (USE_HTTPS) {
    md.setInsecure(true);  // For self-signed certs
  }
  
  if (!md.begin()) {
    Serial.println("Failed to init MicroDashboard client");
    return;
  }
  
  // Fetch dashboard definition
  Serial.print("Fetching dashboard ");
  Serial.println(DASHBOARD_ID);
  if (md.fetchDashboard(DASHBOARD_ID, dashboard)) {
    dashboardLoaded = true;
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
  
  display.clearDisplay();
  display.setCursor(0, 0);
  display.println("Dashboard loaded");
  display.display();
  delay(1000);
}

void loop() {
  if (!dashboardLoaded) return;
  
  unsigned long now = millis();
  
  // Fetch rendered dashboard periodically
  if (now - lastFetch >= dashboard.refreshInterval) {
    if (md.fetchRenderedDashboard(DASHBOARD_ID, rendered)) {
      lastFetch = now;
      Serial.printf("Rendered dashboard at %lu, %d widgets\n", rendered.renderedAt, rendered.widgetCount);
      for (int i = 0; i < rendered.widgetCount; i++) {
        RenderedWidget& w = rendered.widgets[i];
        Serial.printf("  %s: value=%.1f text='%s' status=%d\n", 
                      w.id.c_str(), w.value, w.text.c_str(), w.status);
      }
    } else {
      Serial.println("Fetch failed: " + md.getLastError());
    }
  }
  
  // Render to display
  if (now - lastRender >= 1000) {  // Render at 1Hz max
    renderDashboard();
    lastRender = now;
  }
  
  // Handle WiFi reconnection
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("WiFi disconnected, reconnecting...");
    WiFi.reconnect();
    delay(1000);
  }
}

void renderDashboard() {
  display.clearDisplay();
  
  if (!rendered.widgets || rendered.widgetCount == 0) {
    display.setCursor(0, 0);
    display.println("No data");
    display.display();
    return;
  }
  
  for (int i = 0; i < rendered.widgetCount; i++) {
    RenderedWidget& w = rendered.widgets[i];
    
    // Position on screen (use widget coordinates)
    int16_t x = w.x;
    int16_t y = w.y;
    int16_t w_px = w.width;
    int16_t h_px = w.height;
    
    // Clamp to screen bounds
    if (x + w_px > SCREEN_WIDTH) w_px = SCREEN_WIDTH - x;
    if (y + h_px > SCREEN_HEIGHT) h_px = SCREEN_HEIGHT - y;
    if (w_px <= 0 || h_px <= 0) continue;
    
    // Dispatch to type-specific renderer
    switch (w.type) {
      case WIDGET_GAUGE:
        renderGauge(w, x + w_px/2, y + h_px/2, min(w_px, h_px)/2);
        break;
      case WIDGET_NUMERIC:
        display.setCursor(x, y);
        display.print(w.text);
        break;
      case WIDGET_STATUS:
        renderStatusIndicator(w, x, y, w_px, h_px);
        break;
      case WIDGET_PROGRESS:
        renderProgressBar(w, x, y, w_px, h_px);
        break;
      case WIDGET_TEXT:
        display.setCursor(x, y);
        display.print(w.text);
        break;
      case WIDGET_SPARKLINE:
        // Sparkline needs history - skip for now
        break;
    }
  }
  
  display.display();
}

// Custom renderers for SSD1306
void renderGauge(RenderedWidget& w, int16_t cx, int16_t cy, int16_t r) {
  if (r < 5) return;
  
  // Draw gauge background arc
  for (int angle = -90; angle <= 90; angle += 5) {
    float rad = angle * DEG_TO_RAD;
    int16_t x1 = cx + cos(rad) * (r - 2);
    int16_t y1 = cy + sin(rad) * (r - 2);
    int16_t x2 = cx + cos(rad) * r;
    int16_t y2 = cy + sin(rad) * r;
    display.drawLine(x1, y1, x2, y2, WHITE);
  }
  
  // Calculate needle position (0-100% mapped to -90 to +90 degrees)
  float pct = 0;
  if (w.maxVal != w.minVal) {
    pct = (w.value - w.minVal) / (w.maxVal - w.minVal) * 100.0;
    pct = constrain(pct, 0, 100);
  }
  float needleAngle = -90 + (pct / 100.0) * 180;
  float rad = needleAngle * DEG_TO_RAD;
  
  // Draw needle
  int16_t nx = cx + cos(rad) * (r - 4);
  int16_t ny = cy + sin(rad) * (r - 4);
  display.drawLine(cx, cy, nx, ny, WHITE);
  display.drawCircle(cx, cy, 2, WHITE);
  
  // Draw value text
  display.setCursor(cx - 15, cy + r + 2);
  display.print(w.text);
}

void renderStatusIndicator(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  uint16_t color = WHITE;
  switch (w.status) {
    case STATUS_OK: color = WHITE; break;
    case STATUS_WARN: color = WHITE; break;  // SSD1306 is monochrome
    case STATUS_CRIT: color = WHITE; break;
    default: color = WHITE;
  }
  
  // Draw status box
  display.drawRect(x, y, w_px, h_px, color);
  display.setCursor(x + 2, y + 2);
  display.print(w.text);
}

void renderProgressBar(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  if (w_px < 10 || h_px < 4) return;
  
  float pct = 0;
  if (w.maxVal != w.minVal) {
    pct = (w.value - w.minVal) / (w.maxVal - w.minVal);
    pct = constrain(pct, 0, 1);
  }
  
  int16_t fillW = w_px * pct;
  
  // Border
  display.drawRect(x, y, w_px, h_px, WHITE);
  // Fill
  if (fillW > 2) {
    display.fillRect(x + 1, y + 1, fillW - 2, h_px - 2, WHITE);
  }
  
  // Label
  if (h_px > 10) {
    display.setCursor(x, y + h_px + 2);
    display.print(w.text);
  }
}
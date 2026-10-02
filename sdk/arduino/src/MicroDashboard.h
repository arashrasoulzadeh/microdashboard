#ifndef MICRODASHBOARD_H
#define MICRODASHBOARD_H

#include <Arduino.h>
#include <ArduinoJson.h>
#include <WiFiClient.h>
#include <WiFiClientSecure.h>

#define MICRODASHBOARD_VERSION "1.0.0"

// Widget types
enum WidgetType {
  WIDGET_GAUGE = 0,
  WIDGET_SPARKLINE = 1,
  WIDGET_STATUS = 2,
  WIDGET_NUMERIC = 3,
  WIDGET_PROGRESS = 4,
  WIDGET_TEXT = 5,
  WIDGET_UNKNOWN = 255
};

// Widget status
enum WidgetStatus {
  STATUS_OK = 0,
  STATUS_WARN = 1,
  STATUS_CRIT = 2,
  STATUS_PENDING = 3
};

// Widget configuration
struct WidgetConfig {
  String id;
  WidgetType type;
  int x, y, width, height;
  String expression;
  String unit;
  float minVal = 0;
  float maxVal = 100;
  float thresholdWarn = 0;
  float thresholdCrit = 0;
  int precision = 0;
  String label;
};

// Rendered widget with computed value
struct RenderedWidget {
  String id;
  WidgetType type;
  int x, y, width, height;
  float value;
  String text;
  String unit;
  WidgetStatus status;
  unsigned long renderedAt;
};

// Dashboard configuration
struct DashboardConfig {
  String id;
  String name;
  int width, height;
  int refreshInterval;
  WidgetConfig* widgets;
  int widgetCount;
  unsigned long createdAt;
};

// Rendered dashboard
struct RenderedDashboard {
  String id;
  String name;
  int width, height;
  RenderedWidget* widgets;
  int widgetCount;
  unsigned long renderedAt;
};

class MicroDashboard {
public:
  MicroDashboard(const char* serverUrl, const char* apiKey);
  MicroDashboard(const char* serverUrl, const char* apiKey, bool useHTTPS);
  
  // Initialize connection
  bool begin();
  
  // Fetch dashboard definition
  bool fetchDashboard(const char* dashboardId, DashboardConfig& dashboard);
  
  // Fetch rendered dashboard (with computed values)
  bool fetchRenderedDashboard(const char* dashboardId, RenderedDashboard& dashboard);
  
  // Free dashboard memory
  void freeDashboard(DashboardConfig& dashboard);
  void freeRenderedDashboard(RenderedDashboard& dashboard);
  
  // Widget rendering helpers (for LCD/OLED)
  void renderWidgetGauge(RenderedWidget& w, int16_t centerX, int16_t centerY, int16_t radius);
  void renderWidgetNumeric(RenderedWidget& w, int16_t x, int16_t y);
  void renderWidgetStatus(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px);
  void renderWidgetProgress(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px);
  void renderWidgetText(RenderedWidget& w, int16_t x, int16_t y);
  void renderWidgetSparkline(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px);
  
  // Generic render dispatcher
  void renderWidget(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px);
  
  // Connection management
  void setInsecure(bool insecure);  // Skip cert verification (for self-signed)
  void setTimeout(uint32_t timeoutMs);
  bool isConnected();
  
  // Getters
  const char* getServerUrl() const { return _serverUrl.c_str(); }
  const char* getApiKey() const { return _apiKey.c_str(); }
  String getLastError() const { return _lastError; }

private:
  String _serverUrl;
  String _apiKey;
  bool _useHTTPS;
  bool _insecure;
  uint32_t _timeout;
  String _lastError;
  
  WiFiClient* _client;
  WiFiClientSecure* _secureClient;
  
  bool _makeRequest(const char* method, const char* path, const char* body, String& response);
  bool _parseDashboard(const String& json, DashboardConfig& dashboard);
  bool _parseRenderedDashboard(const String& json, RenderedDashboard& dashboard);
  WidgetType _parseWidgetType(const String& typeStr);
  WidgetStatus _parseStatus(const String& statusStr);
  void _freeWidgets(WidgetConfig* widgets, int count);
  void _freeRenderedWidgets(RenderedWidget* widgets, int count);
};

#endif
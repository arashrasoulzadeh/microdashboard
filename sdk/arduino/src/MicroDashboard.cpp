#include "MicroDashboard.h"

MicroDashboard::MicroDashboard(const char* serverUrl, const char* apiKey)
  : _serverUrl(serverUrl), _apiKey(apiKey), _useHTTPS(false), _insecure(false), _timeout(10000), _client(nullptr), _secureClient(nullptr) {
}

MicroDashboard::MicroDashboard(const char* serverUrl, const char* apiKey, bool useHTTPS)
  : _serverUrl(serverUrl), _apiKey(apiKey), _useHTTPS(useHTTPS), _insecure(false), _timeout(10000), _client(nullptr), _secureClient(nullptr) {
}

bool MicroDashboard::begin() {
  if (_useHTTPS) {
    _secureClient = new WiFiClientSecure();
    if (_insecure) {
      _secureClient->setInsecure();
    }
    _client = _secureClient;
  } else {
    _client = new WiFiClient();
  }
  return _client != nullptr;
}

void MicroDashboard::setInsecure(bool insecure) {
  _insecure = insecure;
  if (_secureClient) {
    _secureClient->setInsecure();
  }
}

void MicroDashboard::setTimeout(uint32_t timeoutMs) {
  _timeout = timeoutMs;
}

bool MicroDashboard::isConnected() {
  return _client && _client->connected();
}

bool MicroDashboard::fetchDashboard(const char* dashboardId, DashboardConfig& dashboard) {
  String path = "/api/dashboards/" + String(dashboardId) + "/render";
  String response;
  
  if (!_makeRequest("GET", path.c_str(), nullptr, response)) {
    return false;
  }
  
  return _parseDashboard(response, dashboard);
}

bool MicroDashboard::fetchRenderedDashboard(const char* dashboardId, RenderedDashboard& dashboard) {
  String path = "/dashboards/" + String(dashboardId) + "/render";
  String response;
  
  if (!_makeRequest("GET", path.c_str(), nullptr, response)) {
    return false;
  }
  
  return _parseRenderedDashboard(response, dashboard);
}

void MicroDashboard::freeDashboard(DashboardConfig& dashboard) {
  if (dashboard.widgets) {
    _freeWidgets(dashboard.widgets, dashboard.widgetCount);
    dashboard.widgets = nullptr;
    dashboard.widgetCount = 0;
  }
}

void MicroDashboard::freeRenderedDashboard(RenderedDashboard& dashboard) {
  if (dashboard.widgets) {
    _freeRenderedWidgets(dashboard.widgets, dashboard.widgetCount);
    dashboard.widgets = nullptr;
    dashboard.widgetCount = 0;
  }
}

bool MicroDashboard::_makeRequest(const char* method, const char* path, const char* body, String& response) {
  if (!_client) {
    _lastError = "Client not initialized";
    return false;
  }
  
  // Parse host and port from server URL
  String url = _serverUrl;
  if (url.startsWith("http://")) url = url.substring(7);
  else if (url.startsWith("https://")) url = url.substring(8);
  
  int port = _useHTTPS ? 443 : 80;
  int colonIdx = url.indexOf(':');
  int slashIdx = url.indexOf('/');
  
  String host;
  if (colonIdx > 0 && (slashIdx == -1 || colonIdx < slashIdx)) {
    host = url.substring(0, colonIdx);
    port = url.substring(colonIdx + 1, slashIdx > 0 ? slashIdx : url.length()).toInt();
  } else {
    host = url.substring(0, slashIdx > 0 ? slashIdx : url.length());
  }
  
  String requestPath = slashIdx > 0 ? url.substring(slashIdx) : "/";
  requestPath += path;
  
  if (!_client->connect(host.c_str(), port)) {
    _lastError = "Connection failed to " + host + ":" + String(port);
    return false;
  }
  
  // Send request
  _client->print(String(method) + " " + requestPath + " HTTP/1.1\r\n");
  _client->print("Host: " + host + "\r\n");
  _client->print("X-API-Key: " + _apiKey + "\r\n");
  _client->print("Connection: close\r\n");
  
  if (body) {
    _client->print("Content-Type: application/json\r\n");
    _client->print("Content-Length: " + String(strlen(body)) + "\r\n");
    _client->print("\r\n");
    _client->print(body);
  } else {
    _client->print("\r\n");
  }
  
  // Read response with timeout
  unsigned long startTime = millis();
  while (_client->connected() && millis() - startTime < _timeout) {
    if (_client->available()) break;
    delay(1);
  }
  
  // Read headers
  String headers;
  while (_client->connected() || _client->available()) {
    String line = _client->readStringUntil('\n');
    headers += line;
    if (line == "\r") break;
    if (millis() - startTime > _timeout) {
      _client->stop();
      _lastError = "Timeout reading headers";
      return false;
    }
  }
  
  // Read body
  response = "";
  while (_client->connected() || _client->available()) {
    char c = _client->read();
    if (c >= 0) response += c;
    if (millis() - startTime > _timeout) break;
  }
  
  _client->stop();
  
  // Check HTTP status
  if (headers.indexOf("200") == -1 && headers.indexOf("201") == -1) {
    _lastError = "HTTP error: " + headers.substring(0, headers.indexOf('\r'));
    return false;
  }
  
  return true;
}

bool MicroDashboard::_parseDashboard(const String& json, DashboardConfig& dashboard) {
  DynamicJsonDocument doc(8192);
  DeserializationError err = deserializeJson(doc, json);
  if (err) {
    _lastError = "JSON parse error: " + String(err.c_str());
    return false;
  }
  
  dashboard.id = doc["id"].as<String>();
  dashboard.name = doc["name"].as<String>();
  dashboard.width = doc["width"] | 128;
  dashboard.height = doc["height"] | 64;
  dashboard.refreshInterval = doc["refresh_interval"] | 15000;
  dashboard.createdAt = doc["created_at"] | 0;
  
  JsonArray widgets = doc["widgets"];
  dashboard.widgetCount = widgets.size();
  if (dashboard.widgetCount > 0) {
    dashboard.widgets = new WidgetConfig[dashboard.widgetCount];
    for (int i = 0; i < dashboard.widgetCount; i++) {
      JsonObject w = widgets[i];
      dashboard.widgets[i].id = w["id"].as<String>();
      dashboard.widgets[i].type = _parseWidgetType(w["type"].as<String>());
      dashboard.widgets[i].x = w["x"] | 0;
      dashboard.widgets[i].y = w["y"] | 0;
      dashboard.widgets[i].width = w["width"] | 32;
      dashboard.widgets[i].height = w["height"] | 16;
      dashboard.widgets[i].expression = w["expression"].as<String>();
      dashboard.widgets[i].unit = w["unit"].as<String>();
      dashboard.widgets[i].minVal = w["options"]["min"] | 0;
      dashboard.widgets[i].maxVal = w["options"]["max"] | 100;
      dashboard.widgets[i].thresholdWarn = w["options"]["threshold_warn"] | 0;
      dashboard.widgets[i].thresholdCrit = w["options"]["threshold_crit"] | 0;
      dashboard.widgets[i].precision = w["options"]["precision"] | 0;
      dashboard.widgets[i].label = w["options"]["label"].as<String>();
    }
  } else {
    dashboard.widgets = nullptr;
  }
  
  return true;
}

bool MicroDashboard::_parseRenderedDashboard(const String& json, RenderedDashboard& dashboard) {
  DynamicJsonDocument doc(8192);
  DeserializationError err = deserializeJson(doc, json);
  if (err) {
    _lastError = "JSON parse error: " + String(err.c_str());
    return false;
  }
  
  dashboard.id = doc["id"].as<String>();
  dashboard.name = doc["name"].as<String>();
  dashboard.width = doc["width"] | 128;
  dashboard.height = doc["height"] | 64;
  dashboard.renderedAt = doc["rendered_at"] | 0;
  
  JsonArray widgets = doc["widgets"];
  dashboard.widgetCount = widgets.size();
  if (dashboard.widgetCount > 0) {
    dashboard.widgets = new RenderedWidget[dashboard.widgetCount];
    for (int i = 0; i < dashboard.widgetCount; i++) {
      JsonObject w = widgets[i];
      dashboard.widgets[i].id = w["id"].as<String>();
      dashboard.widgets[i].type = _parseWidgetType(w["type"].as<String>());
      dashboard.widgets[i].x = w["x"] | 0;
      dashboard.widgets[i].y = w["y"] | 0;
      dashboard.widgets[i].width = w["width"] | 32;
      dashboard.widgets[i].height = w["height"] | 16;
      dashboard.widgets[i].value = w["value"] | 0.0;
      dashboard.widgets[i].text = w["text"].as<String>();
      dashboard.widgets[i].unit = w["unit"].as<String>();
      dashboard.widgets[i].status = _parseStatus(w["status"].as<String>());
    }
  } else {
    dashboard.widgets = nullptr;
  }
  
  return true;
}

MicroDashboard::WidgetType MicroDashboard::_parseWidgetType(const String& typeStr) {
  if (typeStr == "gauge") return WIDGET_GAUGE;
  if (typeStr == "sparkline") return WIDGET_SPARKLINE;
  if (typeStr == "status") return WIDGET_STATUS;
  if (typeStr == "numeric") return WIDGET_NUMERIC;
  if (typeStr == "progress") return WIDGET_PROGRESS;
  if (typeStr == "text") return WIDGET_TEXT;
  return WIDGET_UNKNOWN;
}

MicroDashboard::WidgetStatus MicroDashboard::_parseStatus(const String& statusStr) {
  if (statusStr == "ok") return STATUS_OK;
  if (statusStr == "warn") return STATUS_WARN;
  if (statusStr == "crit") return STATUS_CRIT;
  return STATUS_PENDING;
}

void MicroDashboard::_freeWidgets(WidgetConfig* widgets, int count) {
  delete[] widgets;
}

void MicroDashboard::_freeRenderedWidgets(RenderedWidget* widgets, int count) {
  delete[] widgets;
}

// --- Widget Rendering (stubs - implement for your display library) ---

void MicroDashboard::renderWidgetGauge(RenderedWidget& w, int16_t centerX, int16_t centerY, int16_t radius) {
  // Implement for your display (Adafruit_GFX, U8g2, TFT_eSPI, etc.)
  // Example using Adafruit_GFX:
  // float pct = (w.value - w.minVal) / (w.maxVal - w.minVal) * 100;
  // pct = constrain(pct, 0, 100);
  // Draw arc, needle, labels
}

void MicroDashboard::renderWidgetNumeric(RenderedWidget& w, int16_t x, int16_t y) {
  // Implement: draw w.text at (x,y)
}

void MicroDashboard::renderWidgetStatus(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  // Implement: draw colored indicator based on w.status
}

void MicroDashboard::renderWidgetProgress(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  // Implement: draw progress bar
}

void MicroDashboard::renderWidgetText(RenderedWidget& w, int16_t x, int16_t y) {
  // Implement: draw w.text at (x,y)
}

void MicroDashboard::renderWidgetSparkline(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  // Implement: draw mini line chart (requires historical data)
}

void MicroDashboard::renderWidget(RenderedWidget& w, int16_t x, int16_t y, int16_t w_px, int16_t h_px) {
  switch (w.type) {
    case WIDGET_GAUGE:      renderWidgetGauge(w, x + w_px/2, y + h_px/2, min(w_px, h_px)/2); break;
    case WIDGET_NUMERIC:    renderWidgetNumeric(w, x, y); break;
    case WIDGET_STATUS:     renderWidgetStatus(w, x, y, w_px, h_px); break;
    case WIDGET_PROGRESS:   renderWidgetProgress(w, x, y, w_px, h_px); break;
    case WIDGET_TEXT:       renderWidgetText(w, x, y); break;
    case WIDGET_SPARKLINE:  renderWidgetSparkline(w, x, y, w_px, h_px); break;
    default: break;
  }
}
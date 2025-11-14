package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/a-h/templ"

	"go-log/internal/api/logics"
	"go-log/internal/api/models"
	"go-log/internal/config"
	"go-log/internal/utils"
	webstatic "go-log/web"
	"go-log/web/views"
)

type TokenClaims struct {
	BusinessID int `json:"business_id"`
}

type FilterRequest struct {
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	TableName string `json:"table_name,omitempty"`
}

var remoteConfigHTTPClient = &http.Client{Timeout: 10 * time.Second}

func MonitoringRoutes() {
	// Initialize monitoring configuration at startup
	logics.InitMonitoringConfig()

	// Serve embedded assets
	jsDir := http.StripPrefix("/js/", webstatic.GetJSHandler())
	assetsDir := http.StripPrefix("/assets/", webstatic.GetAssetsHandler())

	configHandler := func(w http.ResponseWriter, r *http.Request) {
		cfg := logics.GetMonitoringConfig()
		
		// Handle remote server config requests (proxy requests)
		if remoteTarget := strings.TrimSpace(r.URL.Query().Get("remote")); remoteTarget != "" {
			// Only allow proxy requests if local dashboard is enabled
			if !IsDashboardEnabled() {
				http.NotFound(w, r)
				return
			}
			proxyRemoteServerConfig(w, remoteTarget, cfg)
			return
		}

		// Handle direct config requests - allow these even when dashboard is disabled
		// because remote servers need to serve their config to other servers

		refresh := 2.0
		if d, err := time.ParseDuration(cfg.RefreshTime); err == nil && d > 0 {
			refresh = d.Seconds()
		}

        payload := map[string]any{
            "refresh_interval_seconds": refresh,
            "heartbeat":                cfg.Heartbeat,
            "servers":                  cfg.Servers,
            "storage":                  cfg.Storage,
            "path":                     cfg.Path,
            "persist_server_logs":      cfg.PersistServerLogs,
        }

		jsonData, err := json.Marshal(payload)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal config"}`)
			return
		}

		setHeader(w, http.StatusOK, string(jsonData))
	}

	// Serve dashboard UI via templ
	http.HandleFunc("/", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if !IsDashboardEnabled() {
			http.NotFound(w, r)
			return
		}

		cfg := logics.GetMonitoringConfig()
		defaultRange := config.GetEnvConfig().GetDashboardDefaultRange()
		dashboard := views.DashboardPage(views.DashboardProps{Config: cfg, DefaultRangePreset: defaultRange})
		templ.Handler(dashboard).ServeHTTP(w, r)
	}))))

	// Serve HTMX component fragments using templ
	registerComponent := func(path string, builder func() templ.Component) {
		http.HandleFunc(path, RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
			if !IsDashboardEnabled() {
				http.NotFound(w, r)
				return
			}
			templ.Handler(builder()).ServeHTTP(w, r)
		}))))
	}

	registerComponent("/components/background.html", views.BackgroundComponent)
	registerComponent("/components/initial-loading.html", views.InitialLoadingOverlay)
	registerComponent("/components/charts.html", views.ChartsSection)
	registerComponent("/components/metrics.html", func() templ.Component {
		return views.MetricsSection()
	})
	registerComponent("/components/heartbeats.html", func() templ.Component {
		return views.HeartbeatSection(false)
	})
	registerComponent("/components/chrome.html", func() templ.Component {
		return views.ChromeComponent()
	})
	registerComponent("/components/hero.html", func() templ.Component {
		cfg := logics.GetMonitoringConfig()
		defaultRange := config.GetEnvConfig().GetDashboardDefaultRange()
		return views.HeroSection(views.HeroProps{RefreshLabel: refreshLabelFromConfig(cfg), DefaultRangePreset: defaultRange})
	})

	// Serve dashboard JavaScript bundle
	http.HandleFunc("/js/", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		if !IsDashboardEnabled() {
			http.NotFound(w, r)
			return
		}
		jsDir.ServeHTTP(w, r)
	}))))

	// Serve compiled assets (CSS)
	http.HandleFunc("/assets/", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		if !IsDashboardEnabled() {
			http.NotFound(w, r)
			return
		}
		assetsDir.ServeHTTP(w, r)
	}))))

	// Serve monitoring configuration for UI
	http.HandleFunc("/api/v1/server-config", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet, http.MethodOptions)(configHandler))))

	// Serve available tables endpoint
	tablesHandler := func(w http.ResponseWriter, r *http.Request) {
		tables := utils.GetAvailableTables()

		payload := map[string]any{
			"tables": tables,
			"count":  len(tables),
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal tables data"}`)
			return
		}

		setHeader(w, http.StatusOK, string(jsonData))
	}

	http.HandleFunc("/api/v1/tables", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodGet, http.MethodOptions)(tablesHandler))))

	monitoringHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check token only in production if CHECK_TOKEN_IN_PRODUCTION is enabled
		if IsProduction() && ShouldCheckTokenInProduction() {
			_, err := ValidateTokenAndParseGeneric[TokenClaims](r)
			if err != nil {
				setHeader(w, http.StatusUnauthorized, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
				return
			}
		}

		// Parse optional filter from request body
		var filter FilterRequest
		if r.ContentLength > 0 {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				setHeader(w, http.StatusBadRequest, `{"status":false, "error": "Failed to read request body"}`)
				return
			}
			defer r.Body.Close()

			if len(body) > 0 {
				err = json.Unmarshal(body, &filter)
				if err != nil {
					setHeader(w, http.StatusBadRequest, `{"status":false, "error": "Invalid JSON format"}`)
					return
				}
			}
		}

		// Generate monitoring data based on filter
		var responseArray []any
		var err error

		if filter.From != "" || filter.To != "" || filter.TableName != "" {
			// Use filtered data from database (with optional table specification)
			filteredData, err := logics.MonitoringDataGeneratorWithTableFilter(filter.TableName, filter.From, filter.To)
			if err != nil {
				setHeader(w, http.StatusInternalServerError, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
				return
			}
			responseArray = filteredData
		} else {
			// Use current metrics and wrap in array
			currentData, err := logics.MonitoringDataGenerator()
			if err != nil {
				setHeader(w, http.StatusInternalServerError, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
				return
			}
			responseArray = []any{currentData}
		}

		// Convert to JSON
		jsonData, err := json.Marshal(responseArray)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal data"}`)
			return
		}

		setHeader(w, http.StatusOK, string(jsonData))
	}

	// Apply middleware to restrict to POST method only
	http.HandleFunc("/monitoring", RateLimitMiddleware(CORSMiddleware(MethodMiddleware(http.MethodPost, http.MethodOptions)(monitoringHandler))))
}

func proxyRemoteServerConfig(w http.ResponseWriter, target string, cfg *models.MonitoringConfig) {
	normalized, err := normalizeRemoteAddress(target)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid remote address: %v", err))
		return
	}

	if cfg == nil || !isRemoteServerAllowed(normalized, cfg.Servers) {
		writeJSONError(w, http.StatusForbidden, "remote server is not allowed")
		return
	}

	// Try to fetch config from remote server first
	remoteURL := fmt.Sprintf("%s/api/v1/server-config", strings.TrimRight(normalized, "/"))
	req, err := http.NewRequest(http.MethodGet, remoteURL, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("failed to create remote request: %v", err))
		return
	}

	resp, err := remoteConfigHTTPClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("remote config request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("failed to read remote response: %v", err))
		return
	}

	// If remote server returns 404, it's likely an older version without the endpoint
	// Fall back to generating a compatible config based on local configuration
	if resp.StatusCode == http.StatusNotFound {
		fallbackConfig := generateFallbackRemoteConfig(normalized, cfg)
		jsonData, err := json.Marshal(fallbackConfig)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to marshal fallback config: %v", err))
			return
		}
		setHeader(w, http.StatusOK, string(jsonData))
		return
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if len(body) == 0 {
			writeJSONError(w, resp.StatusCode, fmt.Sprintf("remote server returned status %d", resp.StatusCode))
			return
		}
		setHeader(w, resp.StatusCode, string(body))
		return
	}

	setHeader(w, http.StatusOK, string(body))
}

// generateFallbackRemoteConfig creates a compatible configuration for older remote servers
// that don't have the /api/v1/server-config endpoint
func generateFallbackRemoteConfig(_ string, localCfg *models.MonitoringConfig) map[string]any {
	// Use local config as base but adapt for remote server context
	refresh := 2.0
	if d, err := time.ParseDuration(localCfg.RefreshTime); err == nil && d > 0 {
		refresh = d.Seconds()
	}

	// For older remote servers, provide a minimal config that works with the frontend
	// Since they don't have their own server list, we show empty servers and heartbeat
    return map[string]any{
        "refresh_interval_seconds": refresh,
        "heartbeat":                []any{}, // Older servers likely don't have complex heartbeat configs
        "servers":                  []any{}, // Older servers don't monitor other servers
        "storage":                  localCfg.Storage,
        "path":                     localCfg.Path,
        "persist_server_logs":      localCfg.PersistServerLogs,
    }
}

func normalizeRemoteAddress(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty remote address")
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "http://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("missing host")
	}
	cleanPath := path.Clean(parsed.Path)
	if cleanPath == "." {
		cleanPath = ""
	}
	if cleanPath == "/" {
		cleanPath = ""
	}
	cleanPath = strings.TrimRight(cleanPath, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	base := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	if cleanPath != "" {
		base += cleanPath
	}
	return base, nil
}

func isRemoteServerAllowed(target string, servers []models.ServerEndpoint) bool {
	for _, server := range servers {
		normalized, err := normalizeRemoteAddress(server.Address)
		if err != nil {
			continue
		}
		if normalized == target {
			return true
		}
	}
	return false
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	resp, _ := json.Marshal(map[string]any{
		"status": false,
		"error":  message,
	})
	setHeader(w, status, string(resp))
}

func refreshLabelFromConfig(cfg *models.MonitoringConfig) string {
	if cfg == nil || cfg.RefreshTime == "" {
		return "2s"
	}
	if d, err := time.ParseDuration(cfg.RefreshTime); err == nil && d > 0 {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return cfg.RefreshTime
}

// Individual API Handlers for Chi Router

// ServerConfigHandler handles server configuration requests
func ServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	cfg := logics.GetMonitoringConfig()
	
	// Handle remote server config requests (proxy requests)
	if remoteTarget := strings.TrimSpace(r.URL.Query().Get("remote")); remoteTarget != "" {
		// Only allow proxy requests if local dashboard is enabled
		if !IsDashboardEnabled() {
			http.NotFound(w, r)
			return
		}
		proxyRemoteServerConfig(w, remoteTarget, cfg)
		return
	}

	// Handle direct config requests - allow these even when dashboard is disabled
	// because remote servers need to serve their config to other servers
	refresh := 2.0
	if d, err := time.ParseDuration(cfg.RefreshTime); err == nil && d > 0 {
		refresh = d.Seconds()
	}

    payload := map[string]any{
        "refresh_interval_seconds": refresh,
        "heartbeat":                cfg.Heartbeat,
        "servers":                  cfg.Servers,
        "storage":                  cfg.Storage,
        "path":                     cfg.Path,
        "persist_server_logs":      cfg.PersistServerLogs,
    }

	jsonData, err := json.Marshal(payload)
	if err != nil {
		setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal config"}`)
		return
	}

	setHeader(w, http.StatusOK, string(jsonData))
}

// TablesHandler serves available tables endpoint
func TablesHandler(w http.ResponseWriter, r *http.Request) {
	tables := utils.GetAvailableTables()

	payload := map[string]any{
		"tables": tables,
		"count":  len(tables),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal tables data"}`)
		return
	}

	setHeader(w, http.StatusOK, string(jsonData))
}

// MonitoringHandler handles monitoring data requests
func MonitoringHandler(w http.ResponseWriter, r *http.Request) {
	// Check token only in production if CHECK_TOKEN_IN_PRODUCTION is enabled
	if IsProduction() && ShouldCheckTokenInProduction() {
		_, err := ValidateTokenAndParseGeneric[TokenClaims](r)
		if err != nil {
			setHeader(w, http.StatusUnauthorized, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
			return
		}
	}

	// Parse optional filter from request body
	var filter FilterRequest
	if r.ContentLength > 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			setHeader(w, http.StatusBadRequest, `{"status":false, "error": "Failed to read request body"}`)
			return
		}
		defer r.Body.Close()

		if len(body) > 0 {
			err = json.Unmarshal(body, &filter)
			if err != nil {
				setHeader(w, http.StatusBadRequest, `{"status":false, "error": "Invalid JSON format"}`)
				return
			}
		}
	}

	// Generate monitoring data based on filter
	var responseArray []any
	var err error

	if filter.From != "" || filter.To != "" || filter.TableName != "" {
		// Use filtered data from database (with optional table specification)
		filteredData, err := logics.MonitoringDataGeneratorWithTableFilter(filter.TableName, filter.From, filter.To)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
			return
		}
		responseArray = filteredData
	} else {
		// Use current metrics and wrap in array
		currentData, err := logics.MonitoringDataGenerator()
		if err != nil {
			setHeader(w, http.StatusInternalServerError, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
			return
		}
		responseArray = []any{currentData}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(responseArray)
	if err != nil {
		setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal data"}`)
		return
	}

	setHeader(w, http.StatusOK, string(jsonData))
}

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/a-h/templ"

	"go-log/internal/api/logics"
	"go-log/internal/api/models"
	"go-log/web/views"
)

type TokenClaims struct {
	BusinessID int `json:"business_id"`
}

type FilterRequest struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

func MonitoringRoutes() {
	// Initialize monitoring configuration at startup
	logics.InitMonitoringConfig()

	jsDir := http.StripPrefix("/js/", http.FileServer(http.Dir(filepath.Join("web", "js"))))
	assetsDir := http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join("web", "assets"))))

	configHandler := func(w http.ResponseWriter, r *http.Request) {
		cfg := logics.GetMonitoringConfig()
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
	http.HandleFunc("/", CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		cfg := logics.GetMonitoringConfig()
		dashboard := views.DashboardPage(views.DashboardProps{Config: cfg})
		templ.Handler(dashboard).ServeHTTP(w, r)
	})))

	// Serve HTMX component fragments using templ
	registerComponent := func(path string, builder func() templ.Component) {
		http.HandleFunc(path, CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
			templ.Handler(builder()).ServeHTTP(w, r)
		})))
	}

	registerComponent("/components/background.html", views.BackgroundComponent)
	registerComponent("/components/initial-loading.html", views.InitialLoadingOverlay)
	registerComponent("/components/charts.html", views.ChartsSection)
	registerComponent("/components/metrics.html", func() templ.Component {
		return views.MetricsSection()
	})
	registerComponent("/components/heartbeats.html", func() templ.Component {
		return views.HeartbeatSection()
	})
	registerComponent("/components/chrome.html", func() templ.Component {
		return views.ChromeComponent()
	})
	registerComponent("/components/hero.html", func() templ.Component {
		cfg := logics.GetMonitoringConfig()
		return views.HeroSection(views.HeroProps{RefreshLabel: refreshLabelFromConfig(cfg)})
	})

	// Serve dashboard JavaScript bundle
	http.HandleFunc("/js/", CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		jsDir.ServeHTTP(w, r)
	})))

	// Serve compiled assets (CSS)
	http.HandleFunc("/assets/", CORSMiddleware(MethodMiddleware(http.MethodGet)(func(w http.ResponseWriter, r *http.Request) {
		assetsDir.ServeHTTP(w, r)
	})))

	// Serve monitoring configuration for UI
	http.HandleFunc("/api/v1/server-config", CORSMiddleware(MethodMiddleware(http.MethodGet, http.MethodOptions)(configHandler)))

	monitoringHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check token only in production
		if IsProduction() {
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

		if filter.From != "" || filter.To != "" {
			// Use filtered data from database
			filteredData, err := logics.MonitoringDataGeneratorWithFilter(filter.From, filter.To)
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
	http.HandleFunc("/monitoring", CORSMiddleware(MethodMiddleware(http.MethodPost, http.MethodOptions)(monitoringHandler)))
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

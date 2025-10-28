package handlers

import (
	"encoding/json"
	"fmt"
	"go-log/internal/api/logics"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

type TokenClaims struct {
	BusinessID int `json:"business_id"`
}

type FilterRequest struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

func MonitoringRoutes() {
	// Initialize servers configuration at startup
	logics.InitServersConfig()

	dashboardPath := filepath.Join("web", "dashboard.html")

	dashboardHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, dashboardPath)
	}

	// Serve dashboard UI
	http.HandleFunc("/", MethodMiddleware(http.MethodGet)(dashboardHandler))

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

		cfg := logics.GetMonitoringConfig()
		refresh := 2.0
		if d, err := time.ParseDuration(cfg.RefreshTime); err == nil && d > 0 {
			refresh = d.Seconds()
		}

		payload := map[string]any{
			"data":                     responseArray,
			"refresh_interval_seconds": refresh,
		}

		// Convert to JSON
		jsonData, err := json.Marshal(payload)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal data"}`)
			return
		}

		setHeader(w, http.StatusOK, string(jsonData))
	}

	// Apply middleware to restrict to POST method only
	http.HandleFunc("/monitoring", MethodMiddleware(http.MethodPost)(monitoringHandler))
}

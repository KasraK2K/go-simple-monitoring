package handlers

import (
	"encoding/json"
	"fmt"
	"go-log/internal/api/logics"
	"net/http"
)

type TokenClaims struct {
	BusinessID int `json:"business_id"`
}

func MonitoringRoutes() {
	// Initialize servers configuration at startup
	logics.InitServersConfig()

	http.HandleFunc("/monitoring", func(w http.ResponseWriter, r *http.Request) {
		// Check token and method only in production
		if IsProduction() {
			_, err := ValidateTokenAndParseGeneric[TokenClaims](r)
			if err != nil {
				setHeader(w, http.StatusUnauthorized, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
				return
			}

			if !checkMethod(r, w, http.MethodGet) {
				return
			}
		}

		// Generate monitoring data
		monitoringData, err := logics.MonitoringDataGenerator()
		if err != nil {
			setHeader(w, http.StatusInternalServerError, fmt.Sprintf(`{"status":false, "error": "%s"}`, err.Error()))
			return
		}

		// Convert to JSON
		jsonData, err := json.Marshal(monitoringData)
		if err != nil {
			setHeader(w, http.StatusInternalServerError, `{"status":false, "error": "Failed to marshal data"}`)
			return
		}

		setHeader(w, http.StatusOK, string(jsonData))
	})
}

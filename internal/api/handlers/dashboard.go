package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"go-log/internal/api/logics"
	"go-log/internal/api/models"
	"go-log/internal/config"
	"go-log/web/views"
)

// DashboardHandler serves the main dashboard page
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	cfg := logics.GetMonitoringConfig()
	defaultRange := config.GetEnvConfig().GetDashboardDefaultRange()
	dashboard := views.DashboardPage(views.DashboardProps{Config: cfg, DefaultRangePreset: defaultRange})
	templ.Handler(dashboard).ServeHTTP(w, r)
}

// Component handlers for HTMX endpoints
func BackgroundComponentHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.BackgroundComponent()).ServeHTTP(w, r)
}

func InitialLoadingOverlayHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.InitialLoadingOverlay()).ServeHTTP(w, r)
}

func ChartsSectionHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.ChartsSection()).ServeHTTP(w, r)
}

func MetricsSectionHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.MetricsSection()).ServeHTTP(w, r)
}

func HeartbeatSectionHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.HeartbeatSection(false)).ServeHTTP(w, r)
}

func ChromeComponentHandler(w http.ResponseWriter, r *http.Request) {
	templ.Handler(views.ChromeComponent()).ServeHTTP(w, r)
}

func HeroSectionHandler(w http.ResponseWriter, r *http.Request) {
	cfg := logics.GetMonitoringConfig()
	defaultRange := config.GetEnvConfig().GetDashboardDefaultRange()
	templ.Handler(views.HeroSection(views.HeroProps{RefreshLabel: refreshLabelFromConfigForRoutes(cfg), DefaultRangePreset: defaultRange})).ServeHTTP(w, r)
}


// refreshLabelFromConfigForRoutes generates refresh label from config
// Note: This is a local version to avoid conflicts with the existing function
func refreshLabelFromConfigForRoutes(cfg *models.MonitoringConfig) string {
	if cfg == nil || cfg.RefreshTime == "" {
		return "2s"
	}
	if d, err := time.ParseDuration(cfg.RefreshTime); err == nil && d > 0 {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	return cfg.RefreshTime
}
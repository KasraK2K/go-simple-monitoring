package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go-log/internal/api/handlers"
	"go-log/internal/api/logics"
	webstatic "go-log/web"
)

// NewRouter creates and configures the main Chi router
func NewRouter() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Initialize monitoring configuration at startup
	logics.InitMonitoringConfig()

	// Setup route groups
	setupDashboardRoutes(r)
	setupAPIRoutes(r)
	setupStaticRoutes(r)

	return r
}

// setupDashboardRoutes configures all dashboard-related routes
func setupDashboardRoutes(r chi.Router) {
	// Dashboard routes group - only active when dashboard is enabled
	r.Group(func(r chi.Router) {
		// Dashboard-specific middleware
		r.Use(dashboardMiddleware)
		r.Use(wrapHandlerFuncMiddleware(handlers.RateLimitMiddleware))
		r.Use(wrapHandlerFuncMiddleware(handlers.CORSMiddleware))

		// Main dashboard page
		r.Get("/", handlers.DashboardHandler)

		// HTMX component endpoints
		r.Get("/components/background.html", handlers.BackgroundComponentHandler)
		r.Get("/components/initial-loading.html", handlers.InitialLoadingOverlayHandler)
		r.Get("/components/charts.html", handlers.ChartsSectionHandler)
		r.Get("/components/metrics.html", handlers.MetricsSectionHandler)
		r.Get("/components/heartbeats.html", handlers.HeartbeatSectionHandler)
		r.Get("/components/chrome.html", handlers.ChromeComponentHandler)
		r.Get("/components/hero.html", handlers.HeroSectionHandler)
	})
}

// setupAPIRoutes configures all API endpoints
func setupAPIRoutes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		// API middleware
		r.Use(wrapHandlerFuncMiddleware(handlers.RateLimitMiddleware))
		r.Use(wrapHandlerFuncMiddleware(handlers.CORSMiddleware))

		// Server configuration endpoint - available regardless of dashboard status
		r.With(methodMiddleware("GET", "OPTIONS")).Get("/server-config", handlers.ServerConfigHandler)
		
		// Tables endpoint - requires dashboard enabled
		r.Group(func(r chi.Router) {
			r.Use(dashboardMiddleware)
			r.With(methodMiddleware("GET", "OPTIONS")).Get("/tables", handlers.TablesHandler)
		})

		// Monitoring endpoint - core functionality, always available
		r.With(methodMiddleware("POST", "OPTIONS")).Post("/monitoring", handlers.MonitoringHandler)
	})
}

// setupStaticRoutes configures static file serving
func setupStaticRoutes(r chi.Router) {
	// Static files group - only active when dashboard is enabled
	r.Group(func(r chi.Router) {
		r.Use(dashboardMiddleware)
		r.Use(wrapHandlerFuncMiddleware(handlers.RateLimitMiddleware))
		r.Use(wrapHandlerFuncMiddleware(handlers.CORSMiddleware))

		// JavaScript files
		r.Handle("/js/*", http.StripPrefix("/js/", webstatic.GetJSHandler()))
		
		// CSS and other assets
		r.Handle("/assets/*", http.StripPrefix("/assets/", webstatic.GetAssetsHandler()))
	})
}

// dashboardMiddleware checks if dashboard is enabled
func dashboardMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !handlers.IsDashboardEnabled() {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// methodMiddleware restricts HTTP methods for endpoints
func methodMiddleware(allowedMethods ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if method is allowed
			for _, method := range allowedMethods {
				if r.Method == method {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Method not allowed
			w.Header().Set("Allow", joinMethods(allowedMethods))
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		})
	}
}

// joinMethods joins HTTP methods with comma separator
func joinMethods(methods []string) string {
	if len(methods) == 0 {
		return ""
	}
	if len(methods) == 1 {
		return methods[0]
	}
	
	result := methods[0]
	for _, method := range methods[1:] {
		result += ", " + method
	}
	return result
}

// wrapHandlerFuncMiddleware adapts http.HandlerFunc middleware to work with Chi's http.Handler middleware
func wrapHandlerFuncMiddleware(middleware func(http.HandlerFunc) http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return middleware(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
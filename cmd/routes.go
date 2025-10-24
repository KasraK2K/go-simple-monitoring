package main

import "go-log/internal/api/handlers"

func RegisterRouter() {
	handlers.MonitoringRoutes()
}

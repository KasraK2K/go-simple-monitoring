package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-log/internal/api/handlers"
	"go-log/internal/utils"
)

func main() {
	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Setup cleanup on exit
	go func() {
		<-c
		log.Println("Shutting down server...")
		utils.CloseDatabase() // Close database connection if open
		os.Exit(0)
	}()

	handlers.MonitoringRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3500"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Println("Server running on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

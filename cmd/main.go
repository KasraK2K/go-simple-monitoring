package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"go-log/internal/api/handlers"
	"go-log/internal/api/logics"
	"go-log/internal/config"
	"go-log/internal/utils"
)

func loadEnvFile() {
	// Try multiple possible locations for .env file
	possiblePaths := []string{
		".env",
		"../.env",
		"../../.env",
		"../../../.env",
	}
	
	// Also try based on executable location
	if ex, err := os.Executable(); err == nil {
		exDir := filepath.Dir(ex)
		possiblePaths = append(possiblePaths, filepath.Join(exDir, ".env"))
	}
	
	// Try current working directory
	if wd, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(wd, ".env"))
	}
	
	for _, envPath := range possiblePaths {
		if file, err := os.Open(envPath); err == nil {
			defer file.Close()
			log.Printf("Loading .env from: %s", envPath)
			
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					os.Setenv(key, value)
				}
			}
			return // Found and loaded .env file
		}
	}
}

func main() {
	// Load .env file before anything else
	loadEnvFile()
	
	// Initialize environment configuration
	config.InitEnvConfig()
	envConfig := config.GetEnvConfig()
	
	// Initialize timezone configuration
	utils.InitTimeConfig()
	
	// Initialize HTTP client configuration
	utils.InitHTTPConfig()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Setup cleanup on exit
	go func() {
		<-c
		log.Println("Shutting down server...")
		
		// Clean up all monitoring goroutines
		logics.CleanupAllGoroutines()
		
		// Close HTTP client connections
		utils.CloseHTTPClient()
		
		// Close database connection if open
		utils.CloseDatabase()
		
		log.Println("Server shutdown completed")
		os.Exit(0)
	}()

	handlers.MonitoringRoutes()

	addr := fmt.Sprintf(":%s", envConfig.Port)
	log.Println("Server running on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"go-log/internal/utils"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

var (
	monitoringConfig   *models.MonitoringConfig
	serversConfigOnce  sync.Once
	serversConfigErr   error
	serversConfigMutex sync.RWMutex
	lastConfigModTime  time.Time
	loggingTicker      *time.Ticker
	loggingStopChan    chan struct{}
)

// InitServersConfig loads the servers configuration once at startup
func InitServersConfig() {
	serversConfigOnce.Do(func() {
		reloadServersConfig()
	})
}

// InitServersConfigCLI loads configuration for CLI mode without auto-logging
func InitServersConfigCLI() {
	serversConfigOnce.Do(func() {
		reloadServersConfigCLI()
	})
}

// GetServersConfig returns the cached servers configuration
// It automatically checks if the config file has been modified and reloads if needed
func GetServersConfig() []models.ServerConfig {
	InitServersConfig() // Ensure config is loaded

	// Check if we should reload (every 30 seconds max)
	serversConfigMutex.RLock()
	shouldCheck := time.Since(lastConfigModTime) > 30*time.Second
	serversConfigMutex.RUnlock()

	if shouldCheck {
		checkAndReloadConfig()
	}

	serversConfigMutex.RLock()
	defer serversConfigMutex.RUnlock()
	if monitoringConfig != nil {
		return monitoringConfig.Servers
	}
	return []models.ServerConfig{}
}

// GetMonitoringConfig returns the full monitoring configuration
func GetMonitoringConfig() *models.MonitoringConfig {
	InitServersConfig() // Ensure config is loaded

	serversConfigMutex.RLock()
	defer serversConfigMutex.RUnlock()
	return monitoringConfig
}

// ReloadServersConfig forces a reload of the servers configuration
func ReloadServersConfig() {
	reloadServersConfig()
}

func reloadServersConfig() {
	newConfig, err := readConfigFromFile()

	serversConfigMutex.Lock()
	defer serversConfigMutex.Unlock()

	if err != nil {
		// Keep existing config on error, or use empty config if first load
		if monitoringConfig == nil {
			monitoringConfig = &models.MonitoringConfig{
				Path:        "./logs",
				RefreshTime: "2s",
				Servers:     []models.ServerConfig{},
			}
		}
		serversConfigErr = nil // Don't propagate errors for monitoring
	} else {
		// Stop existing logging if refresh time changed
		if monitoringConfig != nil && monitoringConfig.RefreshTime != newConfig.RefreshTime {
			stopAutoLogging()
		}

		monitoringConfig = newConfig
		serversConfigErr = nil

		// Initialize logger and start auto-logging
		utils.InitLogger(monitoringConfig)
		startAutoLogging()
	}
	lastConfigModTime = time.Now()
}

func reloadServersConfigCLI() {
	newConfig, err := readConfigFromFile()

	serversConfigMutex.Lock()
	defer serversConfigMutex.Unlock()

	if err != nil {
		// Keep existing config on error, or use empty config if first load
		if monitoringConfig == nil {
			monitoringConfig = &models.MonitoringConfig{
				Path:        "./logs",
				RefreshTime: "2s",
				Servers:     []models.ServerConfig{},
			}
		}
		serversConfigErr = nil // Don't propagate errors for monitoring
	} else {
		monitoringConfig = newConfig
		serversConfigErr = nil

		// CLI mode: DO NOT start auto-logging
	}
	lastConfigModTime = time.Now()
}

func checkAndReloadConfig() {
	// Get current file modification time
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	projectRoot := findProjectRoot(cwd)
	configPath := filepath.Join(projectRoot, "configs.json")

	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	serversConfigMutex.RLock()
	lastCheck := lastConfigModTime
	serversConfigMutex.RUnlock()

	// If file is newer than our last reload, reload it
	if fileInfo.ModTime().After(lastCheck) {
		reloadServersConfig()
	} else {
		// Update last check time even if no reload needed
		serversConfigMutex.Lock()
		lastConfigModTime = time.Now()
		serversConfigMutex.Unlock()
	}
}

func MonitoringDataGenerator() (*models.SystemMonitoring, error) {
	monitoring := &models.SystemMonitoring{
		Timestamp: time.Now(),
	}

	// Get CPU usage
	cpuData, err := getCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}
	monitoring.CPU = cpuData

	// Get disk space
	diskSpace, err := getDiskSpace("/")
	if err != nil {
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}
	monitoring.DiskSpace = diskSpace

	// Get RAM usage
	ramUsage, err := getRAMUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get RAM usage: %w", err)
	}
	monitoring.RAM = ramUsage

	// Get heartbeat data
	servers := GetServersConfig()
	heartbeat := checkServerHeartbeats(servers)
	monitoring.Heartbeat = heartbeat

	return monitoring, nil
}

func getCPUInfo() (models.CPU, error) {
	cpuInfo := models.CPU{
		CoreCount:    runtime.NumCPU(),
		Goroutines:   runtime.NumGoroutine(),
		Architecture: runtime.GOARCH,
	}

	// Get CPU usage percentage
	cpuUsage, err := getCPUUsagePercent()
	if err != nil {
		// Fallback to goroutine-based estimation if real CPU usage fails
		cpuUsage = float64(cpuInfo.Goroutines) / float64(cpuInfo.CoreCount*10) * 100
		if cpuUsage > 100 {
			cpuUsage = 100
		}
	}
	cpuInfo.UsagePercent = math.Round(cpuUsage*100) / 100

	// Get load average (Unix-like systems)
	loadAvg, err := getLoadAverage()
	if err != nil {
		cpuInfo.LoadAverage = "unavailable"
	} else {
		cpuInfo.LoadAverage = loadAvg
	}

	return cpuInfo, nil
}

func getCPUUsagePercent() (float64, error) {
	// This is a simplified CPU usage calculation
	// For macOS/Linux, we can use system commands
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return getCPUUsageUnix()
	}

	// Fallback for other systems
	return 0, fmt.Errorf("CPU usage monitoring not implemented for %s", runtime.GOOS)
}

func getCPUUsageUnix() (float64, error) {
	// Use 'top' command to get CPU usage (works on macOS and Linux)
	cmd := exec.Command("top", "-l", "1", "-n", "0")
	if runtime.GOOS == "linux" {
		cmd = exec.Command("top", "-bn1")
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if runtime.GOOS == "darwin" {
			// macOS format: "CPU usage: 12.34% user, 5.67% sys, 81.99% idle"
			if strings.Contains(line, "CPU usage:") {
				parts := strings.Split(line, ",")
				if len(parts) >= 3 {
					idlePart := strings.TrimSpace(parts[2])
					if strings.Contains(idlePart, "% idle") {
						idleStr := strings.Fields(idlePart)[0]
						idleStr = strings.TrimSuffix(idleStr, "%")
						idle, err := strconv.ParseFloat(idleStr, 64)
						if err == nil {
							return 100 - idle, nil
						}
					}
				}
			}
		} else {
			// Linux format: "%Cpu(s):  1.2 us,  0.8 sy,  0.0 ni, 98.0 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st"
			if strings.Contains(line, "%Cpu(s):") {
				parts := strings.SplitSeq(line, ",")
				for part := range parts {
					part = strings.TrimSpace(part)
					if strings.Contains(part, " id") {
						fields := strings.Fields(part)
						if len(fields) >= 1 {
							idleStr := fields[0]
							idle, err := strconv.ParseFloat(idleStr, 64)
							if err == nil {
								return 100 - idle, nil
							}
						}
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse CPU usage from top output")
}

func getLoadAverage() (string, error) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		cmd := exec.Command("uptime")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}

		outputStr := strings.TrimSpace(string(output))
		// Look for load averages pattern: "load averages: 1.23 1.45 1.67" or "load average: 1.23, 1.45, 1.67"
		if idx := strings.Index(outputStr, "load average"); idx != -1 {
			loadPart := outputStr[idx:]
			// Extract the numbers after "load average:"
			colonIdx := strings.Index(loadPart, ":")
			if colonIdx != -1 {
				loadNumbers := strings.TrimSpace(loadPart[colonIdx+1:])
				// Clean up the format
				loadNumbers = strings.ReplaceAll(loadNumbers, ",", "")
				parts := strings.Fields(loadNumbers)
				if len(parts) >= 3 {
					return fmt.Sprintf("%s, %s, %s", parts[0], parts[1], parts[2]), nil
				}
			}
		}
	}

	return "", fmt.Errorf("load average not available on %s", runtime.GOOS)
}

func getDiskSpace(path string) (models.DiskSpace, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return models.DiskSpace{}, err
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - availableBytes
	usedPct := float64(usedBytes) / float64(totalBytes) * 100

	// Convert to human-readable format
	totalGB := bytesToGB(totalBytes)
	usedGB := bytesToGB(usedBytes)
	availGB := bytesToGB(availableBytes)

	return models.DiskSpace{
		Total:     formatBytes(totalBytes),
		Used:      formatBytes(usedBytes),
		Available: formatBytes(availableBytes),
		UsedPct:   math.Round(usedPct*100) / 100, // Round to 2 decimal places
		TotalGB:   totalGB,
		UsedGB:    usedGB,
		AvailGB:   availGB,
	}, nil
}

func getRAMUsage() (models.RAM, error) {
	// Use gopsutil for accurate system memory stats in production
	vmem, err := mem.VirtualMemory()
	if err != nil {
		// Fallback to Go runtime stats if gopsutil fails
		return getRAMUsageFallback()
	}

	totalBytes := vmem.Total
	usedBytes := vmem.Used
	availableBytes := vmem.Available
	usedPct := vmem.UsedPercent
	bufferCacheBytes := vmem.Buffers + vmem.Cached

	// Convert to human-readable format
	totalGB := bytesToGB(totalBytes)
	usedGB := bytesToGB(usedBytes)
	availGB := bytesToGB(availableBytes)

	return models.RAM{
		Total:       formatBytes(totalBytes),
		Used:        formatBytes(usedBytes),
		Available:   formatBytes(availableBytes),
		UsedPct:     math.Round(usedPct*100) / 100, // Round to 2 decimal places
		BufferCache: formatBytes(bufferCacheBytes),
		TotalGB:     totalGB,
		UsedGB:      usedGB,
		AvailGB:     availGB,
	}, nil
}

// getRAMUsageFallback provides fallback RAM monitoring using Go runtime stats
func getRAMUsageFallback() (models.RAM, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Using Go runtime memory stats as approximation
	totalBytes := m.Sys  // Total memory obtained from system
	usedBytes := m.Alloc // Currently allocated memory
	availableBytes := totalBytes - usedBytes
	usedPct := float64(usedBytes) / float64(totalBytes) * 100
	bufferCacheBytes := m.HeapIdle // Approximate buffer/cache

	// Convert to human-readable format
	totalGB := bytesToGB(totalBytes)
	usedGB := bytesToGB(usedBytes)
	availGB := bytesToGB(availableBytes)

	return models.RAM{
		Total:       formatBytes(totalBytes),
		Used:        formatBytes(usedBytes),
		Available:   formatBytes(availableBytes),
		UsedPct:     math.Round(usedPct*100) / 100, // Round to 2 decimal places
		BufferCache: formatBytes(bufferCacheBytes),
		TotalGB:     totalGB,
		UsedGB:      usedGB,
		AvailGB:     availGB,
	}, nil
}

func checkServerHeartbeats(servers []models.ServerConfig) []models.ServerCheck {
	if len(servers) == 0 {
		return []models.ServerCheck{}
	}

	// Use channels to collect results from parallel goroutines
	resultChan := make(chan models.ServerCheck, len(servers))

	// Launch all requests in parallel
	for _, server := range servers {
		go func(s models.ServerConfig) {
			result := checkSingleServer(s)
			resultChan <- result
		}(server)
	}

	// Collect all results
	var results []models.ServerCheck
	for range servers {
		result := <-resultChan
		results = append(results, result)
	}

	return results
}

func checkSingleServer(server models.ServerConfig) models.ServerCheck {
	start := time.Now()

	timeout := time.Duration(server.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		responseTime := time.Since(start)
		return models.ServerCheck{
			Name:         server.Name,
			URL:          server.URL,
			Status:       models.ServerStatusDown,
			ResponseTime: formatDuration(responseTime),
			ResponseMs:   responseTime.Milliseconds(),
			LastChecked:  time.Now(),
			Error:        err.Error(),
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return models.ServerCheck{
			Name:         server.Name,
			URL:          server.URL,
			Status:       models.ServerStatusDown,
			ResponseTime: formatDuration(responseTime),
			ResponseMs:   responseTime.Milliseconds(),
			LastChecked:  time.Now(),
			Error:        err.Error(),
		}
	}
	defer resp.Body.Close()

	status := models.ServerStatusUp
	errorMsg := ""

	if resp.StatusCode >= 400 {
		status = models.ServerStatusDown
		errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return models.ServerCheck{
		Name:         server.Name,
		URL:          server.URL,
		Status:       status,
		ResponseTime: formatDuration(responseTime),
		ResponseMs:   responseTime.Milliseconds(),
		LastChecked:  time.Now(),
		Error:        errorMsg,
	}
}

func readConfigFromFile() (*models.MonitoringConfig, error) {
	// Get the current working directory to find the root folder
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find the project root (look for go.mod file)
	projectRoot := findProjectRoot(cwd)
	configPath := filepath.Join(projectRoot, "configs.json")

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configs.json: %w", err)
	}

	// Parse the JSON
	var config models.MonitoringConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configs.json: %w", err)
	}

	return &config, nil
}

func findProjectRoot(startPath string) string {
	current := startPath
	for {
		// Check if go.mod exists in current directory
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return current
		}

		// Move to parent directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root, use starting path as fallback
			return startPath
		}
		current = parent
	}
}

// AddCustomServers allows adding custom servers to existing configuration
func AddCustomServers(servers []models.ServerConfig) []models.ServerConfig {
	configServers := GetServersConfig()
	return append(configServers, servers...)
}

// Utility functions for formatting

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func bytesToGB(bytes uint64) float64 {
	return math.Round(float64(bytes)/1024/1024/1024*100) / 100
}

func formatDuration(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else if d >= time.Millisecond {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d >= time.Microsecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	} else {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}

// Auto-logging functions

// startAutoLogging starts the automatic logging based on refresh_time
func startAutoLogging() {
	if monitoringConfig == nil {
		return
	}

	// Stop existing ticker if running
	stopAutoLogging()

	// Parse refresh time
	refreshDuration, err := time.ParseDuration(monitoringConfig.RefreshTime)
	if err != nil {
		fmt.Printf("Warning: invalid refresh_time '%s', using default 2s\n", monitoringConfig.RefreshTime)
		refreshDuration = 2 * time.Second
	}

	// Start new ticker
	loggingTicker = time.NewTicker(refreshDuration)
	loggingStopChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-loggingTicker.C:
				// Generate monitoring data and log it
				if data, err := MonitoringDataGenerator(); err == nil {
					if logErr := utils.LogMonitoringData(data); logErr != nil {
						fmt.Printf("Warning: failed to log monitoring data: %v\n", logErr)
					}
				}
			case <-loggingStopChan:
				return
			}
		}
	}()
}

// stopAutoLogging stops the automatic logging
func stopAutoLogging() {
	if loggingTicker != nil {
		loggingTicker.Stop()
		loggingTicker = nil
	}
	if loggingStopChan != nil {
		close(loggingStopChan)
		loggingStopChan = nil
	}
}

// StopAutoLogging provides external access to stop auto-logging
func StopAutoLogging() {
	stopAutoLogging()
}

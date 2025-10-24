package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
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
)

var (
	serversConfig     []models.ServerConfig
	serversConfigOnce sync.Once
	serversConfigErr  error
)

// InitServersConfig loads the servers configuration once at startup
func InitServersConfig() {
	serversConfigOnce.Do(func() {
		serversConfig, serversConfigErr = readServersFromFile()
		if serversConfigErr != nil {
			// Return empty array if file read fails
			serversConfig = []models.ServerConfig{}
			serversConfigErr = nil
		}
	})
}

// GetServersConfig returns the cached servers configuration
func GetServersConfig() []models.ServerConfig {
	InitServersConfig() // Ensure config is loaded
	return serversConfig
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
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For cross-platform compatibility, we'll use Go's runtime stats
	// In production, consider using a library like gopsutil for more accurate system stats

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
	var results []models.ServerCheck

	for _, server := range servers {
		result := checkSingleServer(server)
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

func readServersFromFile() ([]models.ServerConfig, error) {
	// Get the current working directory to find the root folder
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find the project root (look for go.mod file)
	projectRoot := findProjectRoot(cwd)
	configPath := filepath.Join(projectRoot, "servers.json")

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read servers.json: %w", err)
	}

	// Parse the JSON
	var config models.MonitoringConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse servers.json: %w", err)
	}

	return config.Servers, nil
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

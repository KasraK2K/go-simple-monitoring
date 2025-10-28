package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"go-log/internal/utils"
	"log"
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

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	monitoringConfig     *models.MonitoringConfig
	monitoringConfigOnce sync.Once
	monitoringConfigMu   sync.RWMutex
	lastConfigModTime    time.Time
	loggingTicker        *time.Ticker
	loggingStopChan      chan struct{}
)

// InitMonitoringConfig loads the monitoring configuration once at startup
func InitMonitoringConfig() {
	monitoringConfigOnce.Do(func() {
		newConfig, err := readConfigFromFile()

		monitoringConfigMu.Lock()
		defer monitoringConfigMu.Unlock()

		if err != nil {
			// Use default config on error
			if monitoringConfig == nil {
				monitoringConfig = getDefaultConfig()
			}
		} else {
			// Stop existing logging if refresh time changed
			if monitoringConfig != nil && monitoringConfig.RefreshTime != newConfig.RefreshTime {
				stopAutoLogging()
			}

			monitoringConfig = newConfig

			// Initialize logger and database for API server mode
			utils.InitLogger(monitoringConfig)

			if monitoringConfig.Storage == "db" || monitoringConfig.Storage == "both" {
				if err := utils.InitDatabase(); err != nil {
					log.Printf("Failed to initialize database: %v", err)
				}
			}
			startAutoLogging()
		}
		lastConfigModTime = time.Now()
	})
}

// InitMonitoringConfigCLI loads configuration for CLI mode without auto-logging
func InitMonitoringConfigCLI() {
	monitoringConfigOnce.Do(func() {
		newConfig, err := readConfigFromFile()

		monitoringConfigMu.Lock()
		defer monitoringConfigMu.Unlock()

		if err != nil {
			// Use default config on error
			if monitoringConfig == nil {
				monitoringConfig = getDefaultConfig()
			}
		} else {
			monitoringConfig = newConfig
			// CLI mode: NO auto-logging, NO database initialization
		}
		lastConfigModTime = time.Now()
	})
}

// GetHeartbeatConfig returns the cached heartbeat configuration
func GetHeartbeatConfig() []models.ServerConfig {
	ensureConfigLoaded()

	monitoringConfigMu.RLock()
	defer monitoringConfigMu.RUnlock()
	if monitoringConfig != nil {
		return monitoringConfig.Heartbeat
	}
	return []models.ServerConfig{}
}

// GetMonitoringConfig returns the current monitoring configuration, ensuring defaults if unset.
func GetMonitoringConfig() *models.MonitoringConfig {
	ensureConfigLoaded()

	monitoringConfigMu.RLock()
	defer monitoringConfigMu.RUnlock()
	if monitoringConfig != nil {
		return monitoringConfig
	}
	return getDefaultConfig()
}

// ensureConfigLoaded checks if config needs reloading and handles it
func ensureConfigLoaded() {
	InitMonitoringConfig()

	// Check if we should reload (every 30 seconds max)
	monitoringConfigMu.RLock()
	shouldCheck := time.Since(lastConfigModTime) > 30*time.Second
	monitoringConfigMu.RUnlock()

	if shouldCheck {
		// Check if config file was modified and reload if needed
		configPath := getConfigPath()
		if configPath != "" {
			if fileInfo, err := os.Stat(configPath); err == nil {
				monitoringConfigMu.RLock()
				lastCheck := lastConfigModTime
				monitoringConfigMu.RUnlock()

				if fileInfo.ModTime().After(lastCheck) {
					// Reload config with auto-logging enabled (for API server)
					newConfig, err := readConfigFromFile()

					monitoringConfigMu.Lock()
					if err == nil {
						if monitoringConfig != nil && monitoringConfig.RefreshTime != newConfig.RefreshTime {
							stopAutoLogging()
						}
						monitoringConfig = newConfig
						utils.InitLogger(monitoringConfig)

						if monitoringConfig.Storage == "db" || monitoringConfig.Storage == "both" {
							if err := utils.InitDatabase(); err != nil {
								log.Printf("Failed to initialize database: %v", err)
							}
						}
						startAutoLogging()
					}
					lastConfigModTime = time.Now()
					monitoringConfigMu.Unlock()
				} else {
					monitoringConfigMu.Lock()
					lastConfigModTime = time.Now()
					monitoringConfigMu.Unlock()
				}
			}
		}
	}
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *models.MonitoringConfig {
	return &models.MonitoringConfig{
		Path:        "./logs",
		RefreshTime: "2s",
		Storage:     "file",
		Heartbeat:   []models.ServerConfig{},
	}
}

func MonitoringDataGenerator() (*models.SystemMonitoring, error) {
	monitoring := &models.SystemMonitoring{
		Timestamp: time.Now(),
	}

	// Collect all system metrics in parallel for better performance
	type result struct {
		cpu       models.CPU
		disk      models.DiskSpace
		ram       models.RAM
		networkIO models.NetworkIO
		diskIO    models.DiskIO
		process   models.Process
		heartbeat []models.ServerCheck
		err       error
	}

	resultChan := make(chan result, 1)

	go func() {
		var r result

		// Get system metrics
		r.cpu, r.err = getCPUInfo()
		if r.err != nil {
			resultChan <- r
			return
		}

		r.disk, r.err = getDiskSpace("/")
		if r.err != nil {
			resultChan <- r
			return
		}

		r.ram, r.err = getRAMUsage()
		if r.err != nil {
			resultChan <- r
			return
		}

		r.networkIO, r.err = getNetworkIO()
		if r.err != nil {
			resultChan <- r
			return
		}

		r.diskIO, r.err = getDiskIO()
		if r.err != nil {
			resultChan <- r
			return
		}

		r.process, r.err = getProcessStats()
		if r.err != nil {
			resultChan <- r
			return
		}

		// Get heartbeat data
		servers := GetHeartbeatConfig()
		r.heartbeat = checkServerHeartbeats(servers)

		resultChan <- r
	}()

	r := <-resultChan
	if r.err != nil {
		return nil, r.err
	}

	monitoring.CPU = r.cpu
	monitoring.DiskSpace = r.disk
	monitoring.RAM = r.ram
	monitoring.NetworkIO = r.networkIO
	monitoring.DiskIO = r.diskIO
	monitoring.Process = r.process
	monitoring.Heartbeat = r.heartbeat

	return monitoring, nil
}

func MonitoringDataGeneratorWithFilter(from, to string) ([]any, error) {
	// Check if database is initialized and accessible
	if !utils.IsDatabaseInitialized() {
		return []any{}, fmt.Errorf("database is not accessible or not initialized")
	}

	// Query filtered data from database
	filteredData, err := utils.QueryFilteredMonitoringData(from, to)
	if err != nil {
		return []any{}, fmt.Errorf("failed to query filtered monitoring data: %w", err)
	}

	if len(filteredData) == 0 {
		currentData, err := MonitoringDataGenerator()
		if err != nil {
			return []any{}, fmt.Errorf("failed to generate current monitoring data: %w", err)
		}
		fallbackEntry := utils.BuildMonitoringLogEntry(currentData)
		return []any{fallbackEntry}, nil
	}

	// Convert to []any
	result := make([]any, len(filteredData))
	for i, entry := range filteredData {
		result[i] = entry
	}

	return result, nil
}

func getCPUInfo() (models.CPU, error) {
	cpuInfo := models.CPU{
		CoreCount:    runtime.NumCPU(),
		Goroutines:   runtime.NumGoroutine(),
		Architecture: runtime.GOARCH,
	}

	// Get CPU usage and load average concurrently
	type cpuMetrics struct {
		usage   float64
		loadAvg string
	}

	metricsChan := make(chan cpuMetrics, 1)
	go func() {
		var metrics cpuMetrics

		// Get CPU usage
		if usage, err := getCPUUsagePercent(); err == nil {
			metrics.usage = usage
		} else {
			// Fallback calculation
			metrics.usage = float64(cpuInfo.Goroutines) / float64(cpuInfo.CoreCount*10) * 100
			if metrics.usage > 100 {
				metrics.usage = 100
			}
		}

		// Get load average
		if loadAvg, err := getLoadAverage(); err == nil {
			metrics.loadAvg = loadAvg
		} else {
			metrics.loadAvg = "unavailable"
		}

		metricsChan <- metrics
	}()

	metrics := <-metricsChan
	cpuInfo.UsagePercent = math.Round(metrics.usage*100) / 100
	cpuInfo.LoadAverage = metrics.loadAvg

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

	return models.DiskSpace{
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		UsedPct:        math.Round(usedPct*100) / 100, // Round to 2 decimal places
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

	return models.RAM{
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		UsedPct:        math.Round(usedPct*100) / 100, // Round to 2 decimal places
		BufferBytes:    bufferCacheBytes,
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

	return models.RAM{
		TotalBytes:     totalBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		UsedPct:        math.Round(usedPct*100) / 100, // Round to 2 decimal places
		BufferBytes:    bufferCacheBytes,
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
	configPath := getConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("could not locate configs.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configs.json: %w", err)
	}

	var config models.MonitoringConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configs.json: %w", err)
	}

	return &config, nil
}

// getConfigPath returns the path to configs.json
func getConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	projectRoot := findProjectRoot(cwd)
	return filepath.Join(projectRoot, "configs.json")
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

// getNetworkIO returns network I/O statistics
func getNetworkIO() (models.NetworkIO, error) {
	ioStats, err := net.IOCounters(false) // false = per interface, true = summary
	if err != nil {
		return models.NetworkIO{}, err
	}

	// Sum up all interfaces for total system network I/O
	var totalIO models.NetworkIO
	for _, stat := range ioStats {
		totalIO.BytesSent += stat.BytesSent
		totalIO.BytesRecv += stat.BytesRecv
		totalIO.PacketsSent += stat.PacketsSent
		totalIO.PacketsRecv += stat.PacketsRecv
		totalIO.ErrorsIn += stat.Errin
		totalIO.ErrorsOut += stat.Errout
		totalIO.DropsIn += stat.Dropin
		totalIO.DropsOut += stat.Dropout
	}

	return totalIO, nil
}

// getDiskIO returns disk I/O statistics
func getDiskIO() (models.DiskIO, error) {
	ioStats, err := disk.IOCounters()
	if err != nil {
		return models.DiskIO{}, err
	}

	// Sum up all disks for total system disk I/O
	var totalIO models.DiskIO
	for _, stat := range ioStats {
		totalIO.ReadBytes += stat.ReadBytes
		totalIO.WriteBytes += stat.WriteBytes
		totalIO.ReadCount += stat.ReadCount
		totalIO.WriteCount += stat.WriteCount
		totalIO.ReadTime += stat.ReadTime
		totalIO.WriteTime += stat.WriteTime
		totalIO.IOTime += stat.IoTime
	}

	return totalIO, nil
}

// getProcessStats returns process statistics
func getProcessStats() (models.Process, error) {
	// Get load averages
	loadStats, err := load.Avg()
	if err != nil {
		// Fallback to manual load average calculation if gopsutil fails
		loadStats = &load.AvgStat{Load1: 0, Load5: 0, Load15: 0}
	}

	// Get all processes
	processes, err := process.Processes()
	if err != nil {
		return models.Process{}, err
	}

	// Count process states
	var running, sleeping, zombie, stopped int
	for _, p := range processes {
		status, err := p.Status()
		if err != nil {
			continue // Skip processes we can't read
		}

		// status is []string, so join it or use the first element
		var statusStr string
		if len(status) > 0 {
			statusStr = status[0]
		} else {
			statusStr = "unknown"
		}

		switch strings.ToLower(statusStr) {
		case "running", "r":
			running++
		case "sleeping", "s", "interruptible":
			sleeping++
		case "zombie", "z":
			zombie++
		case "stopped", "t":
			stopped++
		default:
			sleeping++ // Default unknown states to sleeping
		}
	}

	return models.Process{
		TotalProcesses: len(processes),
		RunningProcs:   running,
		SleepingProcs:  sleeping,
		ZombieProcs:    zombie,
		StoppedProcs:   stopped,
		LoadAvg1:       math.Round(loadStats.Load1*100) / 100,
		LoadAvg5:       math.Round(loadStats.Load5*100) / 100,
		LoadAvg15:      math.Round(loadStats.Load15*100) / 100,
	}, nil
}

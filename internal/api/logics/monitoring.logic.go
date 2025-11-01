package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"go-log/internal/utils"
	"io"
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
	logRotateTicker      *time.Ticker
	logRotateStopChan    chan struct{}
	serverMetricsCache   = map[string]cachedServerMetric{}
	serverMetricsCacheMu sync.RWMutex
)

type cachedServerMetric struct {
	metric    models.ServerMetrics
	fetchedAt time.Time
}

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
		Path:              "./logs",
		RefreshTime:       "2s",
		Storage:           "file",
		PersistServerLogs: false,
		Heartbeat:         []models.ServerConfig{},
		Servers:           []models.ServerEndpoint{},
		LogRotate: &models.LogRotateConfig{
			Enabled:    true,
			MaxAgeDays: 30,
		},
	}
}

func MonitoringDataGenerator() (*models.SystemMonitoring, error) {
	cfg := GetMonitoringConfig()
	monitoring := &models.SystemMonitoring{
		Timestamp: time.Now(),
	}

	// Collect all system metrics in parallel for better performance
	type result struct {
		cpu       models.CPU
		disk      []models.DiskSpace
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

		r.disk, r.err = getAllDiskSpaces()
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
	monitoring.ServerMetrics = collectServerMetrics(cfg)

	return monitoring, nil
}

func collectServerMetrics(cfg *models.MonitoringConfig) []models.ServerMetrics {
	if cfg == nil || len(cfg.Servers) == 0 {
		return nil
	}

	refreshDuration := defaultRefreshDuration(cfg.RefreshTime)

	results := make([]models.ServerMetrics, len(cfg.Servers))
	var wg sync.WaitGroup

	for idx, server := range cfg.Servers {
		wg.Add(1)
		go func(i int, srv models.ServerEndpoint) {
			defer wg.Done()
			results[i] = buildServerMetricSnapshot(srv, refreshDuration)
		}(idx, server)
	}

	wg.Wait()

	filtered := make([]models.ServerMetrics, 0, len(results))
	for _, metric := range results {
		if metric.Name == "" && metric.Address == "" && metric.Status == "" {
			continue
		}
		filtered = append(filtered, metric)
	}

	return filtered
}

func buildServerMetricSnapshot(server models.ServerEndpoint, refresh time.Duration) models.ServerMetrics {
	normalized := normalizeServerAddress(server.Address)
	metric := models.ServerMetrics{
		Name:    server.Name,
		Address: normalized,
	}

	if normalized == "" {
		metric.Status = "error"
		metric.Message = "server address is missing"
		metric.Timestamp = time.Now().Format(time.RFC3339Nano)
		return metric
	}

	if cached, ok := getCachedServerMetric(normalized); ok && !isCacheStale(cached, refresh) {
		existing := cached.metric
		if existing.Name == "" {
			existing.Name = metric.Name
		}
		if existing.Address == "" {
			existing.Address = metric.Address
		}
		if existing.Status == "" {
			existing.Status = "ok"
		}
		return existing
	}

	fetched, err := fetchAndCacheServerMetric(server)
	if err != nil {
		metric.Status = "error"
		metric.Message = err.Error()
		metric.Timestamp = time.Now().Format(time.RFC3339Nano)
		return metric
	}

	result := *fetched
	if result.Name == "" {
		result.Name = metric.Name
	}
	if result.Address == "" {
		result.Address = metric.Address
	}
	if result.Status == "" {
		result.Status = "ok"
	}

	return result
}

func defaultRefreshDuration(value string) time.Duration {
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return d
	}
	return 2 * time.Second
}

func getCachedServerMetric(address string) (cachedServerMetric, bool) {
	serverMetricsCacheMu.RLock()
	defer serverMetricsCacheMu.RUnlock()
	entry, ok := serverMetricsCache[address]
	return entry, ok
}

func isCacheStale(entry cachedServerMetric, refresh time.Duration) bool {
	if refresh <= 0 {
		refresh = 2 * time.Second
	}
	staleness := refresh * 2
	if staleness <= 0 {
		staleness = 5 * time.Second
	}
	return time.Since(entry.fetchedAt) > staleness
}

func fetchAndCacheServerMetric(server models.ServerEndpoint) (*models.ServerMetrics, error) {
	normalized := normalizeServerAddress(server.Address)
	if normalized == "" {
		return nil, fmt.Errorf("server address is empty")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload, err := fetchServerMonitoring(client, normalized)
	if err != nil {
		return nil, err
	}

	return updateServerMetricsCache(server, payload)
}

func updateServerMetricsCache(server models.ServerEndpoint, payload []byte) (*models.ServerMetrics, error) {
	metric, err := processServerMetricsPayload(server, payload)
	if err != nil {
		return nil, err
	}

	normalized := normalizeServerAddress(server.Address)
	if metric.Name == "" {
		metric.Name = server.Name
	}
	if metric.Address == "" {
		metric.Address = normalized
	}
	if metric.Status == "" {
		metric.Status = "ok"
	}
	if metric.Timestamp == "" {
		metric.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	serverMetricsCacheMu.Lock()
	serverMetricsCache[normalized] = cachedServerMetric{
		metric:    *metric,
		fetchedAt: time.Now(),
	}
	serverMetricsCacheMu.Unlock()

	return metric, nil
}

func processServerMetricsPayload(server models.ServerEndpoint, payload []byte) (*models.ServerMetrics, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload for server %s", server.Address)
	}

	var snapshots []models.SystemMonitoring
	if err := json.Unmarshal(payload, &snapshots); err == nil && len(snapshots) > 0 {
		metric := buildMetricsFromSnapshot(server, snapshots[0])
		return &metric, nil
	}

	var generic []map[string]any
	if err := json.Unmarshal(payload, &generic); err == nil && len(generic) > 0 {
		if metric, err := buildMetricsFromGenericMap(server, generic[0]); err == nil {
			return metric, nil
		}
	}

	var logs []models.MonitoringLogEntry
	if err := json.Unmarshal(payload, &logs); err == nil && len(logs) > 0 {
		for _, entry := range logs {
			if metric, err := buildMetricsFromGenericMap(server, entry.Body); err == nil {
				if metric.Timestamp == "" {
					metric.Timestamp = entry.Time
				}
				return metric, nil
			}
		}
	}

	var wrapper map[string]any
	if err := json.Unmarshal(payload, &wrapper); err == nil {
		if data, ok := wrapper["data"]; ok {
			nested, err := json.Marshal(data)
			if err == nil {
				return processServerMetricsPayload(server, nested)
			}
		}
	}

	return nil, fmt.Errorf("failed to parse server payload for %s", server.Address)
}

func buildMetricsFromSnapshot(server models.ServerEndpoint, snapshot models.SystemMonitoring) models.ServerMetrics {
	metric := models.ServerMetrics{
		Name:              server.Name,
		Address:           normalizeServerAddress(server.Address),
		CPUUsage:          snapshot.CPU.UsagePercent,
		MemoryUsedPercent: snapshot.RAM.UsedPct,
		DiskUsedPercent:   computeDiskUsedPercent(snapshot.DiskSpace),
		NetworkInBytes:    snapshot.NetworkIO.BytesRecv,
		NetworkOutBytes:   snapshot.NetworkIO.BytesSent,
		LoadAverage:       snapshot.CPU.LoadAverage,
		Status:            "ok",
	}

	if !snapshot.Timestamp.IsZero() {
		metric.Timestamp = snapshot.Timestamp.Format(time.RFC3339Nano)
	} else {
		metric.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	return metric
}

func buildMetricsFromGenericMap(server models.ServerEndpoint, body map[string]any) (*models.ServerMetrics, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty server payload body")
	}

	if data, err := json.Marshal(body); err == nil {
		var snapshot models.SystemMonitoring
		if err := json.Unmarshal(data, &snapshot); err == nil {
			metric := buildMetricsFromSnapshot(server, snapshot)
			return &metric, nil
		}
	}

	metric := models.ServerMetrics{
		Name:      server.Name,
		Address:   normalizeServerAddress(server.Address),
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}

	if value, ok := body["cpu_usage_percent"]; ok {
		metric.CPUUsage = toFloat64(value)
	} else if value, ok := body["cpu_usage"]; ok {
		metric.CPUUsage = toFloat64(value)
	}

	if value, ok := body["ram_used_percent"]; ok {
		metric.MemoryUsedPercent = toFloat64(value)
	} else if ram, ok := body["ram"].(map[string]any); ok {
		metric.MemoryUsedPercent = toFloat64(ram["used_pct"])
	}

	if value, ok := body["disk_used_percent"]; ok {
		metric.DiskUsedPercent = toFloat64(value)
	}

	if value, ok := body["network_bytes_recv"]; ok {
		metric.NetworkInBytes = toUint64(value)
	}

	if value, ok := body["network_bytes_sent"]; ok {
		metric.NetworkOutBytes = toUint64(value)
	}

	if value, ok := body["cpu_load_average"]; ok {
		metric.LoadAverage = fmt.Sprint(value)
	}

	return &metric, nil
}

func computeDiskUsedPercent(disks []models.DiskSpace) float64 {
	if len(disks) == 0 {
		return 0
	}

	for _, disk := range disks {
		if disk.Path == "/" {
			return disk.UsedPct
		}
	}

	return disks[0].UsedPct
}

func toFloat64(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

func toUint64(value any) uint64 {
	f := toFloat64(value)
	if f <= 0 {
		return 0
	}
	return uint64(f)
}

func normalizeServerAddress(address string) string {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return ""
	}
	return strings.TrimRight(trimmed, "/")
}

func MonitoringDataGeneratorWithTableFilter(tableName, from, to string) ([]any, error) {
	// Check if database is initialized and accessible
	if !utils.IsDatabaseInitialized() {
		return []any{}, fmt.Errorf("database is not accessible or not initialized")
	}

	// Determine which table to query
	var filteredData []models.MonitoringLogEntry
	var err error

	if utils.IsEmptyOrWhitespace(tableName) || tableName == "default" {
		// Query default table (handle both empty string and "default" API parameter)
		filteredData, err = utils.QueryFilteredTableData(utils.DefaultTableName, from, to)
	} else {
		// Query specific table
		filteredData, err = utils.QueryFilteredTableData(tableName, from, to)
	}

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

func getAllDiskSpaces() ([]models.DiskSpace, error) {
	// Use gopsutil to get all disk partitions
	partitions, err := disk.Partitions(false) // false = exclude pseudo-filesystems
	if err != nil {
		// Fallback to root filesystem only
		rootDisk, rootErr := getDiskSpace("/")
		if rootErr != nil {
			return nil, fmt.Errorf("failed to get disk partitions and root disk: %v, %v", err, rootErr)
		}
		return []models.DiskSpace{rootDisk}, nil
	}

	var diskSpaces []models.DiskSpace
	seenPaths := make(map[string]bool)
	seenStorageSignatures := make(map[string]models.DiskSpace) // Deduplicate by storage signature

	for _, partition := range partitions {
		// Skip if we've already processed this mount point
		if seenPaths[partition.Mountpoint] {
			continue
		}
		seenPaths[partition.Mountpoint] = true

		// Skip pseudo filesystems and network filesystems
		if shouldSkipFileSystem(partition.Fstype, partition.Mountpoint) {
			continue
		}

		diskSpace, err := getDiskSpaceForPartition(partition)
		if err != nil {
			// Log error but continue with other partitions
			log.Printf("Warning: failed to get disk space for %s: %v", partition.Mountpoint, err)
			continue
		}

		// Create a signature to identify duplicate storage (same device or same size+usage)
		signature := createStorageSignature(diskSpace)

		// Check if we already have this storage device/pool
		if existingDisk, exists := seenStorageSignatures[signature]; exists {
			// If this is a more "important" mount point, replace the existing one
			if isMoreImportantMountPoint(diskSpace.Path, existingDisk.Path) {
				seenStorageSignatures[signature] = diskSpace
			}
			// Otherwise skip this duplicate
			continue
		}

		seenStorageSignatures[signature] = diskSpace
	}

	// Convert map to slice
	for _, diskSpace := range seenStorageSignatures {
		diskSpaces = append(diskSpaces, diskSpace)
	}

	// If no valid partitions found, fallback to root
	if len(diskSpaces) == 0 {
		rootDisk, err := getDiskSpace("/")
		if err != nil {
			return nil, fmt.Errorf("no valid disk partitions found and failed to get root disk: %v", err)
		}
		diskSpaces = append(diskSpaces, rootDisk)
	}

	return diskSpaces, nil
}

func createStorageSignature(diskSpace models.DiskSpace) string {
	// For APFS and other shared storage pools, multiple volumes can have the same total size and usage
	// We need to deduplicate these by treating them as one logical storage unit

	// If multiple volumes have identical total_bytes and used_pct, they're likely sharing the same storage pool
	// Use size+usage pattern as signature to group them
	sizeUsageSignature := fmt.Sprintf("size:%d:usage:%.2f", diskSpace.TotalBytes, diskSpace.UsedPct)

	// But still prefer device name for truly separate devices
	if diskSpace.Device != "" && diskSpace.Device != "unknown" {
		// For now, use the size+usage pattern for better deduplication
		// This helps with APFS containers where multiple volumes share space
		return sizeUsageSignature
	}

	return sizeUsageSignature
}

func isMoreImportantMountPoint(newPath, existingPath string) bool {
	// Priority order for mount points (higher priority = more important)
	priorities := map[string]int{
		"/":                    1000, // Root is most important
		"/home":                900,  // Home directory
		"/usr":                 800,  // User programs
		"/var":                 700,  // Variable data
		"/tmp":                 600,  // Temporary files
		"/boot":                500,  // Boot files
		"/System/Volumes/Data": 450,  // macOS data volume
	}

	newPriority := priorities[newPath]
	existingPriority := priorities[existingPath]

	// If neither has a defined priority, prefer shorter paths (closer to root)
	if newPriority == 0 && existingPriority == 0 {
		return len(newPath) < len(existingPath)
	}

	return newPriority > existingPriority
}

func shouldSkipFileSystem(fstype, mountpoint string) bool {
	// Skip pseudo filesystems and network filesystems
	skipFSTypes := []string{
		"devfs", "devtmpfs", "tmpfs", "proc", "sysfs", "debugfs", "securityfs",
		"cgroup", "cgroup2", "pstore", "bpf", "tracefs", "configfs", "fusectl",
		"selinuxfs", "systemd-1", "mqueue", "hugetlbfs", "autofs", "nfs", "nfs4",
		"cifs", "smbfs", "fuse", "overlay", "squashfs", "iso9660",
	}

	for _, skipType := range skipFSTypes {
		if strings.Contains(strings.ToLower(fstype), strings.ToLower(skipType)) {
			return true
		}
	}

	// Skip certain mount points
	skipMountpoints := []string{
		"/dev", "/proc", "/sys", "/run", "/boot/efi", "/snap",
	}

	for _, skipMount := range skipMountpoints {
		if strings.HasPrefix(mountpoint, skipMount) {
			return true
		}
	}

	// Skip macOS system volumes that are not useful for monitoring
	macOSSystemVolumes := []string{
		"/System/Volumes/xarts",
		"/System/Volumes/iSCPreboot",
		"/System/Volumes/Hardware",
		"/System/Volumes/Preboot",
		"/System/Volumes/Update",
		"/System/Volumes/VM", // Virtual memory - not a real storage volume
	}

	for _, systemVolume := range macOSSystemVolumes {
		if mountpoint == systemVolume {
			return true
		}
	}

	return false
}

func getDiskSpaceForPartition(partition disk.PartitionStat) (models.DiskSpace, error) {
	usage, err := disk.Usage(partition.Mountpoint)
	if err != nil {
		return models.DiskSpace{}, err
	}

	return models.DiskSpace{
		Path:           partition.Mountpoint,
		Device:         partition.Device,
		FileSystem:     partition.Fstype,
		TotalBytes:     usage.Total,
		UsedBytes:      usage.Used,
		AvailableBytes: usage.Free,
		UsedPct:        math.Round(usage.UsedPercent*100) / 100, // Round to 2 decimal places
	}, nil
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
		Path:           path,
		Device:         "unknown",
		FileSystem:     "unknown",
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
	configureLogRotation()

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

				persistServerLogs()
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
	stopLogRotation()
}

func configureLogRotation() {
	stopLogRotation()

	if monitoringConfig == nil {
		return
	}

	if strings.EqualFold(monitoringConfig.Storage, "none") {
		return
	}

	rotateCfg := monitoringConfig.LogRotate
	if rotateCfg == nil || !rotateCfg.Enabled {
		return
	}

	maxAge := rotateCfg.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 30
	}

	performCleanup := func(retention int) {
		if err := utils.CleanOldLogs(retention); err != nil {
			fmt.Printf("Warning: log cleanup failed: %v\n", err)
		}

		cutoff := time.Now().AddDate(0, 0, -retention)
		if utils.IsDatabaseInitialized() {
			if err := utils.CleanOldDatabaseEntries(cutoff); err != nil {
				fmt.Printf("Warning: database cleanup failed: %v\n", err)
			}
		}
	}

	performCleanup(maxAge)

	logRotateTicker = time.NewTicker(24 * time.Hour)
	logRotateStopChan = make(chan struct{})

	go func(retention int) {
		for {
			select {
			case <-logRotateTicker.C:
				performCleanup(retention)
			case <-logRotateStopChan:
				return
			}
		}
	}(maxAge)
}

func stopLogRotation() {
	if logRotateTicker != nil {
		logRotateTicker.Stop()
		logRotateTicker = nil
	}
	if logRotateStopChan != nil {
		close(logRotateStopChan)
		logRotateStopChan = nil
	}
}

func persistServerLogs() {
	monitoringConfigMu.RLock()
	cfg := monitoringConfig
	monitoringConfigMu.RUnlock()

	if cfg == nil || !cfg.PersistServerLogs {
		return
	}

	storage := strings.ToLower(strings.TrimSpace(cfg.Storage))
	if storage == "none" {
		return
	}

	writeFile := storage == "file" || storage == "both"
	writeDB := (storage == "db" || storage == "both") && utils.IsDatabaseInitialized()

	if writeFile && utils.IsEmptyOrWhitespace(cfg.Path) {
		fmt.Printf("Warning: persist_server_logs enabled but log path is empty; skipping file persistence\n")
		writeFile = false
	}

	if !writeFile && !writeDB {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, server := range cfg.Servers {
		if utils.IsEmptyOrWhitespace(server.TableName) {
			continue
		}

		if utils.IsEmptyOrWhitespace(server.Address) {
			continue
		}

		payload, err := fetchServerMonitoring(client, server.Address)
		if err != nil {
			fmt.Printf("Warning: failed to fetch monitoring data from %s: %v\n", server.Address, err)
			continue
		}

		if _, err := updateServerMetricsCache(server, payload); err != nil {
			fmt.Printf("Warning: failed to parse server metrics from %s: %v\n", server.Address, err)
		}

		if writeFile {
			if err := utils.WriteServerLogToFile(cfg.Path, server, payload); err != nil {
				fmt.Printf("Warning: failed to write server log file for %s: %v\n", server.Address, err)
			}
		}

		if writeDB {
			if err := utils.WriteServerLogToDatabase(server.TableName, payload); err != nil {
				fmt.Printf("Warning: failed to write server log to database for %s: %v\n", server.Address, err)
			}
		}
	}
}

func fetchServerMonitoring(client *http.Client, baseAddress string) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is not configured")
	}

	endpoint := strings.TrimRight(baseAddress, "/") + "/monitoring"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
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

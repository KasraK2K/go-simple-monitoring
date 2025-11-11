package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"go-log/internal/config"
	"go-log/internal/utils"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
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
	loggingMu            sync.Mutex
	logRotateTicker      *time.Ticker
	logRotateStopChan    chan struct{}
	logRotateMu          sync.Mutex
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
		lastConfigModTime = utils.NowUTC()
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
		lastConfigModTime = utils.NowUTC()
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
					lastConfigModTime = utils.NowUTC()
					monitoringConfigMu.Unlock()
				} else {
					monitoringConfigMu.Lock()
					lastConfigModTime = utils.NowUTC()
					monitoringConfigMu.Unlock()
				}
			}
		}
	}
}

// getDefaultLogPath returns the default log path from environment
func getDefaultLogPath() string {
	envConfig := config.GetEnvConfig()
	return envConfig.BaseLogFolder
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *models.MonitoringConfig {
	return &models.MonitoringConfig{
		Path:              getDefaultLogPath(),
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
		Timestamp: utils.NowUTC(),
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
			defer func() {
				// Recover from any panics in server monitoring to prevent system crash
				if r := recover(); r != nil {
					utils.LogErrorWithContext("server-monitoring",
						fmt.Sprintf("Server monitoring panic for '%s' (%s)", srv.Name, srv.Address),
						fmt.Errorf("panic: %v", r))

					// Create an error metric for the failed server
					results[i] = models.ServerMetrics{
						Name:      srv.Name,
						Address:   srv.Address,
						Status:    "error",
						Message:   fmt.Sprintf("monitoring panic: %v", r),
						Timestamp: utils.FormatTimestampUTC(utils.NowUTC()),
					}
				}
			}()

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
		metric.Timestamp = utils.FormatTimestampUTC(utils.NowUTC())
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
		metric.Timestamp = utils.FormatTimestampUTC(utils.NowUTC())

		// Log the server connection failure for monitoring purposes
		utils.LogWarnWithContext("server-monitoring",
			fmt.Sprintf("Server '%s' (%s) is unavailable", server.Name, server.Address), err)

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

	// Use the shared HTTP client for resource efficiency
	payload, err := fetchServerMonitoring(normalized)
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
		metric.Timestamp = utils.FormatTimestampUTC(utils.NowUTC())
	}

	serverMetricsCacheMu.Lock()
	serverMetricsCache[normalized] = cachedServerMetric{
		metric:    *metric,
		fetchedAt: utils.NowUTC(),
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
		DiskSpace:         snapshot.DiskSpace,
		NetworkInBytes:    snapshot.NetworkIO.BytesRecv,
		NetworkOutBytes:   snapshot.NetworkIO.BytesSent,
		LoadAverage:       snapshot.CPU.LoadAverage,
		Status:            "ok",
	}

	if !snapshot.Timestamp.IsZero() {
		metric.Timestamp = utils.FormatTimestampUTC(snapshot.Timestamp)
	} else {
		metric.Timestamp = utils.FormatTimestampUTC(utils.NowUTC())
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
		Timestamp: utils.FormatTimestampUTC(utils.NowUTC()),
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

	metric.DiskSpace = extractDiskSpaces(body)

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

func extractDiskSpaces(body map[string]any) []models.DiskSpace {
	value, ok := body["disk_space"]
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case []models.DiskSpace:
		return typed
	case []map[string]any:
		return convertDiskMaps(typed)
	case []any:
		converted := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if entry, ok := item.(map[string]any); ok {
				converted = append(converted, entry)
			}
		}
		return convertDiskMaps(converted)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var disks []models.DiskSpace
		if err := json.Unmarshal(data, &disks); err != nil {
			return nil
		}
		return disks
	}
}

func convertDiskMaps(items []map[string]any) []models.DiskSpace {
	if len(items) == 0 {
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return nil
	}
	var disks []models.DiskSpace
	if err := json.Unmarshal(data, &disks); err != nil {
		return nil
	}
	return disks
}

func convertDiskValueToModels(value any) []models.DiskSpace {
	switch typed := value.(type) {
	case []models.DiskSpace:
		return typed
	case []map[string]any:
		return convertDiskMaps(typed)
	case []any:
		maps := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if entry, ok := item.(map[string]any); ok {
				maps = append(maps, entry)
			}
		}
		return convertDiskMaps(maps)
	default:
		if value == nil {
			return nil
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		var disks []models.DiskSpace
		if err := json.Unmarshal(data, &disks); err != nil {
			return nil
		}
		return disks
	}
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
		currentData, err := MonitoringDataGenerator()
		if err != nil {
			return []any{}, err
		}
		if currentData != nil {
			return []any{currentData}, nil
		}
		return []any{}, nil
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
		// When no historical data is found, return empty array to allow frontend to handle gracefully
		return []any{}, nil
	}

	result := make([]any, 0, len(filteredData))
	for _, entry := range filteredData {
		snapshot, convErr := convertLogEntryToSystemMonitoring(entry)
		if convErr == nil {
			result = append(result, snapshot)
		} else {
			result = append(result, entry.Body)
		}
	}

	return result, nil
}

func convertLogEntryToSystemMonitoring(entry models.MonitoringLogEntry) (*models.SystemMonitoring, error) {
	if entry.Body == nil {
		return nil, fmt.Errorf("empty log entry body")
	}

	// Check if this is a remote server entry with nested payload structure
	if payload, hasPayload := entry.Body["payload"]; hasPayload {
		return convertServerLogEntryToSystemMonitoring(entry, payload)
	}

	// Handle local monitoring data with flat structure
	return convertFlatLogEntryToSystemMonitoring(entry)
}

func convertServerLogEntryToSystemMonitoring(entry models.MonitoringLogEntry, payload any) (*models.SystemMonitoring, error) {
	// Remote server data is stored as: {"time": "...", "payload": [{"timestamp": "...", "cpu": {...}, ...}]}
	payloadArray, ok := payload.([]any)
	if !ok || len(payloadArray) == 0 {
		return nil, fmt.Errorf("invalid payload format")
	}

	// Take the first (and typically only) SystemMonitoring entry from the payload
	firstEntry, ok := payloadArray[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload entry format")
	}

	// Convert the nested structure to SystemMonitoring
	data, err := json.Marshal(firstEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload entry: %w", err)
	}

	var snapshot models.SystemMonitoring
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to SystemMonitoring: %w", err)
	}

	// Use the entry time if timestamp is not set
	if snapshot.Timestamp.IsZero() && entry.Time != "" {
		if ts, err := utils.ParseTimestampUTC(entry.Time); err == nil {
			snapshot.Timestamp = ts
		}
	}

	return &snapshot, nil
}

func convertFlatLogEntryToSystemMonitoring(entry models.MonitoringLogEntry) (*models.SystemMonitoring, error) {
	snapshot := &models.SystemMonitoring{}

	// Parse timestamp
	if entry.Time != "" {
		if ts, err := utils.ParseTimestampUTC(entry.Time); err == nil {
			snapshot.Timestamp = ts
		}
	}

	// Helper function to safely convert to float64
	toFloat64 := func(v any) float64 {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case uint64:
			return float64(val)
		default:
			return 0
		}
	}

	// Helper function to safely convert to uint64
	toUint64 := func(v any) uint64 {
		switch val := v.(type) {
		case uint64:
			return val
		case int64:
			if val >= 0 {
				return uint64(val)
			}
			return 0
		case int:
			if val >= 0 {
				return uint64(val)
			}
			return 0
		case float64:
			if val >= 0 {
				return uint64(val)
			}
			return 0
		default:
			return 0
		}
	}

	// Helper function to safely convert to int
	toInt := func(v any) int {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		default:
			return 0
		}
	}

	// Helper function to safely convert to string
	toString := func(v any) string {
		if v == nil {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}

	// Map CPU fields
	snapshot.CPU = models.CPU{
		UsagePercent: toFloat64(entry.Body["cpu_usage_percent"]),
		CoreCount:    toInt(entry.Body["cpu_cores"]),
		Goroutines:   toInt(entry.Body["cpu_goroutines"]),
		LoadAverage:  toString(entry.Body["cpu_load_average"]),
		Architecture: toString(entry.Body["cpu_architecture"]),
	}

	// Map RAM fields
	snapshot.RAM = models.RAM{
		TotalBytes:     toUint64(entry.Body["ram_total_bytes"]),
		UsedBytes:      toUint64(entry.Body["ram_used_bytes"]),
		AvailableBytes: toUint64(entry.Body["ram_available_bytes"]),
		UsedPct:        toFloat64(entry.Body["ram_used_percent"]),
		BufferBytes:    0, // Not stored in flat format
	}

	// Map NetworkIO fields
	snapshot.NetworkIO = models.NetworkIO{
		BytesSent:   toUint64(entry.Body["network_bytes_sent"]),
		BytesRecv:   toUint64(entry.Body["network_bytes_recv"]),
		PacketsSent: toUint64(entry.Body["network_packets_sent"]),
		PacketsRecv: toUint64(entry.Body["network_packets_recv"]),
		ErrorsIn:    toUint64(entry.Body["network_errors_in"]),
		ErrorsOut:   toUint64(entry.Body["network_errors_out"]),
		DropsIn:     toUint64(entry.Body["network_drops_in"]),
		DropsOut:    toUint64(entry.Body["network_drops_out"]),
	}

	// Map DiskIO fields
	snapshot.DiskIO = models.DiskIO{
		ReadBytes:  toUint64(entry.Body["diskio_read_bytes"]),
		WriteBytes: toUint64(entry.Body["diskio_write_bytes"]),
		ReadCount:  toUint64(entry.Body["diskio_read_count"]),
		WriteCount: toUint64(entry.Body["diskio_write_count"]),
		ReadTime:   toUint64(entry.Body["diskio_read_time"]),
		WriteTime:  toUint64(entry.Body["diskio_write_time"]),
		IOTime:     toUint64(entry.Body["diskio_io_time"]),
	}

	// Map Process fields
	snapshot.Process = models.Process{
		TotalProcesses: toInt(entry.Body["process_total"]),
		RunningProcs:   toInt(entry.Body["process_running"]),
		SleepingProcs:  toInt(entry.Body["process_sleeping"]),
		ZombieProcs:    toInt(entry.Body["process_zombie"]),
		StoppedProcs:   toInt(entry.Body["process_stopped"]),
		LoadAvg1:       toFloat64(entry.Body["process_load_avg_1"]),
		LoadAvg5:       toFloat64(entry.Body["process_load_avg_5"]),
		LoadAvg15:      toFloat64(entry.Body["process_load_avg_15"]),
	}

	// Map DiskSpace fields - try to get from disk_spaces array first, fallback to flat fields
	if diskSpaces, ok := entry.Body["disk_spaces"]; ok {
		if diskArray, ok := diskSpaces.([]any); ok {
			for _, diskItem := range diskArray {
				if diskMap, ok := diskItem.(map[string]any); ok {
					disk := models.DiskSpace{
						Path:           toString(diskMap["path"]),
						Device:         toString(diskMap["device"]),
						FileSystem:     toString(diskMap["filesystem"]),
						TotalBytes:     toUint64(diskMap["total_bytes"]),
						UsedBytes:      toUint64(diskMap["used_bytes"]),
						AvailableBytes: toUint64(diskMap["available_bytes"]),
						UsedPct:        toFloat64(diskMap["used_pct"]),
					}
					snapshot.DiskSpace = append(snapshot.DiskSpace, disk)
				}
			}
		}
	}

	// If no disk_spaces array, create one from flat fields for backward compatibility
	if len(snapshot.DiskSpace) == 0 {
		snapshot.DiskSpace = []models.DiskSpace{
			{
				Path:           "/", // Assume root
				Device:         "unknown",
				FileSystem:     "unknown",
				TotalBytes:     toUint64(entry.Body["disk_total_bytes"]),
				UsedBytes:      toUint64(entry.Body["disk_used_bytes"]),
				AvailableBytes: toUint64(entry.Body["disk_available_bytes"]),
				UsedPct:        toFloat64(entry.Body["disk_used_percent"]),
			},
		}
	}

	// Map Heartbeat and ServerMetrics if present
	if heartbeat, ok := entry.Body["heartbeat"]; ok && heartbeat != nil {
		// Convert heartbeat data if needed
		// This would require additional conversion logic
	}

	if serverMetrics, ok := entry.Body["server_metrics"]; ok && serverMetrics != nil {
		if metricArray, ok := serverMetrics.([]any); ok {
			for _, rawMetric := range metricArray {
				metricMap, ok := rawMetric.(map[string]any)
				if !ok {
					continue
				}
				metric := models.ServerMetrics{
					Name:              toString(metricMap["name"]),
					Address:           normalizeServerAddress(toString(metricMap["address"])),
					CPUUsage:          toFloat64(metricMap["cpu_usage"]),
					MemoryUsedPercent: toFloat64(metricMap["memory_used_percent"]),
					DiskUsedPercent:   toFloat64(metricMap["disk_used_percent"]),
					NetworkInBytes:    toUint64(metricMap["network_in_bytes"]),
					NetworkOutBytes:   toUint64(metricMap["network_out_bytes"]),
					LoadAverage:       toString(metricMap["load_average"]),
					Timestamp:         toString(metricMap["timestamp"]),
					Status:            toString(metricMap["status"]),
					Message:           toString(metricMap["message"]),
					DiskSpace:         convertDiskValueToModels(metricMap["disk_space"]),
				}
				snapshot.ServerMetrics = append(snapshot.ServerMetrics, metric)
			}
		}
	}

	return snapshot, nil
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
	// Use gopsutil for secure CPU usage monitoring instead of external commands
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data available")
	}

	// Return overall CPU usage percentage
	return percentages[0], nil
}

func getLoadAverage() (string, error) {
	// Use gopsutil for secure load average monitoring instead of external commands
	loadAvg, err := load.Avg()
	if err != nil {
		return "", fmt.Errorf("failed to get load average: %w", err)
	}

	// Format load averages to match expected format
	return fmt.Sprintf("%.2f, %.2f, %.2f", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15), nil
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
	start := utils.NowUTC()

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

	// Use shared HTTP client with timeout for server checks
	client := utils.GetHTTPClientWithTimeout(timeout)
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
		return nil, fmt.Errorf("could not locate configuration file")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", configPath, err)
	}

	var monitoringConfig models.MonitoringConfig
	if err = json.Unmarshal(data, &monitoringConfig); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file %s: %w", configPath, err)
	}

	// Override path with environment variable if set
	envConfig := config.GetEnvConfig()
	if envConfig.BaseLogFolder != "./logs" {
		monitoringConfig.Path = envConfig.BaseLogFolder
	}

	return &monitoringConfig, nil
}

// getConfigPath returns the path to configs.json
func getConfigPath() string {
	envConfig := config.GetEnvConfig()
	if override := strings.TrimSpace(envConfig.MonitorConfigPath); override != "" {
		return filepath.Clean(override)
	}

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
		utils.LogWarnWithContext("auto-logging", fmt.Sprintf("invalid refresh_time '%s', using default 2s", monitoringConfig.RefreshTime), err)
		refreshDuration = 2 * time.Second
	}

	// Start new ticker with proper synchronization
	loggingMu.Lock()
	loggingTicker = time.NewTicker(refreshDuration)
	loggingStopChan = make(chan struct{})

	// Create local copies to avoid race conditions
	ticker := loggingTicker
	stopChan := loggingStopChan
	loggingMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.LogErrorWithContext("auto-logging", "goroutine panic recovered", fmt.Errorf("%v", r))
			}
		}()

		for {
			select {
			case <-ticker.C:
				// Generate monitoring data and log it - local monitoring should never fail the entire system
				func() {
					defer func() {
						if r := recover(); r != nil {
							utils.LogErrorWithContext("auto-logging", "local monitoring panic recovered", fmt.Errorf("%v", r))
						}
					}()
					
					if data, err := MonitoringDataGenerator(); err == nil {
						if logErr := utils.LogMonitoringData(data); logErr != nil {
							utils.LogWarnWithContext("auto-logging", "failed to log monitoring data", logErr)
						}
					} else {
						utils.LogWarnWithContext("auto-logging", "failed to generate monitoring data", err)
					}
				}()

				// Persist remote server logs - this should never block or crash local monitoring
				go func() {
					defer func() {
						if r := recover(); r != nil {
							utils.LogErrorWithContext("server-persistence", "server logging panic recovered", fmt.Errorf("%v", r))
						}
					}()
					persistServerLogs()
				}()
			case <-stopChan:
				return
			}
		}
	}()
}

// stopAutoLogging stops the automatic logging with proper synchronization
func stopAutoLogging() {
	loggingMu.Lock()
	defer loggingMu.Unlock()

	if loggingTicker != nil {
		loggingTicker.Stop()
		loggingTicker = nil
	}
	if loggingStopChan != nil {
		// Safe close - check if channel is already closed
		select {
		case <-loggingStopChan:
			// Channel already closed
		default:
			close(loggingStopChan)
		}
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
			utils.LogWarnWithContext("log-rotation", "log cleanup failed", err)
		}

		cutoff := time.Now().AddDate(0, 0, -retention)
		if utils.IsDatabaseInitialized() {
			if err := utils.CleanOldDatabaseEntries(cutoff); err != nil {
				utils.LogWarnWithContext("log-rotation", "database cleanup failed", err)
			}
		}
	}

	performCleanup(maxAge)

	// Start log rotation with proper synchronization
	logRotateMu.Lock()
	logRotateTicker = time.NewTicker(24 * time.Hour)
	logRotateStopChan = make(chan struct{})

	// Create local copies to avoid race conditions
	ticker := logRotateTicker
	stopChan := logRotateStopChan
	logRotateMu.Unlock()

	go func(retention int) {
		defer func() {
			if r := recover(); r != nil {
				utils.LogErrorWithContext("log-rotation", "goroutine panic recovered", fmt.Errorf("%v", r))
			}
		}()

		for {
			select {
			case <-ticker.C:
				performCleanup(retention)
			case <-stopChan:
				return
			}
		}
	}(maxAge)
}

func stopLogRotation() {
	logRotateMu.Lock()
	defer logRotateMu.Unlock()

	if logRotateTicker != nil {
		logRotateTicker.Stop()
		logRotateTicker = nil
	}
	if logRotateStopChan != nil {
		// Safe close - check if channel is already closed
		select {
		case <-logRotateStopChan:
			// Channel already closed
		default:
			close(logRotateStopChan)
		}
		logRotateStopChan = nil
	}
}

// CleanupAllGoroutines stops all running goroutines and cleans up resources
// This function should be called during application shutdown
func CleanupAllGoroutines() {
	utils.LogInfo("cleaning up all monitoring goroutines...")

	// Stop auto-logging goroutines
	stopAutoLogging()

	utils.LogInfo("all monitoring goroutines cleaned up successfully")
}

// IsAutoLoggingActive checks if auto-logging is currently running
func IsAutoLoggingActive() bool {
	loggingMu.Lock()
	defer loggingMu.Unlock()
	return loggingTicker != nil && loggingStopChan != nil
}

// IsLogRotationActive checks if log rotation is currently running
func IsLogRotationActive() bool {
	logRotateMu.Lock()
	defer logRotateMu.Unlock()
	return logRotateTicker != nil && logRotateStopChan != nil
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
		utils.LogWarn("persist_server_logs enabled but log path is empty; skipping file persistence")
		writeFile = false
	}

	if !writeFile && !writeDB {
		return
	}

	// Process each server concurrently with individual timeout handling
	// This prevents one slow/failed server from blocking others
	var wg sync.WaitGroup
	
	for _, server := range cfg.Servers {
		if utils.IsEmptyOrWhitespace(server.TableName) || utils.IsEmptyOrWhitespace(server.Address) {
			continue
		}

		wg.Add(1)
		go func(srv models.ServerEndpoint) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					utils.LogErrorWithContext("server-persistence", 
						fmt.Sprintf("Server persistence panic for '%s' (%s)", srv.Name, srv.Address),
						fmt.Errorf("panic: %v", r))
				}
			}()

			// Add timeout context for each server individually
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Use context-aware fetch with individual server timeout
			payload, err := fetchServerMonitoringWithContext(ctx, srv.Address)
			if err != nil {
				utils.LogWarnWithContext("server-monitoring", fmt.Sprintf("failed to fetch monitoring data from %s", srv.Address), err)
				return
			}

			// Update cache - this should be fast and not block
			if _, err := updateServerMetricsCache(srv, payload); err != nil {
				utils.LogWarnWithContext("server-monitoring", fmt.Sprintf("failed to parse server metrics from %s", srv.Address), err)
			}

			// File and database operations with error isolation
			if writeFile {
				if err := utils.WriteServerLogToFile(cfg.Path, srv, payload); err != nil {
					utils.LogWarnWithContext("server-monitoring", fmt.Sprintf("failed to write server log file for %s", srv.Address), err)
				}
			}

			if writeDB {
				if err := utils.WriteServerLogToDatabase(srv.TableName, payload); err != nil {
					utils.LogWarnWithContext("server-monitoring", fmt.Sprintf("failed to write server log to database for %s", srv.Address), err)
				}
			}
		}(server)
	}

	// Wait for all servers to complete with overall timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All servers completed successfully
	case <-time.After(60 * time.Second):
		utils.LogWarn("server persistence timed out after 60 seconds; some servers may still be processing")
	}
}

func fetchServerMonitoring(baseAddress string) ([]byte, error) {
	// Get timeout from environment configuration
	envConfig := config.GetEnvConfig()
	timeout := envConfig.ServerMonitoringTimeout

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return fetchServerMonitoringWithContext(ctx, baseAddress)
}

func fetchServerMonitoringWithContext(ctx context.Context, baseAddress string) ([]byte, error) {
	endpoint := strings.TrimRight(baseAddress, "/") + "/monitoring"

	// Use the centralized HTTP utility with resource limits
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	body := strings.NewReader("{}")
	payload, err := utils.MakeHTTPRequestWithLimits(ctx, http.MethodPost, endpoint, body, headers)

	if err != nil {
		// Provide more specific error messages for different failure types
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return nil, fmt.Errorf("server timeout: %w", err)
		}
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("server unavailable (connection refused): %w", err)
		}
		if strings.Contains(err.Error(), "no such host") {
			return nil, fmt.Errorf("server host not found: %w", err)
		}
		if strings.Contains(err.Error(), "network is unreachable") {
			return nil, fmt.Errorf("server network unreachable: %w", err)
		}
		return nil, fmt.Errorf("server communication failed: %w", err)
	}

	return payload, nil
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

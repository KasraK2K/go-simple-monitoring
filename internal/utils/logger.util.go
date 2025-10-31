package utils

import (
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	logConfig *models.MonitoringConfig
)

// InitLogger initializes the logger with configuration
func InitLogger(config *models.MonitoringConfig) {
	logConfig = config
}

// BuildMonitoringLogEntry converts a SystemMonitoring snapshot into the log entry structure used for persistence.
func BuildMonitoringLogEntry(data *models.SystemMonitoring) models.MonitoringLogEntry {
	if data == nil {
		return models.MonitoringLogEntry{}
	}

	return models.MonitoringLogEntry{
		Time: data.Timestamp.Format(time.RFC3339Nano),
		Body: map[string]any{
			"cpu_usage_percent":    data.CPU.UsagePercent,
			"cpu_cores":            data.CPU.CoreCount,
			"cpu_goroutines":       data.CPU.Goroutines,
			"cpu_load_average":     data.CPU.LoadAverage,
			"cpu_architecture":     data.CPU.Architecture,
			"ram_used_percent":     data.RAM.UsedPct,
			"ram_total_bytes":      data.RAM.TotalBytes,
			"ram_used_bytes":       data.RAM.UsedBytes,
			"ram_available_bytes":  data.RAM.AvailableBytes,
			"disk_used_percent":    getRootDiskMetric(data.DiskSpace, "used_percent"),
			"disk_total_bytes":     getRootDiskMetric(data.DiskSpace, "total_bytes"),
			"disk_used_bytes":      getRootDiskMetric(data.DiskSpace, "used_bytes"),
			"disk_available_bytes": getRootDiskMetric(data.DiskSpace, "available_bytes"),
			"disk_spaces":          data.DiskSpace, // Full disk array for detailed info
			"network_bytes_sent":   data.NetworkIO.BytesSent,
			"network_bytes_recv":   data.NetworkIO.BytesRecv,
			"network_packets_sent": data.NetworkIO.PacketsSent,
			"network_packets_recv": data.NetworkIO.PacketsRecv,
			"network_errors_in":    data.NetworkIO.ErrorsIn,
			"network_errors_out":   data.NetworkIO.ErrorsOut,
			"network_drops_in":     data.NetworkIO.DropsIn,
			"network_drops_out":    data.NetworkIO.DropsOut,
			"diskio_read_bytes":    data.DiskIO.ReadBytes,
			"diskio_write_bytes":   data.DiskIO.WriteBytes,
			"diskio_read_count":    data.DiskIO.ReadCount,
			"diskio_write_count":   data.DiskIO.WriteCount,
			"diskio_read_time":     data.DiskIO.ReadTime,
			"diskio_write_time":    data.DiskIO.WriteTime,
			"diskio_io_time":       data.DiskIO.IOTime,
			"process_total":        data.Process.TotalProcesses,
			"process_running":      data.Process.RunningProcs,
			"process_sleeping":     data.Process.SleepingProcs,
			"process_zombie":       data.Process.ZombieProcs,
			"process_stopped":      data.Process.StoppedProcs,
			"process_load_avg_1":   data.Process.LoadAvg1,
			"process_load_avg_5":   data.Process.LoadAvg5,
			"process_load_avg_15":  data.Process.LoadAvg15,
			"heartbeat":            formatHeartbeatForLog(data.Heartbeat),
		},
	}
}

// LogMonitoringData writes monitoring data to daily log file
func LogMonitoringData(data *models.SystemMonitoring) error {
	if logConfig == nil {
		return fmt.Errorf("logger not initialized")
	}

	// Create log entry
	logEntry := BuildMonitoringLogEntry(data)

	// Write to storage based on configuration
	switch logConfig.Storage {
	case "none":
		return nil
	case "file":
		return writeLogEntry(logEntry)
	case "db":
		return WriteToDatabase(logEntry)
	case "both":
		if err := writeLogEntry(logEntry); err != nil {
			return err
		}
		return WriteToDatabase(logEntry)
	default:
		return fmt.Errorf("invalid storage type: %s", logConfig.Storage)
	}
}

// formatHeartbeatForLog converts heartbeat data to log-friendly format
func formatHeartbeatForLog(heartbeat []models.ServerCheck) []map[string]any {
	var result []map[string]any

	for _, server := range heartbeat {
		serverData := map[string]any{
			"name":          server.Name,
			"url":           server.URL,
			"status":        string(server.Status),
			"response_ms":   server.ResponseMs,
			"response_time": server.ResponseTime,
		}

		result = append(result, serverData)
	}

	return result
}

// writeLogEntry writes a single log entry to the daily log file in JSON array format
func writeLogEntry(entry models.MonitoringLogEntry) error {
	// Generate filename based on current date
	now := time.Now()
	filename := fmt.Sprintf("%s.log", now.Format("2006-01-02"))
	logPath := filepath.Join(logConfig.Path, filename)

	// Ensure log directory exists
	if err := os.MkdirAll(logConfig.Path, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Read existing log entries
	var entries []models.MonitoringLogEntry

	// Check if file exists
	if _, err := os.Stat(logPath); err == nil {
		// File exists, read existing entries
		data, err := os.ReadFile(logPath)
		if err != nil {
			return fmt.Errorf("failed to read existing log file: %w", err)
		}

		// If file is not empty, unmarshal existing entries
		if len(data) > 0 {
			if err := json.Unmarshal(data, &entries); err != nil {
				// If unmarshal fails, start with empty array (file might be corrupted)
				entries = []models.MonitoringLogEntry{}
			}
		}
	}

	// Append new entry
	entries = append(entries, entry)

	// Marshal all entries to JSON (compact format for production)
	jsonData, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal log entries: %w", err)
	}

	// Write the complete JSON array to file
	if err := os.WriteFile(logPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

// WriteServerLogToFile persists remote server payloads into per-server log files.
func WriteServerLogToFile(basePath string, server models.ServerEndpoint, payload []byte) error {
	if strings.TrimSpace(basePath) == "" {
		return fmt.Errorf("log path is not configured")
	}

	if strings.TrimSpace(server.TableName) == "" {
		return nil
	}

	dirName := SanitizeFilesystemName(server.TableName)
	if dirName == "" {
		return nil
	}

	now := time.Now()
	serverDir := filepath.Join(basePath, "servers", dirName)
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return fmt.Errorf("failed to create server log directory: %w", err)
	}

	filename := fmt.Sprintf("%s.log", now.Format("2006-01-02"))
	logPath := filepath.Join(serverDir, filename)

	var entries []models.ServerLogEntry

	if data, err := os.ReadFile(logPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &entries); err != nil {
			entries = []models.ServerLogEntry{}
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read server log file: %w", err)
	}

	entry := models.ServerLogEntry{
		Time:    now.Format(time.RFC3339Nano),
		Payload: json.RawMessage(payload),
	}
	entries = append(entries, entry)

	jsonData, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal server log entries: %w", err)
	}

	if err := os.WriteFile(logPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write server log file: %w", err)
	}

	return nil
}

// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	if logConfig == nil {
		return ""
	}

	now := time.Now()
	filename := fmt.Sprintf("%s.log", now.Format("2006-01-02"))
	return filepath.Join(logConfig.Path, filename)
}

// CleanOldLogs removes log files older than specified days
func CleanOldLogs(daysToKeep int) error {
	if logConfig == nil {
		return fmt.Errorf("logger not initialized")
	}

	if logConfig.Path == "" {
		return nil
	}

	if _, err := os.Stat(logConfig.Path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to access log directory: %w", err)
	}

	// Read log directory
	files, err := os.ReadDir(logConfig.Path)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".log" {
			// Parse date from filename (YYYY-MM-DD.log)
			dateStr := file.Name()[:len(file.Name())-4] // Remove .log extension
			fileDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue // Skip files that don't match date format
			}

			// Remove file if older than cutoff
			if fileDate.Before(cutoffDate) {
				filePath := filepath.Join(logConfig.Path, file.Name())
				if err := os.Remove(filePath); err != nil {
					fmt.Printf("Warning: failed to remove old log file %s: %v\n", filePath, err)
				}
			}
		}
	}

	return nil
}

// getRootDiskMetric extracts a specific metric from the root disk (/) for backwards compatibility
func getRootDiskMetric(diskSpaces []models.DiskSpace, metric string) interface{} {
	// Find root disk (path="/") or use the first disk as fallback
	var rootDisk *models.DiskSpace
	for i := range diskSpaces {
		if diskSpaces[i].Path == "/" {
			rootDisk = &diskSpaces[i]
			break
		}
	}

	// If no root disk found, use the first disk
	if rootDisk == nil && len(diskSpaces) > 0 {
		rootDisk = &diskSpaces[0]
	}

	// If no disks at all, return zero values
	if rootDisk == nil {
		switch metric {
		case "used_percent":
			return float64(0)
		default:
			return uint64(0)
		}
	}

	// Return the requested metric
	switch metric {
	case "used_percent":
		return rootDisk.UsedPct
	case "total_bytes":
		return rootDisk.TotalBytes
	case "used_bytes":
		return rootDisk.UsedBytes
	case "available_bytes":
		return rootDisk.AvailableBytes
	default:
		return nil
	}
}

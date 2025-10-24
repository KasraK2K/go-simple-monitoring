package models

import "time"

type SystemMonitoring struct {
	Timestamp time.Time     `json:"timestamp"`
	CPU       CPU           `json:"cpu"`
	DiskSpace DiskSpace     `json:"disk_space"`
	RAM       RAM           `json:"ram"`
	Heartbeat []ServerCheck `json:"heartbeat"`
}

type CPU struct {
	UsagePercent float64 `json:"usage_percent"` // Overall CPU usage percentage
	CoreCount    int     `json:"core_count"`    // Number of CPU cores
	Goroutines   int     `json:"goroutines"`    // Number of active goroutines
	LoadAverage  string  `json:"load_average"`  // System load average (1m, 5m, 15m)
	Architecture string  `json:"architecture"`  // CPU architecture (e.g., "amd64")
}

type DiskSpace struct {
	Total     string  `json:"total"`     // Total disk space (e.g., "500 GB")
	Used      string  `json:"used"`      // Used disk space (e.g., "450 GB")
	Available string  `json:"available"` // Available disk space (e.g., "50 GB")
	UsedPct   float64 `json:"used_pct"`  // Used percentage
	TotalGB   float64 `json:"total_gb"`  // Total in GB for easy reference
	UsedGB    float64 `json:"used_gb"`   // Used in GB for easy reference
	AvailGB   float64 `json:"avail_gb"`  // Available in GB for easy reference
}

type RAM struct {
	Total       string  `json:"total"`        // Total RAM (e.g., "16 GB")
	Used        string  `json:"used"`         // Used RAM (e.g., "8 GB")
	Available   string  `json:"available"`    // Available RAM (e.g., "8 GB")
	UsedPct     float64 `json:"used_pct"`     // Used percentage
	BufferCache string  `json:"buffer_cache"` // Buffer/Cache (e.g., "2 GB")
	TotalGB     float64 `json:"total_gb"`     // Total in GB for easy reference
	UsedGB      float64 `json:"used_gb"`      // Used in GB for easy reference
	AvailGB     float64 `json:"avail_gb"`     // Available in GB for easy reference
}

type ServerCheck struct {
	Name         string       `json:"name"`
	URL          string       `json:"url"`
	Status       ServerStatus `json:"status"`
	ResponseTime string       `json:"response_time"` // Human-readable (e.g., "150ms")
	ResponseMs   int64        `json:"response_ms"`   // Response time in milliseconds
	LastChecked  time.Time    `json:"last_checked"`
	Error        string       `json:"error,omitempty"`
}

type ServerStatus string

const (
	ServerStatusUp   ServerStatus = "up"
	ServerStatusDown ServerStatus = "down"
)

type MonitoringConfig struct {
	Path        string         `json:"path"`         // Log file destination path
	RefreshTime string         `json:"refresh_time"` // Refresh interval (e.g., "2s", "30s")
	Servers     []ServerConfig `json:"servers"`
}

type ServerConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Timeout int    `json:"timeout"` // Timeout in seconds
}

type MonitoringLogEntry struct {
	Time string         `json:"time"`
	Body map[string]any `json:"body"`
}

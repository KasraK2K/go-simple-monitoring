package models

import "time"

type SystemMonitoring struct {
	Timestamp time.Time     `json:"timestamp"`
	CPU       CPU           `json:"cpu"`
	DiskSpace DiskSpace     `json:"disk_space"`
	RAM       RAM           `json:"ram"`
	NetworkIO NetworkIO     `json:"network_io"`
	DiskIO    DiskIO        `json:"disk_io"`
	Process   Process       `json:"process"`
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
	TotalBytes     uint64  `json:"total_bytes"`     // Total disk space in bytes
	UsedBytes      uint64  `json:"used_bytes"`      // Used disk space in bytes
	AvailableBytes uint64  `json:"available_bytes"` // Available disk space in bytes
	UsedPct        float64 `json:"used_pct"`        // Used percentage
}

type RAM struct {
	TotalBytes     uint64  `json:"total_bytes"`     // Total RAM in bytes
	UsedBytes      uint64  `json:"used_bytes"`      // Used RAM in bytes
	AvailableBytes uint64  `json:"available_bytes"` // Available RAM in bytes
	UsedPct        float64 `json:"used_pct"`        // Used percentage
	BufferBytes    uint64  `json:"buffer_bytes"`    // Buffer/Cache in bytes
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
	Path        string           `json:"path"`         // Log file destination path
	RefreshTime string           `json:"refresh_time"` // Refresh interval (e.g., "2s", "30s")
	Storage     string           `json:"storage"`      // Storage type: "file", "db", or "both"
	Heartbeat   []ServerConfig   `json:"heartbeat"`
	Servers     []ServerEndpoint `json:"servers"`
}

type ServerConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Timeout int    `json:"timeout"` // Timeout in seconds
}

type NetworkIO struct {
	BytesSent   uint64 `json:"bytes_sent"`   // Total bytes sent
	BytesRecv   uint64 `json:"bytes_recv"`   // Total bytes received
	PacketsSent uint64 `json:"packets_sent"` // Total packets sent
	PacketsRecv uint64 `json:"packets_recv"` // Total packets received
	ErrorsIn    uint64 `json:"errors_in"`    // Input errors
	ErrorsOut   uint64 `json:"errors_out"`   // Output errors
	DropsIn     uint64 `json:"drops_in"`     // Input drops
	DropsOut    uint64 `json:"drops_out"`    // Output drops
}

type ServerEndpoint struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type DiskIO struct {
	ReadBytes  uint64 `json:"read_bytes"`  // Total bytes read
	WriteBytes uint64 `json:"write_bytes"` // Total bytes written
	ReadCount  uint64 `json:"read_count"`  // Total read operations
	WriteCount uint64 `json:"write_count"` // Total write operations
	ReadTime   uint64 `json:"read_time"`   // Time spent reading (ms)
	WriteTime  uint64 `json:"write_time"`  // Time spent writing (ms)
	IOTime     uint64 `json:"io_time"`     // Time spent doing I/Os (ms)
}

type Process struct {
	TotalProcesses int     `json:"total_processes"` // Total number of processes
	RunningProcs   int     `json:"running_procs"`   // Running processes
	SleepingProcs  int     `json:"sleeping_procs"`  // Sleeping processes
	ZombieProcs    int     `json:"zombie_procs"`    // Zombie processes
	StoppedProcs   int     `json:"stopped_procs"`   // Stopped processes
	LoadAvg1       float64 `json:"load_avg_1"`      // 1-minute load average
	LoadAvg5       float64 `json:"load_avg_5"`      // 5-minute load average
	LoadAvg15      float64 `json:"load_avg_15"`     // 15-minute load average
}

type MonitoringLogEntry struct {
	Time string         `json:"time"`
	Body map[string]any `json:"body"`
}

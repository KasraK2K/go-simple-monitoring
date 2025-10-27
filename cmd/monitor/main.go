package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go-log/internal/api/logics"
	"go-log/internal/api/models"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
)

type Config struct {
	ServerURL   string
	RefreshRate time.Duration
	AuthToken   string
	ShowDetails bool
	CompactMode bool
}

type DisplayState struct {
	initialized bool
	lines       int
	startTime   time.Time
}

const (
	metricsFieldWidth  = 28
	metricsFieldSpacer = 2
	metricsValueWidth  = 12
	metricsStartRow    = 7
	statusLabelPrefix  = "   Status: "
)

const (
	colFirstValue  = 1 + metricsFieldWidth - metricsValueWidth
	colSecondValue = colFirstValue + metricsFieldWidth + metricsFieldSpacer
	colThirdValue  = colSecondValue + metricsFieldWidth + metricsFieldSpacer
	colFourthValue = colThirdValue + metricsFieldWidth + metricsFieldSpacer
)

var metricsTableRows = [][]string{
	{"CPU Usage:", "CPU Cores:", "CPU Arch:", "Goroutines:"},
	{"RAM Usage:", "RAM Total:", "RAM Used:", "RAM Available:"},
	{"Disk Usage:", "Disk Total:", "Disk Used:", "Disk Available:"},
	{"Network Sent:", "Network Received:", "Packets Sent:", "Packets Received:"},
	{"Disk I/O Read:", "Disk I/O Write:", "Read Operations:", "Write Operations:"},
	{"Processes Total:", "Processes Running:", "Processes Sleeping:", "Processes Zombie:"},
	{"Load Avg 1m:", "Load Avg 5m:", "Load Avg 15m:", "CPU Load Avg:"},
}

var (
	heartbeatTitleRow        = metricsStartRow + len(metricsTableRows) + 1
	heartbeatStatusRow       = heartbeatTitleRow + 1
	heartbeatServersStartRow = heartbeatStatusRow + 1
)

func main() {
	config := parseFlags()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Initialize display state
	state := &DisplayState{
		startTime: time.Now(),
	}

	// Setup exit handler
	go func() {
		<-c
		cleanupAndExit()
	}()

	if config.ServerURL == "" {
		// Use local monitoring logic if no server URL provided
		runLocalMonitoring(config, state)
	} else {
		// Fetch from remote monitoring endpoint
		runRemoteMonitoring(config, state)
	}
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.ServerURL, "url", "", "Monitoring server URL (e.g., http://localhost:3500/monitoring)")
	flag.DurationVar(&config.RefreshRate, "refresh", 2*time.Second, "Refresh rate (e.g., 2s, 500ms)")
	flag.StringVar(&config.AuthToken, "token", "", "Authentication token for remote monitoring")
	flag.BoolVar(&config.ShowDetails, "details", false, "Show detailed information")
	flag.BoolVar(&config.CompactMode, "compact", false, "Compact display mode")

	flag.Parse()
	return config
}

func runLocalMonitoring(config Config, state *DisplayState) {
	// Initialize servers configuration for CLI mode (no auto-logging)
	logics.InitServersConfigCLI()

	ticker := time.NewTicker(config.RefreshRate)
	defer ticker.Stop()

	// Initial display
	updateDisplay(nil, config, state, true)

	for range ticker.C {
		// Get monitoring data directly from local logic
		data, err := logics.MonitoringDataGenerator()
		if err != nil {
			updateErrorDisplay(fmt.Sprintf("Failed to get monitoring data: %v", err), state)
			continue
		}

		// Update display in place
		updateDisplay(data, config, state, false)
	}
}

func runRemoteMonitoring(config Config, state *DisplayState) {
	ticker := time.NewTicker(config.RefreshRate)
	defer ticker.Stop()

	// Initial display
	updateDisplay(nil, config, state, true)

	for range ticker.C {
		// Fetch from remote endpoint
		data, err := fetchRemoteData(config)
		if err != nil {
			updateErrorDisplay(fmt.Sprintf("Failed to fetch remote data: %v", err), state)
			continue
		}

		// Update display in place
		updateDisplay(data, config, state, false)
	}
}

func fetchRemoteData(config Config) (*models.SystemMonitoring, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", config.ServerURL, nil)
	if err != nil {
		return nil, err
	}

	if config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.AuthToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var data models.SystemMonitoring
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func updateDisplay(data *models.SystemMonitoring, config Config, state *DisplayState, initial bool) {
	if initial || !state.initialized {
		// Clear screen and draw initial layout
		clearScreen()
		drawInitialLayout(config)
		state.initialized = true
	}

	if data != nil {
		// Move cursor to data sections and update values
		updateTimestamp(data.Timestamp)
		updateCPUMetrics(data.CPU, config)
		updateRAMMetrics(data.RAM, config)
		updateDiskMetrics(data.DiskSpace, config)
		updateNetworkMetrics(data.NetworkIO, config)
		updateDiskIOMetrics(data.DiskIO, config)
		updateProcessMetrics(data.Process, config)
		updateLoadAverage(data.CPU, data.Process)
		updateHeartbeat(data.Heartbeat, config)
		updateUptime(state.startTime)
	}

	// Always return cursor to bottom
	fmt.Print("\033[999;1H")
}

func updateErrorDisplay(err string, state *DisplayState) {
	if !state.initialized {
		return
	}

	// Move to status line and show error
	saveCursor()
	moveCursor(2, 50)
	color.Red("ERROR: %s", err)
	restoreCursor()
}

func drawInitialLayout(config Config) {
	title := color.New(color.FgCyan, color.Bold)

	// Header
	title.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	title.Printf("║                            SYSTEM MONITORING                                 ║\n")
	title.Println("╠══════════════════════════════════════════════════════════════════════════════╣")
	title.Printf("║ Last Updated: %-30s │ Uptime: %-21s ║\n", "Loading...", "Starting...")
	title.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Static labels for metrics
	for _, row := range metricsTableRows {
		fmt.Printf("%-*s | %-*s| %-*s| %-*s\n",
			metricsFieldWidth, row[0],
			metricsFieldWidth, row[1],
			metricsFieldWidth, row[2],
			metricsFieldWidth, row[3],
		)
	}
	fmt.Println()

	// Heartbeat section
	fmt.Printf("🔍 HEARTBEAT MONITORING:\n")
	fmt.Printf("%s%-40s\n", statusLabelPrefix, "Checking servers...")
	// Get the actual number of configured servers
	servers := logics.GetServersConfig()
	serverCount := len(servers)
	if serverCount == 0 {
		fmt.Printf("   No servers configured\n")
	} else {
		// Only print loading lines for actual servers
		for range serverCount {
			fmt.Printf("   %-35s\n", "Loading...")
		}
	}
	// Reserve some extra blank lines for clean display updates
	for range 10 - serverCount {
		fmt.Printf("   %-35s\n", "")
	}

	fmt.Println()
	fmt.Printf("Controls: Ctrl+C to exit | Refresh: %v", config.RefreshRate)
	if config.ServerURL != "" {
		fmt.Printf(" | Remote: %s", config.ServerURL)
	} else {
		fmt.Printf(" | Mode: Local")
	}
	fmt.Println()
}

func updateTimestamp(timestamp time.Time) {
	moveCursor(4, 15)
	title := color.New(color.FgCyan, color.Bold)
	title.Printf(": %-30s", timestamp.Format("2006-01-02 15:04:05"))
}

func updateUptime(startTime time.Time) {
	uptime := time.Since(startTime)
	uptimeStr := fmt.Sprintf("%02d:%02d:%02d",
		int(uptime.Hours()),
		int(uptime.Minutes())%60,
		int(uptime.Seconds())%60)

	moveCursor(4, 57)
	title := color.New(color.FgCyan, color.Bold)
	title.Printf(" %-21s", uptimeStr)
}

func metricsRow(index int) int {
	return metricsStartRow + index
}

func updateCPUMetrics(cpu models.CPU, _ Config) {
	row := metricsRow(0)
	usageColor := getStatusColor(cpu.UsagePercent, 80, 60)
	usageStr := fmt.Sprintf("%*.2f", metricsValueWidth, cpu.UsagePercent)
	printValue(row, colFirstValue, metricsValueWidth, usageColor.Sprint(usageStr))

	coresStr := fmt.Sprintf("%*d", metricsValueWidth, cpu.CoreCount)
	printValue(row, colSecondValue, metricsValueWidth, coresStr)

	archStr := fmt.Sprintf("%*s", metricsValueWidth, truncateString(cpu.Architecture, metricsValueWidth))
	printValue(row, colThirdValue, metricsValueWidth, archStr)

	goroutinesStr := fmt.Sprintf("%*d", metricsValueWidth, cpu.Goroutines)
	printValue(row, colFourthValue, metricsValueWidth, goroutinesStr)
}

func updateRAMMetrics(ram models.RAM, _ Config) {
	row := metricsRow(1)
	usageColor := getStatusColor(ram.UsedPct, 80, 60)
	usageStr := fmt.Sprintf("%*.2f", metricsValueWidth, ram.UsedPct)
	printValue(row, colFirstValue, metricsValueWidth, usageColor.Sprint(usageStr))

	totalStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(ram.TotalBytes))
	printValue(row, colSecondValue, metricsValueWidth, totalStr)

	usedStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(ram.UsedBytes))
	printValue(row, colThirdValue, metricsValueWidth, usedStr)

	availStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(ram.AvailableBytes))
	printValue(row, colFourthValue, metricsValueWidth, availStr)
}

func updateDiskMetrics(disk models.DiskSpace, _ Config) {
	row := metricsRow(2)
	usageColor := getStatusColor(disk.UsedPct, 90, 70)
	usageStr := fmt.Sprintf("%*.2f", metricsValueWidth, disk.UsedPct)
	printValue(row, colFirstValue, metricsValueWidth, usageColor.Sprint(usageStr))

	totalStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(disk.TotalBytes))
	printValue(row, colSecondValue, metricsValueWidth, totalStr)

	usedStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(disk.UsedBytes))
	printValue(row, colThirdValue, metricsValueWidth, usedStr)

	availStr := fmt.Sprintf("%*s", metricsValueWidth, formatBytes(disk.AvailableBytes))
	printValue(row, colFourthValue, metricsValueWidth, availStr)
}

func updateLoadAverage(cpu models.CPU, process models.Process) {
	row := metricsRow(6)
	printValue(row, colFirstValue, metricsValueWidth, fmt.Sprintf("%*.2f", metricsValueWidth, process.LoadAvg1))
	printValue(row, colSecondValue, metricsValueWidth, fmt.Sprintf("%*.2f", metricsValueWidth, process.LoadAvg5))
	printValue(row, colThirdValue, metricsValueWidth, fmt.Sprintf("%*.2f", metricsValueWidth, process.LoadAvg15))

	if cpu.LoadAverage != "unavailable" {
		parts := strings.Split(cpu.LoadAverage, ",")
		if len(parts) > 0 {
			printValue(row, colFourthValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, strings.TrimSpace(parts[0])))
			return
		}
	}
	printValue(row, colFourthValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, "N/A"))
}

func updateNetworkMetrics(network models.NetworkIO, _ Config) {
	row := metricsRow(3)
	printValue(row, colFirstValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, formatBytes(network.BytesSent)))
	printValue(row, colSecondValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, formatBytes(network.BytesRecv)))
	printValue(row, colThirdValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, network.PacketsSent))
	printValue(row, colFourthValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, network.PacketsRecv))
}

func updateDiskIOMetrics(diskIO models.DiskIO, _ Config) {
	row := metricsRow(4)
	printValue(row, colFirstValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, formatBytes(diskIO.ReadBytes)))
	printValue(row, colSecondValue, metricsValueWidth, fmt.Sprintf("%*s", metricsValueWidth, formatBytes(diskIO.WriteBytes)))
	printValue(row, colThirdValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, diskIO.ReadCount))
	printValue(row, colFourthValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, diskIO.WriteCount))
}

func updateProcessMetrics(process models.Process, _ Config) {
	row := metricsRow(5)
	printValue(row, colFirstValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, process.TotalProcesses))
	printValue(row, colSecondValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, process.RunningProcs))
	printValue(row, colThirdValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, process.SleepingProcs))
	printValue(row, colFourthValue, metricsValueWidth, fmt.Sprintf("%*d", metricsValueWidth, process.ZombieProcs))
}

func updateHeartbeat(servers []models.ServerCheck, _ Config) {
	// Update status line
	moveCursor(heartbeatStatusRow, len(statusLabelPrefix)+1)
	if len(servers) == 0 {
		fmt.Printf("%-40s", "No servers configured")
		// Clear all server lines
		for i := range 10 {
			moveCursor(heartbeatServersStartRow+i, 1)
			fmt.Printf("   %-35s", "")
		}
		return
	}

	upCount := 0
	downCount := 0
	for _, server := range servers {
		if server.Status == models.ServerStatusUp {
			upCount++
		} else {
			downCount++
		}
	}

	statusText := fmt.Sprintf("%d servers: %d UP, %d DOWN", len(servers), upCount, downCount)
	fmt.Printf("%-40s", statusText)

	// Update individual server lines
	for i := range 10 {
		moveCursor(heartbeatServersStartRow+i, 1)
		if i < len(servers) {
			server := servers[i]
			statusIcon := "✅"
			statusColor := color.New(color.FgGreen)
			if server.Status == models.ServerStatusDown {
				statusIcon = "❌"
				statusColor = color.New(color.FgRed)
			}

			name := truncateString(server.Name, 20)
			status := statusColor.Sprintf("%-4s", strings.ToUpper(string(server.Status)))
			responseTime := fmt.Sprintf("%-8s", server.ResponseTime)

			fmt.Printf("   %s %-20s %s %s", statusIcon, name, status, responseTime)
		} else {
			// Clear unused lines - just empty space
			fmt.Printf("   %-35s", "")
		}
	}
}

func printValue(row, col, width int, value string) {
	moveCursor(row, col)
	fmt.Printf("%-*s", width, "")
	moveCursor(row, col)
	fmt.Print(value)
}

func getStatusColor(value, critical, warning float64) *color.Color {
	if value >= critical {
		return color.New(color.FgRed, color.Bold)
	} else if value >= warning {
		return color.New(color.FgYellow, color.Bold)
	}
	return color.New(color.FgGreen, color.Bold)
}

// Cursor control functions
func moveCursor(row, col int) {
	fmt.Printf("\033[%d;%dH", row, col)
}

func saveCursor() {
	fmt.Print("\033[s")
}

func restoreCursor() {
	fmt.Print("\033[u")
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func cleanupAndExit() {
	fmt.Print("\033[?25h") // Show cursor
	fmt.Print("\033[0m")   // Reset colors
	fmt.Println("\nGoodbye!")
	os.Exit(0)
}

// Utility functions
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
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

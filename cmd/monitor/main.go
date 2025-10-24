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
		updateLoadAverage(data.CPU)
		updateRAMMetrics(data.RAM, config)
		updateDiskMetrics(data.DiskSpace, config)
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
	fmt.Printf("🖥️  CPU:        %%     │ Cores:        │ Arch:         │ Goroutines:    \n")
	fmt.Printf("💾 RAM:        %%     │ Total:        │ Used:         │ Available:     \n")
	fmt.Printf("💽 DISK:       %%     │ Total:        │ Used:         │ Available:     \n")
	fmt.Printf("🔗 LOAD AVG:           │\n")
	fmt.Println()

	// Heartbeat section
	fmt.Printf("🔍 HEARTBEAT MONITORING:\n")
	fmt.Printf("   Status: Checking servers...\n")
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

func updateCPUMetrics(cpu models.CPU, _ Config) {
	// CPU usage percentage
	moveCursor(7, 12)
	color := getStatusColor(cpu.UsagePercent, 80, 60)
	fmt.Printf("%s", color.Sprintf("%6.2f", cpu.UsagePercent))

	// Cores
	moveCursor(7, 28)
	fmt.Printf("%8d", cpu.CoreCount)

	// Architecture
	moveCursor(7, 43)
	fmt.Printf("%-9s", cpu.Architecture)

	// Goroutines
	moveCursor(7, 67)
	fmt.Printf("%8d", cpu.Goroutines)
}

func updateRAMMetrics(ram models.RAM, _ Config) {
	// RAM usage percentage
	moveCursor(8, 12)
	color := getStatusColor(ram.UsedPct, 80, 60)
	fmt.Printf("%s", color.Sprintf("%6.2f", ram.UsedPct))

	// Total
	moveCursor(8, 28)
	fmt.Printf("%8s", formatShort(ram.Total))

	// Used
	moveCursor(8, 43)
	fmt.Printf("%-9s", formatShort(ram.Used))

	// Available
	moveCursor(8, 67)
	fmt.Printf("%8s", formatShort(ram.Available))
}

func updateDiskMetrics(disk models.DiskSpace, _ Config) {
	// Disk usage percentage
	moveCursor(9, 12)
	color := getStatusColor(disk.UsedPct, 90, 70)
	fmt.Printf("%s", color.Sprintf("%6.2f", disk.UsedPct))

	// Total
	moveCursor(9, 28)
	fmt.Printf("%8s", formatShort(disk.Total))

	// Used
	moveCursor(9, 43)
	fmt.Printf("%-9s", formatShort(disk.Used))

	// Available
	moveCursor(9, 67)
	fmt.Printf("%8s", formatShort(disk.Available))
}

func updateLoadAverage(cpu models.CPU) {
	moveCursor(10, 13)
	if cpu.LoadAverage != "unavailable" {
		fmt.Printf("%-20s", cpu.LoadAverage)
	} else {
		fmt.Printf("%-20s", "N/A")
	}
}

func updateHeartbeat(servers []models.ServerCheck, _ Config) {
	// Update status line
	moveCursor(13, 12)
	if len(servers) == 0 {
		fmt.Printf("%-40s", "No servers configured")
		// Clear all server lines
		for i := range 10 {
			moveCursor(14+i, 1)
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
		moveCursor(14+i, 1)
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
func formatShort(s string) string {
	// Convert "123.45 GB" to "123GB" for compact display
	parts := strings.Fields(s)
	if len(parts) >= 2 {
		return fmt.Sprintf("%.0f%s", parseFloat(parts[0]), parts[1])
	}
	return s
}

func parseFloat(s string) float64 {
	val := 0.0
	fmt.Sscanf(s, "%f", &val)
	return val
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

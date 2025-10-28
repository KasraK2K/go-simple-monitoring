package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sql.DB
)

// InitDatabase initializes the SQLite database
func InitDatabase() error {
	if db != nil {
		return nil // Already initialized
	}

	var err error
	db, err = sql.Open("sqlite3", "./monitoring.db")
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if not exists
	if err = createTable(); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// createTable creates the monitoring_logs table
func createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS monitoring_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_timestamp ON monitoring_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_created_at ON monitoring_logs(created_at);
	`

	_, err := db.Exec(query)
	return err
}

// WriteToDatabase writes a log entry to SQLite database
func WriteToDatabase(entry models.MonitoringLogEntry) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry for database: %w", err)
	}

	// Insert into database
	query := `INSERT INTO monitoring_logs (timestamp, data) VALUES (?, ?)`
	_, err = db.Exec(query, entry.Time, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write to database: %w", err)
	}

	return nil
}

// CloseDatabase closes the database connection if open
func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// GetFromDatabase retrieves a log entry from SQLite by timestamp
func GetFromDatabase(timestamp string) (*models.MonitoringLogEntry, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT timestamp, data FROM monitoring_logs WHERE timestamp = ? LIMIT 1`
	row := db.QueryRow(query, timestamp)

	var dbTimestamp, jsonData string
	err := row.Scan(&dbTimestamp, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to get from database: %w", err)
	}

	var entry models.MonitoringLogEntry
	err = json.Unmarshal([]byte(jsonData), &entry)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &entry, nil
}

// ListDatabaseTimestamps returns all timestamps in the database with optional prefix filter
func ListDatabaseTimestamps(prefix string) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT timestamp FROM monitoring_logs WHERE timestamp LIKE ? ORDER BY created_at DESC`
	rows, err := db.Query(query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var timestamps []string
	for rows.Next() {
		var timestamp string
		err := rows.Scan(&timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		timestamps = append(timestamps, timestamp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return timestamps, nil
}

// CleanOldDatabaseEntries removes database entries older than specified date
func CleanOldDatabaseEntries(cutoffDate time.Time) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `DELETE FROM monitoring_logs WHERE created_at < ?`
	result, err := db.Exec(query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old database entries\n", rowsAffected)
	return nil
}

// GetDatabaseStats returns basic statistics about the database
func GetDatabaseStats() (map[string]any, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	stats := make(map[string]any)

	// Count total entries
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM monitoring_logs").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count entries: %w", err)
	}
	stats["total_entries"] = count

	// Get oldest entry
	var oldestTimestamp sql.NullString
	err = db.QueryRow("SELECT timestamp FROM monitoring_logs ORDER BY created_at ASC LIMIT 1").Scan(&oldestTimestamp)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get oldest entry: %w", err)
	}
	if oldestTimestamp.Valid {
		stats["oldest_entry"] = oldestTimestamp.String
	}

	// Get newest entry
	var newestTimestamp sql.NullString
	err = db.QueryRow("SELECT timestamp FROM monitoring_logs ORDER BY created_at DESC LIMIT 1").Scan(&newestTimestamp)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get newest entry: %w", err)
	}
	if newestTimestamp.Valid {
		stats["newest_entry"] = newestTimestamp.String
	}

	return stats, nil
}

// IsDatabaseInitialized checks if the database is initialized and accessible
func IsDatabaseInitialized() bool {
	if db == nil {
		return false
	}

	// Test if database is still accessible
	err := db.Ping()
	return err == nil
}

// QueryFilteredMonitoringData retrieves monitoring data within a date range
func QueryFilteredMonitoringData(from, to string) ([]models.MonitoringLogEntry, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []any

	// Build query based on provided filters
	if from != "" && to != "" {
		query = `SELECT timestamp, data FROM monitoring_logs 
				WHERE created_at >= ? AND created_at <= ? 
				ORDER BY created_at DESC`
		args = []any{from, to}
	} else if from != "" {
		query = `SELECT timestamp, data FROM monitoring_logs 
				WHERE created_at >= ? 
				ORDER BY created_at DESC`
		args = []any{from}
	} else if to != "" {
		query = `SELECT timestamp, data FROM monitoring_logs 
				WHERE created_at <= ? 
				ORDER BY created_at DESC`
		args = []any{to}
	} else {
		return nil, fmt.Errorf("either from or to filter must be provided")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query filtered data: %w", err)
	}
	defer rows.Close()

	var entries []models.MonitoringLogEntry
	for rows.Next() {
		var timestamp, jsonData string
		err := rows.Scan(&timestamp, &jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var entry models.MonitoringLogEntry
		err = json.Unmarshal([]byte(jsonData), &entry)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal data: %w", err)
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, nil
}

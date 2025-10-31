package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const DefaultTableName = "[default]"

var (
	db              *sql.DB
	serverLogTables sync.Map
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

	// Create default table directly
	if err = ensureTable(DefaultTableName); err != nil {
		return fmt.Errorf("failed to create default table: %w", err)
	}

	return nil
}

// ensureTable creates a table with the given name if it doesn't exist
func ensureTable(tableName string) error {
	// Get clean name for index naming (remove brackets, quotes etc.)
	cleanName := SanitizeTableName(tableName)
	
	statements := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`, tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s(timestamp);`, cleanName, tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s(created_at);`, cleanName, tableName),
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to ensure table %s: %w", tableName, err)
		}
	}

	return nil
}

// writeToTableInternal is the internal implementation for writing to any table
func writeToTableInternal(tableName string, entry models.MonitoringLogEntry) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Convert entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry for database: %w", err)
	}

	// Insert into table
	query := fmt.Sprintf(`INSERT INTO %s (timestamp, data) VALUES (?, ?)`, tableName)
	_, err = db.Exec(query, entry.Time, string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to write to database: %w", err)
	}

	return nil
}

// WriteToDatabase writes a log entry to the default table
func WriteToDatabase(entry models.MonitoringLogEntry) error {
	return writeToTableInternal(DefaultTableName, entry)
}

// WriteToTable writes a log entry to a specific table
func WriteToTable(tableName string, entry models.MonitoringLogEntry) error {
	return writeToTableInternal(tableName, entry)
}

// WriteServerLogToDatabase writes remote server payloads into a dedicated table.
func WriteServerLogToDatabase(tableName string, payload []byte) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	sanitized, err := ensureServerLogTable(tableName)
	if err != nil {
		return err
	}

	entry := models.ServerLogEntry{
		Time:    time.Now().Format(time.RFC3339Nano),
		Payload: json.RawMessage(payload),
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal server log entry: %w", err)
	}

	query := fmt.Sprintf("INSERT INTO %s (timestamp, data) VALUES (?, ?)", sanitized)
	if _, err := db.Exec(query, entry.Time, string(jsonData)); err != nil {
		return fmt.Errorf("failed to write server log to database: %w", err)
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
	return GetFromTable(DefaultTableName, timestamp)
}

// GetFromTable retrieves a log entry from a specific table by timestamp
func GetFromTable(tableName, timestamp string) (*models.MonitoringLogEntry, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := fmt.Sprintf(`SELECT timestamp, data FROM %s WHERE timestamp = ? LIMIT 1`, tableName)
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
	return ListTableTimestamps(DefaultTableName, prefix)
}

// ListTableTimestamps returns all timestamps in a specific table with optional prefix filter
func ListTableTimestamps(tableName, prefix string) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := fmt.Sprintf(`SELECT timestamp FROM %s WHERE timestamp LIKE ? ORDER BY created_at DESC`, tableName)
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
	return CleanOldTableEntries(DefaultTableName, cutoffDate)
}

// CleanOldTableEntries removes entries from a specific table older than specified date
func CleanOldTableEntries(tableName string, cutoffDate time.Time) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE created_at < ?`, tableName)
	result, err := db.Exec(query, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("Cleaned up %d old entries from table %s\n", rowsAffected, tableName)
	return nil
}

// GetDatabaseStats returns basic statistics about the database
func GetDatabaseStats() (map[string]any, error) {
	return GetTableStats(DefaultTableName)
}

// GetTableStats returns basic statistics about a specific table
func GetTableStats(tableName string) (map[string]any, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	stats := make(map[string]any)

	// Count total entries
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count entries: %w", err)
	}
	stats["total_entries"] = count

	// Get oldest entry
	var oldestTimestamp sql.NullString
	err = db.QueryRow(fmt.Sprintf("SELECT timestamp FROM %s ORDER BY created_at ASC LIMIT 1", tableName)).Scan(&oldestTimestamp)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get oldest entry: %w", err)
	}
	if oldestTimestamp.Valid {
		stats["oldest_entry"] = oldestTimestamp.String
	}

	// Get newest entry
	var newestTimestamp sql.NullString
	err = db.QueryRow(fmt.Sprintf("SELECT timestamp FROM %s ORDER BY created_at DESC LIMIT 1", tableName)).Scan(&newestTimestamp)
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
	return QueryFilteredTableData(DefaultTableName, from, to)
}

// QueryFilteredTableData retrieves data from a specific table within a date range
func QueryFilteredTableData(tableName, from, to string) ([]models.MonitoringLogEntry, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []any

	// Build query based on provided filters
	if from != "" && to != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at >= ? AND created_at <= ? 
				ORDER BY created_at DESC`, tableName)
		args = []any{from, to}
	} else if from != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at >= ? 
				ORDER BY created_at DESC`, tableName)
		args = []any{from}
	} else if to != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at <= ? 
				ORDER BY created_at DESC`, tableName)
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

func ensureServerLogTable(rawName string) (string, error) {
	sanitized := SanitizeTableName(rawName)
	if sanitized == "" {
		return "", fmt.Errorf("invalid table name")
	}

	if _, exists := serverLogTables.Load(sanitized); exists {
		return sanitized, nil
	}

	if err := ensureTable(sanitized); err != nil {
		return "", err
	}

	serverLogTables.Store(sanitized, struct{}{})
	return sanitized, nil
}

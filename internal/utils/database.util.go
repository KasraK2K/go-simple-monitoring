package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-log/internal/api/models"
	"strings"
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

// CleanOldDatabaseEntries removes database entries older than specified date from all tables
func CleanOldDatabaseEntries(cutoffDate time.Time) error {
	var totalCleaned int64
	var errors []string
	var checkedTables []string

	fmt.Printf("Starting database cleanup for entries older than %s\n", cutoffDate.Format("2006-01-02 15:04:05"))

	// Clean default table first
	fmt.Printf("Checking default table: %s\n", DefaultTableName)
	checkedTables = append(checkedTables, DefaultTableName)
	if err := cleanTableEntries(DefaultTableName, cutoffDate, &totalCleaned); err != nil {
		errors = append(errors, fmt.Sprintf("default table: %v", err))
	}

	// Clean all server tables
	serverLogTables.Range(func(key, value any) bool {
		tableName := key.(string)
		fmt.Printf("Checking server table: %s\n", tableName)
		checkedTables = append(checkedTables, tableName)
		if err := cleanTableEntries(tableName, cutoffDate, &totalCleaned); err != nil {
			errors = append(errors, fmt.Sprintf("table %s: %v", tableName, err))
		}
		return true // continue iteration
	})

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed for some tables: %v", errors)
	}

	fmt.Printf("Database cleanup completed: %d old entries removed from %d tables (%s)\n", 
		totalCleaned, len(checkedTables), strings.Join(checkedTables, ", "))
	return nil
}

// cleanTableEntries is an internal helper that cleans a single table and accumulates the count
func cleanTableEntries(tableName string, cutoffDate time.Time, totalCleaned *int64) error {
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

	*totalCleaned += rowsAffected
	if rowsAffected > 0 {
		fmt.Printf("  ✓ Cleaned %d old entries from table %s\n", rowsAffected, tableName)
	} else {
		fmt.Printf("  ✓ No old entries found in table %s\n", tableName)
	}
	return nil
}

// CleanOldTableEntries removes entries from a specific table older than specified date
func CleanOldTableEntries(tableName string, cutoffDate time.Time) error {
	var totalCleaned int64
	return cleanTableEntries(tableName, cutoffDate, &totalCleaned)
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
		// No date filters, get all entries from the table
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s ORDER BY created_at DESC`, tableName)
		args = []any{}
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

// GetAvailableTables returns a list of available table names for querying
func GetAvailableTables() []string {
	tables := []string{"default"} // Return clean name for API
	
	// Add all server tables
	serverLogTables.Range(func(key, value any) bool {
		tableName := key.(string)
		tables = append(tables, tableName)
		return true // continue iteration
	})
	
	return tables
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

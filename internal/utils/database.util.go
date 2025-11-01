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

const DefaultTableName = "`default`"

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
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	tables, err := collectCleanupTables()
	if err != nil {
		return err
	}

	var totalCleaned int64
	var errors []string
	var checkedTables []string

	fmt.Printf("Starting database cleanup for entries older than %s\n", cutoffDate.Format("2006-01-02 15:04:05"))

	for _, tableName := range tables {
		displayName := displayTableName(tableName)
		if tableName == DefaultTableName {
			fmt.Printf("Checking default table: %s\n", displayName)
		} else {
			fmt.Printf("Checking table: %s\n", displayName)
		}

		checkedTables = append(checkedTables, displayName)
		if err := cleanTableEntries(tableName, cutoffDate, &totalCleaned); err != nil {
			errors = append(errors, fmt.Sprintf("table %s: %v", tableName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed for some tables: %v", errors)
	}

	fmt.Printf("Database cleanup completed: %d old entries removed from %d tables (%s)\n",
		totalCleaned, len(checkedTables), strings.Join(checkedTables, ", "))
	return nil
}

func collectCleanupTables() ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list database tables: %w", err)
	}
	defer rows.Close()

	tables := []string{DefaultTableName}
	existing := map[string]struct{}{
		strings.Trim(DefaultTableName, "`"): {},
	}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}

		if _, skip := existing[name]; skip {
			continue
		}

		tables = append(tables, name)
		serverLogTables.Store(name, struct{}{})
		existing[name] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate table names: %w", err)
	}

	return tables, nil
}

func displayTableName(tableName string) string {
	return strings.Trim(tableName, "`")
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
		fmt.Printf("  ✓ Cleaned %d old entries from table %s\n", rowsAffected, displayTableName(tableName))
	} else {
		fmt.Printf("  ✓ No old entries found in table %s\n", displayTableName(tableName))
	}
	return nil
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

// QueryFilteredTableData retrieves data from a specific table within a date range
func QueryFilteredTableData(tableName, from, to string) ([]models.MonitoringLogEntry, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []any

	fromNormalized, err := normalizeTimestampInput(from)
	if err != nil {
		return nil, fmt.Errorf("invalid from timestamp: %w", err)
	}

	toNormalized, err := normalizeTimestampInput(to)
	if err != nil {
		return nil, fmt.Errorf("invalid to timestamp: %w", err)
	}

	// Build query based on provided filters
	if fromNormalized != "" && toNormalized != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at >= ? AND created_at <= ? 
				ORDER BY created_at DESC`, tableName)
		args = []any{fromNormalized, toNormalized}
	} else if fromNormalized != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at >= ? 
				ORDER BY created_at DESC`, tableName)
		args = []any{fromNormalized}
	} else if toNormalized != "" {
		query = fmt.Sprintf(`SELECT timestamp, data FROM %s 
				WHERE created_at <= ? 
				ORDER BY created_at DESC`, tableName)
		args = []any{toNormalized}
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

func normalizeTimestampInput(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, value)
		if err == nil {
			localized := parsed.In(time.Local)
			return localized.Format("2006-01-02 15:04:05"), nil
		}
	}

	return "", fmt.Errorf("unsupported time format")
}

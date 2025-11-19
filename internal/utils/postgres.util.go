package utils

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "go-log/internal/api/models"
    "go-log/internal/config"
    "strings"
    "sync"
    "time"
    _ "github.com/jackc/pgx/v5/stdlib"
    _ "github.com/lib/pq"
)

var (
    pgdb *sql.DB
    pgMu sync.RWMutex
    timescaleDBAvailable bool
    timescaleCapabilitiesChecked bool
    timescaleCapabilitiesMu sync.RWMutex
    hypertableCache map[string]bool
    hypertableCacheMu sync.RWMutex
    tableStatusCache map[string]string // tracks table creation and hypertable conversion status
    tableStatusCacheMu sync.RWMutex
)

func pqQuoteIdent(name string) string {
    return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// InitPostgres initializes a PostgreSQL connection using synthesized DSN from POSTGRES_*.
// It tries the "pgx" driver name first, then falls back to "postgres".
func InitPostgres() error {
    dsn := strings.TrimSpace(config.GetEnvConfig().GetPostgresDSN())
    if IsEmptyOrWhitespace(dsn) {
        return fmt.Errorf("postgres configuration is incomplete")
    }

    // Open with pgx first
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        // Fallback to lib/pq driver name
        if strings.Contains(strings.ToLower(err.Error()), "unknown driver") {
            var err2 error
            db, err2 = sql.Open("postgres", dsn)
            if err2 != nil {
                return fmt.Errorf("failed to open postgres: %w", err2)
            }
        } else {
            return fmt.Errorf("failed to open postgres: %w", err)
        }
    }

    // Configure pool
    maxConn, connTimeout, idleTimeout := getDatabaseConfig()
    db.SetMaxOpenConns(maxConn)
    db.SetMaxIdleConns(maxConn / 2)
    db.SetConnMaxLifetime(time.Duration(connTimeout) * time.Second)
    db.SetConnMaxIdleTime(time.Duration(idleTimeout) * time.Second)

    // Ping
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        _ = db.Close()
        return fmt.Errorf("failed to ping postgres: %w", err)
    }

    pgMu.Lock()
    pgdb = db
    pgMu.Unlock()

    LogInfo("postgres initialized with max_connections=%d, connection_timeout=%ds, idle_timeout=%ds",
        maxConn, connTimeout, idleTimeout)
    
    // Initialize caches
    hypertableCacheMu.Lock()
    hypertableCache = make(map[string]bool)
    hypertableCacheMu.Unlock()
    
    tableStatusCacheMu.Lock()
    tableStatusCache = make(map[string]string)
    tableStatusCacheMu.Unlock()
    
    // Check TimescaleDB capabilities after successful connection
    checkTimescaleDBCapabilities()
    return nil
}

func IsPostgresInitialized() bool {
    pgMu.RLock()
    defer pgMu.RUnlock()
    return pgdb != nil
}

func ClosePostgres() error {
    pgMu.Lock()
    defer pgMu.Unlock()
    if pgdb != nil {
        err := pgdb.Close()
        pgdb = nil
        
        // Reset TimescaleDB capability cache
        timescaleCapabilitiesMu.Lock()
        timescaleDBAvailable = false
        timescaleCapabilitiesChecked = false
        timescaleCapabilitiesMu.Unlock()
        
        // Reset caches
        hypertableCacheMu.Lock()
        hypertableCache = nil
        hypertableCacheMu.Unlock()
        
        tableStatusCacheMu.Lock()
        tableStatusCache = nil
        tableStatusCacheMu.Unlock()
        
        if err != nil {
            LogWarnWithContext("postgres-close", "error closing PostgreSQL connection", err)
        } else {
            LogInfo("PostgreSQL connection closed successfully")
        }
        return err
    }
    return nil
}

// checkTimescaleDBCapabilities detects if TimescaleDB extension is available and enabled
func checkTimescaleDBCapabilities() {
    timescaleCapabilitiesMu.Lock()
    defer timescaleCapabilitiesMu.Unlock()
    
    if timescaleCapabilitiesChecked {
        return
    }
    
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    
    if db == nil {
        return
    }
    
    // Check if TimescaleDB extension is available
    var exists bool
    err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&exists)
    if err != nil {
        LogWarnWithContext("timescaledb-check", "failed to check TimescaleDB extension availability", err)
        timescaleDBAvailable = false
    } else {
        timescaleDBAvailable = exists
        if exists {
            LogInfo("TimescaleDB extension detected and available")
        } else {
            LogInfo("TimescaleDB extension not available, using standard PostgreSQL features")
        }
    }
    
    timescaleCapabilitiesChecked = true
}

// IsTimescaleDBAvailable returns true if TimescaleDB extension is available
func IsTimescaleDBAvailable() bool {
    timescaleCapabilitiesMu.RLock()
    defer timescaleCapabilitiesMu.RUnlock()
    
    if !timescaleCapabilitiesChecked {
        // Trigger capability check if not done yet
        timescaleCapabilitiesMu.RUnlock()
        checkTimescaleDBCapabilities()
        timescaleCapabilitiesMu.RLock()
    }
    
    return timescaleDBAvailable
}

// isHypertable checks if a table is already a TimescaleDB hypertable (with caching)
func isHypertable(db *sql.DB, tableName string) bool {
    // Check cache first
    hypertableCacheMu.RLock()
    if hypertableCache != nil {
        if cached, exists := hypertableCache[tableName]; exists {
            hypertableCacheMu.RUnlock()
            return cached
        }
    }
    hypertableCacheMu.RUnlock()
    
    // Query database
    var exists bool
    query := `SELECT EXISTS(
        SELECT 1 FROM timescaledb_information.hypertables 
        WHERE hypertable_name = $1
    )`
    err := db.QueryRow(query, tableName).Scan(&exists)
    isHypertableResult := err == nil && exists
    
    // Cache the result
    hypertableCacheMu.Lock()
    if hypertableCache != nil {
        hypertableCache[tableName] = isHypertableResult
    }
    hypertableCacheMu.Unlock()
    
    return isHypertableResult
}

// ClearPostgresCaches clears all PostgreSQL-related caches (useful for development/testing)
func ClearPostgresCaches() {
    hypertableCacheMu.Lock()
    if hypertableCache != nil {
        hypertableCache = make(map[string]bool)
    }
    hypertableCacheMu.Unlock()
    
    tableStatusCacheMu.Lock()
    if tableStatusCache != nil {
        tableStatusCache = make(map[string]string)
    }
    tableStatusCacheMu.Unlock()
    
    LogInfo("PostgreSQL caches cleared")
}

// getApproximateRowCount returns an approximate row count for performance
func getApproximateRowCount(db *sql.DB, tableName string) (int64, error) {
    var count int64
    
    // First try getting statistics from pg_class (fast but approximate)
    query := `SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE relname = $1`
    err := db.QueryRow(query, tableName).Scan(&count)
    if err != nil {
        // Fallback to exact count for small tables or if pg_class fails
        exactQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s", pqQuoteIdent(tableName))
        err = db.QueryRow(exactQuery).Scan(&count)
        if err != nil {
            return 0, fmt.Errorf("failed to get row count: %w", err)
        }
    }
    
    // If estimate is too low (table might be small), do exact count
    if count < 1000 {
        exactQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s", pqQuoteIdent(tableName))
        err = db.QueryRow(exactQuery).Scan(&count)
        if err != nil {
            return 0, fmt.Errorf("failed to get exact row count: %w", err)
        }
    }
    
    return count, nil
}

func pgSanitizeTable(name string) (string, error) {
    if IsEmptyOrWhitespace(name) {
        return "", fmt.Errorf("table name cannot be empty")
    }
    raw := strings.Trim(strings.TrimSpace(name), "`")
    sanitized := SanitizeTableName(raw)
    if sanitized == "" {
        return "", fmt.Errorf("invalid table name")
    }
    return sanitized, nil
}

func ensurePGTable(tableName string) (string, error) {
    name, err := pgSanitizeTable(tableName)
    if err != nil {
        return "", err
    }
    
    // Check if table setup is already complete
    tableStatusCacheMu.RLock()
    if tableStatusCache != nil {
        if status, exists := tableStatusCache[name]; exists && status == "complete" {
            tableStatusCacheMu.RUnlock()
            // Table setup already complete, skip all operations
            return name, nil
        }
    }
    tableStatusCacheMu.RUnlock()
    
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    if db == nil {
        return "", fmt.Errorf("postgres not initialized")
    }

    nameQuoted := pqQuoteIdent(name)
    stmts := []string{
        fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
            id SERIAL PRIMARY KEY,
            timestamp timestamptz NOT NULL,
            data jsonb NOT NULL
        );`, nameQuoted),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s (timestamp);`, name, nameQuoted),
    }

    for _, s := range stmts {
        if _, err := db.Exec(s); err != nil {
            return "", fmt.Errorf("failed to ensure table %s: %w", name, err)
        }
    }
    
    // Handle hypertable conversion if TimescaleDB is available
    if IsTimescaleDBAvailable() {
        // Check if table is already a hypertable
        if isHypertable(db, name) {
            // Table is already a hypertable, mark as complete
            tableStatusCacheMu.Lock()
            if tableStatusCache != nil {
                tableStatusCache[name] = "complete"
            }
            tableStatusCacheMu.Unlock()
            return name, nil
        }
        
        // Try to convert to hypertable
        hypertableQuery := fmt.Sprintf("SELECT create_hypertable('%s', 'timestamp', if_not_exists => TRUE)", name)
        if _, err := db.Exec(hypertableQuery); err != nil {
            // Check if error is due to existing data
            if strings.Contains(err.Error(), "not empty") {
                LogDebug("table %s contains data, skipping hypertable conversion (this is expected for existing tables)", name)
                // Mark as complete to avoid repeated attempts
                tableStatusCacheMu.Lock()
                if tableStatusCache != nil {
                    tableStatusCache[name] = "complete"
                }
                tableStatusCacheMu.Unlock()
            } else {
                // Other error - log as warning but still mark as complete to avoid spam
                LogWarnWithContext("timescaledb-hypertable", 
                    fmt.Sprintf("failed to convert table %s to hypertable", name), err)
                tableStatusCacheMu.Lock()
                if tableStatusCache != nil {
                    tableStatusCache[name] = "complete"
                }
                tableStatusCacheMu.Unlock()
            }
        } else {
            LogInfo("successfully converted table %s to TimescaleDB hypertable", name)
            // Update caches
            hypertableCacheMu.Lock()
            if hypertableCache != nil {
                hypertableCache[name] = true
            }
            hypertableCacheMu.Unlock()
            
            tableStatusCacheMu.Lock()
            if tableStatusCache != nil {
                tableStatusCache[name] = "complete"
            }
            tableStatusCacheMu.Unlock()
        }
    } else {
        // No TimescaleDB, just mark as complete
        tableStatusCacheMu.Lock()
        if tableStatusCache != nil {
            tableStatusCache[name] = "complete"
        }
        tableStatusCacheMu.Unlock()
    }
    
    return name, nil
}

// PreparePostgresServerTable ensures a server log table exists in Postgres.
func PreparePostgresServerTable(tableName string) error {
    if !IsPostgresInitialized() {
        return nil
    }
    _, err := ensurePGTable(tableName)
    return err
}

// WriteToPostgres writes a MonitoringLogEntry into the specified Postgres table.
func WriteToPostgres(tableName string, entry models.MonitoringLogEntry) error {
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    if db == nil {
        return fmt.Errorf("postgres not initialized")
    }

    sanitized, err := ensurePGTable(tableName)
    if err != nil {
        return err
    }

    jsonData, err := json.Marshal(entry)
    if err != nil {
        return fmt.Errorf("failed to marshal log entry: %w", err)
    }

    ts := NowUTC()
    if entry.Time != "" {
        if parsed, err := ParseTimestampUTC(entry.Time); err == nil {
            ts = parsed
        }
    }

    q := fmt.Sprintf(`INSERT INTO %s (timestamp, data) VALUES ($1, $2)`, pqQuoteIdent(sanitized))
    if _, err := db.Exec(q, ts, jsonData); err != nil {
        return fmt.Errorf("failed to write to postgres: %w", err)
    }
    return nil
}

// WriteServerLogToPostgres writes raw server payload into a server-specific Postgres table.
func WriteServerLogToPostgres(tableName string, payload []byte) error {
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    if db == nil {
        return fmt.Errorf("postgres not initialized")
    }

    sanitized, err := ensurePGTable(tableName)
    if err != nil {
        return err
    }

    entry := models.ServerLogEntry{
        Time:    FormatTimestampUTC(NowUTC()),
        Payload: json.RawMessage(payload),
    }

    jsonData, err := json.Marshal(entry)
    if err != nil {
        return fmt.Errorf("failed to marshal server log entry: %w", err)
    }

    q := fmt.Sprintf(`INSERT INTO %s (timestamp, data) VALUES ($1, $2)`, pqQuoteIdent(sanitized))
    if _, err := db.Exec(q, NowUTC(), jsonData); err != nil {
        return fmt.Errorf("failed to write server log to postgres: %w", err)
    }
    return nil
}

    // CleanOldPostgresEntries deletes rows older than cutoffDate from all non-system tables.
    func CleanOldPostgresEntries(cutoffDate time.Time) error {
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    if db == nil {
        return fmt.Errorf("postgres not initialized")
    }

    tables, err := collectPGTables(db)
    if err != nil {
        return err
    }

    var total int64
    for _, t := range tables {
        q := fmt.Sprintf(`DELETE FROM %s WHERE timestamp < $1`, pqQuoteIdent(t))
        res, err := db.Exec(q, cutoffDate)
        if err != nil {
            LogWarnWithContext("pg-cleanup", fmt.Sprintf("failed to clean table %s", t), err)
            continue
        }
        if n, _ := res.RowsAffected(); n > 0 {
            total += n
            LogInfo("  ✓ cleaned %d old entries from table %s", n, t)
        }
    }
    LogInfo("postgres cleanup completed: %d old entries removed from %d tables", total, len(tables))
    return nil
}

func collectPGTables(db *sql.DB) ([]string, error) {
    rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`)
    if err != nil {
        return nil, fmt.Errorf("failed to list postgres tables: %w", err)
    }
    defer rows.Close()

    var tables []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            return nil, fmt.Errorf("failed to scan table name: %w", err)
        }
        // Skip systemy names
        if strings.HasPrefix(name, "pg_") || strings.HasPrefix(name, "sql_") {
            continue
        }
        tables = append(tables, name)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("failed to iterate table names: %w", err)
    }
    return tables, nil
}

// QueryFilteredPostgresData retrieves data from Postgres within a date range with smart downsampling.
func QueryFilteredPostgresData(tableName, from, to string) ([]models.MonitoringLogEntry, error) {
    pgMu.RLock()
    db := pgdb
    pgMu.RUnlock()
    if db == nil {
        return nil, fmt.Errorf("postgres not initialized")
    }

    name, err := ensurePGTable(tableName)
    if err != nil {
        return nil, err
    }

    // Normalize timestamps
    fromNormalized, err := NormalizeTimestampForDB(from)
    if err != nil {
        return nil, fmt.Errorf("invalid from timestamp: %w", err)
    }
    toNormalized, err := NormalizeTimestampForDB(to)
    if err != nil {
        return nil, fmt.Errorf("invalid to timestamp: %w", err)
    }

    tbl := pqQuoteIdent(name)
    envCfg := config.GetEnvConfig()
    
    // If downsampling is disabled via boolean flag, return raw data
    if !envCfg.EnableDownsampling {
        return queryRawData(db, tbl, fromNormalized, toNormalized)
    }
    
    maxPointsCfg := envCfg.DownsampleMaxPoints

    // If downsampling is disabled via maxPoints being 0, return raw data
    if maxPointsCfg <= 0 {
        return queryRawData(db, tbl, fromNormalized, toNormalized)
    }

    // Use approximate row count for better performance
    totalRows, err := getApproximateRowCount(db, name)
    if err != nil {
        LogWarnWithContext("postgres-query", "failed to get approximate row count, using raw query", err)
        return queryRawData(db, tbl, fromNormalized, toNormalized)
    }

    shouldDownsample := totalRows > int64(maxPointsCfg)
    
    if !shouldDownsample {
        return queryRawData(db, tbl, fromNormalized, toNormalized)
    }

    // Try TimescaleDB time_bucket downsampling first, fallback to ntile
    if IsTimescaleDBAvailable() {
        entries, err := queryWithTimeBucket(db, tbl, fromNormalized, toNormalized, maxPointsCfg, totalRows)
        if err == nil {
            return entries, nil
        }
        LogWarnWithContext("postgres-query", "TimescaleDB time_bucket query failed, falling back to ntile", err)
    }

    // Fallback to ntile-based downsampling
    return queryWithNtile(db, tbl, fromNormalized, toNormalized, maxPointsCfg)
}

// queryRawData retrieves raw data without downsampling
func queryRawData(db *sql.DB, tbl, fromNormalized, toNormalized string) ([]models.MonitoringLogEntry, error) {
    var query string
    var args []any

    switch {
    case fromNormalized != "" && toNormalized != "":
        query = fmt.Sprintf("SELECT timestamp, data FROM %s WHERE timestamp >= $1 AND timestamp <= $2 ORDER BY timestamp DESC", tbl)
        args = []any{fromNormalized, toNormalized}
    case fromNormalized != "":
        query = fmt.Sprintf("SELECT timestamp, data FROM %s WHERE timestamp >= $1 ORDER BY timestamp DESC", tbl)
        args = []any{fromNormalized}
    case toNormalized != "":
        query = fmt.Sprintf("SELECT timestamp, data FROM %s WHERE timestamp <= $1 ORDER BY timestamp DESC", tbl)
        args = []any{toNormalized}
    default:
        query = fmt.Sprintf("SELECT timestamp, data FROM %s ORDER BY timestamp DESC LIMIT 1000", tbl)
        args = []any{}
    }

    return executeQuery(db, query, args)
}

// queryWithTimeBucket uses TimescaleDB's time_bucket for efficient downsampling
func queryWithTimeBucket(db *sql.DB, tbl, fromNormalized, toNormalized string, maxPoints int, _ int64) ([]models.MonitoringLogEntry, error) {
    // Calculate optimal bucket interval
    bucketInterval := calculateOptimalBucketInterval(fromNormalized, toNormalized, int64(maxPoints))
    
    var query string
    var args []any

    // Build time_bucket query with aggregated data reconstruction
    switch {
    case fromNormalized != "" && toNormalized != "":
        query = fmt.Sprintf(`
WITH bucketed AS (
  SELECT 
    time_bucket('%s', timestamp) as bucket_time,
    timestamp,
    data,
    ROW_NUMBER() OVER (PARTITION BY time_bucket('%s', timestamp) ORDER BY timestamp DESC) as rn
  FROM %s 
  WHERE timestamp >= $1 AND timestamp <= $2
)
SELECT timestamp, data
FROM bucketed
WHERE rn = 1
ORDER BY bucket_time DESC`, bucketInterval, bucketInterval, tbl)
        args = []any{fromNormalized, toNormalized}
    case fromNormalized != "":
        query = fmt.Sprintf(`
WITH bucketed AS (
  SELECT 
    time_bucket('%s', timestamp) as bucket_time,
    timestamp,
    data,
    ROW_NUMBER() OVER (PARTITION BY time_bucket('%s', timestamp) ORDER BY timestamp DESC) as rn
  FROM %s 
  WHERE timestamp >= $1
)
SELECT timestamp, data
FROM bucketed
WHERE rn = 1
ORDER BY bucket_time DESC`, bucketInterval, bucketInterval, tbl)
        args = []any{fromNormalized}
    case toNormalized != "":
        query = fmt.Sprintf(`
WITH bucketed AS (
  SELECT 
    time_bucket('%s', timestamp) as bucket_time,
    timestamp,
    data,
    ROW_NUMBER() OVER (PARTITION BY time_bucket('%s', timestamp) ORDER BY timestamp DESC) as rn
  FROM %s 
  WHERE timestamp <= $1
)
SELECT timestamp, data
FROM bucketed
WHERE rn = 1
ORDER BY bucket_time DESC`, bucketInterval, bucketInterval, tbl)
        args = []any{toNormalized}
    default:
        query = fmt.Sprintf(`
WITH bucketed AS (
  SELECT 
    time_bucket('%s', timestamp) as bucket_time,
    timestamp,
    data,
    ROW_NUMBER() OVER (PARTITION BY time_bucket('%s', timestamp) ORDER BY timestamp DESC) as rn
  FROM %s
)
SELECT timestamp, data
FROM bucketed
WHERE rn = 1
ORDER BY bucket_time DESC
LIMIT %d`, bucketInterval, bucketInterval, tbl, maxPoints)
        args = []any{}
    }

    return executeQuery(db, query, args)
}

// queryWithNtile uses ntile-based downsampling as fallback
func queryWithNtile(db *sql.DB, tbl, fromNormalized, toNormalized string, maxPoints int) ([]models.MonitoringLogEntry, error) {
    var query string
    var args []any
    var where string

    if fromNormalized != "" && toNormalized != "" {
        where = "WHERE timestamp >= $1 AND timestamp <= $2"
        args = []any{fromNormalized, toNormalized}
    } else if fromNormalized != "" {
        where = "WHERE timestamp >= $1"
        args = []any{fromNormalized}
    } else if toNormalized != "" {
        where = "WHERE timestamp <= $1"
        args = []any{toNormalized}
    } else {
        where = ""
        args = []any{}
    }

    tilesParam := len(args) + 1
    query = fmt.Sprintf(`
WITH q AS (
  SELECT timestamp, data,
         ntile($%d) OVER (ORDER BY timestamp DESC) AS bucket,
         ROW_NUMBER() OVER (ORDER BY timestamp DESC) AS rn
  FROM %s %s
), ranked AS (
  SELECT timestamp, data, bucket,
         ROW_NUMBER() OVER (PARTITION BY bucket ORDER BY rn) AS rnk
  FROM q
)
SELECT timestamp, data
FROM ranked
WHERE rnk = 1
ORDER BY bucket DESC`, tilesParam, tbl, where)
    
    args = append(args, maxPoints)
    return executeQuery(db, query, args)
}

// executeQuery executes a query and returns MonitoringLogEntry results
func executeQuery(db *sql.DB, query string, args []any) ([]models.MonitoringLogEntry, error) {
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to execute query: %w", err)
    }
    defer rows.Close()

    var entries []models.MonitoringLogEntry
    for rows.Next() {
        var ts time.Time
        var jsonData []byte
        if err := rows.Scan(&ts, &jsonData); err != nil {
            return nil, fmt.Errorf("failed to scan row: %w", err)
        }

        var entry models.MonitoringLogEntry
        if err := json.Unmarshal(jsonData, &entry); err != nil {
            LogWarnWithContext("postgres-query", "failed to unmarshal data, skipping row", err)
            continue
        }
        entries = append(entries, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return entries, nil
}

// calculateOptimalBucketInterval calculates the optimal time bucket interval
func calculateOptimalBucketInterval(fromNormalized, toNormalized string, targetPoints int64) string {
    if targetPoints <= 0 {
        return "5 minutes"
    }

    // Calculate time span between from and to
    var span time.Duration
    if fromNormalized != "" && toNormalized != "" {
        fromTime, err1 := time.Parse("2006-01-02 15:04:05", fromNormalized)
        toTime, err2 := time.Parse("2006-01-02 15:04:05", toNormalized)
        
        if err1 == nil && err2 == nil {
            if toTime.After(fromTime) {
                span = toTime.Sub(fromTime)
            }
        }
    }

    // If we can't determine span, use default
    if span <= 0 {
        return "5 minutes"
    }

    // Calculate ideal bucket size
    idealDuration := span / time.Duration(targetPoints)
    
    // Round to sensible intervals
    switch {
    case idealDuration >= 24*time.Hour:
        days := int(idealDuration.Hours() / 24)
        if days <= 0 {
            days = 1
        }
        return fmt.Sprintf("%d day", days)
    case idealDuration >= time.Hour:
        hours := int(idealDuration.Hours())
        if hours <= 0 {
            hours = 1
        }
        return fmt.Sprintf("%d hour", hours)
    case idealDuration >= time.Minute:
        minutes := int(idealDuration.Minutes())
        if minutes <= 0 {
            minutes = 1
        }
        return fmt.Sprintf("%d minute", minutes)
    default:
        return "1 minute"
    }
}


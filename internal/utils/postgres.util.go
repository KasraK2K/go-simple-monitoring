package utils

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "go-log/internal/api/models"
    "go-log/internal/config"
    "strconv"
    "strings"
    "sync"
    "time"
    _ "github.com/jackc/pgx/v5/stdlib"
    _ "github.com/lib/pq"
)

var (
    pgdb *sql.DB
    pgMu sync.RWMutex
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
        return err
    }
    return nil
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
    // Best-effort: convert to hypertable on 'timestamp' column if TimescaleDB is available
    // Ignore errors if extension is missing
    _, _ = db.Exec(fmt.Sprintf("SELECT create_hypertable('%s', 'timestamp', if_not_exists => TRUE)", name))
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

// QueryFilteredPostgresData retrieves data from Postgres within a date range.
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

    var query string
    var args []any

    fromNormalized, err := NormalizeTimestampForDB(from)
    if err != nil {
        return nil, fmt.Errorf("invalid from timestamp: %w", err)
    }
    toNormalized, err := NormalizeTimestampForDB(to)
    if err != nil {
        return nil, fmt.Errorf("invalid to timestamp: %w", err)
    }

    tbl := pqQuoteIdent(name)

    // Decide whether to downsample based on raw row count and env threshold
    maxPointsCfg := config.GetEnvConfig().DownsampleMaxPoints
    // If <= 0, disable downsampling entirely
    constDisabled := maxPointsCfg <= 0
    var countQuery string
    var countArgs []any
    switch {
    case fromNormalized != "" && toNormalized != "":
        countQuery = fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE timestamp >= $1 AND timestamp <= $2", tbl)
        countArgs = []any{fromNormalized, toNormalized}
    case fromNormalized != "":
        countQuery = fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE timestamp >= $1", tbl)
        countArgs = []any{fromNormalized}
    case toNormalized != "":
        countQuery = fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE timestamp <= $1", tbl)
        countArgs = []any{toNormalized}
    default:
        countQuery = fmt.Sprintf("SELECT COUNT(1) FROM %s", tbl)
        countArgs = []any{}
    }

    var totalRows int64
    if err := db.QueryRow(countQuery, countArgs...).Scan(&totalRows); err != nil {
        return nil, fmt.Errorf("failed to count rows: %w", err)
    }

    shouldBucket := !constDisabled && totalRows > int64(maxPointsCfg)

    if !shouldBucket {
        // Return raw rows without downsampling
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
            query = fmt.Sprintf("SELECT timestamp, data FROM %s ORDER BY timestamp DESC", tbl)
            args = []any{}
        }
        rows, err := db.Query(query, args...)
        if err != nil {
            return nil, fmt.Errorf("failed to query postgres data: %w", err)
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
                return nil, fmt.Errorf("failed to unmarshal data: %w", err)
            }
            entries = append(entries, entry)
        }
        if err := rows.Err(); err != nil {
            return nil, fmt.Errorf("row iteration error: %w", err)
        }
        return entries, nil
    }

    // Downsample by row count: use ntile(maxPoints) and pick 1 per tile to target exactly ~maxPoints
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
    args = append(args, maxPointsCfg)
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to query postgres data: %w", err)
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
            return nil, fmt.Errorf("failed to unmarshal data: %w", err)
        }
        entries = append(entries, entry)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("row iteration error: %w", err)
    }

    return entries, nil
}

// dynamicBucketInterval returns a Timescale interval string (e.g. "5 minutes") aiming for ~targetPoints
// based on the span between from and to and the observed totalRows.
func dynamicBucketInterval(fromNormalized, toNormalized string, totalRows int64, targetPoints int64) string {
    // Default to 1 minute if we cannot compute span
    span := 0 * time.Second
    const layout = "2006-01-02 15:04:05"
    if fromNormalized != "" && toNormalized != "" {
        if fromT, err1 := time.Parse(layout, fromNormalized); err1 == nil {
            if toT, err2 := time.Parse(layout, toNormalized); err2 == nil {
                if toT.Before(fromT) {
                    fromT, toT = toT, fromT
                }
                span = toT.Sub(fromT)
            }
        }
    }
    if span <= 0 || targetPoints <= 0 {
        return "1 minute"
    }

    // Compute desired seconds per bucket so that buckets ~= targetPoints (ceil to avoid under-bucketing)
    spanSec := int64(span.Seconds())
    desired := spanSec / targetPoints
    if spanSec%targetPoints != 0 {
        desired++
    }
    if desired <= 0 {
        desired = 60 // 1 minute minimum
    }

    // Express interval in a compact unit without big jumps (e.g., 3 days when needed)
    if desired >= 86400 {
        days := (desired + 86400 - 1) / 86400 // ceil
        return fmt.Sprintf("%d days", days)
    }
    if desired >= 3600 {
        hours := (desired + 3600 - 1) / 3600 // ceil
        return fmt.Sprintf("%d hours", hours)
    }
    if desired >= 60 {
        minutes := (desired + 60 - 1) / 60 // ceil
        return fmt.Sprintf("%d minutes", minutes)
    }
    return fmt.Sprintf("%d seconds", desired)
}

// dateTruncUnitForInterval maps an interval string like "5 minutes" to a date_trunc unit
// for fallback when Timescale time_bucket is unavailable.
func dateTruncUnitForInterval(interval string) string {
    s := strings.ToLower(strings.TrimSpace(interval))
    if strings.Contains(s, "second") {
        return "second"
    }
    if strings.Contains(s, "minute") {
        return "minute"
    }
    if strings.Contains(s, "hour") {
        return "hour"
    }
    if strings.Contains(s, "day") {
        return "day"
    }
    return ""
}

// intervalToSeconds parses an interval string produced by dynamicBucketInterval
// like "5 minutes", "2 hours", "3 days" to seconds.
func intervalToSeconds(interval string) int64 {
    s := strings.TrimSpace(strings.ToLower(interval))
    if s == "" {
        return 0
    }
    // Expect format: "<number> <unit>"
    parts := strings.Fields(s)
    if len(parts) < 2 {
        // maybe it's just a number
        if n, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
            return n
        }
        return 0
    }
    n, err := strconv.ParseInt(parts[0], 10, 64)
    if err != nil || n <= 0 {
        return 0
    }
    u := parts[1]
    switch {
    case strings.HasPrefix(u, "second"):
        return n
    case strings.HasPrefix(u, "minute"):
        return n * 60
    case strings.HasPrefix(u, "hour"):
        return n * 3600
    case strings.HasPrefix(u, "day"):
        return n * 86400
    default:
        return 0
    }
}

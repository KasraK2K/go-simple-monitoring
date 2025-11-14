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
            data jsonb NOT NULL,
            created_at timestamptz NOT NULL DEFAULT now()
        );`, nameQuoted),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s (timestamp);`, name, nameQuoted),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s (created_at);`, name, nameQuoted),
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
        q := fmt.Sprintf(`DELETE FROM %s WHERE created_at < $1`, pqQuoteIdent(t))
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
    if fromNormalized != "" && toNormalized != "" {
        query = fmt.Sprintf(`SELECT timestamp, data FROM %s WHERE created_at >= $1 AND created_at <= $2 ORDER BY created_at DESC`, tbl)
        args = []any{fromNormalized, toNormalized}
    } else if fromNormalized != "" {
        query = fmt.Sprintf(`SELECT timestamp, data FROM %s WHERE created_at >= $1 ORDER BY created_at DESC`, tbl)
        args = []any{fromNormalized}
    } else if toNormalized != "" {
        query = fmt.Sprintf(`SELECT timestamp, data FROM %s WHERE created_at <= $1 ORDER BY created_at DESC`, tbl)
        args = []any{toNormalized}
    } else {
        query = fmt.Sprintf(`SELECT timestamp, data FROM %s ORDER BY created_at DESC`, tbl)
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

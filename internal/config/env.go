package config

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

// EnvConfig holds all environment variable configurations
type EnvConfig struct {
	// Server Configuration
	Port string

	// Security
	AESSecret string
	JWTSecret string

	// Environment
	Environment string

	// CORS
	CORSAllowedOrigins string

	// Rate Limiting
	RateLimitEnabled bool
	RateLimitRPS     float64
	RateLimitBurst   int

	// Logging
	LogLevel string

	// Token Validation
	CheckToken bool

	// Dashboard
	HasDashboard          bool
	DashboardDefaultRange string

    // Paths
    BaseLogFolder string
    SQLiteDSN     string

	// Database
	DBMaxConnections    int
	DBConnectionTimeout int
    DBIdleTimeout       int

    // Postgres
    PostgresUser string
    PostgresPassword string
    PostgresHost string
    PostgresPort string
    PostgresDB   string

    // Monitoring
    MonitorConfigPath       string
    ServerMonitoringTimeout time.Duration

    // Downsampling
    // If 0 or unset: disable server-side downsampling for Postgres historical queries
    // If >0: target approximately this many points via bucketing
    DownsampleMaxPoints int

	// HTTP Client
	HTTPMaxConnsPerHost       int
	HTTPMaxIdleConns          int
	HTTPMaxIdleConnsPerHost   int
	HTTPIdleConnTimeout       time.Duration
	HTTPConnectTimeout        time.Duration
	HTTPRequestTimeout        time.Duration
	HTTPResponseHeaderTimeout time.Duration
	HTTPMaxResponseSize       int64
	HTTPTLSHandshakeTimeout   time.Duration

	// Time Configuration
	DisableUTCEnforcement bool
	DefaultTimezone       string
}

var allowedDashboardRanges = map[string]struct{}{
	"1h":  {},
	"6h":  {},
	"24h": {},
	"7d":  {},
	"30d": {},
}

var envConfig *EnvConfig

// InitEnvConfig initializes the environment configuration
func InitEnvConfig() {
    envConfig = &EnvConfig{
		// Server Configuration
		Port: getEnvString("PORT", "3500"),

		// Security
		AESSecret: getEnvString("AES_SECRET", ""),
		JWTSecret: getEnvString("JWT_SECRET", ""),

		// Environment
		Environment: getEnvironment(),

		// CORS
		CORSAllowedOrigins: getEnvString("CORS_ALLOWED_ORIGINS", "http://localhost:3500,http://127.0.0.1:3500"),

		// Rate Limiting
		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRPS:     getEnvFloat("RATE_LIMIT_RPS", 10.0),
		RateLimitBurst:   getEnvInt("RATE_LIMIT_BURST", 20),

		// Logging
		LogLevel: getEnvString("LOG_LEVEL", "INFO"),

		// Token Validation
		CheckToken: getEnvBool("CHECK_TOKEN", false),

		// Dashboard
		HasDashboard:          getEnvBool("HAS_DASHBOARD", true),
		DashboardDefaultRange: sanitizeDashboardRange(getEnvString("DASHBOARD_DEFAULT_RANGE", "")),

		// Paths
        BaseLogFolder: getEnvString("BASE_LOG_FOLDER", "./logs"),
        SQLiteDSN:     getEnvString("SQLITE_DNS", "./monitoring.db"),

		// Database
		DBMaxConnections:    getEnvInt("DB_MAX_CONNECTIONS", 10),
		DBConnectionTimeout: getEnvInt("DB_CONNECTION_TIMEOUT", 30),
        DBIdleTimeout:       getEnvInt("DB_IDLE_TIMEOUT", 300),

        // Postgres
        PostgresUser: getEnvString("POSTGRES_USER", "monitoring"),
        PostgresPassword: getEnvString("POSTGRES_PASSWORD", "monitoring"),
        PostgresHost: getEnvString("POSTGRES_HOST", "localhost"),
        PostgresPort: getEnvString("POSTGRES_PORT", "5432"),
        PostgresDB: getEnvString("POSTGRES_DB", "monitoring"),

        // Monitoring
        MonitorConfigPath:       getEnvString("MONITOR_CONFIG_PATH", ""),
        ServerMonitoringTimeout: getEnvDuration("SERVER_MONITORING_TIMEOUT", 15*time.Second),

        // Downsampling
        DownsampleMaxPoints: getEnvInt("MONITORING_DOWNSAMPLE_MAX_POINTS", 0),

		// HTTP Client
		HTTPMaxConnsPerHost:       getEnvInt("HTTP_MAX_CONNS_PER_HOST", 10),
		HTTPMaxIdleConns:          getEnvInt("HTTP_MAX_IDLE_CONNS", 100),
		HTTPMaxIdleConnsPerHost:   getEnvInt("HTTP_MAX_IDLE_CONNS_PER_HOST", 5),
		HTTPIdleConnTimeout:       getEnvDuration("HTTP_IDLE_CONN_TIMEOUT", 90*time.Second),
		HTTPConnectTimeout:        getEnvDuration("HTTP_CONNECT_TIMEOUT", 10*time.Second),
		HTTPRequestTimeout:        getEnvDuration("HTTP_REQUEST_TIMEOUT", 30*time.Second),
		HTTPResponseHeaderTimeout: getEnvDuration("HTTP_RESPONSE_HEADER_TIMEOUT", 10*time.Second),
		HTTPMaxResponseSize:       getEnvInt64("HTTP_MAX_RESPONSE_SIZE", 10485760), // 10MB
		HTTPTLSHandshakeTimeout:   getEnvDuration("HTTP_TLS_HANDSHAKE_TIMEOUT", 10*time.Second),

		// Time Configuration
		DisableUTCEnforcement: getEnvBool("DISABLE_UTC_ENFORCEMENT", false),
		DefaultTimezone:       getEnvString("DEFAULT_TIMEZONE", "UTC"),
	}
}

// GetEnvConfig returns the current environment configuration
func GetEnvConfig() *EnvConfig {
	if envConfig == nil {
		InitEnvConfig()
	}
	return envConfig
}

// Helper functions for reading environment variables with defaults

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

func getEnvironment() string {
	// Check multiple possible environment variable names
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	if env == "" {
		env = "development" // Default
	}
	return env
}

func sanitizeDashboardRange(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	if _, ok := allowedDashboardRanges[trimmed]; ok {
		return trimmed
	}
	return ""
}

// Convenience methods for common checks

// IsProduction returns true if the environment is production
func (c *EnvConfig) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// ShouldCheckTokenInProduction returns true if token checking is enabled
func (c *EnvConfig) ShouldCheckTokenInProduction() bool {
	return c.CheckToken
}

// IsDashboardEnabled returns true if dashboard is enabled
func (c *EnvConfig) IsDashboardEnabled() bool {
	return c.HasDashboard
}

// GetDashboardDefaultRange returns the sanitized default range preset for the dashboard
func (c *EnvConfig) GetDashboardDefaultRange() string {
	return c.DashboardDefaultRange
}

// GetDatabasePath returns the full path to the database file
func (c *EnvConfig) GetDatabasePath() string {
    // Return SQLite DSN/path directly
    return c.SQLiteDSN
}

// GetPostgresDSN returns DSN if set, otherwise synthesizes one from POSTGRES_* vars.
func (c *EnvConfig) GetPostgresDSN() string {
    user := strings.TrimSpace(c.PostgresUser)
    pass := strings.TrimSpace(c.PostgresPassword)
    host := strings.TrimSpace(c.PostgresHost)
    port := strings.TrimSpace(c.PostgresPort)
    db := strings.TrimSpace(c.PostgresDB)
    if user == "" || host == "" || port == "" || db == "" {
        return ""
    }
    return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}

// IsRateLimitEnabled returns true if rate limiting is enabled
func (c *EnvConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

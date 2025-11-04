package utils

import (
	"fmt"
	"go-log/internal/config"
	"time"
)

// TimeConfig holds timezone configuration
type TimeConfig struct {
	UseUTC      bool
	DefaultZone *time.Location
}

var (
	timeConfig = &TimeConfig{
		UseUTC:      true, // Default to UTC for consistency
		DefaultZone: time.UTC,
	}
)

// InitTimeConfig initializes timezone configuration from environment
func InitTimeConfig() {
	envConfig := config.GetEnvConfig()
	
	// Check if UTC enforcement is disabled
	if envConfig.DisableUTCEnforcement {
		timeConfig.UseUTC = false
		timeConfig.DefaultZone = time.Local
		LogInfo("UTC enforcement disabled, using local timezone")
	} else {
		timeConfig.UseUTC = true
		timeConfig.DefaultZone = time.UTC
		LogInfo("using UTC timezone for consistency")
	}

	// Allow custom timezone configuration
	if envConfig.DefaultTimezone != "" && envConfig.DefaultTimezone != "UTC" {
		if loc, err := time.LoadLocation(envConfig.DefaultTimezone); err == nil {
			timeConfig.DefaultZone = loc
			LogInfo("using custom timezone: %s", envConfig.DefaultTimezone)
		} else {
			LogWarnWithContext("time-config", fmt.Sprintf("invalid timezone '%s', falling back to UTC", envConfig.DefaultTimezone), err)
		}
	}
}

// NowUTC returns the current time in UTC
func NowUTC() time.Time {
	return time.Now().UTC()
}

// NowDefault returns the current time in the configured default timezone
func NowDefault() time.Time {
	if timeConfig.UseUTC {
		return NowUTC()
	}
	return time.Now().In(timeConfig.DefaultZone)
}

// FormatTimestamp formats a time consistently using RFC3339Nano in the default timezone
func FormatTimestamp(t time.Time) string {
	if timeConfig.UseUTC {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return t.In(timeConfig.DefaultZone).Format(time.RFC3339Nano)
}

// FormatTimestampUTC formats a time consistently using RFC3339Nano in UTC
func FormatTimestampUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// ParseTimestamp parses a timestamp string and returns it in the default timezone
func ParseTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	var parsed time.Time
	var err error
	
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, value)
		if err == nil {
			// Always return in the configured default timezone
			return parsed.In(timeConfig.DefaultZone), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

// ParseTimestampUTC parses a timestamp string and returns it in UTC
func ParseTimestampUTC(value string) (time.Time, error) {
	parsed, err := ParseTimestamp(value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

// NormalizeTimestampForDB normalizes a timestamp for database storage (always in UTC)
func NormalizeTimestampForDB(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	parsed, err := ParseTimestamp(value)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Always store in UTC for database consistency
	return parsed.UTC().Format("2006-01-02 15:04:05"), nil
}

// NormalizeTimestampInput normalizes user input timestamps with timezone awareness
func NormalizeTimestampInput(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	parsed, err := ParseTimestamp(value)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Always store in UTC for consistency, but format for database
	return parsed.UTC().Format("2006-01-02 15:04:05"), nil
}

// ConvertToUserTimezone converts a UTC timestamp to user's preferred timezone for display
func ConvertToUserTimezone(utcTime time.Time, userTZ string) time.Time {
	if userTZ == "" || userTZ == "UTC" {
		return utcTime.UTC()
	}

	if loc, err := time.LoadLocation(userTZ); err == nil {
		return utcTime.In(loc)
	}

	// Fallback to default timezone
	return utcTime.In(timeConfig.DefaultZone)
}

// GetDefaultTimezone returns the configured default timezone
func GetDefaultTimezone() *time.Location {
	return timeConfig.DefaultZone
}

// IsUTCEnforced returns whether UTC enforcement is enabled
func IsUTCEnforced() bool {
	return timeConfig.UseUTC
}

// ValidateTimezone validates if a timezone string is valid
func ValidateTimezone(tz string) error {
	if tz == "" {
		return fmt.Errorf("timezone cannot be empty")
	}

	if tz == "UTC" || tz == "Local" {
		return nil
	}

	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone '%s': %w", tz, err)
	}

	return nil
}
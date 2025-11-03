package utils

import (
	"errors"
	"fmt"
)

// Custom error types for better error categorization and handling

// Authentication and Authorization errors
var (
	ErrInvalidToken        = errors.New("invalid or malformed token")
	ErrTokenExpired        = errors.New("token has expired")
	ErrInvalidCredentials  = errors.New("invalid credentials provided")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrInvalidClaims       = errors.New("invalid token claims")
	ErrTokenParsingFailed  = errors.New("failed to parse token")
)

// Data Processing errors
var (
	ErrInvalidDataFormat   = errors.New("invalid data format")
	ErrDataMarshalFailed   = errors.New("failed to marshal data")
	ErrDataUnmarshalFailed = errors.New("failed to unmarshal data")
	ErrTypeCastFailed      = errors.New("type assertion failed")
	ErrValidationFailed    = errors.New("data validation failed")
)

// Configuration errors
var (
	ErrConfigNotFound      = errors.New("configuration not found")
	ErrInvalidConfig       = errors.New("invalid configuration")
	ErrConfigurationError  = errors.New("configuration error")
)

// Network and HTTP errors
var (
	ErrNetworkError        = errors.New("network error")
	ErrHTTPRequestFailed   = errors.New("HTTP request failed")
	ErrConnectionTimeout   = errors.New("connection timeout")
	ErrResponseTooLarge    = errors.New("response size exceeds limit")
)

// File System errors
var (
	ErrFileNotFound        = errors.New("file not found")
	ErrFileAccessDenied    = errors.New("file access denied")
	ErrInvalidPath         = errors.New("invalid file path")
	ErrDirectoryTraversal  = errors.New("directory traversal detected")
)

// Database errors
var (
	ErrDatabaseNotInit     = errors.New("database not initialized")
	ErrDatabaseConnection  = errors.New("database connection error")
	ErrQueryFailed         = errors.New("database query failed")
	ErrTransactionFailed   = errors.New("database transaction failed")
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypeData       ErrorType = "data_processing"
	ErrorTypeConfig     ErrorType = "configuration"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeFileSystem ErrorType = "filesystem"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeInternal   ErrorType = "internal"
)

// CategorizedError wraps an error with additional context and categorization
type CategorizedError struct {
	Type    ErrorType
	Code    string
	Message string
	Cause   error
	Context map[string]any
}

func (e *CategorizedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Type, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

func (e *CategorizedError) Unwrap() error {
	return e.Cause
}

func (e *CategorizedError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	if categorizedTarget, ok := target.(*CategorizedError); ok {
		return e.Type == categorizedTarget.Type && e.Code == categorizedTarget.Code
	}
	
	return errors.Is(e.Cause, target)
}

// NewCategorizedError creates a new categorized error
func NewCategorizedError(errorType ErrorType, code, message string, cause error) *CategorizedError {
	return &CategorizedError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]any),
	}
}

// WithContext adds context information to the error
func (e *CategorizedError) WithContext(key string, value any) *CategorizedError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// Common error creation functions for consistency

// NewAuthError creates an authentication/authorization error
func NewAuthError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeAuth, code, message, cause)
}

// NewDataError creates a data processing error
func NewDataError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeData, code, message, cause)
}

// NewConfigError creates a configuration error
func NewConfigError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeConfig, code, message, cause)
}

// NewNetworkError creates a network error
func NewNetworkError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeNetwork, code, message, cause)
}

// NewFileSystemError creates a file system error
func NewFileSystemError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeFileSystem, code, message, cause)
}

// NewDatabaseError creates a database error
func NewDatabaseError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeDatabase, code, message, cause)
}

// NewValidationError creates a validation error
func NewValidationError(code, message string, cause error) *CategorizedError {
	return NewCategorizedError(ErrorTypeValidation, code, message, cause)
}

// Error type checking functions

// IsAuthError checks if error is authentication/authorization related
func IsAuthError(err error) bool {
	var categorizedErr *CategorizedError
	return errors.As(err, &categorizedErr) && categorizedErr.Type == ErrorTypeAuth
}

// IsDataError checks if error is data processing related
func IsDataError(err error) bool {
	var categorizedErr *CategorizedError
	return errors.As(err, &categorizedErr) && categorizedErr.Type == ErrorTypeData
}

// IsNetworkError checks if error is network related
func IsNetworkError(err error) bool {
	var categorizedErr *CategorizedError
	return errors.As(err, &categorizedErr) && categorizedErr.Type == ErrorTypeNetwork
}

// IsValidationError checks if error is validation related
func IsValidationError(err error) bool {
	var categorizedErr *CategorizedError
	return errors.As(err, &categorizedErr) && categorizedErr.Type == ErrorTypeValidation
}

// GetErrorCode extracts error code from categorized error
func GetErrorCode(err error) string {
	var categorizedErr *CategorizedError
	if errors.As(err, &categorizedErr) {
		return categorizedErr.Code
	}
	return "unknown"
}

// GetErrorType extracts error type from categorized error
func GetErrorType(err error) ErrorType {
	var categorizedErr *CategorizedError
	if errors.As(err, &categorizedErr) {
		return categorizedErr.Type
	}
	return ErrorTypeInternal
}

// WrapError wraps an existing error with categorization
func WrapError(err error, errorType ErrorType, code, message string) *CategorizedError {
	if err == nil {
		return nil
	}
	return NewCategorizedError(errorType, code, message, err)
}
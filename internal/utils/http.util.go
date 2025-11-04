package utils

import (
	"context"
	"fmt"
	"go-log/internal/config"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPConfig holds HTTP client configuration
type HTTPConfig struct {
	MaxConnsPerHost      int
	MaxIdleConns         int
	MaxIdleConnsPerHost  int
	IdleConnTimeout      time.Duration
	ConnectTimeout       time.Duration
	RequestTimeout       time.Duration
	ResponseHeaderTimeout time.Duration
	MaxResponseSize      int64
	TLSHandshakeTimeout  time.Duration
}

var (
	httpConfig     *HTTPConfig
	httpClient     *http.Client
	httpClientOnce sync.Once
	httpClientMu   sync.RWMutex
)

// InitHTTPConfig initializes HTTP client configuration from environment
func InitHTTPConfig() {
	envConfig := config.GetEnvConfig()
	httpConfig = &HTTPConfig{
		MaxConnsPerHost:       envConfig.HTTPMaxConnsPerHost,
		MaxIdleConns:          envConfig.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:   envConfig.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:       envConfig.HTTPIdleConnTimeout,
		ConnectTimeout:        envConfig.HTTPConnectTimeout,
		RequestTimeout:        envConfig.HTTPRequestTimeout,
		ResponseHeaderTimeout: envConfig.HTTPResponseHeaderTimeout,
		MaxResponseSize:       envConfig.HTTPMaxResponseSize,
		TLSHandshakeTimeout:   envConfig.HTTPTLSHandshakeTimeout,
	}

	// Create the shared HTTP client
	httpClientOnce.Do(func() {
		httpClient = createHTTPClient(httpConfig)
	})

	LogInfo("HTTP client initialized with max_conns_per_host=%d, timeout=%v, max_response_size=%d",
		httpConfig.MaxConnsPerHost, httpConfig.RequestTimeout, httpConfig.MaxResponseSize)
}

// createHTTPClient creates a properly configured HTTP client
func createHTTPClient(config *HTTPConfig) *http.Client {
	transport := &http.Transport{
		MaxConnsPerHost:     config.MaxConnsPerHost,
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		TLSHandshakeTimeout: config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		MaxResponseHeaderBytes: 4096, // 4KB header limit
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.RequestTimeout,
	}
}

// GetHTTPClient returns the shared HTTP client
func GetHTTPClient() *http.Client {
	httpClientMu.RLock()
	defer httpClientMu.RUnlock()
	
	if httpClient == nil {
		// Initialize if not already done
		InitHTTPConfig()
	}
	
	return httpClient
}

// GetHTTPClientWithTimeout returns an HTTP client with a custom timeout
func GetHTTPClientWithTimeout(timeout time.Duration) *http.Client {
	baseClient := GetHTTPClient()
	
	// Create a client with custom timeout but shared transport
	return &http.Client{
		Transport: baseClient.Transport,
		Timeout:   timeout,
	}
}

// MakeHTTPRequest makes an HTTP request with proper resource management
func MakeHTTPRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "go-monitoring/1.0")
	req.Header.Set("Accept", "application/json")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Get HTTP client and make request
	client := GetHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// MakeHTTPRequestWithLimits makes an HTTP request with response size limits
func MakeHTTPRequestWithLimits(ctx context.Context, method, url string, body io.Reader, headers map[string]string) ([]byte, error) {
	resp, err := MakeHTTPRequest(ctx, method, url, body, headers)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Read response with size limit
	limited := io.LimitReader(resp.Body, httpConfig.MaxResponseSize)
	responseData, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response was truncated
	if int64(len(responseData)) >= httpConfig.MaxResponseSize {
		return nil, fmt.Errorf("response size exceeds limit of %d bytes", httpConfig.MaxResponseSize)
	}

	return responseData, nil
}

// GetHTTPConfig returns the current HTTP configuration
func GetHTTPConfig() *HTTPConfig {
	return httpConfig
}

// CloseHTTPClient properly closes the HTTP client and its connections
func CloseHTTPClient() {
	httpClientMu.Lock()
	defer httpClientMu.Unlock()
	
	if httpClient != nil && httpClient.Transport != nil {
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		LogInfo("HTTP client connections closed")
	}
}

// Helper functions for environment variable parsing
// ValidateHTTPConfig validates HTTP configuration values
func ValidateHTTPConfig() error {
	if httpConfig == nil {
		return fmt.Errorf("HTTP config not initialized")
	}

	if httpConfig.MaxConnsPerHost <= 0 {
		return fmt.Errorf("max connections per host must be positive")
	}

	if httpConfig.MaxIdleConns <= 0 {
		return fmt.Errorf("max idle connections must be positive")
	}

	if httpConfig.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if httpConfig.MaxResponseSize <= 0 {
		return fmt.Errorf("max response size must be positive")
	}

	return nil
}

// GetHTTPClientStats returns statistics about the HTTP client
func GetHTTPClientStats() map[string]any {
	stats := map[string]any{
		"max_conns_per_host":       httpConfig.MaxConnsPerHost,
		"max_idle_conns":           httpConfig.MaxIdleConns,
		"max_idle_conns_per_host":  httpConfig.MaxIdleConnsPerHost,
		"idle_conn_timeout":        httpConfig.IdleConnTimeout.String(),
		"connect_timeout":          httpConfig.ConnectTimeout.String(),
		"request_timeout":          httpConfig.RequestTimeout.String(),
		"response_header_timeout":  httpConfig.ResponseHeaderTimeout.String(),
		"max_response_size":        httpConfig.MaxResponseSize,
		"tls_handshake_timeout":    httpConfig.TLSHandshakeTimeout.String(),
	}

	if httpClient != nil && httpClient.Transport != nil {
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			stats["transport_type"] = "http.Transport"
			stats["force_attempt_http2"] = transport.ForceAttemptHTTP2
			stats["disable_keep_alives"] = transport.DisableKeepAlives
		}
	}

	return stats
}
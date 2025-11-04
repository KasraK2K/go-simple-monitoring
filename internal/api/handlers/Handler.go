package handlers

import (
	"go-log/internal/utils"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

func getAESSecret() string {
	return getRequiredEnv("AES_SECRET")
}

func getJWTSecret() string {
	return getRequiredEnv("JWT_SECRET")
}

func setHeader(w http.ResponseWriter, status int, responseData string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(responseData))
}

func getCORSOrigins() string {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		// Default to localhost for development
		if IsProduction() {
			log.Fatal("CORS_ALLOWED_ORIGINS must be set in production environment")
		}
		return "http://localhost:3500,http://127.0.0.1:3500"
	}
	return origins
}

func isOriginAllowed(origin string, allowedOrigins string) bool {
	if allowedOrigins == "*" {
		return true
	}

	origins := strings.SplitSeq(allowedOrigins, ",")
	for allowed := range origins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := getCORSOrigins()

		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if allowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func getTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

func ValidateTokenAndParseGeneric[T any](r *http.Request) (*T, error) {
	encryptedToken := getTokenFromHeader(r)
	if encryptedToken == "" {
		return nil, utils.ErrMissingToken
	}

	claims, err := utils.DecryptAndParseToken[T](encryptedToken, getAESSecret(), getJWTSecret())
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func IsProduction() bool {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = os.Getenv("APP_ENV")
	}

	return env == "production" || env == "prod"
}

func ShouldCheckTokenInProduction() bool {
	checkToken := os.Getenv("CHECK_TOKEN")
	if checkToken == "" {
		return false // Default: false
	}
	return checkToken == "true" || checkToken == "1"
}

// Rate limiting structures
type clientEntry struct {
	tokens     float64
	lastRefill time.Time
	mutex      sync.Mutex
}

var (
	rateLimitClients = make(map[string]*clientEntry)
	clientMutex      sync.RWMutex
)

// getRateLimitConfig returns rate limiting configuration from environment
func getRateLimitConfig() (requestsPerSecond float64, burstSize int) {
	rpsStr := os.Getenv("RATE_LIMIT_RPS")
	if rpsStr == "" {
		rpsStr = "10" // Default: 10 requests per second
	}
	
	burstStr := os.Getenv("RATE_LIMIT_BURST")
	if burstStr == "" {
		burstStr = "20" // Default: 20 request burst
	}
	
	rps, err := strconv.ParseFloat(rpsStr, 64)
	if err != nil || rps <= 0 {
		rps = 10
	}
	
	burst, err := strconv.Atoi(burstStr)
	if err != nil || burst <= 0 {
		burst = 20
	}
	
	return rps, burst
}

// getClientKey extracts client identifier for rate limiting
func getClientKey(r *http.Request) string {
	// Try X-Forwarded-For first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// isRateLimitEnabled checks if rate limiting is enabled
func isRateLimitEnabled() bool {
	enabled := os.Getenv("RATE_LIMIT_ENABLED")
	if enabled == "" {
		return true // Default: enabled
	}
	return enabled != "false" && enabled != "0"
}

// TokenBucket implements rate limiting using token bucket algorithm
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if rate limiting is enabled
		if !isRateLimitEnabled() {
			next(w, r)
			return
		}
		
		clientKey := getClientKey(r)
		rps, burst := getRateLimitConfig()
		
		clientMutex.RLock()
		client, exists := rateLimitClients[clientKey]
		clientMutex.RUnlock()
		
		if !exists {
			client = &clientEntry{
				tokens:     float64(burst),
				lastRefill: utils.NowUTC(),
			}
			clientMutex.Lock()
			rateLimitClients[clientKey] = client
			clientMutex.Unlock()
		}
		
		client.mutex.Lock()
		defer client.mutex.Unlock()
		
		now := utils.NowUTC()
		elapsed := now.Sub(client.lastRefill).Seconds()
		
		// Refill tokens based on elapsed time
		client.tokens += elapsed * rps
		if client.tokens > float64(burst) {
			client.tokens = float64(burst)
		}
		client.lastRefill = now
		
		// Check if we have tokens available
		if client.tokens < 1 {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(burst))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Second).Unix(), 10))
			setHeader(w, http.StatusTooManyRequests, `{"status":false, "error": "Rate limit exceeded"}`)
			return
		}
		
		// Consume a token
		client.tokens--
		
		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(burst))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(client.tokens)))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Second).Unix(), 10))
		
		next(w, r)
	}
}

func MethodMiddleware(allowedMethods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			methodAllowed := slices.Contains(allowedMethods, r.Method)

			if !methodAllowed {
				setHeader(w, http.StatusMethodNotAllowed, `{"status":false, "error": "Method not allowed"}`)
				return
			}

			next(w, r)
		}
	}
}

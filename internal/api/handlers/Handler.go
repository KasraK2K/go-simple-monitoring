package handlers

import (
	"go-log/internal/utils"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
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

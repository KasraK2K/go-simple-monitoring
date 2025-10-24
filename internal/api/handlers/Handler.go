package handlers

import (
	"go-log/internal/utils"
	"net/http"
	"os"
	"strings"
)

var aesSecret string = "jVuTFhObFk0SmxkQzFyWlhrNmlNalZ1VEZoT2JGazBTbXhrUXpHeVdsaHJObVa3JObVY0c0luUlNqRmpNbF"
var jwtSecret string = "QiOjObFkrNmV4FhObFk0SmxkQ0N3UDMTmlNalZ1V"

func setHeader(w http.ResponseWriter, status int, responseData string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(responseData))
}

func checkMethod(r *http.Request, w http.ResponseWriter, method string) bool {
	if r.Method != method {
		setHeader(w, http.StatusMethodNotAllowed, `{"status":false, "error": "Method not allowed"}`)
		return false
	}
	return true
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

func ValidateTokenAndGetBusinessID(r *http.Request) (int, error) {
	encryptedToken := getTokenFromHeader(r)
	if encryptedToken == "" {
		return 0, utils.ErrMissingToken
	}

	businessID, err := utils.DecryptAndExtractBusinessID(encryptedToken, aesSecret, jwtSecret)
	if err != nil {
		return 0, err
	}

	return businessID, nil
}

func ValidateTokenAndParseGeneric[T any](r *http.Request) (*T, error) {
	encryptedToken := getTokenFromHeader(r)
	if encryptedToken == "" {
		return nil, utils.ErrMissingToken
	}

	claims, err := utils.DecryptAndParseToken[T](encryptedToken, aesSecret, jwtSecret)
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

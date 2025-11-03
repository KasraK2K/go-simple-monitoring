package utils

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

func ParseJWT(tokenStr, jwtSecret string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// Ensure it's HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, NewAuthError("INVALID_SIGNING_METHOD", "unexpected JWT signing method", errors.New("expected HMAC signing method"))
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, NewAuthError("TOKEN_PARSE_FAILED", "failed to parse JWT token", err)
	}

	if !token.Valid {
		return nil, NewAuthError("INVALID_TOKEN", "JWT token is invalid", ErrInvalidToken)
	}

	return token, nil
}

func ParseBusinessIDFromJWT(tokenStr, jwtSecret string) (int, error) {
	token, err := ParseJWT(tokenStr, jwtSecret)
	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if bid, ok := claims["business_id"].(float64); ok {
			return int(bid), nil
		}
		return 0, NewDataError("BUSINESS_ID_NOT_FOUND", "business_id claim not found in token", ErrInvalidClaims)
	}

	return 0, NewDataError("INVALID_CLAIMS", "failed to extract claims from JWT token", ErrInvalidClaims)
}

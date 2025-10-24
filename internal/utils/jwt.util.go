package utils

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

func ParseJWT(tokenStr, jwtSecret string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// Ensure it's HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
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
		return 0, errors.New("business_id not found")
	}

	return 0, errors.New("invalid token claims")
}

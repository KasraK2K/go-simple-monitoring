package utils

import (
	"encoding/json"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var ErrMissingToken = errors.New("missing or invalid token")

func DecryptAndExtractBusinessID(encryptedToken, aesSecret, jwtSecret string) (int, error) {
	// 1. AES decrypt
	jwtToken, err := DecryptCryptoJSAES(encryptedToken, aesSecret)
	if err != nil {
		return 0, err
	}

	// 2. Parse JWT and get business_id
	return ParseBusinessIDFromJWT(jwtToken, jwtSecret)
}

func DecryptAndParseToken[T any](encryptedToken, aesSecret, jwtSecret string) (*T, error) {
	// 1. AES decrypt
	jwtToken, err := DecryptCryptoJSAES(encryptedToken, aesSecret)
	if err != nil {
		return nil, err
	}

	// 2. Parse JWT
	token, err := ParseJWT(jwtToken, jwtSecret)
	if err != nil {
		return nil, err
	}

	// 3. Extract claims and marshal to JSON
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, NewDataError("INVALID_CLAIMS", "failed to extract claims from token", ErrInvalidClaims)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, NewDataError("MARSHAL_FAILED", "failed to marshal token claims", err)
	}

	// 4. Unmarshal to target struct
	var result T
	err = json.Unmarshal(claimsJSON, &result)
	if err != nil {
		return nil, NewDataError("UNMARSHAL_FAILED", "failed to unmarshal claims to target type", err)
	}

	return &result, nil
}

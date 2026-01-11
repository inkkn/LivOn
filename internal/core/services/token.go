package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secretKey []byte
	issuer    string
}

func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secretKey: []byte(secret),
		issuer:    "livon-backend",
	}
}

func (s *TokenService) GenerateToken(phone string) (string, error) {
	claims := jwt.MapClaims{
		"sub": phone,                                 // Subject
		"iat": time.Now().Unix(),                     // Issued At
		"exp": time.Now().Add(24 * time.Hour).Unix(), // Expiration
		"iss": s.issuer,                              // Issuer
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken parses and validates the JWT string
func (s *TokenService) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Ensure signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}
	// Extract the phone number from 'sub'
	phone, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("subject not found in token")
	}
	return phone, nil
}

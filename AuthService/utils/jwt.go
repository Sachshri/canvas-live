package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID   string `json:"user_id`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte("my_super_secret_key")

func CreateToken(userID string, email string, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	// create custom claims object
	claims := &CustomClaims{
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtSecret)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// KeyFunc provides the secret key to the library for verification
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	// Check for parsing errors
	if err != nil {
		return nil, err
	}

	// Check if the token is valid (signature and standard claims are checked here)
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// If valid, return the claims
	return claims, nil
}

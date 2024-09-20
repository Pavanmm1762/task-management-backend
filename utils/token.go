// utils/auth.go

package utils

import (
	"errors"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gocql/gocql"
)

var jwtSecret = []byte("secure_secret_key") // Change this to a secure secret key

// Claims represents the claims that will be encoded into the JWT token
type Claims struct {
	UserID gocql.UUID `json:"user_id"`
	jwt.StandardClaims
}

// GenerateToken generates a JWT token for the given user ID
func GenerateToken(userID gocql.UUID) (string, error) {
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 12).Unix(), // Token expires in 24 hours
			IssuedAt:  time.Now().Unix(),
			Issuer:    "task_management",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func getUserId(tokenString string) (gocql.UUID, error) {

	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return jwtSecret, nil
	})

	// Check for errors
	if err != nil {
		return gocql.UUID{}, err
	}

	// Check if the token is valid
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return gocql.UUID{}, errors.New("Invalid token")
}

// middleware/authentication.go

package middleware

import (
	"errors"
	"fmt"
	"strings"

	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
)

var jwtSecret = []byte("secure_secret_key") // Change this to your secret key

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Token not provided"})
			return
		}

		// Verify the token using your verification logic
		validToken, err := VerifyToken(tokenString)
		if err != nil || !validToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Invalid token"})
			return
		}

		c.Next()
	}
}

// VerifyToken is a function to verify the JWT token
func VerifyToken(tokenString string) (bool, error) {

	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

	if jwtSecret == nil {
		return false, errors.New("jwtSecret is nil")
	}

	token, err := jwt.ParseWithClaims(tokenString, &utils.Claims{}, func(token *jwt.Token) (interface{}, error) {

		return jwtSecret, nil
	})

	if err != nil {
		fmt.Println(err)
		return false, err
	}

	// Check if the token is valid
	return token.Valid, nil
}

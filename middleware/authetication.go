// middleware/authentication.go

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go/task_management/backend/utils"
)

func AuthMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Token not provided"})
			return
		}

		// Verify the token using your verification logic
		validToken, claims, err := utils.VerifyToken(tokenString)
		if err != nil || !validToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Invalid token"})
			return
		}

		// Check if the role in the token matches the required role
		if claims["role"] != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden - Insufficient permissions"})
			return
		}

		// Set user info in context (optional)
		c.Set("userID", claims["user_id"])
		c.Set("role", claims["role"])

		c.Next()
	}
}

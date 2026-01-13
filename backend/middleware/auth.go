package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"alert-manager-backend/utils"
)

// AuthMiddleware parses JWT from Authorization header and sets user info in context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization"})
			return
		}

		token := auth
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
		}

		claims, err := utils.ParseToken(token)
		if err != nil || claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userID", int(claims.UserID))
		c.Set("username", claims.Username)
		c.Next()
	}
}

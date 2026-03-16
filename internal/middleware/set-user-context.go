package middleware

import (
	"prime-erp-core/internal/models"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// allow public routes
		switch c.Request.URL.Path {
		case "/authen/login", "/authen/create-user", "/author/get-requester":
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")

		if authHeader != "" {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &models.AuthenJWTClaims{})
			if err == nil {
				if claims, ok := token.Claims.(*models.AuthenJWTClaims); ok {
					c.Set("user", claims.User)
				}
			}
		}

		c.Next()
	}
}

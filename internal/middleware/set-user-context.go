package middleware

import (
	"prime-erp-core/internal/models"
	"prime-erp-core/internal/requestcontext"
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
					// เก็บใน gin ไว้ให้ service เดิมที่ยังรับ *gin.Context อ่านได้
					c.Set("user", claims.User)

					ctx := c.Request.Context()

					// เก็บ user ให้ service ที่รับ context.Context
					ctx = requestcontext.WithUser(ctx, claims.User)

					// เก็บ token เดิมไว้ส่งต่อไป service อื่น
					ctx = requestcontext.WithToken(ctx, authHeader)

					c.Request = c.Request.WithContext(ctx)
				}
			}
		}

		c.Next()
	}
}

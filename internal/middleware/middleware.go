package middleware

import "github.com/gin-gonic/gin"

func RegisterMiddlewares(ctx *gin.Engine) {
	ctx.Use(CORSMiddleware())
	ctx.Use(AuthMiddleware())
}

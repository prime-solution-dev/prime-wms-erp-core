package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterHealthz adds GET /healthz (liveness: the process is up and serving).
// Call it BEFORE middleware.RegisterMiddlewares: Gin binds middleware to a route when the
// route is registered, so registering first keeps /healthz outside auth and request logging.
func RegisterHealthz(engine *gin.Engine) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

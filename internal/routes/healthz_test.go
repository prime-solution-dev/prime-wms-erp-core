package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthzBypassesLaterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterHealthz(engine)
	// stands in for AuthMiddleware: anything registered after RegisterHealthz must not run for /healthz
	engine.Use(func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) })
	engine.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK || w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("/healthz: got %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("/protected should still be guarded, got %d", w.Code)
	}
}

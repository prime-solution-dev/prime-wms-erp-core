package routes

import (
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
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

func TestHealthzVersion(t *testing.T) {
	got := healthzVersion(&debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "5caba398"},
		{Key: "vcs.time", Value: "2026-11-16T04:45:21Z"},
		{Key: "vcs.modified", Value: "true"},
		{Key: "GOOS", Value: "linux"},
	}})
	if got["sha"] != "5caba398" || got["time"] != "2026-11-16T04:45:21Z" || got["modified"] != true || got["status"] != "ok" {
		t.Fatalf("unexpected version: %v", got)
	}
	if got := healthzVersion(nil); got["sha"] != "" || got["modified"] != false {
		t.Fatalf("nil build info should give empty sha: %v", got)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterHealthz(engine)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz?action=version", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"sha":`) {
		t.Fatalf("/healthz?action=version: got %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK || w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("/healthz without action: got %d %s", w.Code, w.Body.String())
	}
}

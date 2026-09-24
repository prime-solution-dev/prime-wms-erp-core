package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prime-erp-core/internal/apperr"
	"prime-erp-core/internal/middleware"
	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	return router
}

// JWT ที่ claim user = somchai (ไม่ได้เซ็น เพราะ middleware ใช้ ParseUnverified)
// header {"alg":"none","typ":"JWT"} . payload {"user":"somchai","company_code":"CM"} .
const tokenOfSomchai = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." +
	"eyJ1c2VyIjoic29tY2hhaSIsImNvbXBhbnlfY29kZSI6IkNNIn0."

// เส้นเต็ม: middleware แกะ JWT แล้ว service ที่รับ context.Context ต้องเห็นทั้ง user และ token
func TestProcessContextRequestCarriesUserAndTokenToService(t *testing.T) {
	gotUser := ""
	gotToken := ""

	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			gotUser = requestcontext.GetUserOrDefault(ctx)
			gotToken, _ = requestcontext.GetToken(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenOfSomchai)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotUser != "somchai" {
		t.Fatalf("user ที่ service เห็น = %q, ต้องการ somchai", gotUser)
	}

	if gotToken != "Bearer "+tokenOfSomchai {
		t.Fatalf("token ที่ service เห็น = %q", gotToken)
	}
}

// ไม่มี token เลย ต้องไม่พัง แต่ได้ user เป็นค่าว่าง
func TestProcessContextRequestFallsBackToEmptyUser(t *testing.T) {
	gotUser := ""

	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			gotUser = requestcontext.GetUserOrDefault(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))

	if gotUser != "" {
		t.Fatalf("user = %q, ต้องการค่าว่าง", gotUser)
	}
}

// middleware ตัวเก่าที่เก็บ user ด้วย c.Set อย่างเดียว ต้องยังใช้งานได้
func TestProcessContextRequestBridgesLegacyGinUser(t *testing.T) {
	gotUser := ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", "legacy-user")
		c.Next()
	})
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			gotUser = requestcontext.GetUserOrDefault(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))

	if gotUser != "legacy-user" {
		t.Fatalf("user = %q, ต้องการ legacy-user", gotUser)
	}
}

// error ที่เป็น apperr ต้องได้ status ตามที่กำหนด ไม่ใช่ 500 ทั้งหมด
func TestProcessContextRequestMapsAppErrorStatus(t *testing.T) {
	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			return nil, apperr.BadRequest("delivery_codes is required")
		})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, ต้องการ 400", rec.Code)
	}

	body := map[string]string{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("อ่าน body ไม่ได้: %v", err)
	}

	if body["error"] != "delivery_codes is required" || body["code"] != "BAD_REQUEST" {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

// error ธรรมดายังเป็น 500 รูปแบบเดิม
func TestProcessContextRequestPlainErrorStays500(t *testing.T) {
	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			return nil, context.DeadlineExceeded
		})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, ต้องการ 500", rec.Code)
	}
}

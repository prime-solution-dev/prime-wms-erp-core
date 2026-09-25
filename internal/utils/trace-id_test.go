package utils_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prime-erp-core/internal/requestcontext"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
)

// buildContext ต้องไม่ทับ trace id ที่ถูกใส่ไว้ใน context มาก่อนแล้ว (RequestLogMiddleware
// เป็นคนใส่ให้จริงตอน production) ไม่งั้นทุก service ในเส้นเดียวกันจะเห็นคนละเลข
//
// เทสสองตัวที่เคยยืนอยู่ตรงนี้ (เดิมชื่อ TestProcessContextRequestKeepsIncomingTraceID กับ
// TestProcessContextRequestGeneratesTraceIDWhenMissing) ย้ายไปอยู่
// internal/middleware/request-log-middleware_test.go แล้ว เพราะพฤติกรรม "อ่าน X-Trace-ID เดิม /
// ออกเลขใหม่เมื่อไม่มี" ย้ายไปทำที่ RequestLogMiddleware ก่อน buildContext แล้ว เพื่อให้ทุก route
// มี trace id แม้ AuthMiddleware จะปฏิเสธ request ก็ตาม ที่เหลือให้ buildContext ทำหน้าที่แค่
// ตาข่ายรองเหมือนที่ทำกับ user/token อยู่แล้ว: ถ้า context มี trace id อยู่แล้วห้ามทับ
func TestProcessContextRequestPreservesTraceIDAlreadyInContext(t *testing.T) {
	gotTraceID := ""

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(requestcontext.WithTraceID(c.Request.Context(), "trace-จาก-middleware-ก่อนหน้า"))
		c.Next()
	})
	router.POST("/x", func(c *gin.Context) {
		utils.ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			gotTraceID, _ = requestcontext.GetTraceID(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	// ไม่ส่ง X-Trace-ID header มาเลย แต่ context มี trace id อยู่แล้วจาก middleware ก่อนหน้า
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotTraceID != "trace-จาก-middleware-ก่อนหน้า" {
		t.Fatalf("trace id = %q, buildContext ต้องไม่ทับเลขที่มีอยู่แล้วใน context", gotTraceID)
	}
}

// ตอนยิงต่อไป service อื่น trace id ต้องติดไปกับ header ไม่งั้นปลายทางจะออกเลขใหม่
// แล้วเส้นทางจะขาดตรงนั้น
func TestNewRequestForwardsTraceID(t *testing.T) {
	gotHeader := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(utils.TraceIDHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	ctx := requestcontext.WithTraceID(context.Background(), "trace-ของเส้นนี้")

	req, err := utils.NewRequest(ctx, http.MethodPost, downstream.URL, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotHeader != "trace-ของเส้นนี้" {
		t.Fatalf("ปลายทางได้ %s = %q", utils.TraceIDHeader, gotHeader)
	}
}

// ไม่มี trace id ใน context ก็ไม่ต้องแปะ header เปล่าๆ ให้ปลายทางออกเลขเอง
func TestNewRequestWithoutTraceIDSendsNoHeader(t *testing.T) {
	req, err := utils.NewRequest(context.Background(), http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get(utils.TraceIDHeader); got != "" {
		t.Fatalf("%s = %q, ต้องว่าง", utils.TraceIDHeader, got)
	}
}

// เส้นที่รับไฟล์ก็ต้องได้ trace id เหมือนกัน ไม่ใช่มีเฉพาะเส้น JSON
func TestProcessContextRequestMultipartCarriesTraceID(t *testing.T) {
	gotTraceID := ""

	router := newTestRouter()
	router.POST("/upload", func(c *gin.Context) {
		utils.ProcessContextRequestMultipart(c, func(ctx context.Context, input utils.MultipartInput) (interface{}, error) {
			gotTraceID, _ = requestcontext.GetTraceID(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "pricelist.xlsx")
	part.Write([]byte("dummy"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(utils.TraceIDHeader, "trace-อัปโหลด")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotTraceID != "trace-อัปโหลด" {
		t.Fatalf("trace id = %q", gotTraceID)
	}
}

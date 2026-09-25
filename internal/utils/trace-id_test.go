package utils

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prime-erp-core/internal/requestcontext"

	"github.com/gin-gonic/gin"
)

// service ต้นทางส่ง trace id มา ต้องใช้เลขเดิม ไม่ออกเลขใหม่
// ไม่งั้นไล่ดูเส้นทางของ request เดียวกันข้าม service ไม่ได้
func TestProcessContextRequestKeepsIncomingTraceID(t *testing.T) {
	gotTraceID := ""

	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			gotTraceID, _ = requestcontext.GetTraceID(ctx)

			return gin.H{"ok": true}, nil
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`))
	req.Header.Set(TraceIDHeader, "trace-จากต้นทาง")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotTraceID != "trace-จากต้นทาง" {
		t.Fatalf("trace id = %q, ต้องเป็นเลขเดิมที่ต้นทางส่งมา", gotTraceID)
	}
}

// ไม่มี trace id มาด้วย แปลว่าเราเป็นต้นทาง ต้องออกเลขใหม่ให้ ไม่ปล่อยว่าง
func TestProcessContextRequestGeneratesTraceIDWhenMissing(t *testing.T) {
	first := ""
	second := ""

	router := newTestRouter()
	router.POST("/x", func(c *gin.Context) {
		ProcessContextRequest(c, func(ctx context.Context, payload string) (interface{}, error) {
			traceID, _ := requestcontext.GetTraceID(ctx)
			if first == "" {
				first = traceID
			} else {
				second = traceID
			}

			return gin.H{"ok": true}, nil
		})
	})

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{}`)))
	}

	if first == "" || second == "" {
		t.Fatalf("trace id ว่าง: first=%q second=%q", first, second)
	}

	if first == second {
		t.Fatalf("สอง request คนละครั้งต้องได้คนละ trace id แต่ได้ %q ทั้งคู่", first)
	}
}

// ตอนยิงต่อไป service อื่น trace id ต้องติดไปกับ header ไม่งั้นปลายทางจะออกเลขใหม่
// แล้วเส้นทางจะขาดตรงนั้น
func TestNewRequestForwardsTraceID(t *testing.T) {
	gotHeader := ""

	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(TraceIDHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	ctx := requestcontext.WithTraceID(context.Background(), "trace-ของเส้นนี้")

	req, err := NewRequest(ctx, http.MethodPost, downstream.URL, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotHeader != "trace-ของเส้นนี้" {
		t.Fatalf("ปลายทางได้ %s = %q", TraceIDHeader, gotHeader)
	}
}

// ไม่มี trace id ใน context ก็ไม่ต้องแปะ header เปล่าๆ ให้ปลายทางออกเลขเอง
func TestNewRequestWithoutTraceIDSendsNoHeader(t *testing.T) {
	req, err := NewRequest(context.Background(), http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get(TraceIDHeader); got != "" {
		t.Fatalf("%s = %q, ต้องว่าง", TraceIDHeader, got)
	}
}

// เส้นที่รับไฟล์ก็ต้องได้ trace id เหมือนกัน ไม่ใช่มีเฉพาะเส้น JSON
func TestProcessContextRequestMultipartCarriesTraceID(t *testing.T) {
	gotTraceID := ""

	router := newTestRouter()
	router.POST("/upload", func(c *gin.Context) {
		ProcessContextRequestMultipart(c, func(ctx context.Context, input MultipartInput) (interface{}, error) {
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
	req.Header.Set(TraceIDHeader, "trace-อัปโหลด")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotTraceID != "trace-อัปโหลด" {
		t.Fatalf("trace id = %q", gotTraceID)
	}
}

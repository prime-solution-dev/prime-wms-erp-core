package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"prime-erp-core/internal/requestcontext"
	"prime-erp-core/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/prime-solution-dev/prime-service-x/servicelog"
)

// fakeServiceLogSink จับ Create/Update ที่ RequestLogMiddleware ยิงเข้ามา ไว้เทสได้โดยไม่ต้อง
// พึ่ง MongoDB จริง — Init คืน nil เสมอ (ไม่ต่อ network) ปลอดภัยกับการรันเทสหลายตัวในไฟล์นี้
// เพราะ utils.SetServiceLogSinkForTest ไม่ thread-safe จึงห้ามใช้ t.Parallel() กับเทสที่สลับ sink
type fakeServiceLogSink struct {
	mu      sync.Mutex
	created []servicelog.ServiceLog
	updated []capturedUpdate
}

type capturedUpdate struct {
	requestID  string
	response   interface{}
	httpStatus int
	status     string
	errMessage string
}

func (f *fakeServiceLogSink) Init(cfg servicelog.Config) error {
	return nil
}

func (f *fakeServiceLogSink) Create(ctx context.Context, entry servicelog.ServiceLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, entry)
	return nil
}

func (f *fakeServiceLogSink) Update(ctx context.Context, requestID string, response interface{}, httpStatus int, status string, errMessage string, durationMs int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updated = append(f.updated, capturedUpdate{
		requestID:  requestID,
		response:   response,
		httpStatus: httpStatus,
		status:     status,
		errMessage: errMessage,
	})
	return nil
}

// setFakeServiceLogSink เปิด logging ผ่าน sink ปลอม (ไม่ต่อ MongoDB จริง) restore ให้เองผ่าน
// t.Cleanup — repo นี้ใช้ servicelog ตัวเดียว ไม่มี outbound logger ให้ปลอมอีกตัวแล้ว
func setFakeServiceLogSink(t *testing.T) *fakeServiceLogSink {
	t.Helper()

	fake := &fakeServiceLogSink{}
	restore := utils.SetServiceLogSinkForTest(fake)
	t.Cleanup(restore)

	t.Setenv("API_LOG_ENABLED", "true")
	t.Setenv("API_LOG_SERVICE", "erp-core-test")
	t.Setenv("API_LOG_MONGODB_URI", "mongodb://unused-in-test")
	t.Setenv("API_LOG_DATABASE", "api_logs_test")

	return fake
}

func newLogTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// middleware ต้องออก request id ใหม่ทุก request แล้ว echo กลับเป็น response header X-Request-ID
func TestRequestLogMiddlewareEchoesRequestIDHeader(t *testing.T) {
	t.Setenv("API_LOG_ENABLED", "false")

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("X-Request-ID ว่าง, ต้องมีค่า")
	}
}

// ต้นทางส่ง X-Trace-ID มา → context ต้องเห็นเลขเดิม ไม่ออกเลขใหม่
// (ย้ายมาจาก internal/utils/trace-id_test.go — เดิมชื่อ TestProcessContextRequestKeepsIncomingTraceID
// ตอนนี้ trace id ถูกกำหนดที่ RequestLogMiddleware แทน utils.buildContext แล้ว)
func TestRequestLogMiddlewareReusesIncomingTraceID(t *testing.T) {
	t.Setenv("API_LOG_ENABLED", "false")

	gotTraceID := ""

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.GET("/x", func(c *gin.Context) {
		gotTraceID, _ = requestcontext.GetTraceID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(utils.TraceIDHeader, "trace-จากต้นทาง")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if gotTraceID != "trace-จากต้นทาง" {
		t.Fatalf("trace id = %q, ต้องเป็นเลขเดิมที่ต้นทางส่งมา", gotTraceID)
	}
}

// ไม่มี X-Trace-ID มาด้วย แปลว่าเราเป็นต้นทาง ต้องออกเลขใหม่ให้ทุกครั้ง ไม่ปล่อยว่าง และสอง
// request คนละครั้งต้องได้คนละเลข
// (ย้ายมาจาก internal/utils/trace-id_test.go — เดิมชื่อ TestProcessContextRequestGeneratesTraceIDWhenMissing)
func TestRequestLogMiddlewareGeneratesTraceIDWhenMissing(t *testing.T) {
	t.Setenv("API_LOG_ENABLED", "false")

	var traceIDs []string

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.GET("/x", func(c *gin.Context) {
		traceID, _ := requestcontext.GetTraceID(c.Request.Context())
		traceIDs = append(traceIDs, traceID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	}

	if len(traceIDs) != 2 || traceIDs[0] == "" || traceIDs[1] == "" {
		t.Fatalf("trace id ว่าง: %v", traceIDs)
	}

	if traceIDs[0] == traceIDs[1] {
		t.Fatalf("สอง request คนละครั้งต้องได้คนละ trace id แต่ได้ %q ทั้งคู่", traceIDs[0])
	}
}

// path ที่ตรงกับ API_LOG_EXCLUDE ต้องไม่ถูกบันทึก ส่วน path อื่นต้องถูกบันทึกตามปกติ
func TestRequestLogMiddlewareExcludesConfiguredPathPrefix(t *testing.T) {
	fake := setFakeServiceLogSink(t)
	t.Setenv("API_LOG_EXCLUDE", "/cronjob/,/health")

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.POST("/cronjob/credit-request", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.POST("/orders/create", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, httptest.NewRequest(http.MethodPost, "/cronjob/credit-request", strings.NewReader(`{}`)))
	if rec1.Code != http.StatusOK {
		t.Fatalf("cronjob status = %d, body = %s", rec1.Code, rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/orders/create", strings.NewReader(`{}`)))
	if rec2.Code != http.StatusOK {
		t.Fatalf("orders status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()

	if len(fake.created) != 1 {
		t.Fatalf("created logs = %d, ต้องการ 1 รายการ (เฉพาะ /orders/create) ได้: %+v", len(fake.created), fake.created)
	}

	if fake.created[0].Endpoint != "/orders/create" {
		t.Fatalf("endpoint ที่ถูก log = %q, ต้องการ /orders/create", fake.created[0].Endpoint)
	}
}

// key ที่ดูเป็นข้อมูล sensitive (password, token, secret, authorization, ...) ในตัว body ที่ถูก
// log ต้องถูก redact — และ handler ด้านหลังยังต้องอ่าน body เดิม (ไม่ถูก redact) ได้ตามปกติ
func TestRequestLogMiddlewareRedactsSensitiveKeysInLoggedBody(t *testing.T) {
	fake := setFakeServiceLogSink(t)

	var gotBodyInHandler string

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.POST("/login", func(c *gin.Context) {
		buf := make([]byte, 4096)
		n, _ := c.Request.Body.Read(buf)
		gotBodyInHandler = string(buf[:n])
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	body := `{"username":"somchai","password":"hunter2","meta":{"api_key":"sk-live-123"}}`

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(gotBodyInHandler, "hunter2") {
		t.Fatalf("handler ต้องยังอ่าน body เดิม (ไม่ redact) ได้: %q", gotBodyInHandler)
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()

	if len(fake.created) != 1 {
		t.Fatalf("created logs = %d, ต้องการ 1 รายการ", len(fake.created))
	}

	logged, ok := fake.created[0].Request.(map[string]interface{})
	if !ok {
		t.Fatalf("request ที่ log ต้องเป็น object ได้ %T", fake.created[0].Request)
	}

	if logged["password"] != "[REDACTED]" {
		t.Fatalf("password ต้องถูก redact ได้ %v", logged["password"])
	}

	if logged["username"] != "somchai" {
		t.Fatalf("key ที่ไม่ sensitive ต้องไม่ถูกแตะ ได้ %v", logged["username"])
	}

	meta, ok := logged["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta ต้องเป็น object ได้ %T", logged["meta"])
	}

	if meta["api_key"] != "[REDACTED]" {
		t.Fatalf("api_key ที่ซ้อนอยู่ข้างในก็ต้องถูก redact ได้ %v", meta["api_key"])
	}
}

// ปิด logging ด้วย API_LOG_ENABLED=false → request ต้องยังทำงานได้ตามปกติ ไม่มี log ถูกสร้าง
func TestRequestLogMiddlewareDisabledDoesNotBreakRequest(t *testing.T) {
	fake := &fakeServiceLogSink{}
	restore := utils.SetServiceLogSinkForTest(fake)
	t.Cleanup(restore)

	t.Setenv("API_LOG_ENABLED", "false")

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()

	if len(fake.created) != 0 || len(fake.updated) != 0 {
		t.Fatalf("ปิด logging แล้วต้องไม่มี log ถูกสร้าง ได้ created=%d updated=%d", len(fake.created), len(fake.updated))
	}
}

// servicelog.Init ล้มเหลว (เปิด API_LOG_ENABLED=true แต่ไม่ได้ตั้ง API_LOG_MONGODB_URI) →
// middleware ต้องปิด logging เองเงียบๆ แล้ว request ยังทำงานได้ตามปกติ ไม่ต่อ MongoDB จริง เพราะ
// servicelog.Init ตรวจ MongoDBURI ว่างก่อนจะพยายามต่อ network (ServiceName ตรงนี้ไม่ว่าง เพราะ
// RequestLogMiddleware ใส่ค่า default ให้เมื่อไม่ได้ตั้ง API_LOG_SERVICE)
func TestRequestLogMiddlewareInitFailureDoesNotBreakRequest(t *testing.T) {
	t.Setenv("API_LOG_ENABLED", "true")
	t.Setenv("API_LOG_SERVICE", "")
	t.Setenv("API_LOG_MONGODB_URI", "")

	router := newLogTestRouter()
	router.Use(RequestLogMiddleware())
	router.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

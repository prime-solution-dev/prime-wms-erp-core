package utils_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prime-erp-core/internal/requestcontext"
	"prime-erp-core/internal/utils"

	"github.com/prime-solution-dev/prime-service-x/apilog"
)

// ปลายทางรู้ว่าใครเรียกได้จาก header เท่านั้น เทสนี้จึงยิงของจริงไปหา httptest server
// แล้วอ่าน header ที่ปลายทางได้รับ
func TestNewRequestForwardsTokenFromContext(t *testing.T) {
	gotAuth := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := requestcontext.WithToken(context.Background(), "Bearer jwt-ของสมชาย")

	req, err := utils.NewRequest(ctx, http.MethodPost, server.URL, strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotAuth != "Bearer jwt-ของสมชาย" {
		t.Fatalf("ปลายทางได้ Authorization = %q, ต้องการ token ของคนที่ยิงเข้ามา", gotAuth)
	}
}

// ไม่มี token ใน context และไม่ได้ตั้ง SERVICE_TOKEN ต้องไม่แปะ header มั่ว
func TestNewRequestWithoutTokenSendsNoAuthHeader(t *testing.T) {
	t.Setenv("SERVICE_TOKEN", "")

	req, err := utils.NewRequest(context.Background(), http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, ต้องว่าง", got)
	}

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
}

// ไม่มี token ใน context ต้องไม่ไปหยิบ SERVICE_TOKEN มาแปะเอง
// เส้นที่ต้องยิงในนามระบบต้องใส่ token ลง context เองตั้งแต่จุดเริ่มงาน (ดู cronContext)
func TestNewRequestIgnoresServiceTokenEnv(t *testing.T) {
	t.Setenv("SERVICE_TOKEN", "service-token-abc")

	req, err := utils.NewRequest(context.Background(), http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization = %q, ต้องว่าง", got)
	}
}

// token ที่อยู่ใน context ถูกส่งต่อไปตรงๆ ไม่ถูกดัดแปลง
func TestNewRequestUsesTokenFromContextAsIs(t *testing.T) {
	ctx := requestcontext.WithToken(context.Background(), "Bearer user-token")

	req, err := utils.NewRequest(ctx, http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer user-token" {
		t.Fatalf("Authorization = %q", got)
	}
}

// trace id ที่มีอยู่ใน context ต้องกลายเป็น transaction id ที่ apilog เห็น (จุดเดียวที่เชื่อมสอง
// ระบบนี้เข้าด้วยกันคือ utils.NewRequest — ดูคอมเมนต์ในนั้น) ไม่งั้นจะมีสอง id วิ่งคนละเส้น ไล่ log
// ข้าม service ไม่ได้ (requestcontext ของเราเอง ส่ง X-Trace-ID กับ apilog.WithTransactionID ที่ส่ง
// X-Transaction-ID)
func TestNewRequestMakesTraceIDTheAPILogTransactionID(t *testing.T) {
	ctx := requestcontext.WithTraceID(context.Background(), "trace-เชื่อมสองระบบ")

	req, err := utils.NewRequest(ctx, http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := apilog.TransactionIDFromContext(req.Context()); got != "trace-เชื่อมสองระบบ" {
		t.Fatalf("apilog transaction id = %q, ต้องการเลขเดียวกับ trace id คือ %q", got, "trace-เชื่อมสองระบบ")
	}
}

// ไม่มี trace id ใน context เลย ต้องไม่ปั้น transaction id ขึ้นมาเอง (ค่าว่าง = apilog ไม่แปะ
// X-Transaction-ID header ให้ ดู apilog/transport.go)
func TestNewRequestWithoutTraceIDLeavesAPILogTransactionIDEmpty(t *testing.T) {
	req, err := utils.NewRequest(context.Background(), http.MethodPost, "http://example.local", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := apilog.TransactionIDFromContext(req.Context()); got != "" {
		t.Fatalf("apilog transaction id = %q, ต้องการค่าว่างเมื่อไม่มี trace id", got)
	}
}
